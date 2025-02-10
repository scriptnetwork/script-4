package types

import (
	"bytes"
	"fmt"

	"github.com/pkg/errors"
	"github.com/scripttoken/script/rlp"
)

const maxTxSize = 8 * 1024 * 1024

// ----------------- Common -------------------

func ToBytes(a interface{}) ([]byte, error) {
	return rlp.EncodeToBytes(a)
}

func FromBytes(in []byte, a interface{}) error {
	return rlp.DecodeBytes(in, a)
}

// ----------------- Tx -------------------

type TxType uint16

const (
	TxCoinbase TxType = iota
	TxSend
	TxSmartContract
	TxLicense
)

func Fuzz(data []byte) int {
	if len(data) == 0 {
		return -1
	}
	if data[0]%3 == 0 {
		if _, ok := ParseCoinAmount(string(data[1:])); ok {
			return 1
		}
		return 0
	}
	if data[0]%3 == 1 {
		if _, err := TxFromBytes(data[1:]); err != nil {
			return 1
		}
		return 0
	}
	return -1
}

func TxFromBytes(raw []byte) (Tx, error) {
	var txType TxType
	buff := bytes.NewBuffer(raw)
	s := rlp.NewStream(buff, maxTxSize)
	err := s.Decode(&txType)
	if err != nil {
		return nil, err
	}
	if txType == TxCoinbase {
		data := &CoinbaseTx{}
		err = s.Decode(data)
		return data, err
	} else if txType == TxSend {
		data := &SendTx{}
		err = s.Decode(data)
		return data, err
	} else if txType == TxSmartContract {
		data := &SmartContractTx{}
		err = s.Decode(data)
		return data, err
	} else if txType == TxLicense {
		data := &LicenseTx{}
		err = s.Decode(data)
		return data, err
	} else {
		return nil, fmt.Errorf("unknown TX type: %v", txType)
	}
}

func TxToBytes(t Tx) ([]byte, error) {
	var buf bytes.Buffer
	var txType TxType
	switch t.(type) {
	case *CoinbaseTx:
		txType = TxCoinbase
	case *SendTx:
		txType = TxSend
	case *SmartContractTx:
		txType = TxSmartContract
	case *LicenseTx:
		txType = TxLicense
	default:
		return nil, errors.New("Unsupported message type")
	}
	err := rlp.Encode(&buf, txType)
	if err != nil {
		return nil, err
	}
	err = rlp.Encode(&buf, t)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
