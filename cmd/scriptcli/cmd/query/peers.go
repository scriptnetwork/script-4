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

// peersCmd represents the peers command.
// Example:
//		scriptcli query peers
var peersCmd = &cobra.Command{
	Use:     "peers",
	Short:   "Get currently connected peers",
	Long:    `Get currently connected peers.`,
	Example: `scriptcli query peers`,
	Run: func(cmd *cobra.Command, args []string) {
		client := rpcc.NewRPCClient(viper.GetString(utils.CfgRemoteRPCEndpoint))

		res, err := client.Call("script.GetPeers", rpc.GetPeersArgs{
			SkipEdgeNode: skipEdgeNodeFlag,
		})
		if err != nil {
			utils.Error("Failed to get peers: %v\n", err)
		}
		if res.Error != nil {
			utils.Error("Failed to retrieve peers: %v\n", res.Error)
		}
		json, err := json.MarshalIndent(res.Result, "", "    ")
		if err != nil {
			utils.Error("Failed to parse server response: %v\n%v\n", err, string(json))
		}
		fmt.Println(string(json))
	},
}

func init() {
	peersCmd.Flags().BoolVar(&skipEdgeNodeFlag, "skip_edge_node", true, "skip peer edge nodes")
}
