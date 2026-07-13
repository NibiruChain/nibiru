package wasmext_test

import (
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	wasmvm "github.com/NibiruChain/nibiru/v2/lib/wasmvm-ffi/wvm"

	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/codec"
	sdkcodec "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/codec/types"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/authz"
	bank "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/bank/types"

	"github.com/NibiruChain/nibiru/v2/app/wasmext"
	"github.com/NibiruChain/nibiru/v2/evm"
	"github.com/NibiruChain/nibiru/v2/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testapp"
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
	protoValueBz, err := deps.App.AppCodec().Marshal(ethTxMsg)
	s.Require().NoError(err, "expect ethTxMsg to proto marshal", protoValueBz)

	_, ok := deps.App.AppCodec().(sdkcodec.AnyUnpacker)
	s.Require().True(ok, "codec must be an AnyUnpacker")

	pbAny, err := sdkcodec.NewAnyWithValue(ethTxMsg)
	s.NoError(err)
	pbAnyBz, err := pbAny.Marshal()
	s.NoError(err, pbAnyBz)

	var sdkMsg sdk.Msg
	err = deps.App.AppCodec().UnpackAny(pbAny, &sdkMsg)
	s.Require().NoError(err)
	s.Equal("/eth.evm.v1.MsgEthereumTx", sdk.MsgTypeURL(sdkMsg))

	s.T().Log("Dispatch the Eth tx msg from Wasm (unsuccessfully)")
	_, _, err = wasmMsgHandler.DispatchMsg(
		deps.Ctx(),
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
	err = testapp.FundAccount(deps.App.BankKeeper, deps.Ctx(), deps.Sender.NibiruAddr, coins)
	s.NoError(err)
	txMsg := &bank.MsgSend{
		FromAddress: deps.Sender.NibiruAddr.String(),
		ToAddress:   evmtest.NewEthPrivAcc().NibiruAddr.String(),
		Amount:      []sdk.Coin{sdk.NewInt64Coin(evm.EVMBankDenom, 20)},
	}
	protoValueBz, err = deps.App.AppCodec().Marshal(txMsg)
	s.NoError(err)
	_, _, err = wasmMsgHandler.DispatchMsg(
		deps.Ctx(),
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

func (s *Suite) TestWasmSdkMessageHandlerRejectsBlockedMessages() {
	deps := evmtest.NewTestDeps()
	wasmMsgHandler := wasmext.WasmMessageHandler(deps.App.WasmMsgHandlerArgs)
	contractAddr := deps.Sender.NibiruAddr
	granteeAddr := evmtest.NewEthPrivAcc().NibiruAddr

	to := evmtest.NewEthPrivAcc()
	ethTxMsg, err := evmtest.TxTransferWei{
		Deps:      &deps,
		To:        to.EthAddr,
		AmountWei: evm.NativeToWei(big.NewInt(420)),
	}.Build()
	s.Require().NoError(err)

	execMsg := authz.NewMsgExec(
		contractAddr,
		[]sdk.Msg{
			&bank.MsgSend{
				FromAddress: granteeAddr.String(),
				ToAddress:   contractAddr.String(),
				Amount:      sdk.NewCoins(sdk.NewInt64Coin(evm.EVMBankDenom, 1)),
			},
		},
	)

	expiration := time.Now().Add(time.Hour)
	grantMsg, err := authz.NewMsgGrant(
		contractAddr,
		granteeAddr,
		authz.NewGenericAuthorization(sdk.MsgTypeURL(&bank.MsgSend{})),
		&expiration,
	)
	s.Require().NoError(err)

	revokeMsg := authz.NewMsgRevoke(
		contractAddr,
		granteeAddr,
		sdk.MsgTypeURL(&bank.MsgSend{}),
	)

	testCases := []struct {
		name string
		msg  interface {
			sdk.Msg
			codec.ProtoMarshaler
		}
		wantErr string
	}{
		{
			name:    "evm ethereum tx",
			msg:     ethTxMsg,
			wantErr: "Wasm VM to EVM call pattern is not yet supported",
		},
		{
			name:    "authz exec",
			msg:     &execMsg,
			wantErr: "not allowed",
		},
		{
			name:    "authz grant",
			msg:     grantMsg,
			wantErr: "not allowed",
		},
		{
			name:    "authz revoke",
			msg:     &revokeMsg,
			wantErr: "not allowed",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			protoValueBz, err := deps.App.AppCodec().Marshal(tc.msg)
			s.Require().NoError(err)

			_, _, err = wasmMsgHandler.DispatchMsg(
				deps.Ctx(),
				contractAddr,
				"ibcport-unused",
				wasmvm.CosmosMsg{
					Stargate: &wasmvm.StargateMsg{
						TypeURL: sdk.MsgTypeURL(tc.msg),
						Value:   protoValueBz,
					},
				},
			)
			s.Require().ErrorContains(err, tc.wantErr)
		})
	}
}
