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

	res := tx.Issuer.ValidateBasic()
	if res.IsError() {
		return res
	}

	issuerAccount, res := getInput(view, tx.Issuer)
	if res.IsError() {
		return res
	}

	//Compare issuer with secrets here

	signBytes := tx.SignBytes(chainID)
	if !tx.Issuer.Signature.Verify(signBytes, issuerAccount.Address) {
		return result.Error("Signature verification failed for issuer: %X", signBytes)
	}

	for _, license := range tx.Licenses {
		if license.Issuer.IsEmpty() || license.Licensee.IsEmpty() || len(license.Items) == 0 {
			return result.Error("Invalid license information: %+v", license)
		}
	}

	return result.OK
}

func (exec *LicenseTxExecutor) process(chainID string, view *st.StoreView, viewSel core.ViewSelector, transaction types.Tx) (common.Hash, result.Result) {
	tx := transaction.(*types.LicenseTx)

	issuerAccount := view.GetAccount(tx.Issuer.Address)
	if issuerAccount == nil {
		return common.Hash{}, result.Error("Issuer account %v does not exist!", tx.Issuer.Address)
	}


	for _, license := range tx.Licenses {
		err := core.WriteLicenseFile(license, "")
		if err != nil {
			return common.Hash{}, result.Error("Error writing license to file: %v\n", err)
		}
	}

	_, err := core.ReadLicenseFile("")
	if err != nil {
		return common.Hash{}, result.Error("Error re-reading license file: %v\n", err)
	}

	//Deduct trx fee
	issuerAccount.Balance = issuerAccount.Balance.Minus(tx.Fee)

	view.SetAccount(tx.Issuer.Address, issuerAccount)

	txHash := types.TxID(chainID, tx)
	return txHash, result.OK

}

func (exec *LicenseTxExecutor) getTxInfo(transaction types.Tx) *core.TxInfo {
	tx := transaction.(*types.LicenseTx)
	return &core.TxInfo{
		Address:           tx.Issuer.Address,
		Sequence:          tx.Issuer.Sequence,
		EffectiveGasPrice: exec.calculateEffectiveGasPrice(transaction),
	}
}

func (exec *LicenseTxExecutor) calculateEffectiveGasPrice(transaction types.Tx) *big.Int {
	return new(big.Int).SetUint64(0)
}



