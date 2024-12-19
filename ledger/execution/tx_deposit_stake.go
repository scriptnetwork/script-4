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

var _ TxExecutor = (*DepositStakeExecutor)(nil)

// ------------------------------- DepositStake Transaction -----------------------------------

// DepositStakeExecutor implements the TxExecutor interface
type DepositStakeExecutor struct {
	state *st.LedgerState
}

// NewDepositStakeExecutor creates a new instance of DepositStakeExecutor
func NewDepositStakeExecutor(state *st.LedgerState) *DepositStakeExecutor {
	return &DepositStakeExecutor{
		state: state,
	}
}

func (exec *DepositStakeExecutor) sanityCheck(chainID string, view *st.StoreView, viewSel core.ViewSelector, transaction types.Tx) result.Result {
	// Feature block height check
	blockHeight := view.Height() + 1 // the view points to the parent of the current block
	if _, ok := transaction.(*types.DepositStakeTxV2); ok && blockHeight < common.HeightEnableScript2 {
		return result.Error("Feature lightning is not active yet")
	}

	tx := exec.castTx(transaction)

	res := tx.Source.ValidateBasic()
	if res.IsError() {
		return res
	}

	sourceAccount, success := getInput(view, tx.Source)
	if success.IsError() {
		return result.Error("Failed to get the source account: %v", tx.Source.Address)
	}

	signBytes := tx.SignBytes(chainID)
	res = validateInputAdvanced(sourceAccount, signBytes, tx.Source, blockHeight)
	if res.IsError() {
		logger.Debugf(fmt.Sprintf("validateSourceAdvanced failed on %v: %v", tx.Source.Address.Hex(), res))
		return res
	}

	if minTxFee, success := sanityCheckForFee(tx.Fee, blockHeight); !success {
		return result.Error("Insufficient fee. Transaction fee needs to be at least %v SPAYWei",
			minTxFee).WithErrorCode(result.CodeInvalidFee)
	}

	if !(tx.Purpose == core.StakeForValidator || tx.Purpose == core.StakeForLightning || tx.Purpose == core.StakeForEliteEdgeNode) {
		return result.Error("Invalid stake purpose!").
			WithErrorCode(result.CodeInvalidStakePurpose)
	}

	stake := tx.Source.Coins.NoNil()
	if !stake.IsValid() || !stake.IsNonnegative() {
		return result.Error("Invalid stake for stake deposit!").
			WithErrorCode(result.CodeInvalidStake)
	}

	if (tx.Purpose == core.StakeForValidator || tx.Purpose == core.StakeForLightning) && stake.SPAYWei.Cmp(types.Zero) != 0 {
		return result.Error("SPAY has to be zero for validator or lightning stake deposit!").
			WithErrorCode(result.CodeInvalidStake)
	}

	if tx.Purpose == core.StakeForEliteEdgeNode && stake.SCPTWei.Cmp(types.Zero) != 0 {
		return result.Error("Script has to be zero for elite edge node stake deposit!").
			WithErrorCode(result.CodeInvalidStake)
	}

	// Minimum stake deposit requirement to avoid spamming
	if tx.Purpose == core.StakeForValidator {
		minValidatorStake := core.MinValidatorStakeDeposit
		//if blockHeight >= common.HeightValidatorStakeChangedTo200K {
		//	minValidatorStake = core.MinValidatorStakeDeposit200K
		//}
		if stake.SCPTWei.Cmp(minValidatorStake) < 0 {
			return result.Error("Insufficient amount of stake, at least %v SCPTWei is required for each validator deposit", minValidatorStake).
				WithErrorCode(result.CodeInsufficientStake)
		}
	}

	if tx.Purpose == core.StakeForLightning {
		minLightningStake := core.MinLightningStakeDeposit
		//if blockHeight >= common.HeightLowerGNStakeThresholdTo1000 {
		//	minLightningStake = core.MinLightningStakeDeposit1000
		//}
		if stake.SCPTWei.Cmp(minLightningStake) < 0 {
			return result.Error("Insufficient amount of stake, at least %v SCPTWei is required for each lightning deposit", minLightningStake).
				WithErrorCode(result.CodeInsufficientStake)
		}
	}

	if tx.Purpose == core.StakeForEliteEdgeNode {
		if blockHeight < common.HeightEnableScript3 {
			return result.Error(fmt.Sprintf("Elite Edge Node staking not enabled yet, please wait until block height %v", common.HeightEnableScript3)).WithErrorCode(result.CodeGenericError)
		}

		minEliteEdgeNodeStake := core.MinEliteEdgeNodeStakeDeposit
		maxEliteEdgeNodeStake := core.MaxEliteEdgeNodeStakeDeposit

		if stake.SCPTWei.Cmp(big.NewInt(0)) > 0 {
			return result.Error("Only SPAY can be deposited for elite edge nodes").
				WithErrorCode(result.CodeStakeExceedsCap)
		}

		if stake.SPAYWei.Cmp(minEliteEdgeNodeStake) < 0 {
			return result.Error("Insufficient amount of stake, at least %v SPAYWei is required for each elite edge node deposit", minEliteEdgeNodeStake).
				WithErrorCode(result.CodeInsufficientStake)
		}

		eenAddr := tx.Holder.Address
		currentStake := exec.getEliteEdgeNodeStake(view, eenAddr)
		expectedStake := big.NewInt(0).Add(currentStake, stake.SPAYWei)
		if expectedStake.Cmp(maxEliteEdgeNodeStake) > 0 {
			return result.Error("Stake exceeds the cap, at most %v SPAYWei can be deposited to each elite edge node", maxEliteEdgeNodeStake).
				WithErrorCode(result.CodeStakeExceedsCap)
		}
	}

	minimalBalance := stake.Plus(tx.Fee)
	if !sourceAccount.Balance.IsGTE(minimalBalance) {
		logger.Infof(fmt.Sprintf("DepositStake: Source did not have enough balance %v", tx.Source.Address.Hex()))
		return result.Error("DepositStake: Source balance is %v, but required minimal balance is %v",
			sourceAccount.Balance, minimalBalance).WithErrorCode(result.CodeInsufficientStake)
	}

	return result.OK
}

func (exec *DepositStakeExecutor) process(chainID string, view *st.StoreView, viewSel core.ViewSelector, transaction types.Tx) (common.Hash, result.Result) {
	blockHeight := view.Height() + 1 // the view points to the parent of the current block
    logger.Debugf("TR-job309_REWARDS 00001 tx_deposit_stake. minLightningStake. blockHeight %v", blockHeight)

	tx := exec.castTx(transaction)

	sourceAccount, success := getInput(view, tx.Source)
	if success.IsError() {
		return common.Hash{}, result.Error("Failed to get the source account")
	}

	if !chargeFee(sourceAccount, tx.Fee) {
		return common.Hash{}, result.Error("Failed to charge transaction fee")
	}

	stake := tx.Source.Coins.NoNil()
	if !sourceAccount.Balance.IsGTE(stake) {
		return common.Hash{}, result.Error("Not enough balance to stake").WithErrorCode(result.CodeNotEnoughBalanceToStake)
	}

	sourceAddress := tx.Source.Address
	holderAddress := tx.Holder.Address

	if tx.Purpose == core.StakeForValidator {
		sourceAccount.Balance = sourceAccount.Balance.Minus(stake)
		stakeAmount := stake.SCPTWei
		vcp := view.GetValidatorCandidatePool()
		err := vcp.DepositStake(sourceAddress, holderAddress, stakeAmount, blockHeight)
		if err != nil {
			return common.Hash{}, result.Error("Failed to deposit stake, err: %v", err)
		}
		view.UpdateValidatorCandidatePool(vcp)
	} else if tx.Purpose == core.StakeForLightning {
        logger.Debugf("TR-job309_REWARDS 00007 tx.Purpose = core.StakeForLightning")
		sourceAccount.Balance = sourceAccount.Balance.Minus(stake)
		stakeAmount := stake.SCPTWei
		gcp := view.GetLightningCandidatePool()

		if !gcp.Contains(holderAddress) {
			checkBLSRes := exec.checkBLSSummary(tx)
			if checkBLSRes.IsError() {
				return common.Hash{}, checkBLSRes
			}
		}
        logger.Debugf("TR-job309_REWARDS 00008 gcp.DepositStake")

		err := gcp.DepositStake(sourceAddress, holderAddress, stakeAmount, tx.BlsPubkey, blockHeight)
		if err != nil {
            logger.Debugf("TR-job309_REWARDS 00011 tx_deposit_stake")
			return common.Hash{}, result.Error("Failed to deposit stake, err: %v", err)
		}
        logger.Debugf("TR-job309_REWARDS 00012 tx_deposit_stake -> UpdateLightningCandidatePool gcp.Len %v", gcp.Len())
        
		view.UpdateLightningCandidatePool(gcp)
	} else if tx.Purpose == core.StakeForEliteEdgeNode {
		sourceAccount.Balance = sourceAccount.Balance.Minus(stake)
		stakeAmount := stake.SPAYWei // elite edge node deposits SPAY
		eenp := state.NewEliteEdgeNodePool(view, false)

		if !eenp.Contains(holderAddress) {
			checkBLSRes := exec.checkBLSSummary(tx)
			if checkBLSRes.IsError() {
				return common.Hash{}, checkBLSRes
			}
		}

		err := eenp.DepositStake(sourceAddress, holderAddress, stakeAmount, tx.BlsPubkey, blockHeight)
		if err != nil {
			return common.Hash{}, result.Error("Failed to deposit stake, err: %v", err)
		}
	} else {
		return common.Hash{}, result.Error("Invalid staking purpose").WithErrorCode(result.CodeInvalidStakePurpose)
	}

	// Only update stake transaction height list for validator stake tx.
	if tx.Purpose == core.StakeForValidator {
		hl := view.GetStakeTransactionHeightList()
		if hl == nil {
			hl = &types.HeightList{}
		}
		blockHeight := view.Height() + 1 // the view points to the parent of the current block
		hl.Append(blockHeight)
		view.UpdateStakeTransactionHeightList(hl)
	}

	sourceAccount.Sequence++
	view.SetAccount(sourceAddress, sourceAccount)

	txHash := types.TxID(chainID, tx)
	return txHash, result.OK
}

func (exec *DepositStakeExecutor) checkBLSSummary(tx *types.DepositStakeTxV2) result.Result {
	if tx.BlsPubkey.IsEmpty() {
		return result.Error("Must provide BLS Pubkey")
	}
	if tx.BlsPop.IsEmpty() {
		return result.Error("Must provide BLS POP")
	}
	if tx.HolderSig == nil || tx.HolderSig.IsEmpty() {
		return result.Error("Must provide Holder Signature")
	}

	if !tx.HolderSig.Verify(tx.BlsPop.ToBytes(), tx.Holder.Address) {
		return result.Error("BLS key info is not properly signed")
	}

	if !tx.BlsPop.PopVerify(tx.BlsPubkey) {
		return result.Error("BLS pop is invalid")
	}

	return result.OK
}

func (exec *DepositStakeExecutor) getEliteEdgeNodeStake(view *st.StoreView, eenAddr common.Address) *big.Int {
	eenp := state.NewEliteEdgeNodePool(view, true)

	een := eenp.Get(eenAddr)
	if een != nil {
		return een.TotalStake()
	}

	return big.NewInt(0)
}

func (exec *DepositStakeExecutor) getTxInfo(transaction types.Tx) *core.TxInfo {
	tx := exec.castTx(transaction)
	return &core.TxInfo{
		Address:           tx.Source.Address,
		Sequence:          tx.Source.Sequence,
		EffectiveGasPrice: exec.calculateEffectiveGasPrice(transaction),
	}
}

func (exec *DepositStakeExecutor) calculateEffectiveGasPrice(transaction types.Tx) *big.Int {
	tx := exec.castTx(transaction)
	fee := tx.Fee
	gas := new(big.Int).SetUint64(getRegularTxGas(exec.state))
	effectiveGasPrice := new(big.Int).Div(fee.SPAYWei, gas)
	return effectiveGasPrice
}

func (exec *DepositStakeExecutor) castTx(transaction types.Tx) *types.DepositStakeTxV2 {
	if tx, ok := transaction.(*types.DepositStakeTxV2); ok {
		return tx
	}
	if tx, ok := transaction.(*types.DepositStakeTx); ok {
		return &types.DepositStakeTxV2{
			Fee:     tx.Fee,
			Source:  tx.Source,
			Holder:  tx.Holder,
			Purpose: tx.Purpose,
		}
	}
	panic("Unreachable code")
}
