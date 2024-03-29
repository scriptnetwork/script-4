package key

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/scripttoken/script/cmd/scriptcli/cmd/utils"
	"github.com/scripttoken/script/common"
	"github.com/scripttoken/script/wallet"
	wtypes "github.com/scripttoken/script/wallet/types"
)

// passwordCmd updates the password for the key corresponding to the given address
var passwordCmd = &cobra.Command{
	Use:     "password",
	Short:   "Change the password for a key",
	Long:    `Change the password for a key.`,
	Example: "scriptcli key password 1d8E1191E0a97C1aDa4940B79188D3B1f6f5C695",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			utils.Error("Usage: scriptcli key password <address>\n")
		}
		address := common.HexToAddress(args[0])

		cfgPath := cmd.Flag("config").Value.String()
		wallet, err := wallet.OpenWallet(cfgPath, wtypes.WalletTypeSoft, true)
		if err != nil {
			utils.Error("Failed to open wallet: %v\n", err)
		}

		prompt := fmt.Sprintf("Please enter the current password: ")
		oldPassword, err := utils.GetPassword(prompt)
		if err != nil {
			utils.Error("Failed to get password: %v\n", err)
		}

		prompt = fmt.Sprintf("Please enter a new password: ")
		newPassword, err := utils.GetPassword(prompt)
		if err != nil {
			utils.Error("Failed to get password: %v\n", err)
		}

		err = wallet.UpdatePassword(address, oldPassword, newPassword)
		if err != nil {
			utils.Error("Failed to update password: %v\n", err)
		}

		fmt.Printf("Password updated successfully\n")
	},
}
