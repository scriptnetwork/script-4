package execution

import (
	"fmt"
	"math/big"

	"github.com/scripttoken/script/common"
	"github.com/scripttoken/script/common/result"
	"github.com/scripttoken/script/core"
	"github.com/scripttoken/script/ledger/state"
	st "github.com/scripttoken/script/ledger/state"
	"github.com/scripttoken/script/ledger/types"
)

var _ TxExecutor = (*StakeRewardDistributionTxExecutor)(nil)

// ------------------------------- SplitRule Transaction -----------------------------------

// StakeRewardDistributionTxExecutor implements the TxExecutor interface
type StakeRewardDistributionTxExecutor struct {
	state *st.LedgerState
}

// NewStakeRewardDistributionTxExecutor creates a new instance of StakeRewardDistributionTxExecutor
func NewStakeRewardDistributionTxExecutor(state *st.LedgerState) *StakeRewardDistributionTxExecutor {
	return &StakeRewardDistributionTxExecutor{
		state: state,
	}
}

func (exec *StakeRewardDistributionTxExecutor) sanityCheck(chainID string, view *st.StoreView, viewSel core.ViewSelector, transaction types.Tx) result.Result {
	blockHeight := view.Height() + 1 // the view points to the parent of the current block

	tx := transaction.(*types.StakeRewardDistributionTx)

	res := tx.Holder.ValidateBasic()
	if res.IsError() {
		return res
	}

	// Get inputs
	stakeHolderAccount, res := getInput(view, tx.Holder)
	if res.IsError() {
		return res
	}

	// Validate inputs and outputs, advanced
	signBytes := tx.SignBytes(chainID)
	res = validateInputAdvanced(stakeHolderAccount, signBytes, tx.Holder, blockHeight)
	if res.IsError() {
		return res
	}

	res = validateOutputsBasic([]types.TxOutput{tx.Beneficiary})
	if res.IsError() {
		return res
	}

	if tx.SplitBasisPoint > 1000 { // initially we only allow up to 10.00% reward split
		return result.Error("Only allow at most 10.00%% reward split for the beneficiary for now (i.e., SplitBasisPoint <= 1000)")
	}

	// stakeHolderAddress := tx.Holder.Address
	// beneficiaryAddress := tx.Beneficiary.Address

	// vcp := view.GetValidatorCandidatePool()
	// if vcp.FindStakeDelegate(stakeHolderAddress) != nil {
	// 	// for safety purpose, for now we don't allow reward split for validators, even if the validator is also a lightning
	// 	return result.Error("StakeRewardDistributionTx not allowed for Validators for now")
	// }

	// if tx.Purpose == core.StakeForLightning {
	// 	gcp := view.GetLightningCandidatePool().WithStake()

	// 	var gn *core.Lightning
	// 	if gn = gcp.GetWithHolderAddress(stakeHolderAddress); gn == nil {
	// 		return result.Error("%v is not an staked lightning node", stakeHolderAddress)
	// 	}

	// 	for _, stake := range gn.Stakes {
	// 		if stake.Source == beneficiaryAddress {
	// 			return result.Error("Beneficiary is not allowed to be a staker address")
	// 		}
	// 	}
	// } else if tx.Purpose == core.StakeForEliteEdgeNode {
	// 	eenp := state.NewEliteEdgeNodePool(view, true)

	// 	var een *core.EliteEdgeNode
	// 	if een = eenp.Get(stakeHolderAddress); een == nil {
	// 		return result.Error("%v is not an staked elite edge node", stakeHolderAddress)
	// 	}

	// 	for _, stake := range een.Stakes {
	// 		if stake.Source == beneficiaryAddress {
	// 			return result.Error("Beneficiary is not allowed to be a staker address")
	// 		}
	// 	}
	// } else {
	// 	return result.Error("Invalid purpose: %v", tx.Purpose)
	// }

	if minTxFee, success := sanityCheckForFee(tx.Fee, blockHeight); !success {
		return result.Error("Insufficient fee. Transaction fee needs to be at least %v SPAYWei",
			minTxFee).WithErrorCode(result.CodeInvalidFee)
	}

	minimalBalance := tx.Fee
	if !stakeHolderAccount.Balance.IsGTE(minimalBalance) {
		logger.Infof(fmt.Sprintf("the contract initiator did not have enough to cover the fee %X", tx.Holder.Address))
		return result.Error("the contract initiator account balance is %v, but required minimal balance is %v", stakeHolderAccount.Balance, minimalBalance)
	}

	return result.OK
}

func (exec *StakeRewardDistributionTxExecutor) process(chainID string, view *st.StoreView, viewSel core.ViewSelector, transaction types.Tx) (common.Hash, result.Result) {
	tx := transaction.(*types.StakeRewardDistributionTx)

	stakeHolderAccount, res := getInput(view, tx.Holder)
	if res.IsError() {
		return common.Hash{}, res
	}

	if !chargeFee(stakeHolderAccount, tx.Fee) {
		return common.Hash{}, result.Error("failed to charge transaction fee")
	}

	stakeHolderAddress := tx.Holder.Address
	//if tx.Purpose == core.StakeForLightning || tx.Purpose == core.StakeForEliteEdgeNode {
	srdsr := state.NewStakeRewardDistributionRuleSet(view)

	splitBasisPoint := tx.SplitBasisPoint
	if splitBasisPoint > 10000 {
		splitBasisPoint = 10000
	} else if splitBasisPoint < 0 { // should not happen, but doesn't hurt to have the check
		splitBasisPoint = 0
	}

	if splitBasisPoint == 0 { // considered as removal
		srdsr.Remove(stakeHolderAddress) // no need to check the return value, ok to remove a non-existing reward distribution rule
	} else {
		rd, err := core.NewRewardDistribution(stakeHolderAddress, tx.Beneficiary.Address, splitBasisPoint)
		if err != nil { // should not reach here
			logger.Panicf("Failed to create reward distribution: %v", err)
		}
		srdsr.Upsert(rd)
	}
	// } else {
	// 	return common.Hash{}, result.Error("Invalid purpose").WithErrorCode(result.CodeInvalidStakePurpose)
	// }

	stakeHolderAccount.Sequence++
	view.SetAccount(tx.Holder.Address, stakeHolderAccount)

	txHash := types.TxID(chainID, tx)
	return txHash, result.OK
}

func (exec *StakeRewardDistributionTxExecutor) getTxInfo(transaction types.Tx) *core.TxInfo {
	tx := transaction.(*types.StakeRewardDistributionTx)
	return &core.TxInfo{
		Address:           tx.Holder.Address,
		Sequence:          tx.Holder.Sequence,
		EffectiveGasPrice: exec.calculateEffectiveGasPrice(transaction),
	}
}

func (exec *StakeRewardDistributionTxExecutor) calculateEffectiveGasPrice(transaction types.Tx) *big.Int {
	tx := transaction.(*types.StakeRewardDistributionTx)
	fee := tx.Fee
	gas := new(big.Int).SetUint64(getRegularTxGas(exec.state))
	effectiveGasPrice := new(big.Int).Div(fee.SPAYWei, gas)
	return effectiveGasPrice
}
