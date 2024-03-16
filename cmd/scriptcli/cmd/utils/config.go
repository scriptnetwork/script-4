package utils

import "github.com/spf13/viper"

const (
	CfgRemoteRPCEndpoint = "remoteRPCEndpoint"
	CfgDebug             = "debug"
)

func init() {
	viper.SetDefault(CfgRemoteRPCEndpoint, "http://localhost:15672/rpc")
	viper.SetDefault(CfgDebug, false)
}
