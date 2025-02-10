package execution

import (
	log "github.com/sirupsen/logrus"

	"github.com/scripttoken/script/blockchain"
	"github.com/scripttoken/script/common"
	"github.com/scripttoken/script/common/result"
	"github.com/scripttoken/script/core"
	st "github.com/scripttoken/script/ledger/state"
	"github.com/scripttoken/script/ledger/types"
	"github.com/scripttoken/script/store/database"
)

var logger *log.Entry = log.WithFields(log.Fields{"prefix": "ledger"})

// TxExecutor defines the interface of the transaction executors
type TxExecutor interface {
	sanityCheck(chainID string, view *st.StoreView, viewSel core.ViewSelector, transaction types.Tx) result.Result
	process(chainID string, view *st.StoreView, viewSel core.ViewSelector, transaction types.Tx) (common.Hash, result.Result)
	getTxInfo(transaction types.Tx) *core.TxInfo
}

// Executor executes the transactions
type Executor struct {
	db        database.Database
	chain     *blockchain.Chain
	state     *st.LedgerState
	consensus core.ConsensusEngine
	valMgr    core.ValidatorManager
	ledger    core.Ledger

	coinbaseTxExec      *CoinbaseTxExecutor
	sendTxExec          *SendTxExecutor
	smartContractTxExec *SmartContractTxExecutor
	licenseTxExec       *LicenseTxExecutor

	skipSanityCheck bool
}

// NewExecutor creates a new instance of Executor
func NewExecutor(db database.Database, chain *blockchain.Chain, state *st.LedgerState, consensus core.ConsensusEngine, valMgr core.ValidatorManager, ledger core.Ledger) *Executor {
	executor := &Executor{
		db:                  db,
		chain:               chain,
		state:               state,
		consensus:           consensus,
		valMgr:              valMgr,
		coinbaseTxExec:      NewCoinbaseTxExecutor(db, chain, state, consensus, valMgr),
		sendTxExec:          NewSendTxExecutor(state),
		smartContractTxExec: NewSmartContractTxExecutor(chain, state, ledger),
		licenseTxExec:       NewLicenseTxExecutor(state),
		skipSanityCheck:     false,
	}

	return executor
}

// SetSkipSanityCheck sets the flag for sanity check.
// Skip checks while replaying commmitted blocks.
func (exec *Executor) SetSkipSanityCheck(skip bool) {
	exec.skipSanityCheck = skip
}

// ExecuteTx executes the given transaction
func (exec *Executor) ExecuteTx(tx types.Tx) (common.Hash, result.Result) {
	return exec.processTx(tx, core.DeliveredView)
}

// CheckTx checks the validity of the given transaction
func (exec *Executor) CheckTx(tx types.Tx) (common.Hash, result.Result) {
	return exec.processTx(tx, core.CheckedView)
}

// ScreenTx checks the validity of the given transaction
func (exec *Executor) ScreenTx(tx types.Tx) (common.Hash, result.Result) {
	return exec.processTx(tx, core.ScreenedView)
}

// GetTxInfo extracts tx information used by mempool to sort Txs.
func (exec *Executor) GetTxInfo(tx types.Tx) (*core.TxInfo, result.Result) {
	txExecutor := exec.getTxExecutor(tx)
	if txExecutor == nil {
		return nil, result.Error("Unknown tx type")
	}

	txInfo := txExecutor.getTxInfo(tx)
	return txInfo, result.OK
}

// processTx contains the main logic to process the transaction. If the tx is invalid, a TMSP error will be returned.
func (exec *Executor) processTx(tx types.Tx, viewSel core.ViewSelector) (common.Hash, result.Result) {
	chainID := exec.state.GetChainID()
	var view *st.StoreView
	switch viewSel {
	case core.DeliveredView:
		view = exec.state.Delivered()
	case core.CheckedView:
		view = exec.state.Checked()
	default:
		view = exec.state.Screened()
	}

	sanityCheckResult := exec.sanityCheck(chainID, view, viewSel, tx)
	if sanityCheckResult.IsError() {
		return common.Hash{}, sanityCheckResult
	}

	txHash, processResult := exec.process(chainID, view, viewSel, tx)
	return txHash, processResult
}

func (exec *Executor) sanityCheck(chainID string, view *st.StoreView, viewSel core.ViewSelector, tx types.Tx) result.Result {
	if exec.skipSanityCheck { // Skip checks, e.g. while replaying commmitted blocks.
		return result.OK
	}

	var sanityCheckResult result.Result
	txExecutor := exec.getTxExecutor(tx)
	if txExecutor != nil {
		sanityCheckResult = txExecutor.sanityCheck(chainID, view, viewSel, tx)
	} else {
		sanityCheckResult = result.Error("Unknown tx type")
	}

	return sanityCheckResult
}

func (exec *Executor) process(chainID string, view *st.StoreView, viewSel core.ViewSelector, tx types.Tx) (common.Hash, result.Result) {
	var processResult result.Result
	var txHash common.Hash

	txExecutor := exec.getTxExecutor(tx)
	if txExecutor != nil {
		txHash, processResult = txExecutor.process(chainID, view, viewSel, tx)
		if processResult.IsError() {
			logger.Warnf("Tx processing error: %v", processResult.Message)
		}
	} else {
		processResult = result.Error("Unknown tx type")
	}

	return txHash, processResult
}

func (exec *Executor) getTxExecutor(tx types.Tx) TxExecutor {
	var txExecutor TxExecutor
	switch tx.(type) {
	case *types.CoinbaseTx:
		txExecutor = exec.coinbaseTxExec
	case *types.SendTx:
		txExecutor = exec.sendTxExec
	case *types.SmartContractTx:
		txExecutor = exec.smartContractTxExec
	case *types.LicenseTx:
		txExecutor = exec.licenseTxExec
	default:
		txExecutor = nil
	}
	return txExecutor
}
