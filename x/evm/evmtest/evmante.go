package evmtest

import (
	"math/big"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	gethparams "github.com/ethereum/go-ethereum/params"

	"github.com/NibiruChain/nibiru/v2/x/evm"
)

var NextNoOpAnteHandler sdk.AnteHandler = func(
	ctx sdk.Context, tx sdk.Tx, simulate bool,
) (newCtx sdk.Context, err error) {
	return ctx, nil
}

func HappyTransferTx(deps *TestDeps, nonce uint64) *evm.MsgEthereumTx {
	to := NewEthPrivAcc().EthAddr
	evmTxArgs := &evm.EvmTxArgs{
		ChainID:  deps.App.EvmKeeper.EthChainID(deps.Ctx),
		Nonce:    nonce,
		Amount:   big.NewInt(10),
		GasLimit: GasLimitCreateContract().Uint64(),
		GasPrice: evm.NativeToWei(big.NewInt(1)),
		To:       &to,
	}
	tx := evm.NewTx(evmTxArgs)
	tx.From = deps.Sender.EthAddr.Hex()
	return tx
}

func NonEvmMsgTx(deps *TestDeps) sdk.Tx {
	gasLimit := uint64(10)
	fees := sdk.NewCoins(sdk.NewInt64Coin("unibi", int64(gasLimit)))
	msg := &banktypes.MsgSend{
		FromAddress: deps.Sender.NibiruAddr.String(),
		ToAddress:   NewEthPrivAcc().NibiruAddr.String(),
		Amount:      sdk.NewCoins(sdk.NewInt64Coin("unibi", 1)),
	}
	return BuildTx(deps, true, gasLimit, fees, msg)
}

func BuildTx(
	deps *TestDeps,
	ethExtentions bool,
	gasLimit uint64,
	fees sdk.Coins,
	msgs ...sdk.Msg,
) sdk.FeeTx {
	txBuilder, _ := deps.EncCfg.TxConfig.NewTxBuilder().(authtx.ExtensionOptionsTxBuilder)
	if ethExtentions {
		option, _ := codectypes.NewAnyWithValue(&evm.ExtensionOptionsEthereumTx{})
		txBuilder.SetExtensionOptions(option)
	}
	err := txBuilder.SetMsgs(msgs...)
	if err != nil {
		panic(err)
	}
	txBuilder.SetGasLimit(gasLimit)
	txBuilder.SetFeeAmount(fees)

	return txBuilder.GetTx()
}

func HappyCreateContractTx(deps *TestDeps) *evm.MsgEthereumTx {
	evmTxArgs := &evm.EvmTxArgs{
		ChainID:  deps.App.EvmKeeper.EthChainID(deps.Ctx),
		Nonce:    1,
		Amount:   big.NewInt(10),
		GasLimit: GasLimitCreateContract().Uint64(),
		GasPrice: evm.NativeToWei(big.NewInt(1)),
		To:       nil,
	}
	tx := evm.NewTx(evmTxArgs)
	tx.From = deps.Sender.EthAddr.Hex()
	return tx
}

func GasLimitCreateContract() *big.Int {
	return new(big.Int).SetUint64(
		gethparams.TxGasContractCreation + 700,
	)
}
