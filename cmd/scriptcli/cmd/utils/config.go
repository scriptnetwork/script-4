package utils

import "github.com/spf13/viper"

const (
	CfgRemoteRPCEndpoint = "remoteRPCEndpoint"
	CfgDebug             = "CfgDebug"
        CfgChainID           = "chainID"
        CfgEthChainID        = "ethChainID"
)

func init() {
	viper.SetDefault(CfgRemoteRPCEndpoint, "http://127.0.0.1:10002/rpc")
	viper.SetDefault(CfgDebug, false)
	viper.SetDefault(CfgChainID, "unset")
	viper.SetDefault(CfgEthChainID, 0)
}
