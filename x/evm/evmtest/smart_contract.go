package evmtest

import (
	"math/big"

	// gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/params"
	gethparams "github.com/ethereum/go-ethereum/params"

	gethcore "github.com/ethereum/go-ethereum/core/types"

	"github.com/NibiruChain/nibiru/x/evm"
)

type AssertionSuite interface {
	Require()
}

type ArgsCreateContract struct {
	EthAcc        EthPrivKeyAcc
	EthChainIDInt *big.Int
	GasPrice      *big.Int
	Nonce         uint64
}

func CreateContractTxMsg(
	args ArgsCreateContract,
) (ethTxMsg *evm.MsgEthereumTx, err error) {
	gethTxCreateCntract := &gethcore.AccessListTx{
		GasPrice: args.GasPrice,
		Gas:      gethparams.TxGasContractCreation,
		To:       nil,
		Data:     []byte("contract_data"),
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
