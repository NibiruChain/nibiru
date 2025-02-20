package evmtest

import (
	"math/big"

	gethcommon "github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	gethparams "github.com/ethereum/go-ethereum/params"

	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"

	"github.com/NibiruChain/nibiru/v2/x/evm"
)

// ArgsCreateContract: Arguments to call with `CreateContractTxMsg` to make Ethereum transactions that create
// contracts.
//
// It is recommended to use a gas price of `big.NewInt(1)` for simpler op code
// calculations in gas units.
type ArgsCreateContract struct {
	EthAcc        EthPrivKeyAcc
	EthChainIDInt *big.Int
	GasPrice      *big.Int
	Nonce         uint64
	GasLimit      *big.Int
}

// ArgsExecuteContract: Arguments to call with `ExecuteContractTxMsg`
// to make Ethereum transactions that execute contracts.
type ArgsExecuteContract struct {
	EthAcc          EthPrivKeyAcc
	EthChainIDInt   *big.Int
	ContractAddress *gethcommon.Address
	Data            []byte
	GasPrice        *big.Int
	Nonce           uint64
	GasLimit        *big.Int
}

func CreateContractMsgEthereumTx(
	args ArgsCreateContract,
) (msgEthereumTx *evm.MsgEthereumTx, err error) {
	gasLimit := args.GasLimit
	if gasLimit == nil {
		gasLimit = new(big.Int).SetUint64(gethparams.TxGasContractCreation)
	}
	ethTx := gethcore.NewTx(&gethcore.AccessListTx{
		GasPrice: args.GasPrice,
		Gas:      gasLimit.Uint64(),
		To:       nil,
		Data:     embeds.SmartContract_TestERC20.Bytecode,
		Nonce:    args.Nonce,
	})

	msgEthereumTx = new(evm.MsgEthereumTx)
	err = msgEthereumTx.FromEthereumTx(ethTx)
	if err != nil {
		return msgEthereumTx, err
	}
	msgEthereumTx.From = args.EthAcc.EthAddr.Hex()

	gethSigner := gethcore.LatestSignerForChainID(args.EthChainIDInt)
	return msgEthereumTx, msgEthereumTx.Sign(gethSigner, args.EthAcc.KeyringSigner)
}

func ExecuteContractMsgEthereumTx(args ArgsExecuteContract) (msgEthereumTx *evm.MsgEthereumTx, err error) {
	gasLimit := args.GasLimit
	if gasLimit == nil {
		gasLimit = new(big.Int).SetUint64(gethparams.TxGas)
	}

	coreTx := gethcore.NewTx(&gethcore.AccessListTx{
		GasPrice: args.GasPrice,
		Gas:      gasLimit.Uint64(),
		To:       args.ContractAddress,
		Data:     args.Data,
		Nonce:    args.Nonce,
	})
	msgEthereumTx = new(evm.MsgEthereumTx)
	err = msgEthereumTx.FromEthereumTx(coreTx)
	if err != nil {
		return msgEthereumTx, err
	}
	msgEthereumTx.From = args.EthAcc.EthAddr.Hex()

	gethSigner := gethcore.LatestSignerForChainID(args.EthChainIDInt)
	return msgEthereumTx, msgEthereumTx.Sign(gethSigner, args.EthAcc.KeyringSigner)
}
