package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/scripttoken/script/common"
	"github.com/scripttoken/script/core"
	"github.com/scripttoken/script/ledger/state"
	"github.com/scripttoken/script/ledger/types"
	"github.com/scripttoken/script/rlp"
	"github.com/scripttoken/script/store/database/backend"
	"github.com/scripttoken/script/store/trie"
)

var logger *log.Entry = log.WithFields(log.Fields{"prefix": "genesis"})

const (
	GenBlockHashMode int = iota
	GenGenesisFileMode
)

/*
type StakeDeposit struct {
	Source string `json:"source"`
	Holder string `json:"holder"`
	Amount string `json:"amount"`
}
*/

// Example:
// pushd $SCRIPT_HOME/integration/scriptnet/node
// generate_genesis -chainID=scriptnet -erc20snapshot=./data/genesis_script_erc20_snapshot.json -stake_deposit=./data/genesis_stake_deposit.json -genesis=./genesis
func main() {
	chainID, erc20SnapshotJSONFilePath, validatorsFilePath, hf_file, genesisSnapshotFilePath := parseArguments()

	{
		if hf_file == "" {
			panic("Empty HF file")
		}
		err := common.Initialize_hf_values(hf_file)
		if err != nil {
			panic(fmt.Sprintf("Failed to initialize HF values: %v", err))
		}
	}

	sv, metadata, err := generateGenesisSnapshot(chainID, erc20SnapshotJSONFilePath, validatorsFilePath)
	if err != nil {
		panic(fmt.Sprintf("Failed to generate genesis snapshot: %v", err))
	}

	err = sanityChecks(sv)
	if err != nil {
		panic(fmt.Sprintf("Sanity checks failed: %v", err))
	} else {
		logger.Infof("Sanity checks all passed.")
	}

	err = writeGenesisSnapshot(sv, metadata, genesisSnapshotFilePath)
	if err != nil {
		panic(fmt.Sprintf("Failed to write genesis snapshot: %v", err))
	}

	genesisBlockHeader := metadata.TailTrio.Second.Header
	genesisBlockHash := genesisBlockHeader.Hash()

	fmt.Println("")
	fmt.Printf("--------------------------------------------------------------------------\n")
	fmt.Printf("Genesis block hash: %v\n", genesisBlockHash.Hex())
	fmt.Printf("--------------------------------------------------------------------------\n")
	fmt.Println("")
	fmt.Printf("hf2=%v", common.Height_hf2)
}

func parseArguments() (chainID string, erc20SnapshotJSONFilePath string, validatorsFilePath string, hf_file string, genesisSnapshotFilePath string) {
	chainIDPtr := flag.String("chainID", "local_chain", "the ID of the chain")
	erc20SnapshotJSONFilePathPtr := flag.String("erc20snapshot", "./script_erc20_snapshot.json", "the json file contain the ERC20 balance snapshot")
	validatorsFilePathPtr := flag.String("validators", "./validators.json", "the initial validators")
	hf_filePtr := flag.String("hf_file", "", "Hard fork heights file")
	genesisSnapshotFilePathPtr := flag.String("genesis", "./genesis", "the genesis snapshot")
	flag.Parse()

	chainID = *chainIDPtr
	erc20SnapshotJSONFilePath = *erc20SnapshotJSONFilePathPtr
	validatorsFilePath = *validatorsFilePathPtr
	hf_file = *hf_filePtr
	genesisSnapshotFilePath = *genesisSnapshotFilePathPtr

	return
}

// generateGenesisSnapshot generates the genesis snapshot.
func generateGenesisSnapshot(chainID, erc20SnapshotJSONFilePath, validatorsFilePath string) (*state.StoreView, *core.SnapshotMetadata, error) {
	metadata := &core.SnapshotMetadata{}
	genesisHeight := core.GenesisBlockHeight

	sv := loadInitialBalances(erc20SnapshotJSONFilePath)
	initialValidators(validatorsFilePath, genesisHeight, sv)

	stateHash := sv.Hash()

	genesisBlock := core.NewBlock()
	genesisBlock.ChainID = chainID
	genesisBlock.Height = genesisHeight
	genesisBlock.Epoch = genesisBlock.Height
	genesisBlock.Parent = common.Hash{}
	genesisBlock.StateHash = stateHash
	genesisBlock.Timestamp = big.NewInt(time.Now().Unix())

	metadata.TailTrio = core.SnapshotBlockTrio{
		First:  core.SnapshotFirstBlock{},
		Second: core.SnapshotSecondBlock{Header: genesisBlock.BlockHeader},
		Third:  core.SnapshotThirdBlock{},
	}

	return sv, metadata, nil
}

func loadInitialBalances(erc20SnapshotJSONFilePath string) *state.StoreView {
	initSPAYToScriptRatio := new(big.Int).SetUint64(5)
	sv := state.NewStoreView(0, common.Hash{}, backend.NewMemDatabase())

	erc20SnapshotJSONFile, err := os.Open(erc20SnapshotJSONFilePath)
	if err != nil {
		panic(fmt.Sprintf("failed to open the ERC20 balance snapshot: %v", err))
	}
	defer erc20SnapshotJSONFile.Close()

	var erc20BalanceMap map[string]string
	erc20BalanceMapByteValue, err := ioutil.ReadAll(erc20SnapshotJSONFile)
	if err != nil {
		panic(fmt.Sprintf("failed to read the ERC20 balance snapshot: %v", err))
	}

	json.Unmarshal(erc20BalanceMapByteValue, &erc20BalanceMap)
	for key, val := range erc20BalanceMap {
		if !common.IsHexAddress(key) {
			panic(fmt.Sprintf("Invalid address: %v", key))
		}
		address := common.HexToAddress(key)

		script, success := new(big.Int).SetString(val, 10)
		if !success {
			panic(fmt.Sprintf("Failed to parse SCPTWei amount: %v", val))
		}
		spay := new(big.Int).Mul(initSPAYToScriptRatio, script)
		acc := &types.Account{
			Address:  address,
			Root:     common.Hash{},
			CodeHash: types.EmptyCodeHash,
			Balance: types.Coins{
				SCPTWei: script,
				SPAYWei: spay,
			},
		}
		sv.SetAccount(acc.Address, acc)
		//logger.Infof("address: %v, script: %v, spay: %v", strings.ToLower(address.String()), script, spay)
	}

	return sv
}

func initialValidators(validatorsFilePath string, genesisHeight uint64, sv *state.StoreView) *core.AddressSet {
	//	var stakeDeposits []StakeDeposit
	var addresses []string

	validatorsFile, err := os.Open(validatorsFilePath)
	validatorsByteValue, err := io.ReadAll(validatorsFile)
	if err != nil {
		panic(fmt.Sprintf("failed to read initial stake deposit file: %v", err))
	}

	json.Unmarshal(validatorsByteValue, &addresses)
	var validators core.AddressSet = make(core.AddressSet)

	for _, addr := range addresses {
		if !common.IsHexAddress(addr) {
			panic(fmt.Sprintf("Invalid address: %v", addr))
		}
		a := common.HexToAddress(addr)
		account := sv.GetAccount(a)
		if account == nil {
			panic(fmt.Sprintf("Failed to retrieve account for source address: %v", a))
		}
		validators[a] = struct{}{}
		sv.SetAccount(a, account)
	}

	sv.UpdateValidators(&validators)

	hl := &types.HeightList{}
	hl.Append(genesisHeight)
	sv.UpdateValidatorTransactionHeightList(hl)

	return &validators
}

func proveValidators(sv *state.StoreView) (*core.ValidatorsProof, error) {
	vp := &core.ValidatorsProof{}
	validatorsKey := state.ValidatorsKey()
	err := sv.ProveValidators(validatorsKey, vp)
	return vp, err
}

// writeGenesisSnapshot writes genesis snapshot to file system.
func writeGenesisSnapshot(sv *state.StoreView, metadata *core.SnapshotMetadata, genesisSnapshotFilePath string) error {
	file, err := os.Create(genesisSnapshotFilePath)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	err = core.WriteMetadata(writer, metadata)
	if err != nil {
		return err
	}
	writeStoreView(sv, true, writer)
	return err
}

func writeStoreView(sv *state.StoreView, needAccountStorage bool, writer *bufio.Writer) {
	height := core.Itobytes(sv.Height())
	err := core.WriteRecord(writer, []byte{core.SVStart}, height)
	if err != nil {
		panic(err)
	}
	sv.GetStore().Traverse(nil, func(k, v common.Bytes) bool {
		err = core.WriteRecord(writer, k, v)
		if err != nil {
			panic(err)
		}
		return true
	})
	err = core.WriteRecord(writer, []byte{core.SVEnd}, height)
	if err != nil {
		panic(err)
	}
	writer.Flush()
}

func sanityChecks(sv *state.StoreView) error {
	scriptWeiTotal := new(big.Int).SetUint64(0)
	spayWeiTotal := new(big.Int).SetUint64(0)

	validatorsAnalyzed := false
	sv.GetStore().Traverse(nil, func(key, val common.Bytes) bool {
		if bytes.Compare(key, state.ValidatorsKey()) == 0 {
			var validators core.AddressSet
			err := rlp.DecodeBytes(val, &validators)
			if err != nil {
				panic(fmt.Sprintf("Failed to decode validators: %v", err))
			}
			for _, sc := range validators {
				//logger.Infof("--------------------------------------------------------")
				logger.Infof("Validator Candidate: %v", sc)
				//for _, stake := range sc.Stakes {
				//	scriptWeiTotal = new(big.Int).Add(scriptWeiTotal, stake.Amount)
				//	logger.Infof("     Stake: source = %v, stakeAmount = %v", stake.Source, stake.Amount)
				//}
				//logger.Infof("--------------------------------------------------------")
			}
			validatorsAnalyzed = true
		} else if bytes.Compare(key, state.ValidatorTransactionHeightListKey()) == 0 {
			var hl types.HeightList
			err := rlp.DecodeBytes(val, &hl)
			if err != nil {
				panic(fmt.Sprintf("Failed to decode Height List: %v", err))
			}
			if len(hl.Heights) != 1 {
				panic(fmt.Sprintf("The genesis height list should contain only one height: %v", hl.Heights))
			}
			if hl.Heights[0] != uint64(0) {
				panic(fmt.Sprintf("Only height 0 should be in the genesis height list"))
			}
		} else { // regular account
			var account types.Account
			err := rlp.DecodeBytes(val, &account)
			if err != nil {
				panic(fmt.Sprintf("Failed to decode Account: %v", err))
			}

			scriptWei := account.Balance.SCPTWei
			spayWei := account.Balance.SPAYWei
			scriptWeiTotal = new(big.Int).Add(scriptWeiTotal, scriptWei)
			spayWeiTotal = new(big.Int).Add(spayWeiTotal, spayWei)

			logger.Infof("Account: %v, SCPTWei = %v, SPAYWei = %v", account.Address, scriptWei, spayWei)
		}
		return true
	})

	// Check #1: VCP analyzed
	validatorsProof, err := proveValidators(sv)
	if err != nil {
		panic(fmt.Sprintf("Failed to get Validators proof from storeview"))
	}
	_, _, err = trie.VerifyProof(sv.Hash(), state.ValidatorsKey(), validatorsProof)
	if err != nil {
		panic(fmt.Sprintf("Failed to verify VCP proof in storeview"))
	}
	if !validatorsAnalyzed {
		return fmt.Errorf("VCP not detected in the genesis file")
	}

	// Check #2: Sum(SCPTWei) + Sum(Stake) == 1 * 10^9 * 10^18
	oneBillion := new(big.Int).SetUint64(1000000000)
	fiveBillion := new(big.Int).Mul(new(big.Int).SetUint64(5), oneBillion)
	ten18 := new(big.Int).SetUint64(1000000000000000000)

	expectedSCPTWeiTotal := new(big.Int).Mul(oneBillion, ten18)
	if expectedSCPTWeiTotal.Cmp(scriptWeiTotal) != 0 {
		return fmt.Errorf("Unmatched SCPTWei total: expected = %v, calculated = %v", expectedSCPTWeiTotal, scriptWeiTotal)
	}
	logger.Infof("Expected   SCPTWei total = %v", expectedSCPTWeiTotal)
	logger.Infof("Calculated SCPTWei total = %v", scriptWeiTotal)

	// Check #3: Sum(SPAYWei) == 5 * 10^9 * 10^18
	expectedSPAYWeiTotal := new(big.Int).Mul(fiveBillion, ten18)
	if expectedSPAYWeiTotal.Cmp(spayWeiTotal) != 0 {
		return fmt.Errorf("Unmatched SPAYWei total: expected = %v, calculated = %v", expectedSPAYWeiTotal, spayWeiTotal)
	}
	logger.Infof("Expected   SPAYWei total = %v", expectedSPAYWeiTotal)
	logger.Infof("Calculated SPAYWei total = %v", spayWeiTotal)

	return nil
}
