package rpc

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/viper"
	"github.com/scripttoken/script/cmd/scriptcli/cmd/utils"
	"github.com/scripttoken/script/rpc"
	rpcc "github.com/ybbus/jsonrpc"
)

// ------------------------------- Query Lightning -----------------------------------
type LightningInfoArgs struct {}

type LightningInfoResult struct {
	BLSPubkey string `json:"bls_pubkey"`
	BLSPop    string `json:"bls_pop"`
	Address   string `json:"address"`
	Signature string `json:"signature"`
	Summary string `json:"summary"`
}


func (t *ScriptCliRPCService) LightningInfo(args *LightningInfoArgs, result *LightningInfoResult) (err error) {

	client := rpcc.NewRPCClient(viper.GetString(utils.CfgRemoteRPCEndpoint))

	res, err := client.Call("script.GetLightningInfo", rpc.GetLightningInfoArgs{})
	if err != nil {
		return fmt.Errorf("Failed to get lightning info: %v\n", err)
	}
	if res.Error != nil {
		return fmt.Errorf("Failed to get lightning info: %v\n", res.Error)
	}

	resultMap, ok := res.Result.(map[string]interface{})
	if !ok {
		jsonData, err := json.MarshalIndent(res.Result, "", "    ")
		return fmt.Errorf("Failed to parse server response: %v\n%v", err, string(jsonData))
	}

	address, ok := resultMap["Address"].(string)
	if !ok {
		jsonData, err := json.MarshalIndent(res.Result, "", "    ")
		return fmt.Errorf("Failed to parse server response: %v\n%v\n", err, string(jsonData))
	}
	blsPubkey, ok := resultMap["BLSPubkey"].(string)
	if !ok {
		jsonData, err := json.MarshalIndent(res.Result, "", "    ")
		return fmt.Errorf("Failed to parse server response: %v\n%v\n", err, string(jsonData))
	}
	blsPop, ok := resultMap["BLSPop"].(string)
	if !ok {
		jsonData, err := json.MarshalIndent(res.Result, "", "    ")
		return fmt.Errorf("Failed to parse server response: %v\n%v\n", err, string(jsonData))
	}
	sig, ok := resultMap["Signature"].(string)
	if !ok {
		jsonData, err := json.MarshalIndent(res.Result, "", "    ")
		return fmt.Errorf("Failed to parse server response: %v\n%v\n", err, string(jsonData))
	}
	summary := address + blsPubkey + blsPop + sig
	result.BLSPubkey = blsPubkey
	result.BLSPop = blsPop
	result.Address = address
	result.Signature = sig
	result.Summary = summary

	return nil
}
