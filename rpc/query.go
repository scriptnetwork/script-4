package rpc

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/scripttoken/script/blockchain"
	"github.com/scripttoken/script/crypto/bls"

	"github.com/scripttoken/script/common"
	"github.com/scripttoken/script/core"
	"github.com/scripttoken/script/crypto"
	"github.com/scripttoken/script/ledger/state"
	"github.com/scripttoken/script/ledger/types"
	"github.com/scripttoken/script/mempool"
	"github.com/scripttoken/script/version"
)

// ------------------------------- GetVersion -----------------------------------

type GetVersionArgs struct {
}

type GetVersionResult struct {
	Version   string `json:"version"`
	GitHash   string `json:"git_hash"`
	Timestamp string `json:"timestamp"`
}

func (t *ScriptRPCService) GetVersion(args *GetVersionArgs, result *GetVersionResult) (err error) {
	result.Version = version.Version
	result.GitHash = version.GitHash
	result.Timestamp = version.Timestamp
	return nil
}

// ------------------------------- GetAccount -----------------------------------

type GetAccountArgs struct {
	Name    string            `json:"name"`
	Address string            `json:"address"`
	Height  common.JSONUint64 `json:"height"`
	Preview bool              `json:"preview"` // preview the account balance from the ScreenedView
}

type GetAccountResult struct {
	*types.Account
	Address string `json:"address"`
}

func (t *ScriptRPCService) GetAccount(args *GetAccountArgs, result *GetAccountResult) (err error) {
	if args.Address == "" {
		return errors.New("Address must be specified")
	}
	address := common.HexToAddress(args.Address)
	result.Address = args.Address
	height := uint64(args.Height)

	if height == 0 { // get the latest
		var ledgerState *state.StoreView
		if args.Preview {
			ledgerState, err = t.ledger.GetScreenedSnapshot()
		} else {
			ledgerState, err = t.ledger.GetFinalizedSnapshot()
		}
		if err != nil {
			return err
		}

		account := ledgerState.GetAccount(address)
		if account == nil {
			return fmt.Errorf("Account with address %s is not found", address.Hex())
		}
		account.UpdateToHeight(ledgerState.Height())

		result.Account = account
	} else {
		blocks := t.chain.FindBlocksByHeight(height)
		if len(blocks) == 0 {
			result.Account = nil
			return fmt.Errorf("Historical data at given height is not available on current node")
		}

		deliveredView, err := t.ledger.GetDeliveredSnapshot()
		if err != nil {
			return err
		}
		db := deliveredView.GetDB()

		for _, b := range blocks {
			if b.Status.IsFinalized() {
				stateRoot := b.StateHash
				ledgerState := state.NewStoreView(height, stateRoot, db)
				if ledgerState == nil { // might have been pruned
					return fmt.Errorf("the account details for height %v is not available, it might have been pruned", height)
				}
				account := ledgerState.GetAccount(address)
				if account == nil {
					return fmt.Errorf("Account with address %v is not found", address.Hex())
				}
				result.Account = account
				break
			}
		}

	}

	if result.Account == nil {
		log.Debugf("Account with address %v at height %v is not found", address.Hex(), height)
		return fmt.Errorf("Account with address %v at height %v is not found", address.Hex(), height)
	}

	return nil
}

// ------------------------------ GetTransaction -----------------------------------

type GetTransactionArgs struct {
	Hash string `json:"hash"`
}

type GetTransactionResult struct {
	BlockHash      common.Hash                       `json:"block_hash"`
	BlockHeight    common.JSONUint64                 `json:"block_height"`
	Status         TxStatus                          `json:"status"`
	TxHash         common.Hash                       `json:"hash"`
	Type           byte                              `json:"type"`
	Tx             types.Tx                          `json:"transaction"`
	Receipt        *blockchain.TxReceiptEntry        `json:"receipt"`
	BalanceChanges *blockchain.TxBalanceChangesEntry `json:"blance_changes"`
}

type TxStatus string

const (
	TxStatusNotFound  = "not_found"
	TxStatusPending   = "pending"
	TxStatusFinalized = "finalized"
	TxStatusAbandoned = "abandoned"
)

func (t *ScriptRPCService) GetTransaction(args *GetTransactionArgs, result *GetTransactionResult) (err error) {
	if args.Hash == "" {
		return errors.New("Transanction hash must be specified")
	}
	hash := common.HexToHash(args.Hash)

	raw, block, found := t.chain.FindTxByHash(hash)
	if !found {
		txStatus, exists := t.mempool.GetTransactionStatus(args.Hash)
		if exists {
			if txStatus == mempool.TxStatusAbandoned {
				result.Status = TxStatusAbandoned
			} else {
				result.Status = TxStatusPending
			}
		} else {
			result.Status = TxStatusNotFound
		}
		return nil
	}
	result.BlockHash = block.Hash()
	result.BlockHeight = common.JSONUint64(block.Height)

	if block.Status.IsFinalized() {
		result.Status = TxStatusFinalized
	} else {
		result.Status = TxStatusPending
	}

	tx, err := types.TxFromBytes(raw)
	if err != nil {
		return err
	}
	result.Tx = tx
	result.Type = getTxType(tx)

	// args.Hash maybe an ETH tx hash, need to lookup the receipt using the hash of the corresponding native Smart contract Tx
	canonicalTxHash := hash
	if result.Type == TxTypeSmartContract {
		canonicalTxHash = crypto.Keccak256Hash(raw)
	}
	result.TxHash = canonicalTxHash

	// Add receipt
	blockHash := block.Hash()
	receipt, found := t.chain.FindTxReceiptByHash(blockHash, canonicalTxHash)
	if found {
		result.Receipt = receipt
	}
	balanceChanges, found := t.chain.FindTxBalanceChangesByHash(blockHash, canonicalTxHash)
	if found {
		result.BalanceChanges = balanceChanges
	}

	return nil
}

// ------------------------------ GetPendingTransactions -----------------------------------

type GetPendingTransactionsArgs struct {
}

type GetPendingTransactionsResult struct {
	TxHashes []string `json:"tx_hashes"`
}

func (t *ScriptRPCService) GetPendingTransactions(args *GetPendingTransactionsArgs, result *GetPendingTransactionsResult) (err error) {
	pendingTxHashes := t.mempool.GetCandidateTransactionHashes()
	result.TxHashes = pendingTxHashes
	return nil
}

// ------------------------------ TraceBlocks -----------------------------------

type TraceBlocksArgs struct {
	Hash     common.Hash `json:"hash"`
	Upstream bool        `json:"upstream"`
}

type TraceBlocksResults struct {
	Blocks []common.Hash
}

func (t *ScriptRPCService) TraceBlocks(args *TraceBlocksArgs, result *TraceBlocksResults) (err error) {
	if args.Hash.IsEmpty() {
		return errors.New("Block hash must be specified")
	}

	MaxNumBlocks := 10000 // to prevent abuse

	block, err := t.chain.FindBlock(args.Hash)
	if err != nil {
		return err
	}

	blockHashes := []common.Hash{}
	blockHashes = append(blockHashes, args.Hash)
	numBlocks := 1
	if args.Upstream {
		for {
			parentHash := block.Parent
			parentBlock, err := t.chain.FindBlock(parentHash)
			if err != nil {
				return err
			}
			blockHashes = append(blockHashes, parentHash)
			numBlocks += 1
			if numBlocks > MaxNumBlocks {
				blockHashes = append(blockHashes, common.Hash{0x99}) // special marker
				break
			}
			if parentBlock.Status == core.BlockStatusDirectlyFinalized || parentBlock.Status == core.BlockStatusIndirectlyFinalized {
				break // stop at a finalized block
			}
			block = parentBlock
		}
	} else {
		for {
			childrenHashes := block.Children
			numBlocks += 1
			if numBlocks > MaxNumBlocks {
				blockHashes = append(blockHashes, common.Hash{0x99}) // special marker
				break
			}
			if len(childrenHashes) == 2 {
				blockHashes = append(blockHashes, common.Hash{0x2}) // special marker
				break
			} else if len(childrenHashes) == 1 {
				childHash := childrenHashes[0]
				childBlock, err := t.chain.FindBlock(childHash)
				if err != nil {
					blockHashes = append(blockHashes, common.Hash{0x1}) // special marker
					break
				}
				blockHashes = append(blockHashes, childHash)
				block = childBlock
			} else if len(childrenHashes) == 0 {
				blockHashes = append(blockHashes, common.Hash{0x0}) // special marker
				break
			}
		}
	}

	result.Blocks = blockHashes
	return nil
}

// ------------------------------ GetBlock -----------------------------------

type GetBlockArgs struct {
	Hash               common.Hash `json:"hash"`
	IncludeEthTxHashes bool        `json:"include_eth_tx_hashes"`
}

type Tx struct {
	types.Tx       `json:"raw"`
	Type           byte                              `json:"type"`
	Hash           common.Hash                       `json:"hash"`
	Receipt        *blockchain.TxReceiptEntry        `json:"receipt"`
	BalanceChanges *blockchain.TxBalanceChangesEntry `json:"balance_changes"`
}

type TxWithEthHash struct {
	types.Tx       `json:"raw"`
	Type           byte                              `json:"type"`
	Hash           common.Hash                       `json:"hash"`
	EthTxHash      common.Hash                       `json:"eth_tx_hash"`
	Receipt        *blockchain.TxReceiptEntry        `json:"receipt"`
	BalanceChanges *blockchain.TxBalanceChangesEntry `json:"balance_changes"`
}

type GetBlockResult struct {
	*GetBlockResultInner
}

type GetBlocksResult []*GetBlockResultInner

type GetBlockResultInner struct {
	ChainID        string                 `json:"chain_id"`
	Epoch          common.JSONUint64      `json:"epoch"`
	Height         common.JSONUint64      `json:"height"`
	Parent         common.Hash            `json:"parent"`
	TxHash         common.Hash            `json:"transactions_hash"`
	StateHash      common.Hash            `json:"state_hash"`
	Timestamp      *common.JSONBig        `json:"timestamp"`
	Proposer       common.Address         `json:"proposer"`
	HCC            core.CommitCertificate `json:"hcc"`
	LightningVotes *core.AggregatedVotes  `json:"lightning_votes"`

	Children []common.Hash    `json:"children"`
	Status   core.BlockStatus `json:"status"`

	Hash common.Hash   `json:"hash"`
	Txs  []interface{} `json:"transactions"` // for backward conpatibility, see function ScriptRPCService.gatherTxs()
}

type TxType byte

const (
	TxTypeCoinbase = byte(iota)
	TxTypeSlash
	TxTypeSend
	TxTypeReserveFund
	TxTypeReleaseFund
	TxTypeServicePayment
	TxTypeSmartContract
)

func (t *ScriptRPCService) GetBlock(args *GetBlockArgs, result *GetBlockResult) (err error) {
	if args.Hash.IsEmpty() {
		return errors.New("Block hash must be specified")
	}

	block, err := t.chain.FindBlock(args.Hash)
	if err != nil {
		return err
	}

	result.GetBlockResultInner = &GetBlockResultInner{}
	result.ChainID = block.ChainID
	result.Epoch = common.JSONUint64(block.Epoch)
	result.Height = common.JSONUint64(block.Height)
	result.Parent = block.Parent
	result.TxHash = block.TxHash
	result.StateHash = block.StateHash
	result.Timestamp = (*common.JSONBig)(block.Timestamp)
	result.Proposer = block.Proposer
	result.Children = block.Children
	result.Status = block.Status
	result.HCC = block.HCC
	result.LightningVotes = block.LightningVotes

	result.Hash = block.Hash()

	t.gatherTxs(block, &result.Txs, args.IncludeEthTxHashes)

	return
}

// ------------------------------ GetBlockByHeight -----------------------------------

type GetBlockByHeightArgs struct {
	Height                   common.JSONUint64 `json:"height"`
	IncludeEthTxHashes       bool              `json:"include_eth_tx_hashes"`
	IncludeUnfinalizedBlocks bool              `json:"include_unfinalized_blocks"`
}

func (t *ScriptRPCService) GetBlockByHeight(args *GetBlockByHeightArgs, result *GetBlockResult) (err error) {
	// if args.Height == 0 {
	// 	return errors.New("Block height must be specified")
	// }

	blockHeight := uint64(args.Height)
	blocks := t.chain.FindBlocksByHeight(blockHeight)

	var block *core.ExtendedBlock
	for _, b := range blocks {
		if args.IncludeUnfinalizedBlocks {
			block = b
		} else {
			if b.Status.IsFinalized() {
				block = b
				break
			}
		}
	}

	if blockHeight == 0 && block == nil { // special handling for a node starting from a non-genesis snapshot
		var genesisHash common.Hash
		//if t.consensus.Chain().ChainID == core.MainnetChainID {
		//	genesisHash = common.HexToHash(core.MainnetGenesisBlockHash)
		//} else {
		genesisHash = common.HexToHash(viper.GetString(common.CfgGenesisHash))
		//}

		result.GetBlockResultInner = &GetBlockResultInner{}
		result.ChainID = t.consensus.Chain().ChainID
		result.Children = []common.Hash{}
		result.Status = core.BlockStatusDirectlyFinalized
		result.Timestamp = (*common.JSONBig)(big.NewInt(0))
		result.Hash = genesisHash
		return
	}

	if block == nil {
		return
	}

	result.GetBlockResultInner = &GetBlockResultInner{}
	result.ChainID = block.ChainID
	result.Epoch = common.JSONUint64(block.Epoch)
	result.Height = common.JSONUint64(block.Height)
	result.Parent = block.Parent
	result.TxHash = block.TxHash
	result.StateHash = block.StateHash
	result.Timestamp = (*common.JSONBig)(block.Timestamp)
	result.Proposer = block.Proposer
	result.Children = block.Children
	result.Status = block.Status
	result.HCC = block.HCC
	result.LightningVotes = block.LightningVotes

	result.Hash = block.Hash()

	t.gatherTxs(block, &result.Txs, args.IncludeEthTxHashes)

	return
}

// ------------------------------ GetBlocksByRange -----------------------------------

type GetBlocksByRangeArgs struct {
	Start              common.JSONUint64 `json:"start"`
	End                common.JSONUint64 `json:"end"`
	IncludeEthTxHashes bool              `json:"include_eth_tx_hashes"`
}

func (t *ScriptRPCService) GetBlocksByRange(args *GetBlocksByRangeArgs, result *GetBlocksResult) (err error) {
	// if args.Start == 0 && args.End == 0 {
	// 	return errors.New("Starting block and ending block must be specified")
	// }
	genesisBlock := &GetBlockResultInner{}
	var genesisHash common.Hash
	//if t.consensus.Chain().ChainID == core.MainnetChainID {
	//	genesisHash = common.HexToHash(core.MainnetGenesisBlockHash)
	//} else {
	genesisHash = common.HexToHash(viper.GetString(common.CfgGenesisHash))
	//}
	genesisBlock.ChainID = t.consensus.Chain().ChainID
	genesisBlock.Children = []common.Hash{}
	genesisBlock.Status = core.BlockStatusDirectlyFinalized
	genesisBlock.Timestamp = (*common.JSONBig)(big.NewInt(0))
	genesisBlock.Hash = genesisHash

	if args.End == 0 {
		*result = append([]*GetBlockResultInner{genesisBlock}, *result...)
		return
	}

	if args.Start > args.End {
		return errors.New("starting block must be less than ending block")
	}

	maxBlockRange := common.JSONUint64(5000)
	blockStart := args.Start
	blockEnd := args.End
	queryBlockRange := blockEnd - blockStart
	if queryBlockRange > maxBlockRange {
		return fmt.Errorf("can't retrieve more than %v blocks at a time", maxBlockRange)
	}

	heavyQueryThreshold := viper.GetUint64(common.CfgRPCGetBlocksHeavyQueryThreshold)
	isHeavyQuery := uint64(queryBlockRange) > heavyQueryThreshold
	if isHeavyQuery {
		t.pendingHeavyGetBlocksCounterLock.Lock()
		hasTooManyPendingHeavyQueries := t.pendingHeavyGetBlocksCounter > viper.GetUint64(common.CfgRPCMaxHeavyGetBlocksQueryCount)
		t.pendingHeavyGetBlocksCounterLock.Unlock()

		if hasTooManyPendingHeavyQueries {
			warningMsg := fmt.Sprintf("too many pending heavy getBlocksByRange queries, rejecting getBlocksByRange query from block %v to %v", blockStart, blockEnd)
			logger.Warnf(warningMsg)
			return fmt.Errorf(warningMsg)
		}

		t.pendingHeavyGetBlocksCounterLock.Lock()
		t.pendingHeavyGetBlocksCounter += 1
		t.pendingHeavyGetBlocksCounterLock.Unlock()

		defer func() { // the heavy query has now been processed
			t.pendingHeavyGetBlocksCounterLock.Lock()
			if t.pendingHeavyGetBlocksCounter > 0 {
				t.pendingHeavyGetBlocksCounter -= 1
			}
			t.pendingHeavyGetBlocksCounterLock.Unlock()
		}()
	}

	blocks := t.chain.FindBlocksByHeight(uint64(args.End))

	var block *core.ExtendedBlock
	for _, b := range blocks {
		if b.Status.IsFinalized() {
			block = b
			break
		}
	}

	if block == nil {
		return
	}

	startBlockHeight := args.Start
	if args.Start == 0 {
		startBlockHeight = 1 // genesis block needs special handling
	}
	for common.JSONUint64(block.Height) >= startBlockHeight {
		blkInner := &GetBlockResultInner{}
		blkInner.ChainID = block.ChainID
		blkInner.Epoch = common.JSONUint64(block.Epoch)
		blkInner.Height = common.JSONUint64(block.Height)
		blkInner.Parent = block.Parent
		blkInner.TxHash = block.TxHash
		blkInner.StateHash = block.StateHash
		blkInner.Timestamp = (*common.JSONBig)(block.Timestamp)
		blkInner.Proposer = block.Proposer
		blkInner.Children = block.Children
		blkInner.Status = block.Status
		blkInner.HCC = block.HCC
		blkInner.LightningVotes = block.LightningVotes

		blkInner.Hash = block.Hash()

		t.gatherTxs(block, &blkInner.Txs, args.IncludeEthTxHashes)

		*result = append([]*GetBlockResultInner{blkInner}, *result...)

		block, err = t.chain.FindBlock(block.Parent)
		if err != nil {
			return err
		}
	}
	if args.Start == 0 {
		*result = append([]*GetBlockResultInner{genesisBlock}, *result...)
	}

	return
}

// ------------------------------ GetStatus -----------------------------------

type GetStatusArgs struct{}

type GetStatusResult struct {
	Address                    string            `json:"address"`
	ChainID                    string            `json:"chain_id"`
	EthChainID                 int64             `json:"eth_chain_id"`
	PeerID                     string            `json:"peer_id"`
	LatestFinalizedBlockHash   common.Hash       `json:"latest_finalized_block_hash"`
	LatestFinalizedBlockHeight common.JSONUint64 `json:"latest_finalized_block_height"`
	LatestFinalizedBlockTime   *common.JSONBig   `json:"latest_finalized_block_time"`
	LatestFinalizedBlockEpoch  common.JSONUint64 `json:"latest_finalized_block_epoch"`
	CurrentEpoch               common.JSONUint64 `json:"current_epoch"`
	CurrentHeight              common.JSONUint64 `json:"current_height"`
	CurrentHash                common.Hash       `json:"current_hash"`
	CurrentTime                *common.JSONBig   `json:"current_time"`
	Syncing                    bool              `json:"syncing"`
	TipHash                    common.Hash       `json:"tip_hash"`
	GenesisBlockHash           common.Hash       `json:"genesis_block_hash"`
	SnapshotBlockHeight        common.JSONUint64 `json:"snapshot_block_height"`
	SnapshotBlockHash          common.Hash       `json:"snapshot_block_hash"`
}

func (t *ScriptRPCService) GetStatus(args *GetStatusArgs, result *GetStatusResult) (err error) {
	s := t.consensus.GetSummary()
	result.Address = t.consensus.ID()
	//result.PeerID = t.dispatcher.ID()
	result.PeerID = t.dispatcher.LibP2PID() // TODO: use ID() instead after 1.3.0 upgrade
	result.ChainID = t.consensus.Chain().ChainID
	result.EthChainID = int64(viper.GetUint64(common.CfgGenesisEthChainID))
	latestFinalizedHash := s.LastFinalizedBlock
	var latestFinalizedBlock *core.ExtendedBlock
	if !latestFinalizedHash.IsEmpty() {
		result.LatestFinalizedBlockHash = latestFinalizedHash
		latestFinalizedBlock, err = t.chain.FindBlock(latestFinalizedHash)
		if err != nil {
			return err
		}
		result.LatestFinalizedBlockEpoch = common.JSONUint64(latestFinalizedBlock.Epoch)
		result.LatestFinalizedBlockHeight = common.JSONUint64(latestFinalizedBlock.Height)
		result.LatestFinalizedBlockTime = (*common.JSONBig)(latestFinalizedBlock.Timestamp)
	}
	result.CurrentEpoch = common.JSONUint64(s.Epoch)
	result.CurrentTime = (*common.JSONBig)(big.NewInt(time.Now().Unix()))

	maxVoteHeight := uint64(0)
	epochVotes, err := t.consensus.State().GetEpochVotes()
	if err != nil {
		return err
	}
	if epochVotes != nil {
		for _, v := range epochVotes.Votes() {
			if v.Height > maxVoteHeight {
				maxVoteHeight = v.Height
				result.CurrentHash = v.Block
			}
		}
		result.CurrentHeight = common.JSONUint64(maxVoteHeight - 1) // current finalized height is at most maxVoteHeight-1
	}

	result.Syncing = !t.consensus.HasSynced()
	tipBlock := t.consensus.GetTip(true)
	if tipBlock != nil && tipBlock.BlockHeader != nil {
		result.TipHash = tipBlock.Hash()
	}

	var genesisHash common.Hash
	//if t.consensus.Chain().ChainID == core.MainnetChainID {
	//	genesisHash = common.HexToHash(core.MainnetGenesisBlockHash)
	//} else {
	genesisHash = common.HexToHash(viper.GetString(common.CfgGenesisHash))
	//}
	result.GenesisBlockHash = genesisHash
	result.SnapshotBlockHeight = common.JSONUint64(t.chain.Root().Block.BlockHeader.Height)
	result.SnapshotBlockHash = t.chain.Root().Block.BlockHeader.Hash()

	return
}

// ------------------------------ GetPeerURLs -----------------------------------

type GetPeerURLsArgs struct {
	SkipEdgeNode bool `json:"skip_edge_node"`
}

type GetPeerURLsResult struct {
	PeerURLs []string `json:"peer_urls"`
}

func (t *ScriptRPCService) GetPeerURLs(args *GetPeersArgs, result *GetPeerURLsResult) (err error) {
	peerURLs := t.dispatcher.PeerURLs()

	numPeers := len(peerURLs)
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(numPeers, func(i, j int) { peerURLs[i], peerURLs[j] = peerURLs[j], peerURLs[i] })

	maxNumOfPeers := 256
	if len(peerURLs) < maxNumOfPeers {
		maxNumOfPeers = len(peerURLs)
	}
	result.PeerURLs = peerURLs[0:maxNumOfPeers]

	return
}

// ------------------------------ GetPeers -----------------------------------

type GetPeersArgs struct {
}

type GetPeersResult struct {
	Peers []string `json:"peers"`
}

func (t *ScriptRPCService) GetPeers(args *GetPeersArgs, result *GetPeersResult) (err error) {
	peers := t.dispatcher.Peers()
	result.Peers = peers

	return
}

// ------------------------------ GetVcp -----------------------------------

type GetValidatorsByHeightArgs struct {
	Height common.JSONUint64 `json:"height"`
}

type GetValidatorsResult struct {
	BlockHashValidatorPairs []BlockHashValidatorPair
}

type BlockHashValidatorPair struct {
	BlockHash  common.Hash
	Validators *core.AddressSet
	HeightList *types.HeightList
}

func (t *ScriptRPCService) GetValidatorsByHeight(args *GetValidatorsByHeightArgs, result *GetValidatorsResult) (err error) {
	deliveredView, err := t.ledger.GetDeliveredSnapshot()
	if err != nil {
		return err
	}

	db := deliveredView.GetDB()
	height := uint64(args.Height)

	blockHashValidatorPairs := []BlockHashValidatorPair{}
	blocks := t.chain.FindBlocksByHeight(height)
	for _, b := range blocks {
		blockHash := b.Hash()
		stateRoot := b.StateHash
		blockStoreView := state.NewStoreView(height, stateRoot, db)
		if blockStoreView == nil { // might have been pruned
			return fmt.Errorf("the VCP for height %v does not exists, it might have been pruned", height)
		}
		validators := blockStoreView.GetValidators()
		hl := blockStoreView.GetValidatorTransactionHeightList()
		blockHashValidatorPairs = append(blockHashValidatorPairs, BlockHashValidatorPair{
			BlockHash:  blockHash,
			Validators: validators,
			HeightList: hl,
		})
	}

	result.BlockHashValidatorPairs = blockHashValidatorPairs
	return nil
}

// ------------------------------ GetLightnings -----------------------------------

type GetLightningsByHeightArgs struct {
	Height common.JSONUint64 `json:"height"`
}

type GetLightningsResult struct {
	BlockHashLightningsPairs []BlockHashLightningsPair
}

type BlockHashLightningsPair struct {
	BlockHash  common.Hash
	Lightnings *core.AddressSet
}

func (t *ScriptRPCService) GetLightningsByHeight(args *GetLightningsByHeightArgs, result *GetLightningsResult) (err error) {
	deliveredView, err := t.ledger.GetDeliveredSnapshot()
	if err != nil {
		return err
	}

	db := deliveredView.GetDB()
	height := uint64(args.Height)

	blockHashLightningsPairs := []BlockHashLightningsPair{}
	blocks := t.chain.FindBlocksByHeight(height)
	for _, b := range blocks {
		blockHash := b.Hash()
		stateRoot := b.StateHash
		blockStoreView := state.NewStoreView(height, stateRoot, db)
		if blockStoreView == nil { // might have been pruned
			return fmt.Errorf("the GCP for height %v does not exists, it might have been pruned", height)
		}
		lightnings := blockStoreView.GetLightnings()
		blockHashLightningsPairs = append(blockHashLightningsPairs, BlockHashLightningsPair{
			BlockHash:  blockHash,
			Lightnings: lightnings,
		})
	}

	result.BlockHashLightningsPairs = blockHashLightningsPairs

	return nil
}

// ------------------------------ GetLightningKey -----------------------------------

type GetLightningInfoArgs struct{}

type GetLightningInfoResult struct {
	BLSPubkey string
	BLSPop    string
	Address   string
	Signature string
}

func (t *ScriptRPCService) GetLightningInfo(args *GetLightningInfoArgs, result *GetLightningInfoResult) (err error) {
	privKey := t.consensus.PrivateKey()
	//	blsKey, err := bls.GenKey(strings.NewReader(common.Bytes2Hex(privKey.PublicKey().ToBytes())))   //Script version
	blsKey, err := bls.GenKey(strings.NewReader(common.Bytes2Hex(privKey.ToBytes()))) //Using unpredictable seed, keeping random oracle properties
	if err != nil {
		return fmt.Errorf("Failed to get BLS key: %v", err.Error())
	}

	result.Address = privKey.PublicKey().Address().Hex()
	result.BLSPubkey = hex.EncodeToString(blsKey.PublicKey().ToBytes())
	popBytes := blsKey.PopProve().ToBytes()
	result.BLSPop = hex.EncodeToString(popBytes)

	sig, err := privKey.Sign(popBytes)
	if err != nil {
		return fmt.Errorf("Failed to generate signature: %v", err.Error())
	}
	result.Signature = hex.EncodeToString(sig.ToBytes())

	return nil
}

// ------------------------------- GetCode -----------------------------------

type GetCodeArgs struct {
	Address string            `json:"address"`
	Height  common.JSONUint64 `json:"height"`
}

type GetCodeResult struct {
	Address string `json:"address"`
	Code    string `json:"code"`
}

func (t *ScriptRPCService) GetCode(args *GetCodeArgs, result *GetCodeResult) (err error) {
	if args.Address == "" {
		return errors.New("address must be specified")
	}
	address := common.HexToAddress(args.Address)
	result.Address = args.Address
	height := uint64(args.Height)

	if height == 0 { // get the latest
		var ledgerState *state.StoreView
		ledgerState, err = t.ledger.GetFinalizedSnapshot()
		if err != nil {
			return err
		}
		codeBytes := ledgerState.GetCode(address)
		result.Code = hex.EncodeToString(codeBytes)
	} else {
		blocks := t.chain.FindBlocksByHeight(height)
		if len(blocks) == 0 {
			result.Code = ""
			return nil
		}

		// deliveredView, err := t.ledger.GetDeliveredSnapshot()
		view, err := t.ledger.GetScreenedSnapshot()
		if err != nil {
			return err
		}
		db := view.GetDB()

		for _, b := range blocks {
			if b.Status.IsFinalized() {
				stateRoot := b.StateHash
				ledgerState := state.NewStoreView(height, stateRoot, db)
				if ledgerState == nil { // might have been pruned
					return fmt.Errorf("the account details for height %v is not available, it might have been pruned", height)
				}
				codeBytes := ledgerState.GetCode(address)
				result.Code = hex.EncodeToString(codeBytes)
				break
			}
		}

	}

	return nil
}

// ------------------------------- GetStorageAt -----------------------------------

type GetStorageAtArgs struct {
	Address         string            `json:"address"`
	StoragePosition string            `json:"storage_positon"`
	Height          common.JSONUint64 `json:"height"`
}

type GetStorageAtResult struct {
	Value string `json:"value"`
}

func (t *ScriptRPCService) GetStorageAt(args *GetStorageAtArgs, result *GetStorageAtResult) (err error) {
	if args.Address == "" || args.StoragePosition == "" {
		return fmt.Errorf("address and storage_position must be specified, address: %v, storage_position: %v", args.Address, args.StoragePosition)
	}
	address := common.HexToAddress(args.Address)
	key := common.HexToHash(args.StoragePosition)
	height := uint64(args.Height)

	if height == 0 { // get the latest
		var ledgerState *state.StoreView
		ledgerState, err = t.ledger.GetFinalizedSnapshot()
		if err != nil {
			return err
		}
		value := ledgerState.GetState(address, key)
		result.Value = hex.EncodeToString(value.Bytes())
	} else {
		blocks := t.chain.FindBlocksByHeight(height)
		if len(blocks) == 0 {
			result.Value = ""
			return nil
		}

		deliveredView, err := t.ledger.GetDeliveredSnapshot()
		if err != nil {
			return err
		}
		db := deliveredView.GetDB()

		for _, b := range blocks {
			if b.Status.IsFinalized() {
				stateRoot := b.StateHash
				ledgerState := state.NewStoreView(height, stateRoot, db)
				if ledgerState == nil { // might have been pruned
					return fmt.Errorf("the account details for height %v is not available, it might have been pruned", height)
				}
				value := ledgerState.GetState(address, key)
				result.Value = hex.EncodeToString(value.Bytes())
				break
			}
		}
	}

	return nil
}

// ------------------------------ Utils ------------------------------

func (t *ScriptRPCService) gatherTxs(block *core.ExtendedBlock, txs *[]interface{}, includeEthTxHashes bool) error {
	// Parse and fulfill Txs.
	//var tx types.Tx
	for _, txBytes := range block.Txs {
		tx, err := types.TxFromBytes(txBytes)
		if err != nil {
			return err
		}
		hash := crypto.Keccak256Hash(txBytes)
		blockHash := block.Hash()
		receipt, found := t.chain.FindTxReceiptByHash(blockHash, hash)
		if !found {
			receipt = nil
		}
		balanceChanges, found := t.chain.FindTxBalanceChangesByHash(blockHash, hash)
		if !found {
			balanceChanges = nil
		}

		tp := getTxType(tx)

		var txw interface{}
		if !includeEthTxHashes { // For backward compatibility, return the same tx struct as before
			txw = Tx{
				Tx:             tx,
				Hash:           hash,
				Type:           tp,
				Receipt:        receipt,
				BalanceChanges: balanceChanges,
			}
		} else {
			ethTxHash, _ := blockchain.CalcEthTxHash(block, txBytes) // ignore error, since ethTxHash will be 0x000...000 if the function returns an error
			txw = TxWithEthHash{
				Tx:             tx,
				Hash:           hash,
				EthTxHash:      ethTxHash,
				Type:           tp,
				Receipt:        receipt,
				BalanceChanges: balanceChanges,
			}
		}

		*txs = append(*txs, txw)
	}

	return nil
}

func getTxType(tx types.Tx) byte {
	t := byte(0x0)
	switch tx.(type) {
	case *types.CoinbaseTx:
		t = TxTypeCoinbase
	case *types.SlashTx:
		t = TxTypeSlash
	case *types.SendTx:
		t = TxTypeSend
	case *types.ReserveFundTx:
		t = TxTypeReserveFund
	case *types.ReleaseFundTx:
		t = TxTypeReleaseFund
	case *types.ServicePaymentTx:
		t = TxTypeServicePayment
	case *types.SmartContractTx:
		t = TxTypeSmartContract
	}

	return t
}
