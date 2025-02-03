package tx

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"encoding/json"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/scripttoken/script/cmd/scriptcli/cmd/utils"
	"github.com/scripttoken/script/core"
	"github.com/scripttoken/script/ledger/types"
	"github.com/scripttoken/script/rpc"
	wtypes "github.com/scripttoken/script/wallet/types"

	"github.com/ybbus/jsonrpc"
	rpcc "github.com/ybbus/jsonrpc"
)

var licenseCmd = &cobra.Command{
	Use:     "license",
	Short:   "Create a License transaction",
	Example: `scriptcli tx license --chain="scriptnet" --from=2E833968E5bB786Ae419c4d13189fB081Cc43bab --license='[{"issuer":"2E833968E5bB786Ae419c4d13189fB081Cc43bab","licensee":"5A2C2C8D4D2C6C8B7C5D4F8A6D7C6E6A4E3B2B3A","from":1696732800,"to":1702099200,"items":["VN"],"signature":"signature_example"}]'`,
	Run:     doLicenseCmd,
}

func doLicenseCmd(cmd *cobra.Command, args []string) {
	walletType := getWalletType(cmd)
	if walletType == wtypes.WalletTypeSoft && len(fromFlag) == 0 {
		 utils.Error("The from address cannot be empty")
		 return
	}

	if len(licenseFlag) == 0 {
		 utils.Error("The license file path cannot be empty")
		 return
	}

	wallet, fromAddress, err := walletUnlockWithPath(cmd, fromFlag, pathFlag, passwordFlag)
    if err != nil || wallet == nil {
        return
    }
    defer wallet.Lock(fromAddress)

	var licenses []core.License
	if err := json.Unmarshal([]byte(licenseFlag), &licenses); err != nil {
		utils.Error("Failed to parse license JSON: %v\n", err)
		return
	}

	 licenseTx := &types.LicenseTx{
	    Fee: types.Coins{
	        SCPTWei: new(big.Int).SetUint64(0), // Assuming no script fee for now
	        SPAYWei: new(big.Int).SetUint64(0), // Set appropriate fee here
	    },
	    Licenses: licenses, // Use the parsed licenses
	    Issuer: types.TxInput{
	        Address: fromAddress, // Set the issuer's address
	    },
}

  sig, err := wallet.Sign(fromAddress, licenseTx.SignBytes(chainIDFlag))
  if err != nil {
		utils.Error("Failed to sign transaction: %v\n", err)
		return
  }
  licenseTx.SetSignature(fromAddress, sig)

  raw, err := types.TxToBytes(licenseTx)
  if err != nil {
		utils.Error("Failed to encode transaction: %v\n", err)
		return
  }
  signedTx := hex.EncodeToString(raw)

  client := rpcc.NewRPCClient(viper.GetString(utils.CfgRemoteRPCEndpoint))

  var res *jsonrpc.RPCResponse
  if asyncFlag {
		res, err = client.Call("script.BroadcastRawTransactionAsync", rpc.BroadcastRawTransactionArgs{TxBytes: signedTx})
  } else {
		res, err = client.Call("script.BroadcastRawTransaction", rpc.BroadcastRawTransactionArgs{TxBytes: signedTx})
  }

  if err != nil {
		utils.Error("Failed to broadcast transaction: %v\n", err)
		return
  }
  if res.Error != nil {
		utils.Error("Server returned error: %v\n", res.Error)
		return
  }

  result := &rpc.BroadcastRawTransactionResult{}
    err = res.GetObject(result)
    if err != nil {
        utils.Error("Failed to parse server response: %v\n", err)
        return
    }
    formatted, err := json.MarshalIndent(result, "", "    ")
    if err != nil {
        utils.Error("Failed to parse server response: %v\n", err)
        return
    }
    fmt.Printf("Successfully broadcasted transaction:\n%s\n", formatted)
}

func init() {
	licenseCmd.Flags().StringVar(&chainIDFlag, "chain", "", "Chain ID")
	licenseCmd.Flags().StringVar(&fromFlag, "from", "", "Address to send from")
	licenseCmd.Flags().StringVar(&licenseFlag, "license", "", "License in json")
	licenseCmd.Flags().Uint64Var(&seqFlag, "seq", 0, "Sequence number of the transaction")
	licenseCmd.Flags().StringVar(&feeFlag, "fee", fmt.Sprintf("%dwei", types.MinimumTransactionFeeSPAYWeiJune2021), "Fee")
	licenseCmd.Flags().StringVar(&walletFlag, "wallet", "soft", "Wallet type (soft|nano|trezor)")
	licenseCmd.Flags().BoolVar(&asyncFlag, "async", false, "Block until tx has been included in the blockchain")
	licenseCmd.Flags().StringVar(&passwordFlag, "password", "", "Password to unlock the wallet")

	licenseCmd.MarkFlagRequired("chain")
	licenseCmd.MarkFlagRequired("from")
	licenseCmd.MarkFlagRequired("license")
}