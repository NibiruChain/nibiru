package genmsg

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/require"

	v1 "github.com/NibiruChain/nibiru/x/genmsg/v1"
)

type mockRouter struct {
	handler func(msg sdk.Msg) baseapp.MsgServiceHandler
}

func (m mockRouter) Handler(msg sdk.Msg) baseapp.MsgServiceHandler { return m.handler(msg) }

func makeCodec(_ *testing.T) codec.JSONCodec {
	ir := types.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(ir)
	ir.RegisterInterface(sdk.MsgInterfaceProtoName, (*sdk.Msg)(nil), &banktypes.MsgSend{})
	return cdc
}

func newGenesisFromMsgs(t *testing.T, cdc codec.JSONCodec, msgs ...proto.Message) *v1.GenesisState {
	genesis := new(v1.GenesisState)
	for _, msg := range msgs {
		anyProto, err := types.NewAnyWithValue(msg)
		require.NoError(t, err)
		genesis.Messages = append(genesis.Messages, anyProto)
	}
	genesisJSON, err := cdc.MarshalJSON(genesis)
	require.NoError(t, err)
	genesis = new(v1.GenesisState)
	require.NoError(t, cdc.UnmarshalJSON(genesisJSON, genesis))
	return genesis
}

func Test_initGenesis(t *testing.T) {
	cdc := makeCodec(t)
	ctx := sdk.Context{}

	t.Run("works - no msgs", func(t *testing.T) {
		r := mockRouter{func(msg sdk.Msg) baseapp.MsgServiceHandler {
			return func(ctx sdk.Context, req sdk.Msg) (*sdk.Result, error) {
				return &sdk.Result{}, nil
			}
		}}

		err := initGenesis(ctx, cdc, r, newGenesisFromMsgs(t, cdc))
		require.NoError(t, err)
	})

	t.Run("works - with message", func(t *testing.T) {
		called := false
		r := mockRouter{func(msg sdk.Msg) baseapp.MsgServiceHandler {
			return func(ctx sdk.Context, req sdk.Msg) (*sdk.Result, error) {
				called = true
				return &sdk.Result{}, nil
			}
		}}

		err := initGenesis(ctx, cdc, r, newGenesisFromMsgs(t, cdc, &banktypes.MsgSend{
			FromAddress: sdk.AccAddress("a").String(),
			ToAddress:   sdk.AccAddress("b").String(),
			Amount:      sdk.NewCoins(sdk.NewInt64Coin("test", 1000)),
		}))
		require.NoError(t, err)
		require.True(t, called)
	})

	t.Run("fails - handler is nil", func(t *testing.T) {
		r := mockRouter{func(msg sdk.Msg) baseapp.MsgServiceHandler {
			return nil
		}}

		err := initGenesis(ctx, cdc, r, newGenesisFromMsgs(t, cdc, &banktypes.MsgSend{
			FromAddress: sdk.AccAddress("a").String(),
			ToAddress:   sdk.AccAddress("b").String(),
			Amount:      sdk.NewCoins(sdk.NewInt64Coin("test", 1000)),
		}))
		require.Error(t, err)
	})
}

func Test_validateGenesis(t *testing.T) {
	cdc := makeCodec(t)
	t.Run("works - empty", func(t *testing.T) {
		err := validateGenesis(cdc, &v1.GenesisState{})
		require.NoError(t, err)
	})
	t.Run("works - with messages", func(t *testing.T) {
		genesis := newGenesisFromMsgs(t, cdc, &banktypes.MsgSend{
			FromAddress: sdk.AccAddress("sender").String(),
			ToAddress:   sdk.AccAddress("receiver").String(),
			Amount:      sdk.NewCoins(sdk.NewInt64Coin("test", 1000)),
		})
		err := validateGenesis(cdc, genesis)
		require.NoError(t, err)
	})
	t.Run("fails - validate basic", func(t *testing.T) {
		genesis := newGenesisFromMsgs(t, cdc, &banktypes.MsgSend{})
		err := validateGenesis(cdc, genesis)
		require.Error(t, err)
	})
}
