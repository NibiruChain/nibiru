package evmv1

import (
	fmt "fmt"

	"github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"

	protov2 "google.golang.org/protobuf/proto"
)

// supportedTxs holds the Ethereum transaction types
// supported by Nibiru.
// Use a function to return a new pointer and avoid
// possible reuse or racing conditions when using the same pointer
var supportedTxs = map[string]func() TxDataV2{
	"/eth.evm.v1.DynamicFeeTx": func() TxDataV2 { return &DynamicFeeTx{} },
	"/eth.evm.v1.AccessListTx": func() TxDataV2 { return &AccessListTx{} },
	"/eth.evm.v1.LegacyTx":     func() TxDataV2 { return &LegacyTx{} },
}

// getSender extracts the sender address from the signature values using the latest signer for the given chainID.
func getSender(txData TxDataV2) (common.Address, error) {
	signer := gethcore.LatestSignerForChainID(txData.GetChainID())
	from, err := signer.Sender(gethcore.NewTx(txData.AsEthereumData()))
	if err != nil {
		return common.Address{}, err
	}
	return from, nil
}

func EthereumTxGetSigners(msg protov2.Message) ([][]byte, error) {
	msgEthereumTx, ok := msg.(*MsgEthereumTx)
	if !ok {
		return nil, fmt.Errorf("invalid type, expected MsgConvertERC20 and got %T", msg)
	}

	txDataFn, found := supportedTxs[msgEthereumTx.Data.TypeUrl]
	if !found {
		return nil, fmt.Errorf("invalid TypeUrl %s", msgEthereumTx.Data.TypeUrl)
	}
	txData := txDataFn()

	// msgEthTx.Data is a message (DynamicFeeTx, LegacyTx or AccessListTx)
	if err := msgEthereumTx.Data.UnmarshalTo(txData); err != nil {
		return nil, err
	}

	sender, err := getSender(txData)
	if err != nil {
		return nil, err
	}

	return [][]byte{sender.Bytes()}, nil
}
