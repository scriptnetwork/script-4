package utils

import "github.com/spf13/viper"

const (
	CfgRemoteRPCEndpoint = "remoteRPCEndpoint"
	CfgDebug             = "debug"
)

func init() {
	viper.SetDefault(CfgRemoteRPCEndpoint, "http://localhost:10001/rpc")
	viper.SetDefault(CfgDebug, false)
}
