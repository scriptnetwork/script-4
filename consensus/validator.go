package consensus

import (
	"math/big"
	"math/rand"

	"github.com/scripttoken/script/common"
	"github.com/scripttoken/script/core"
	log "github.com/sirupsen/logrus"
)

const MaxValidatorCount int = 31

// -------------------------------- FixedValidatorManager ----------------------------------
var _ core.ValidatorManager = &FixedValidatorManager{}

// FixedValidatorManager is an implementation of ValidatorManager interface that selects a fixed validator as the proposer.
type FixedValidatorManager struct {
	consensus core.ConsensusEngine
}

// NewFixedValidatorManager creates an instance of FixedValidatorManager.
func NewFixedValidatorManager() *FixedValidatorManager {
	m := &FixedValidatorManager{
		consensus: nil,
	}
	return m
}

// SetConsensusEngine mplements ValidatorManager interface.
func (m *FixedValidatorManager) SetConsensusEngine(consensus core.ConsensusEngine) {
	m.consensus = consensus
}

// GetProposer implements ValidatorManager interface.
func (m *FixedValidatorManager) GetProposer(blockHash common.Hash, _ uint64) common.Address {
	return m.getProposerFromValidators(m.GetValidators(blockHash))
}

// GetNextProposer implements ValidatorManager interface.
func (m *FixedValidatorManager) GetNextProposer(blockHash common.Hash, _ uint64) common.Address {
	return m.getProposerFromValidators(m.GetNextValidators(blockHash))
}

func (m *FixedValidatorManager) getProposerFromValidators(valSet *core.AddressSet) common.Address {
	if len(*valSet) == 0 {
		log.Panic("No validators have been added")
	}
	for key := range *valSet {
		return key
	}
	return common.Address{} //Unreachable code
}

// GetValidatorSet returns the validator set for given block hash.
func (m *FixedValidatorManager) GetValidators(blockHash common.Hash) *core.AddressSet {
	validators, err := m.consensus.GetLedger().GetFinalizedValidators(blockHash, false)
	if err != nil {
		log.Panicf("Failed to get the validators, blockHash: %v, err: %v", blockHash.Hex(), err)
	}
	if validators == nil {
		log.Panicf("Failed to retrieve the validators, blockHash: %v, isNext: %v", blockHash.Hex(), false)
	}
	return validators
}

// GetNextValidatorSet returns the validator set for given block hash's next block.
func (m *FixedValidatorManager) GetNextValidators(blockHash common.Hash) *core.AddressSet {
	validators, err := m.consensus.GetLedger().GetFinalizedValidators(blockHash, true)
	if err != nil {
		log.Panicf("Failed to get the validators, blockHash: %v, err: %v", blockHash.Hex(), err)
	}
	if validators == nil {
		log.Panicf("Failed to retrieve the validators, blockHash: %v, isNext: %v", blockHash.Hex(), true)
	}
	return validators
}

// -------------------------------- RotatingValidatorManager ----------------------------------
var _ core.ValidatorManager = &RotatingValidatorManager{}

// RotatingValidatorManager is an implementation of ValidatorManager interface that selects a random validator as
// the proposer using validator's stake as weight.
type RotatingValidatorManager struct {
	consensus core.ConsensusEngine
}

// NewRotatingValidatorManager creates an instance of RotatingValidatorManager.
func NewRotatingValidatorManager() *RotatingValidatorManager {
	m := &RotatingValidatorManager{}
	return m
}

// SetConsensusEngine mplements ValidatorManager interface.
func (m *RotatingValidatorManager) SetConsensusEngine(consensus core.ConsensusEngine) {
	m.consensus = consensus
}

// GetProposer implements ValidatorManager interface.
func (m *RotatingValidatorManager) GetProposer(blockHash common.Hash, epoch uint64) common.Address {
	return m.getProposerFromValidators(m.GetValidators(blockHash), epoch)
}

// GetNextProposer implements ValidatorManager interface.
func (m *RotatingValidatorManager) GetNextProposer(blockHash common.Hash, epoch uint64) common.Address {
	return m.getProposerFromValidators(m.GetNextValidators(blockHash), epoch)
}

func (m *RotatingValidatorManager) getRandomValidator(valSet *core.AddressSet, epoch uint64) common.Address {
	if len(*valSet) == 0 {
		log.Panic("No validators have been added")
	}
	validators := (*valSet).ToSortedSlice()
	rnd := rand.New(rand.NewSource(int64(epoch)))
	randomIndex := rnd.Intn(len(validators))
	return validators[randomIndex]
}

func (m *RotatingValidatorManager) getProposerFromValidators(valSet *core.AddressSet, epoch uint64) common.Address {
	return m.getRandomValidator(valSet, epoch)
}

// GetValidatorSet returns the validator set for given block.
func (m *RotatingValidatorManager) GetValidators(blockHash common.Hash) *core.AddressSet {
	validators, err := m.consensus.GetLedger().GetFinalizedValidators(blockHash, false)
	if err != nil {
		log.Panicf("Failed to get the validators, blockHash: %v, isNext: %v, err: %v", blockHash.Hex(), true, err)
	}
	if validators == nil {
		log.Panicf("Failed to retrieve the validators, blockHash: %v, isNext: %v", blockHash.Hex(), true)
	}
	return validators
}

// GetNextValidatorSet returns the validator set for given block's next block.
func (m *RotatingValidatorManager) GetNextValidators(blockHash common.Hash) *core.AddressSet {
	validators, err := m.consensus.GetLedger().GetFinalizedValidators(blockHash, true)
	if err != nil {
		log.Panicf("Failed to get the validators, blockHash: %v, isNext: %v, err: %v", blockHash.Hex(), true, err)
	}
	if validators == nil {
		log.Panicf("Failed to retrieve the validators, blockHash: %v, isNext: %v", blockHash.Hex(), true)
	}
	return validators
}

//
// -------------------------------- Utilities ----------------------------------
//
/*
func SelectTopStakeHoldersAsValidators(vcp *core.ValidatorCandidatePool) *core.ValidatorSet {
	maxNumValidators := MaxValidatorCount
	topStakeHolders := vcp.GetTopStakeHolders(maxNumValidators)

        minValidatorTotalStake := new(big.Int).Mul(new(big.Int).SetUint64(1000000), new(big.Int).SetUint64(1000000000000000000))

	valSet := core.NewValidatorSet()
	for _, stakeHolder := range topStakeHolders {
		valAddr := stakeHolder.Holder.Hex()
		valStake := stakeHolder.TotalStake()
		if valStake.Cmp(core.Zero) == 0 {
			continue
		}
	        if valStake.Cmp(minValidatorTotalStake) < 0 {
	                break
                }
		validator := core.NewValidator(valAddr, valStake)
		valSet.AddValidator(validator)
	}

	return valSet
}
*/
/*
func selectTopStakeHoldersAsValidatorsForBlock(consensus core.ConsensusEngine, blockHash common.Hash, isNext bool) *core.ValidatorSet {
	validators, err := consensus.GetLedger().GetFinalizedValidators(blockHash, isNext)
	if err != nil {
		log.Panicf("Failed to get the validators, blockHash: %v, isNext: %v, err: %v", blockHash.Hex(), isNext, err)
	}
	if validators == nil {
		log.Panic("Failed to retrieve the validators, blockHash: %v, isNext: %v", blockHash.Hex(), isNext)
	}

	return SelectTopStakeHoldersAsValidators(vcp)
}
*/
// Generate a random uint64 in [0, max)
func randUint64(rnd *rand.Rand, max uint64) uint64 {
	const maxInt64 uint64 = 1<<63 - 1
	if max <= maxInt64 {
		return uint64(rnd.Int63n(int64(max)))
	}
	for {
		r := rnd.Uint64()
		if r < max {
			return r
		}
	}
}

func scaleDown(x *big.Int, scalingFactor *big.Int) uint64 {
	if scalingFactor.Cmp(common.Big0) == 0 {
		log.Panic("scalingFactor is zero")
	}
	scaledX := new(big.Int).Div(x, scalingFactor)
	scaledXUint64 := scaledX.Uint64()
	return scaledXUint64
}
