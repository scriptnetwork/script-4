package core

import (
	"math/big"

	"github.com/scripttoken/script/common"
	"github.com/scripttoken/script/common/result"
)

type ViewSelector int

const (
	DeliveredView ViewSelector = 1
	CheckedView   ViewSelector = 2
	ScreenedView  ViewSelector = 3
)

// TxInfo encapsulates information used by mempool to sorting.
type TxInfo struct {
	EffectiveGasPrice *big.Int
	Address           common.Address
	Sequence          uint64
}

// Ledger defines the interface of the ledger
type Ledger interface {
	GetCurrentBlock() *Block
	ScreenTxUnsafe(rawTx common.Bytes) result.Result
	ScreenTx(rawTx common.Bytes) (priority *TxInfo, res result.Result)
	ProposeBlockTxs(block *Block, shouldIncludeValidatorUpdateTxs bool) (stateRootHash common.Hash, blockRawTxs []common.Bytes, res result.Result)
	ApplyBlockTxs(block *Block) result.Result
	ApplyBlockTxsForChainCorrection(block *Block) (common.Hash, result.Result)
	//ResetState(height uint64, rootHash common.Hash) result.Result
	ResetState(block *Block) result.Result
	FinalizeState(height uint64, rootHash common.Hash) result.Result
	GetFinalizedValidators(blockHash common.Hash, isNext bool) (*AddressSet, error)
	//GetFinalizedValidatorCandidatePool(blockHash common.Hash, isNext bool) (*ValidatorCandidatePool, error)
	GetLightnings(blockHash common.Hash) (*AddressSet, error)
	//GetEliteEdgeNodePoolOfLastCheckpoint(blockHash common.Hash) (EliteEdgeNodePool, error)
	PruneState(endHeight uint64) error
}
