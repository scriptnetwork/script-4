package query

import (
	"encoding/json"
	"fmt"

	"github.com/scripttoken/script/cmd/scriptcli/cmd/utils"
	"github.com/scripttoken/script/rpc"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	rpcc "github.com/ybbus/jsonrpc"
)

// accountCmd represents the account command.
// Example:
//
//	scriptcli query accounts
var accountsCmd = &cobra.Command{
	Use:     "accounts",
	Short:   "Get all accounts",
	Long:    `Get all accounts.`,
	Example: `scriptcli query accounts`,
	Run:     doAccountsCmd,
}

func doAccountsCmd(cmd *cobra.Command, args []string) {
	client := rpcc.NewRPCClient(viper.GetString(utils.CfgRemoteRPCEndpoint))

	res, err := client.Call("script.GetAccounts", rpc.GetAccountsArgs{})
	if err != nil {
		utils.Error("Failed to get accounts: %v\n", err)
	}
	if res.Error != nil {
		utils.Error("Failed to get accounts: %v\n", res.Error)
	}

	json, err := json.MarshalIndent(res.Result, "", "    ")
	if err != nil {
		utils.Error("Failed to parse server response: %v\n%v\n", err, string(json))
	}
	fmt.Println(string(json))
}

func init() {
}
