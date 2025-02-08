package core

import (
	"encoding/hex"
	"testing"

	"github.com/scripttoken/script/common"
	"github.com/scripttoken/script/crypto"
	"github.com/scripttoken/script/rlp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBlockEncoding(t *testing.T) {
	require := require.New(t)
	/*
		b11 := CreateTestBlock("B1", "")
		b11.Height = 151
		b11.AddTxs([]common.Bytes{common.Hex2Bytes("aaa")})
		b11raw1, _ := rlp.EncodeToBytes(b11)

		hash := b11.Hash().Hex()
		fmt.Printf("Block hash: %s\n", hash)
		fmt.Printf("Block: %s\n", hex.EncodeToString(b11raw1))
	*/
	oldBlockHash := common.HexToHash("0xcd627c7bf28c7b446d7a8e60b165a720f49cd11802042258d2183a1fa5042e57")
	v1, err := hex.DecodeString("f9021cf902168974657374636861696e028197a00000000000000000000000000000000000000000000000000000000000000000e2c0a00000000000000000000000000000000000000000000000000000000000000000a0a2c5e264c417d380c129c80b53f8a2c59c249df85616193e1d98e72423f3b25ba056e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421b9010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000b18467a7291a94662d41cf8d25d51af3d51b7c2e4003962fcfd4beb841d51b8393b3818b9d5ec78e7cb8cf1ba52e5a8d09c707c497360bac29e0f1d928723f319f8c9cd08e2469cb52ca50b07b15a1c246e827e5c47a5ea96b4de7dbaf01c0c0c281aa")

	// Serialized block before Guardian fork.
	//	oldBlockHash := common.HexToHash("0xf1a7fa371f6a108bb4f2ed33de26ac006f0d8cf6a0ed9dc2c1d9547b6cf43cae")
	//v1, err := hex.DecodeString("f90217f902138974657374636861696e0301a035a8f8d3cf9b6da72f72363d53291f9744cab20e420e7e6545235e93a3588e74e2c0a035a8f8d3cf9b6da72f72363d53291f9744cab20e420e7e6545235e93a3588e74a056e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421a056e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421b9010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000b1845dc3ade894c0a4e0c9b349b13b5e882770bfcf20e985691298b841e745098aff2ddbae9aefbb72850f3ff7542dd26bc06252d235f9975a52152a6e257ce7195a31e14b4be0f185b59fd56f5dc3ee444759d16e8ae7964e9370436b00c0")
	require.Nil(err)

	// Should be able to encode/decode blocks before Script2.0 fork.
	b1 := &Block{}
	err = rlp.DecodeBytes(v1, b1)
	require.Nil(err)

	raw, err := rlp.EncodeToBytes(b1)
	require.Nil(err)
	require.Equal(v1, raw)
	// Block hash should remain the same.
	require.Equal(oldBlockHash, b1.Hash())

	// Should be able to encode/decode blocks before Script2.0 fork.
	CreateTestBlock("root", "")
	b2 := CreateTestBlock("b2", "root")
	b2.AddTxs([]common.Bytes{common.Hex2Bytes("aaa")})
	b2raw1, _ := rlp.EncodeToBytes(b2)
	tmp := &Block{}
	err = rlp.DecodeBytes(b2raw1, tmp)
	require.Nil(err)
	b2raw2, _ := rlp.EncodeToBytes(tmp)
	require.Equal(b2raw1, b2raw2)

	// Should be able to encode/decode blocks after Script2.0 fork.
	b2.Height = common.HeightEnableScript2
	b2raw1, _ = rlp.EncodeToBytes(b2)
	err = rlp.DecodeBytes(b2raw1, tmp)
	require.Nil(err)
	b2raw2, _ = rlp.EncodeToBytes(tmp)
	require.Equal(b2raw1, b2raw2)

	// Decode with lightning votes.
	b2.LightningVotes = NewAggregateVotes(b2.Hash(), NewLightningCandidatePool())
	b2raw1, _ = rlp.EncodeToBytes(b2)
	err = rlp.DecodeBytes(b2raw1, tmp)
	require.Nil(err)
	b2raw2, _ = rlp.EncodeToBytes(tmp)
	require.Equal(b2raw1, b2raw2)
	require.Equal(tmp.LightningVotes.Block, b2.LightningVotes.Block)

	// Test ExtendedBlock encoding/decoding
	eb := &ExtendedBlock{}
	eb.Block = b2
	eb.Children = []common.Hash{eb.Hash()}
	eb.Status = BlockStatusCommitted
	eb.HasValidatorUpdate = true
	ebraw1, _ := rlp.EncodeToBytes(eb)

	tmp2 := &ExtendedBlock{}
	err = rlp.DecodeBytes(ebraw1, tmp2)
	require.Nil(err)
	ebraw2, _ := rlp.EncodeToBytes(tmp2)
	require.Equal(ebraw1, ebraw2)

	_, err = rlp.EncodeToBytes(tmp2)
	require.Nil(err)
}

func TestBlockHash(t *testing.T) {
	assert := assert.New(t)

	eb := &ExtendedBlock{}
	assert.Equal(eb.Hash(), common.Hash{})

	eb = &ExtendedBlock{
		Block: &Block{},
	}
	assert.Equal(eb.Hash(), common.Hash{})

	eb = &ExtendedBlock{
		Block: &Block{
			BlockHeader: &BlockHeader{
				Epoch: 1,
			},
		},
	}
	assert.Equal("0x80f3ec4e59cd83a2e8d3041b26a4e5bfed19cf12b418c73fb9255b0c98acf304", eb.Hash().Hex())
}

func TestCreateTestBlock(t *testing.T) {
	assert := assert.New(t)

	b11 := CreateTestBlock("B1", "")
	b12 := CreateTestBlock("b1", "")

	assert.Equal(b11.Hash(), b12.Hash())
}

func TestBlockBasicValidation(t *testing.T) {
	require := require.New(t)
	ResetTestBlocks()

	CreateTestBlock("root", "")
	b1 := CreateTestBlock("B1", "root")
	res := b1.Validate("testchain")
	require.True(res.IsOK())

	res = b1.Validate("anotherchain")
	require.True(res.IsError())
	require.Equal("ChainID mismatch", res.Message)

	oldTS := b1.Timestamp
	b1.Timestamp = nil
	res = b1.Validate("testchain")
	require.True(res.IsError())
	require.Equal("Timestamp is missing", res.Message)
	b1.Timestamp = oldTS

	oldParent := b1.Parent
	b1.Parent = common.Hash{}
	res = b1.Validate("testchain")
	require.True(res.IsError())
	require.Equal("Parent is empty", res.Message)
	b1.Parent = oldParent

	oldProposer := b1.Proposer
	b1.Proposer = common.Address{}
	res = b1.Validate("testchain")
	require.True(res.IsError())
	require.Equal("Proposer is not specified", res.Message)
	b1.Proposer = oldProposer

	oldHCC := b1.HCC
	b1.HCC = CommitCertificate{}
	res = b1.Validate("testchain")
	require.True(res.IsError())
	require.Equal("HCC is empty", res.Message)
	b1.HCC = oldHCC

	oldSig := b1.Signature
	b1.Signature = nil
	res = b1.Validate("testchain")
	require.True(res.IsError())
	require.Equal("Block is not signed", res.Message)
	b1.Signature = oldSig

	oldSig = b1.Signature
	b1.Signature = &crypto.Signature{}
	res = b1.Validate("testchain")
	require.True(res.IsError())
	require.Equal("Block is not signed", res.Message)
	b1.Signature = oldSig

	privKey, _, _ := crypto.GenerateKeyPair()
	sig, _ := privKey.Sign(b1.SignBytes())
	b1.SetSignature(sig)
	res = b1.Validate("testchain")
	require.True(res.IsError())
	require.Equal("Signature verification failed", res.Message)
}
