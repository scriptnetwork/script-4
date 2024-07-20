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

// lightningCmd retreves lightning related information from Script server.
// Example:
//		scriptcli query lightning
var lightningCmd = &cobra.Command{
	Use:     "lightning",
	Short:   "Get lightning info",
	Long:    `Get lightning status.`,
	Example: `scriptcli query lightning`,
	Run:     doLightningCmd,
}

type LightningResult struct {
	Address   string
	BlsPubkey string
	BlsPop    string
	Signature string
	Summary   string
}

func doLightningCmd(cmd *cobra.Command, args []string) {
//	cfgPath := cmd.Flag("config").Value.String()
//panic(fmt.Errorf("fatal error config file: %w", cfgPath))

	client := rpcc.NewRPCClient(viper.GetString(utils.CfgRemoteRPCEndpoint))

	res, err := client.Call("script.GetLightningInfo", rpc.GetLightningInfoArgs{})
	if err != nil {
		utils.Error("Failed to get lightning info: %v\n", err)
	}
	if res.Error != nil {
		utils.Error("Failed to get lightning info: %v\n", res.Error)
	}
	result := res.Result.(map[string]interface{})
	address, ok := result["Address"].(string)
	if !ok {
		json, err := json.MarshalIndent(res.Result, "", "    ")
		utils.Error("Failed to parse server response: %v\n%v\n", err, string(json))
	}
	blsPubkey, ok := result["BLSPubkey"].(string)
	if !ok {
		json, err := json.MarshalIndent(res.Result, "", "    ")
		utils.Error("Failed to parse server response: %v\n%v\n", err, string(json))
	}
	blsPop, ok := result["BLSPop"].(string)
	if !ok {
		json, err := json.MarshalIndent(res.Result, "", "    ")
		utils.Error("Failed to parse server response: %v\n%v\n", err, string(json))
	}
	sig, ok := result["Signature"].(string)
	if !ok {
		json, err := json.MarshalIndent(res.Result, "", "    ")
		utils.Error("Failed to parse server response: %v\n%v\n", err, string(json))
	}
	output := &LightningResult{
		Address:   address,
		BlsPubkey: blsPubkey,
		BlsPop:    blsPop,
		Signature: sig,
	}
	output.Summary = address + blsPubkey + blsPop + sig
	json, err := json.MarshalIndent(output, "", "    ")
	if err != nil {
		utils.Error("Failed to parse server response: %v\n%v\n", err, string(json))
	}
	fmt.Println(string(json))
}

func init() {}
