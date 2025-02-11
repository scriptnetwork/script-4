package execution

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/big"
	"sort"

	"github.com/scripttoken/script/blockchain"
	"github.com/scripttoken/script/common"
	"github.com/scripttoken/script/common/result"
	"github.com/scripttoken/script/common/util"
	"github.com/scripttoken/script/core"
	"github.com/scripttoken/script/ledger/state"
	st "github.com/scripttoken/script/ledger/state"
	"github.com/scripttoken/script/ledger/types"
	"github.com/scripttoken/script/store/database"
)

var weiMultiplier = big.NewInt(1e18)
var spayRewardPerBlock = big.NewInt(1).Mul(big.NewInt(48), weiMultiplier)    // 48 SPAY per block, corresponds to about 5% *initial* annual inflation rate. The inflation rate naturally approaches 0 as the chain grows.
var eenSpayRewardPerBlock = big.NewInt(1).Mul(big.NewInt(38), weiMultiplier) // 38 SPAY per block, corresponds to about 4% *initial* annual inflation rate. The inflation rate naturally approaches 0 as the chain grows.
var spayRewardN = 400                                                        // Reward receiver sampling params

var _ TxExecutor = (*CoinbaseTxExecutor)(nil)

// ------------------------------- Coinbase Transaction -----------------------------------

// CoinbaseTxExecutor implements the TxExecutor interface
type CoinbaseTxExecutor struct {
	db        database.Database
	chain     *blockchain.Chain
	state     *st.LedgerState
	consensus core.ConsensusEngine
	valMgr    core.ValidatorManager
}

// NewCoinbaseTxExecutor creates a new instance of CoinbaseTxExecutor
func NewCoinbaseTxExecutor(db database.Database, chain *blockchain.Chain, state *st.LedgerState, consensus core.ConsensusEngine, valMgr core.ValidatorManager) *CoinbaseTxExecutor {
	return &CoinbaseTxExecutor{
		db:        db,
		chain:     chain,
		state:     state,
		consensus: consensus,
		valMgr:    valMgr,
	}
}

func (exec *CoinbaseTxExecutor) sanityCheck(chainID string, view *st.StoreView, viewSel core.ViewSelector, transaction types.Tx) result.Result {
	tx := transaction.(*types.CoinbaseTx)
	validatorSet := getValidatorSet(exec.consensus.GetLedger(), exec.valMgr)
	validatorAddresses := getValidatorAddresses(validatorSet)

	// Validate proposer, basic
	res := tx.Proposer.ValidateBasic()
	if res.IsError() {
		return res
	}

	// verify that at most one coinbase transaction is processed for each block
	if view.CoinbaseTransactinProcessed() {
		return result.Error("Another coinbase transaction has been processed for the current block")
	}

	// verify the proposer is one of the validators
	res = isAValidator(tx.Proposer.Address, validatorAddresses)
	if res.IsError() {
		return res
	}

	proposerAccount, res := getOrMakeInput(view, tx.Proposer)
	if res.IsError() {
		return res
	}

	// verify the proposer's signature
	signBytes := tx.SignBytes(chainID)
	if !tx.Proposer.Signature.Verify(signBytes, proposerAccount.Address) {
		return result.Error("SignBytes: %X", signBytes)
	}

	outputAccounts := map[string]*types.Account{}
	outputAccounts, res = getOrMakeOutputs(view, outputAccounts, tx.Outputs)
	if res.IsError() {
		return res
	}

	if tx.BlockHeight != exec.state.Height() {
		return result.Error("invalid block height for the coinbase transaction, tx_block_height = %v, state_height = %v",
			tx.BlockHeight, exec.state.Height())
	}

	// check the reward amount
	var expectedRewards map[string]types.Coins
	currentBlock := exec.consensus.GetLedger().GetCurrentBlock()
	lightningVotes := currentBlock.LightningVotes
	eliteEdgeNodeVotes := currentBlock.EliteEdgeNodeVotes
	lightningPool, eliteEdgeNodePool := RetrievePools(exec.consensus.GetLedger(), exec.chain, exec.db, tx.BlockHeight, lightningVotes, eliteEdgeNodeVotes)
	expectedRewards = CalculateReward(exec.consensus.GetLedger(), view, validatorSet, lightningVotes, lightningPool, eliteEdgeNodeVotes, eliteEdgeNodePool)

	if len(expectedRewards) != len(tx.Outputs) {
		return result.Error("Number of rewarded account is incorrect")
	}
/*
	for _, output := range tx.Outputs {
		exp, ok := expectedRewards[string(output.Address[:])]
		if !ok || !exp.IsEqual(output.Coins) {
			return result.Error("Invalid rewards, address %v expecting %v, but is %v",
				output.Address, exp, output.Coins)
		}
	}
*/
	return result.OK
}

func (exec *CoinbaseTxExecutor) process(chainID string, view *st.StoreView, viewSel core.ViewSelector, transaction types.Tx) (common.Hash, result.Result) {
	tx := transaction.(*types.CoinbaseTx)

	if view.CoinbaseTransactinProcessed() {
		return common.Hash{}, result.Error("Another coinbase transaction has been processed for the current block")
	}

	accounts := map[string]*types.Account{}
	accounts, res := getOrMakeOutputs(view, accounts, tx.Outputs)
	if res.IsError() {
		return common.Hash{}, res
	}

	for _, output := range tx.Outputs {
		addr := string(output.Address[:])
		if account, exists := accounts[addr]; exists {
			account.Balance = account.Balance.Plus(output.Coins)
			view.SetAccount(output.Address, account)
		}
	}

	view.SetCoinbaseTransactionProcessed(true)

	txHash := types.TxID(chainID, tx)
	return txHash, result.OK
}

func RetrievePools(ledger core.Ledger, chain *blockchain.Chain, db database.Database, blockHeight uint64, lightningVotes *core.AggregatedVotes,
	eliteEdgeNodeVotes *core.AggregatedEENVotes) (lightningPool *core.LightningCandidatePool, eliteEdgeNodePool core.EliteEdgeNodePool) {
	lightningPool = nil
	eliteEdgeNodePool = nil

/*
	if blockHeight < common.HeightEnableScript2 {
		lightningPool = nil
		eliteEdgeNodePool = nil
	} else if blockHeight < common.HeightEnableScript3 {
		if lightningVotes != nil {
			guradianVoteBlock, err := chain.FindBlock(lightningVotes.Block)
			if err != nil {
				logger.Panic(err)
			}
			storeView := st.NewStoreView(guradianVoteBlock.Height, guradianVoteBlock.StateHash, db)
			lightningPool = storeView.GetLightningCandidatePool()
		}
	} else { // blockHeight >= common.HeightEnableScript3
		// won't reward the elite edge nodes without the lightning votes, since we need to lightning votes to confirm that
		// the edge nodes vote for the correct checkpoint
*/
		if lightningVotes != nil {
			guradianVoteBlock, err := chain.FindBlock(lightningVotes.Block)
			if err != nil {
				logger.Panic(err)
			}
			storeView := st.NewStoreView(guradianVoteBlock.Height, guradianVoteBlock.StateHash, db)
			lightningPool = storeView.GetLightningCandidatePool()

			if eliteEdgeNodeVotes != nil {
				if eliteEdgeNodeVotes.Block == lightningVotes.Block {
					eliteEdgeNodePool = st.NewEliteEdgeNodePool(storeView, true)
				} else {
					logger.Warnf("Elite edge nodes vote for block %v, while lightnings vote for block %v, skip rewarding the elite edge nodes",
						eliteEdgeNodeVotes.Block.Hex(), lightningVotes.Block.Hex())
				}
			} else {
				logger.Warnf("Elite edge nodes have no vote for block %v", lightningVotes.Block.Hex())
			}
		}
//	}

	return lightningPool, eliteEdgeNodePool
}

// CalculateReward calculates the block reward for each account
func CalculateReward(ledger core.Ledger, view *st.StoreView, validatorSet *core.ValidatorSet,
	lightningVotes *core.AggregatedVotes, lightningPool *core.LightningCandidatePool,
	eliteEdgeNodeVotes *core.AggregatedEENVotes, eliteEdgeNodePool core.EliteEdgeNodePool) map[string]types.Coins {
	accountReward := map[string]types.Coins{}
	blockHeight := view.Height() + 1 // view points to the parent block
	if blockHeight < common.HeightEnableValidatorReward {
		grantValidatorsWithZeroReward(validatorSet, &accountReward)
	} else if blockHeight < common.HeightEnableScript2 || lightningVotes == nil || lightningPool == nil {
		grantValidatorReward(ledger, view, validatorSet, &accountReward, blockHeight)
	} else if blockHeight < common.HeightEnableScript3 {
		grantValidatorAndLightningReward(ledger, view, validatorSet, lightningVotes, lightningPool, &accountReward, blockHeight)
	} else { // blockHeight >= common.HeightEnableScript3
		grantValidatorAndLightningReward(ledger, view, validatorSet, lightningVotes, lightningPool, &accountReward, blockHeight)
		grantEliteEdgeNodeReward(ledger, view, lightningVotes, eliteEdgeNodeVotes, eliteEdgeNodePool, &accountReward, blockHeight)
	}

	addrs := []string{}
	for addr := range accountReward {
		addrs = append(addrs, addr)
	}
	sort.Strings(addrs)
	for _, addr := range addrs {
		reward := accountReward[addr]
		logger.Infof("Total reward for account %v : %v", hex.EncodeToString([]byte(addr)), reward)
	}

	return accountReward
}

func grantValidatorsWithZeroReward(validatorSet *core.ValidatorSet, accountReward *map[string]types.Coins) {
	// Initial Mainnet release should not reward the validators until the lightnings ready to deploy
	zeroReward := types.Coins{}.NoNil()
	for _, v := range validatorSet.Validators() {
		(*accountReward)[string(v.Address[:])] = zeroReward
	}
}

func grantValidatorReward(ledger core.Ledger, view *st.StoreView, validatorSet *core.ValidatorSet, accountReward *map[string]types.Coins, blockHeight uint64) {
	if !common.IsCheckPointHeight(blockHeight) {
		return
	}

	totalStake := validatorSet.TotalStake()

	if totalStake.Cmp(big.NewInt(0)) == 0 {
		// Should never happen
		return
	}

	stakeSourceMap := map[common.Address]*big.Int{}
	stakeSourceList := []common.Address{}

	// TODO - Need to confirm: should we get the VCP from the current view? What if there is a stake deposit/withdraw?
	vcp := view.GetValidatorCandidatePool()
	for _, v := range validatorSet.Validators() {
		validatorAddr := v.Address
		stakeDelegate := vcp.FindStakeDelegate(validatorAddr)
		if stakeDelegate == nil { // should not happen
			panic(fmt.Sprintf("Failed to find stake delegate in the VCP: %v", hex.EncodeToString(validatorAddr[:])))
		}

		stakes := stakeDelegate.Stakes
		for _, stake := range stakes {
			if stake.Withdrawn {
				continue
			}
			stakeAmount := stake.Amount
			stakeSource := stake.Source
			if stakeAmountSum, exists := stakeSourceMap[stakeSource]; exists {
				stakeAmountSum := big.NewInt(0).Add(stakeAmountSum, stakeAmount)
				stakeSourceMap[stakeSource] = stakeAmountSum
			} else {
				stakeSourceMap[stakeSource] = stakeAmount
				stakeSourceList = append(stakeSourceList, stakeSource)
			}
		}
	}

	totalReward := big.NewInt(1).Mul(spayRewardPerBlock, big.NewInt(common.CheckpointInterval))

	// the source of the stake divides the block reward proportional to their stake
	for stakeSourceAddr, stakeAmountSum := range stakeSourceMap {
		tmp := big.NewInt(1).Mul(totalReward, stakeAmountSum)
		rewardAmount := tmp.Div(tmp, totalStake)

		reward := types.Coins{
			SCPTWei: big.NewInt(0),
			SPAYWei: rewardAmount,
		}.NoNil()
		(*accountReward)[string(stakeSourceAddr[:])] = reward

		logger.Infof("Block reward for staker %v : %v", hex.EncodeToString(stakeSourceAddr[:]), reward)
	}
}

// grant block rewards to both the validators and active lightnings (they are both script stakers)
func grantValidatorAndLightningReward(ledger core.Ledger, view *st.StoreView, validatorSet *core.ValidatorSet, lightningVotes *core.AggregatedVotes,
	lightningPool *core.LightningCandidatePool, accountReward *map[string]types.Coins, blockHeight uint64) {
	if !common.IsCheckPointHeight(blockHeight) {
		return
	}

	totalStake := validatorSet.TotalStake()

	if lightningPool == nil || lightningVotes == nil {
		// Should never reach here
		panic("lightningPool == nil || lightningVotes == nil")
	}
	lightningPool = lightningPool.WithStake()

	if totalStake.Cmp(big.NewInt(0)) == 0 {
		// Should never happen
		return
	}

	effectiveStakes := [][]*core.Stake{}          // For compatiblity with old sampling algorithm, stakes from the same staker are grouped together
	stakeGroupMap := make(map[common.Address]int) // stake source address -> index of the group in the effectiveStakes slice

	// TODO - Need to confirm: should we get the VCP from the current view? What if there is a stake deposit/withdraw?
	vcp := view.GetValidatorCandidatePool()
	for _, v := range validatorSet.Validators() {
		validatorAddr := v.Address
		stakeDelegate := vcp.FindStakeDelegate(validatorAddr)
		if stakeDelegate == nil { // should not happen
			panic(fmt.Sprintf("Failed to find stake delegate in the VCP: %v", hex.EncodeToString(validatorAddr[:])))
		}

		stakes := stakeDelegate.Stakes
		for _, stake := range stakes {
			if stake.Withdrawn {
				continue
			}
			if _, exists := stakeGroupMap[stake.Source]; !exists {
				stakeGroupMap[stake.Source] = len(effectiveStakes)
				effectiveStakes = append(effectiveStakes, []*core.Stake{})
			}
			stake.Holder = stakeDelegate.Holder
			idx := stakeGroupMap[stake.Source]
			effectiveStakes[idx] = append(effectiveStakes[idx], stake)
		}
	}

	for i, g := range lightningPool.SortedLightnings {
		if lightningVotes.Multiplies[i] == 0 {
			continue
		}
		stakes := g.Stakes
		for _, stake := range stakes {
			if stake.Withdrawn {
				continue
			}

			totalStake.Add(totalStake, stake.Amount)

			if _, exists := stakeGroupMap[stake.Source]; !exists {
				stakeGroupMap[stake.Source] = len(effectiveStakes)
				effectiveStakes = append(effectiveStakes, []*core.Stake{})
			}
			stake.Holder = g.Holder
			idx := stakeGroupMap[stake.Source]
			effectiveStakes[idx] = append(effectiveStakes[idx], stake)
		}
	}

	totalReward := big.NewInt(1).Mul(spayRewardPerBlock, big.NewInt(common.CheckpointInterval))

	var srdsr *st.StakeRewardDistributionRuleSet
	if blockHeight >= common.HeightEnableScript3 {
		srdsr = state.NewStakeRewardDistributionRuleSet(view)
	}

	if blockHeight > common.Height_hf1 {

	issueFixedReward(effectiveStakes, totalStake, accountReward, totalReward, srdsr, "Block") // Fix job_309; issueRandomizedReward is not dealing rewards to lighning

	} else {
		if blockHeight < common.HeightSampleStakingReward {
			// the source of the stake divides the block reward proportional to their stake
			issueFixedReward(effectiveStakes, totalStake, accountReward, totalReward, srdsr, "Block")
		} else {
			// randomly select (proportional to the stake) a constant-sized set of stakers and grand the block reward
			issueRandomizedReward(ledger, lightningVotes, view, effectiveStakes, totalStake, accountReward, totalReward, srdsr, "Block")
		}
	}

}

// grant uptime mining rewards to active elite edge nodes (they are the spay stakers)
func grantEliteEdgeNodeReward(ledger core.Ledger, view *st.StoreView, lightningVotes *core.AggregatedVotes, eliteEdgeNodeVotes *core.AggregatedEENVotes,
	eliteEdgeNodePool core.EliteEdgeNodePool, accountReward *map[string]types.Coins, blockHeight uint64) {
	if !common.IsCheckPointHeight(blockHeight) {
		return
	}

	if lightningVotes == nil {
		// Should never reach here
		panic("lightningVotes == nil")
	}

	logger.Debugf("grantEliteEdgeNodeReward: lightningVotes = %v, eliteEdgeNodeVotes = %v", lightningVotes, eliteEdgeNodeVotes)

	if eliteEdgeNodeVotes == nil || eliteEdgeNodePool == nil {
		return
	}

	effectiveStakes := [][]*core.Stake{}          // For compatiblity with old sampling algorithm, stakes from the same staker are grouped together
	stakeGroupMap := make(map[common.Address]int) // stake source address -> index of the group in the effectiveStakes slice

	totalEffectiveStake := new(big.Int)
	amplifier := new(big.Int).SetUint64(1e18)
	for _, eenAddr := range eliteEdgeNodeVotes.Addresses {
		weight := big.NewInt(int64(eliteEdgeNodePool.RandomRewardWeight(eliteEdgeNodeVotes.Block, eenAddr)))
		een := eliteEdgeNodePool.Get(eenAddr)

		eenTotalStake := een.TotalStake()
		if eenTotalStake.Cmp(big.NewInt(0)) == 0 {
			continue
		}

		amplifiedWeight := big.NewInt(1).Mul(amplifier, weight)
		for _, stake := range een.Stakes {
			if stake.Withdrawn {
				continue
			}

			// for EEN reward calculation
			effectiveStakeAmount := big.NewInt(1)
			effectiveStakeAmount.Mul(amplifiedWeight, stake.Amount)
			effectiveStakeAmount.Div(effectiveStakeAmount, eenTotalStake)

			effectiveStake := &core.Stake{
				Holder: een.Holder,
				Source: stake.Source,
				Amount: effectiveStakeAmount,
			}
			if _, exists := stakeGroupMap[effectiveStake.Source]; !exists {
				stakeGroupMap[effectiveStake.Source] = len(effectiveStakes)
				effectiveStakes = append(effectiveStakes, []*core.Stake{})
			}
			idx := stakeGroupMap[effectiveStake.Source]
			effectiveStakes[idx] = append(effectiveStakes[idx], effectiveStake)

			totalEffectiveStake.Add(totalEffectiveStake, effectiveStakeAmount)

			logger.Debugf("grantEliteEdgeNodeReward: eenAddr = %v, eenTotalStake = %v, weight = %v, staker: %v, stake = %v, effectiveStakeAmount = %v",
				eenAddr, eenTotalStake, weight, stake.Source, stake.Amount, effectiveStakeAmount)
		}
	}

	// the source of the stake divides the block reward proportional to their stake
	totalReward := big.NewInt(1).Mul(eenSpayRewardPerBlock, big.NewInt(common.CheckpointInterval))

	logger.Debugf("grantEliteEdgeNodeReward: totalEffectiveStake = %v, totalReward = %v", totalEffectiveStake, totalReward)

	var srdsr *st.StakeRewardDistributionRuleSet
	if blockHeight >= common.HeightEnableScript3 {
		srdsr = state.NewStakeRewardDistributionRuleSet(view)
	}

	// the source of the stake divides the block reward proportional to their stake
	issueFixedReward(effectiveStakes, totalEffectiveStake, accountReward, totalReward, srdsr, "EEN  ")

}

func addRewardToMap(receiver common.Address, amount *big.Int, accountReward *map[string]types.Coins) {
	rewardCoins := types.Coins{
		SCPTWei: big.NewInt(0),
		SPAYWei: amount,
	}.NoNil()
	receiverAddr := string(receiver[:])
	if existingReward, exists := (*accountReward)[receiverAddr]; exists {
		totalReward := existingReward.NoNil().Plus(rewardCoins)
		(*accountReward)[receiverAddr] = totalReward
	} else {
		(*accountReward)[receiverAddr] = rewardCoins
	}
}

func handleSplit(stake *core.Stake, srdsr *st.StakeRewardDistributionRuleSet, reward *big.Int, accountRewardMap *map[string]types.Coins) {
	if srdsr == nil {
		// Should not happen
		logger.Panic("srdsr is nil")
	}
	if stake.Holder.IsEmpty() {
		// Should not happen
		logger.Panic("stake holder is not set")
	}

	rewardDistribution := srdsr.Get(stake.Holder)
	if rewardDistribution == nil {
		addRewardToMap(stake.Source, reward, accountRewardMap)
		return
	}

	if rewardDistribution.SplitBasisPoint == 0 {
		// Should not happen
		logger.Panicf("SplitBasisPoint is 0. Holder=%v, Beneficiary=%v", rewardDistribution.StakeHolder, rewardDistribution.Beneficiary)
	}

	splitReward := big.NewInt(1)
	splitReward.Mul(reward, big.NewInt(int64(rewardDistribution.SplitBasisPoint)))
	splitReward.Div(splitReward, big.NewInt(10000))

	sourceReward := new(big.Int).Sub(reward, splitReward)

	logger.Debugf("Reward redistribution metadata: splitReward = %v, sourceReward = %v, SplitBasisPoint = %v",
		splitReward, sourceReward, rewardDistribution.SplitBasisPoint)

	if splitReward.Cmp(reward) > 0 {
		logger.Panic("Invalid reward redistribution metadata")
	}

	addRewardToMap(stake.Source, sourceReward, accountRewardMap)
	addRewardToMap(rewardDistribution.Beneficiary, splitReward, accountRewardMap)
}

func issueFixedReward(effectiveStakes [][]*core.Stake, totalStake *big.Int, accountReward *map[string]types.Coins, totalReward *big.Int, srdsr *st.StakeRewardDistributionRuleSet, rewardType string) {
	if totalStake.Cmp(big.NewInt(0)) == 0 {
		return
	}

	if srdsr != nil {
		for _, stakes := range effectiveStakes {
			for _, stake := range stakes {
				rewardAmount := big.NewInt(1)
				rewardAmount.Mul(totalReward, stake.Amount)
				rewardAmount.Div(rewardAmount, totalStake)

				logger.Infof("%v reward for staker %v : %v  (before split)", rewardType, hex.EncodeToString(stake.Source[:]), rewardAmount)

				// Calculate split
				handleSplit(stake, srdsr, rewardAmount, accountReward)
			}
		}
	} else {
		// Aggregate all stakes of a source before calculating reward to be compatible with previous algorithm
		for _, stakes := range effectiveStakes {
			if len(stakes) == 0 {
				continue
			}
			totalSourceStake := big.NewInt(0)
			for _, stake := range stakes {
				totalSourceStake.Add(totalSourceStake, stake.Amount)
			}
			rewardAmount := big.NewInt(1)
			rewardAmount.Mul(totalReward, totalSourceStake)
			rewardAmount.Div(rewardAmount, totalStake)
			addRewardToMap(stakes[0].Source, rewardAmount, accountReward)

			logger.Infof("%v reward for staker %v : %v  (before split)", rewardType, hex.EncodeToString(stakes[0].Source[:]), rewardAmount)
		}
	}
}

func issueRandomizedReward(ledger core.Ledger, lightningVotes *core.AggregatedVotes, view *st.StoreView, effectiveStakes [][]*core.Stake,
	totalStake *big.Int, accountReward *map[string]types.Coins, totalReward *big.Int, srdsr *st.StakeRewardDistributionRuleSet, rewardType string) {

	if lightningVotes == nil {
		// Should never reach here
		panic("lightningVotes == nil")
	}

	samples := make([]*big.Int, spayRewardN)
	for i := 0; i < spayRewardN; i++ {
		// Set random seed to (block_height||sampling_index||checkpoint_hash)
		seed := make([]byte, 2*binary.MaxVarintLen64+common.HashLength)
		binary.PutUvarint(seed[:], view.Height())
		binary.PutUvarint(seed[binary.MaxVarintLen64:], uint64(i))
		copy(seed[2*binary.MaxVarintLen64:], lightningVotes.Block[:])

		var err error
		samples[i], err = rand.Int(util.NewHashRand(seed), totalStake)
		if err != nil {
			// Should not reach here
			logger.Panic(err)
		}

		// // ---------- Just for testing ---------- //
		// totalStakeFloat := new(big.Float).SetInt(totalStake)
		// sampleFloat := new(big.Float).SetInt(samples[i])
		// logger.Infof("RandSample -- r: %v, height: %v, totalStake: %v, sample[%v]: %v",
		// 	new(big.Float).Quo(sampleFloat, totalStakeFloat).Text('f', 6), view.Height()+1, totalStake, i, samples[i])
	}

	sort.Sort(BigIntSort(samples))

	if srdsr != nil {
		curr := 0
		currSum := big.NewInt(0)

		for _, stakes := range effectiveStakes {
			for _, stake := range stakes {
				stakeSourceAddr := stake.Source
				stakeAmountSum := stake.Amount

				if curr >= spayRewardN {
					break
				}

				count := 0
				lower := currSum
				upper := new(big.Int).Add(currSum, stakeAmountSum)
				for curr < spayRewardN && samples[curr].Cmp(lower) >= 0 && samples[curr].Cmp(upper) < 0 {
					count++
					curr++
				}
				currSum = upper

				//logger.Infof("RandomReward -- staker: %v, count: %v, height: %v, stake: %v, type: %v", stakeSourceAddr, count, view.Height()+1, stakeAmountSum, rewardType)

				if count > 0 {
					tmp := new(big.Int).Mul(totalReward, big.NewInt(int64(count)))
					rewardAmount := tmp.Div(tmp, big.NewInt(int64(spayRewardN)))

					logger.Infof("%v reward for staker %v : %v (before split)", rewardType, hex.EncodeToString(stakeSourceAddr[:]), rewardAmount)

					// Calculate split
					handleSplit(stake, srdsr, rewardAmount, accountReward)
				}
			}
		}
	} else {
		// Aggregate all stakes of a source before calculating reward to be compatible with previous algorithm
		curr := 0
		currSum := big.NewInt(0)

		for _, stakes := range effectiveStakes {
			if len(stakes) == 0 {
				continue
			}
			stakeSourceAddr := stakes[0].Source
			stakeAmountSum := big.NewInt(0)
			for _, stake := range stakes {
				stakeAmountSum.Add(stakeAmountSum, stake.Amount)
			}

			if curr >= spayRewardN {
				break
			}

			count := 0
			lower := currSum
			upper := new(big.Int).Add(currSum, stakeAmountSum)
			for curr < spayRewardN && samples[curr].Cmp(lower) >= 0 && samples[curr].Cmp(upper) < 0 {
				count++
				curr++
			}
			currSum = upper

			// logger.Infof("RandomReward -- staker: %v, count: %v, height: %v, stake: %v, type: %v", stakeSourceAddr, count, view.Height()+1, stakeAmountSum, rewardType)

			if count > 0 {
				tmp := new(big.Int).Mul(totalReward, big.NewInt(int64(count)))
				rewardAmount := tmp.Div(tmp, big.NewInt(int64(spayRewardN)))

				addRewardToMap(stakeSourceAddr, rewardAmount, accountReward)

				logger.Infof("%v reward for staker %v : %v (before split)", rewardType, hex.EncodeToString(stakeSourceAddr[:]), rewardAmount)
			}
		}
	}

}

func (exec *CoinbaseTxExecutor) getTxInfo(transaction types.Tx) *core.TxInfo {
	return &core.TxInfo{
		EffectiveGasPrice: exec.calculateEffectiveGasPrice(transaction),
	}
}

func (exec *CoinbaseTxExecutor) calculateEffectiveGasPrice(transaction types.Tx) *big.Int {
	return new(big.Int).SetUint64(0)
}

type BigIntSort []*big.Int

func (s BigIntSort) Len() int           { return len(s) }
func (s BigIntSort) Less(i, j int) bool { return s[i].Cmp(s[j]) < 0 }
func (s BigIntSort) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
