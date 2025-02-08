package softwallet

import (
	"os"
	"sort"
	"testing"

	"github.com/scripttoken/script/common"
	"github.com/stretchr/testify/assert"
)

func TestPlainSoftWalletBasics(t *testing.T) {
	testSoftWalletBasics(t)
}

func TestPlainSoftWalletMultipleKeys(t *testing.T) {
	testSoftWalletMultipleKeys(t)
}

// ---------------- Test Utilities ---------------- //

func testSoftWalletBasics(t *testing.T) {
	assert := assert.New(t)

	tmpdir := createTempDir()
	defer os.RemoveAll(tmpdir)

	wallet, err := NewSoftWallet(tmpdir)
	assert.Nil(err)
	addrs, err := wallet.List()
	assert.Nil(err)
	assert.Equal(0, len(addrs))

	addr, err := wallet.NewKey()
	assert.NotEqual(common.Address{}, addr)
	assert.Nil(err)
	addrs, err = wallet.List()
	assert.Nil(err)
	assert.Equal(1, len(addrs))
	assert.Equal([]common.Address{addr}, addrs)

	// updtpass
	err = wallet.Unlock(addr, nil)
	assert.Nil(err)

	err = wallet.Lock(addr)
	assert.Nil(err)

	addrs, err = wallet.List()
	assert.Nil(err)
	assert.Equal(1, len(addrs))
	assert.Equal([]common.Address{addr}, addrs)

	err = wallet.Unlock(addr, nil)
	assert.Nil(err)
	err = wallet.Lock(addr)
	assert.Nil(err)

	err = wallet.Unlock(addr, nil)
	assert.Nil(err)

	signature, err := wallet.Sign(addr, common.Bytes("hello world"))
	assert.False(signature.IsEmpty())
	assert.Nil(err)

	err = wallet.Lock(addr)
	assert.Nil(err)

	signature, err = wallet.Sign(addr, common.Bytes("hello world"))
	assert.Nil(signature)
	assert.NotNil(err)

	err = wallet.Delete(addr)
	assert.Nil(err)

	err = wallet.Unlock(addr, nil)
	assert.NotNil(err)
}

func testSoftWalletMultipleKeys(t *testing.T) {
	assert := assert.New(t)

	tmpdir := createTempDir()
	defer os.RemoveAll(tmpdir)

	wallet, err := NewSoftWallet(tmpdir)
	assert.Nil(err)
	addrs, err := wallet.List()
	assert.Nil(err)
	assert.Equal(0, len(addrs))

	addr1, err := wallet.NewKey()
	assert.NotEqual(common.Address{}, addr1)
	assert.Nil(err)
	addrs, err = wallet.List()
	assert.Nil(err)
	assert.Equal(1, len(addrs))
	assert.Equal([]common.Address{addr1}, addrs)

	addr2, err := wallet.NewKey()
	assert.NotEqual(common.Address{}, addr2)
	assert.Nil(err)
	addrs, err = wallet.List()
	assert.Nil(err)
	assert.Equal(2, len(addrs))
	assert.Equal(sortAddresses([]common.Address{addr1, addr2}), sortAddresses(addrs))

	addr3, err := wallet.NewKey()
	assert.NotEqual(common.Address{}, addr3)
	assert.Nil(err)
	addrs, err = wallet.List()
	assert.Nil(err)
	assert.Equal(3, len(addrs))
	assert.Equal(sortAddresses([]common.Address{addr1, addr2, addr3}), sortAddresses(addrs))

	addr4, err := wallet.NewKey()
	assert.NotEqual(common.Address{}, addr4)
	assert.Nil(err)
	addrs, err = wallet.List()
	assert.Nil(err)
	assert.Equal(4, len(addrs))
	assert.Equal(sortAddresses([]common.Address{addr1, addr2, addr3, addr4}), sortAddresses(addrs))

	err = wallet.Lock(addr1)
	assert.Nil(err)
	addrs, err = wallet.List()
	assert.Nil(err)
	assert.Equal(4, len(addrs))
	assert.Equal(sortAddresses([]common.Address{addr1, addr2, addr3, addr4}), sortAddresses(addrs))

	err = wallet.Delete(addr3)
	assert.Nil(err)
	addrs, err = wallet.List()
	assert.Nil(err)
	assert.Equal(3, len(addrs))
	assert.Equal(sortAddresses([]common.Address{addr1, addr2, addr4}), sortAddresses(addrs))

	signature3, err := wallet.Sign(addr3, common.Bytes("hello world"))
	assert.Nil(signature3)
	assert.NotNil(err)

	signature2, err := wallet.Sign(addr2, common.Bytes("hello world"))
	assert.NotNil(signature2)
	assert.Nil(err)

	signature4, err := wallet.Sign(addr4, common.Bytes("hello world"))
	assert.NotNil(signature4)
	assert.Nil(err)

	assert.NotEqual(signature2.ToBytes(), signature4.ToBytes())

	err = wallet.Delete(addr2)
	assert.Nil(err)
	addrs, err = wallet.List()
	assert.Nil(err)
	assert.Equal(2, len(addrs))
	assert.Equal(sortAddresses([]common.Address{addr1, addr4}), sortAddresses(addrs))
}

func createTempDir() string {
	dir, err := os.MkdirTemp("", "script-softwallet-test")
	if err != nil {
		panic(err)
	}

	return dir
}

func sortAddresses(addresses []common.Address) []common.Address {
	sortedAddresses := addresses
	sort.Slice(sortedAddresses[:], func(i, j int) bool {
		return sortedAddresses[i].Hex() < sortedAddresses[j].Hex()
	})
	return sortedAddresses
}
