package key

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/scripttoken/script/cmd/scriptcli/cmd/utils"
	"github.com/scripttoken/script/wallet"
	wtypes "github.com/scripttoken/script/wallet/types"
)

// newCmd generates a new key
var importCmd = &cobra.Command{
	Use:     "import",
	Short:   "Import a private key",
	Long:    `Import a private key.`,
	Example: "scriptcli key import",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			utils.Error("Usage: scriptcli import <private key>\n")
		}
		cfgPath := cmd.Flag("config").Value.String()
		wallet, err := wallet.OpenWallet(cfgPath, wtypes.WalletTypeSoft, true)
		if err != nil {
			utils.Error("Failed to open wallet: %v\n", err)
		}
		address, err := wallet.ImportKey(args[0])
		if err != nil {
			utils.Error("Failed to import key: %v\n", err)
		}

		fmt.Printf("Successfully imported key: %v\n", address.Hex())
	},
}
