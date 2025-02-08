package core

import (
	"github.com/scripttoken/script/common"
	"github.com/scripttoken/script/crypto"
)

// ConsensusEngine is the interface of a consensus engine.
type ConsensusEngine interface {
	ID() string
	PrivateKey() *crypto.PrivateKey
	GetTip(includePendingBlockingLeaf bool) *ExtendedBlock
	GetEpoch() uint64
	GetLedger() Ledger
	AddMessage(msg interface{})
	FinalizedBlocks() chan *Block
	GetLastFinalizedBlock() *ExtendedBlock
	GetEpochVotes() (*VoteSet, error)
	//GetValidatorSet(blockHash common.Hash) *ValidatorSet
	GetValidators(blockHash common.Hash) *AddressSet
}

// ValidatorManager is the component for managing validator related logic for consensus engine.
type ValidatorManager interface {
	SetConsensusEngine(consensus ConsensusEngine)
	//	GetProposer(blockHash common.Hash, epoch uint64) Validator
	GetProposer(blockHash common.Hash, epoch uint64) common.Address
	//	GetNextProposer(blockHash common.Hash, epoch uint64) Validator
	GetNextProposer(blockHash common.Hash, epoch uint64) common.Address
	//	GetValidatorSet(blockHash common.Hash) *ValidatorSet
	GetValidators(blockHash common.Hash) *AddressSet
	//	GetNextValidatorSet(blockHash common.Hash) *ValidatorSet
	GetNextValidators(blockHash common.Hash) *AddressSet
}
