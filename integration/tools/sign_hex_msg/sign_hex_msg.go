package main

import (
	"encoding/hex"
	"flag"
	"fmt"

	"github.com/scripttoken/script/common"
	ks "github.com/scripttoken/script/wallet/softwallet/keystore"
)

// Usage:   sign_hex_msg -signer=<signer_address> -keys_dir=<keys_dir> -msg=<hex_msg_to_be_signed> -encrypted=<true/false>
//
// Example: sign_hex_msg -signer=2E833968E5bB786Ae419c4d13189fB081Cc43bab -keys_dir=$HOME/.scriptcli/keys -msg=02f8a4c78085e8d4a51000f86ff86d942e833968e5
func main() {
	signerAddress, keysDir, message := parseArguments()

	var keystore ks.Keystore
	var err error
	keystore, err = ks.NewKeystorePlain(keysDir)
	if err != nil {
		fmt.Printf("\n[ERROR] Failed to create keystore: %v\n", err)
		return
	}

	key, err := keystore.GetKey(signerAddress)
	if err != nil {
		fmt.Printf("\n[ERROR] Failed to get key: %v\n", err)
		return
	}

	msgHex, err := hex.DecodeString(message)
	if err != nil {
		fmt.Printf("\n[ERROR] message %v is not a hex string: %v\n", message, err)
		return
	}

	signature, err := key.Sign(msgHex)
	if err != nil {
		fmt.Printf("\n[ERROR] Failed sign the message: %v\n", err)
		return
	}

	fmt.Println("")
	fmt.Printf("--------------------------------------------------------------------------\n")
	fmt.Printf("Signature: %v\n", hex.EncodeToString(signature.ToBytes()))
	fmt.Printf("--------------------------------------------------------------------------\n")
	fmt.Println("")
}

func parseArguments() (signerAddress common.Address, keysDir, message string) {
	signerAddressPtr := flag.String("signer", "", "the address of the signer")
	keysDirPtr := flag.String("keys_dir", "./keys", "the folder that contains the keys of the signers")
	messagePtr := flag.String("msg", "", "the message to be signed")

	flag.Parse()

	signerAddress = common.HexToAddress(*signerAddressPtr)
	keysDir = *keysDirPtr
	message = *messagePtr
	return
}
