package evmtest

import (
	"math/big"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/params"
	gethparams "github.com/ethereum/go-ethereum/params"

	"github.com/NibiruChain/nibiru/x/evm/embeds"

	gethcore "github.com/ethereum/go-ethereum/core/types"

	"github.com/NibiruChain/nibiru/x/evm"
)

// ArgsCreateContract: Arguments to call with `CreateContractTxMsg` and
// `CreateContractGethCoreMsg` to make Ethereum transactions that create
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

func CreateContractTxMsg(
	args ArgsCreateContract,
) (ethTxMsg *evm.MsgEthereumTx, err error) {
	gasLimit := args.GasLimit
	if gasLimit == nil {
		gasLimit = new(big.Int).SetUint64(gethparams.TxGasContractCreation)
	}
	testContract, err := embeds.SmartContract_TestERC20.Load()
	if err != nil {
		return nil, err
	}
	gethTxCreateCntract := &gethcore.AccessListTx{
		GasPrice: args.GasPrice,
		Gas:      gasLimit.Uint64(),
		To:       nil,
		Data:     testContract.Bytecode,
		Nonce:    args.Nonce,
	}
	ethTx := gethcore.NewTx(gethTxCreateCntract)
	ethTxMsg = new(evm.MsgEthereumTx)
	err = ethTxMsg.FromEthereumTx(ethTx)
	if err != nil {
		return ethTxMsg, err
	}
	fromAcc := args.EthAcc
	ethTxMsg.From = fromAcc.EthAddr.Hex()

	gethSigner := fromAcc.GethSigner(args.EthChainIDInt)
	keyringSigner := fromAcc.KeyringSigner
	return ethTxMsg, ethTxMsg.Sign(gethSigner, keyringSigner)
}

func CreateContractGethCoreMsg(
	args ArgsCreateContract,
	cfg *params.ChainConfig,
	blockHeight *big.Int,
) (gethCoreMsg core.Message, err error) {
	ethTxMsg, err := CreateContractTxMsg(args)
	if err != nil {
		return gethCoreMsg, err
	}

	signer := gethcore.MakeSigner(cfg, blockHeight)
	return ethTxMsg.AsMessage(signer, nil)
}

func ExecuteContractTxMsg(args ArgsExecuteContract) (ethTxMsg *evm.MsgEthereumTx, err error) {
	gasLimit := args.GasLimit
	if gasLimit == nil {
		gasLimit = new(big.Int).SetUint64(gethparams.TxGas)
	}
	gethTxExecuteContract := &gethcore.AccessListTx{
		GasPrice: args.GasPrice,
		Gas:      gasLimit.Uint64(),
		To:       args.ContractAddress,
		Data:     args.Data,
		Nonce:    args.Nonce,
	}
	ethTx := gethcore.NewTx(gethTxExecuteContract)
	ethTxMsg = new(evm.MsgEthereumTx)
	err = ethTxMsg.FromEthereumTx(ethTx)
	if err != nil {
		return ethTxMsg, err
	}
	fromAcc := args.EthAcc
	ethTxMsg.From = fromAcc.EthAddr.Hex()

	gethSigner := fromAcc.GethSigner(args.EthChainIDInt)
	keyringSigner := fromAcc.KeyringSigner
	return ethTxMsg, ethTxMsg.Sign(gethSigner, keyringSigner)
}
