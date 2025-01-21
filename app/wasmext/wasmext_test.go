package wasmext_test

import (
	"math/big"
	"testing"

	wasmvm "github.com/CosmWasm/wasmvm/types"
	sdkcodec "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bank "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/v2/app/wasmext"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
)

type Suite struct {
	suite.Suite
}

func TestWasmExtSuite(t *testing.T) {
	suite.Run(t, new(Suite))
}

// WasmVM to EVM call pattern is not yet supported. This test verifies the
// Nibiru's [wasmkeeper.Option] function as expected.
func (s *Suite) TestEvmFilter() {
	deps := evmtest.NewTestDeps()
	// wk := wasmkeeper.NewDefaultPermissionKeeper(deps.App.WasmKeeper)
	wasmMsgHandler := wasmext.WasmMessageHandler(deps.App.WasmMsgHandlerArgs)

	s.T().Log("Create a valid Ethereum tx msg")

	to := evmtest.NewEthPrivAcc()
	ethTxMsg, err := evmtest.TxTransferWei{
		Deps:      &deps,
		To:        to.EthAddr,
		AmountWei: evm.NativeToWei(big.NewInt(420)),
	}.Build()
	s.NoError(err)

	s.T().Log("Validate Eth tx msg proto encoding as wasmvm.StargateMsg")
	wasmContractAddr := deps.Sender.NibiruAddr
	protoValueBz, err := deps.EncCfg.Codec.Marshal(ethTxMsg)
	s.Require().NoError(err, "expect ethTxMsg to proto marshal", protoValueBz)

	_, ok := deps.EncCfg.Codec.(sdkcodec.AnyUnpacker)
	s.Require().True(ok, "codec must be an AnyUnpacker")

	pbAny, err := sdkcodec.NewAnyWithValue(ethTxMsg)
	s.NoError(err)
	pbAnyBz, err := pbAny.Marshal()
	s.NoError(err, pbAnyBz)

	var sdkMsg sdk.Msg
	err = deps.EncCfg.Codec.UnpackAny(pbAny, &sdkMsg)
	s.Require().NoError(err)
	s.Equal("/eth.evm.v1.MsgEthereumTx", sdk.MsgTypeURL(sdkMsg))

	s.T().Log("Dispatch the Eth tx msg from Wasm (unsuccessfully)")
	_, _, err = wasmMsgHandler.DispatchMsg(
		deps.Ctx,
		wasmContractAddr,
		"ibcport-unused",
		wasmvm.CosmosMsg{
			Stargate: &wasmvm.StargateMsg{
				TypeURL: sdk.MsgTypeURL(ethTxMsg),
				Value:   protoValueBz,
			},
		},
	)
	s.Require().ErrorContains(err, "Wasm VM to EVM call pattern is not yet supported")

	coins := sdk.NewCoins(sdk.NewInt64Coin(evm.EVMBankDenom, 420)) // arbitrary constant
	err = testapp.FundAccount(deps.App.BankKeeper, deps.Ctx, deps.Sender.NibiruAddr, coins)
	s.NoError(err)
	txMsg := &bank.MsgSend{
		FromAddress: deps.Sender.NibiruAddr.String(),
		ToAddress:   evmtest.NewEthPrivAcc().NibiruAddr.String(),
		Amount:      []sdk.Coin{sdk.NewInt64Coin(evm.EVMBankDenom, 20)},
	}
	protoValueBz, err = deps.EncCfg.Codec.Marshal(txMsg)
	s.NoError(err)
	_, _, err = wasmMsgHandler.DispatchMsg(
		deps.Ctx,
		wasmContractAddr,
		"ibcport-unused",
		wasmvm.CosmosMsg{
			Stargate: &wasmvm.StargateMsg{
				TypeURL: sdk.MsgTypeURL(txMsg),
				Value:   protoValueBz,
			},
		},
	)
	s.Require().NoError(err)
}
