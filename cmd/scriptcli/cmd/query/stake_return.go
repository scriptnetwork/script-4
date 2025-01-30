package query

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/scripttoken/script/cmd/scriptcli/cmd/utils"
	"github.com/scripttoken/script/common"
	"github.com/scripttoken/script/rpc"

	rpcc "github.com/ybbus/jsonrpc"
)

// stakeReturnsCmd represents the stake_return command.
// Example:
//		scriptcli query stake_returns --height=10
var stakeReturnsCmd = &cobra.Command{
	Use:     "stake_returns",
	Short:   "Get stake returns",
	Example: `scriptcli query stake_returns, scriptcli query stake_returns --height=800`,
	Run:     doStakeReturnsCmd,
}

func doStakeReturnsCmd(cmd *cobra.Command, args []string) {
	client := rpcc.NewRPCClient(viper.GetString(utils.CfgRemoteRPCEndpoint))

	purpose := purposeFlag
	if purpose != 2 {
		fmt.Println("Only support querying stake return for elite edge nodes (purpose=2) for now")
		return
	}

	height := heightFlag
	var res *rpcc.RPCResponse
	var err error
	if height == 0 {
		res, err = client.Call("script.GetAllPendingEliteEdgeNodeStakeReturns", rpc.GetAllPendingEliteEdgeNodeStakeReturnsArgs{})
	} else {
		res, err = client.Call("script.GetEliteEdgeNodeStakeReturnsByHeight", rpc.GetEliteEdgeNodeStakeReturnsByHeightArgs{Height: common.JSONUint64(height)})
	}
	if err != nil {
		utils.Error("Failed to get stake returns: %v\n", err)
	}
	if res.Error != nil {
		utils.Error("Failed to get stake returns: %v\n", res.Error)
	}
	json, err := json.MarshalIndent(res.Result, "", "    ")
	if err != nil {
		utils.Error("Failed to parse server response: %v\n%s\n", err, string(json))
	}
	fmt.Println(string(json))
}

func init() {
	stakeReturnsCmd.Flags().Uint8Var(&purposeFlag, "purpose", uint8(2), "purpose of the stake return query, validator_node=0, lightning_node=1, elite_edge_node=2")
	stakeReturnsCmd.Flags().Uint64Var(&heightFlag, "height", uint64(0), "height of the block, if height=0 the command returns all the pending stake returns")
	//stakeReturnsCmd.MarkFlagRequired("height")
}
