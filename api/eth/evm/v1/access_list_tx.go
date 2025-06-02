package evmv1

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"

	ethutils "github.com/NibiruChain/nibiru/v2/utils"
)

// GetChainID returns the chain id field from the AccessListTx
func (tx *AccessListTx) GetChainID() *big.Int {
	return stringToBigInt(tx.GetChainId())
}

// GetAccessList returns the AccessList field.
func (tx *AccessListTx) GetAccessList() gethcore.AccessList {
	if tx.Accesses == nil {
		return nil
	}
	var ethAccessList gethcore.AccessList

	for _, tuple := range tx.Accesses {
		storageKeys := make([]common.Hash, len(tuple.StorageKeys))

		for i := range tuple.StorageKeys {
			storageKeys[i] = common.HexToHash(tuple.StorageKeys[i])
		}

		ethAccessList = append(ethAccessList, gethcore.AccessTuple{
			Address:     common.HexToAddress(tuple.Address),
			StorageKeys: storageKeys,
		})
	}

	return ethAccessList
}

// AsEthereumData returns an AccessListTx transaction tx from the proto-formatted
// TxData defined on the Cosmos EVM.
func (tx *AccessListTx) AsEthereumData() gethcore.TxData {
	v, r, s := tx.GetRawSignatureValues()
	return &gethcore.AccessListTx{
		ChainID:    tx.GetChainID(),
		Nonce:      tx.GetNonce(),
		GasPrice:   stringToBigInt(tx.GetGasPrice()),
		Gas:        tx.GetGas(),
		To:         stringToAddress(tx.GetTo()),
		Value:      stringToBigInt(tx.GetValue()),
		Data:       tx.GetData(),
		AccessList: tx.GetAccessList(),
		V:          v,
		R:          r,
		S:          s,
	}
}

// GetRawSignatureValues returns the V, R, S signature values of the transaction.
// The return values should not be modified by the caller.
func (tx *AccessListTx) GetRawSignatureValues() (v, r, s *big.Int) {
	return ethutils.RawSignatureValues(tx.V, tx.R, tx.S)
}
