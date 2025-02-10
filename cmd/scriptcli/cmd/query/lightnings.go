package query

import (
	"encoding/json"
	"fmt"

	"github.com/scripttoken/script/cmd/scriptcli/cmd/utils"
	"github.com/scripttoken/script/common"
	"github.com/scripttoken/script/rpc"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	rpcc "github.com/ybbus/jsonrpc"
)

// gcpCmd represents the lightnings command.
// Example:
//
//	scriptcli query lightnings --height=10
var lightningsCmd = &cobra.Command{
	Use:     "lightnings",
	Short:   "Get lightnings",
	Example: `scriptcli query lightnings --height=10`,
	Run:     doLightningsCmd,
}

func doLightningsCmd(cmd *cobra.Command, args []string) {
	client := rpcc.NewRPCClient(viper.GetString(utils.CfgRemoteRPCEndpoint))

	height := heightFlag
	res, err := client.Call("script.GetLightningsByHeight", rpc.GetLightningsByHeightArgs{Height: common.JSONUint64(height)})
	if err != nil {
		utils.Error("Failed to get lightnings: %v\n", err)
	}
	if res.Error != nil {
		utils.Error("Failed to get lightnings: %v\n", res.Error)
	}
	json, err := json.MarshalIndent(res.Result, "", "    ")
	if err != nil {
		utils.Error("Failed to parse server response: %v\n%s\n", err, string(json))
	}
	fmt.Println(string(json))
}

func init() {
	lightningsCmd.Flags().Uint64Var(&heightFlag, "height", uint64(0), "height of the block")
	lightningsCmd.MarkFlagRequired("height")
}
