package tx

import (
	"github.com/spf13/cobra"
)

// Common flags used in Tx sub commands.
var (
	chainIDFlag       string
	fromFlag          string
	toFlag            string
	pathFlag          string
	seqFlag           uint64
	scriptAmountFlag  string
	spayAmountFlag    string
	gasAmountFlag     uint64
	feeFlag           string
	resourceIDsFlag   []string
	resourceIDFlag    string
	durationFlag      uint64
	reserveSeqFlag    uint64
	addressesFlag     []string
	valueFlag         string
	gasPriceFlag      string
	gasLimitFlag      uint64
	dataFlag          string
	walletFlag        string
	stakeInScriptFlag string
	purposeFlag       uint8
	sourceFlag        string
	asyncFlag         bool
	licenseFlag       string
)

// TxCmd represents the Tx command
var TxCmd = &cobra.Command{
	Use:   "tx",
	Short: "Manage transactions",
	Long:  `Manage transactions.`,
}

func init() {
	TxCmd.AddCommand(sendCmd)
	TxCmd.AddCommand(smartContractCmd)
	TxCmd.AddCommand(licenseCmd)
}
