package types

import (
	"github.com/scripttoken/script/common"
	"github.com/scripttoken/script/crypto"
)

type WalletType int

const (
	WalletTypeSoft WalletType = iota
	WalletTypeColdNano
	WalletTypeColdTrezor
)

type Wallet interface {
	ID() string
	Status() (string, error)
	List() ([]common.Address, error)
	NewKey() (common.Address, error)
	ImportKey(privHex string) (common.Address, error)
	Unlock(address common.Address, derivationPath DerivationPath) error
	Lock(address common.Address) error
	IsUnlocked(address common.Address) bool
	Delete(address common.Address) error
	Derive(path DerivationPath, pin bool) (common.Address, error)
	GetPublicKey(address common.Address) (*crypto.PublicKey, error)
	Sign(address common.Address, txrlp common.Bytes) (*crypto.Signature, error)
}
