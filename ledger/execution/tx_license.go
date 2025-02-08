package execution

import (
	"math/big"

	"github.com/scripttoken/script/common"
	"github.com/scripttoken/script/common/result"
	"github.com/scripttoken/script/core"
	st "github.com/scripttoken/script/ledger/state"
	"github.com/scripttoken/script/ledger/types"
)

var _ TxExecutor = (*LicenseTxExecutor)(nil)

// --------------------------------- License Transaction ----------------------------------

type LicenseTxExecutor struct {
	state *st.LedgerState
}

func NewLicenseTxExecutor(state *st.LedgerState) *LicenseTxExecutor {
	return &LicenseTxExecutor{
		state: state,
	}
}

func (exec *LicenseTxExecutor) sanityCheck(chainID string, view *st.StoreView, viewSel core.ViewSelector, transaction types.Tx) result.Result {
	tx := transaction.(*types.LicenseTx)
	res := tx.ValidateBasic()
	if res.IsError() {
		return res
	}
	signBytes := tx.SignBytes(chainID)
	if !tx.Signature.Verify(signBytes, core.LicenseIssuer) {
		return result.Error("Signature verification failed for issuer: %X", signBytes)
	}
	return result.OK
}

func (exec *LicenseTxExecutor) process(chainID string, view *st.StoreView, viewSel core.ViewSelector, transaction types.Tx) (common.Hash, result.Result) {
	tx := transaction.(*types.LicenseTx)

	if tx.Op == "authorize" {
		if tx.Type == "VN" {
			core.AddValidator(tx.Address)
		} else if tx.Type == "LN" {
			core.AddLightning(tx.Address)
		} else {
			return common.Hash{}, result.Error("Unknown license type: %v", tx.Type)
		}
	} else if tx.Op == "revoke" {
		if tx.Type == "VN" {
			core.RemoveValidator(tx.Address)
		} else if tx.Type == "LN" {
			core.RemoveLightning(tx.Address)
		} else {
			return common.Hash{}, result.Error("Unknown license type: %v", tx.Type)
		}
	} else {
		return common.Hash{}, result.Error("Unknown operation: %v", tx.Op)
	}
	txHash := types.TxID(chainID, tx)
	return txHash, result.OK
}

func (exec *LicenseTxExecutor) getTxInfo(transaction types.Tx) *core.TxInfo {
	tx := transaction.(*types.LicenseTx)
	return &core.TxInfo{
		Address:           tx.Address,
		Sequence:          uint64(0),
		EffectiveGasPrice: new(big.Int).SetUint64(0),
	}
}
