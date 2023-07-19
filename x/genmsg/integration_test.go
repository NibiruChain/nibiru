package genmsg_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"

	"github.com/archway-network/archway/app"
	e2eTesting "github.com/archway-network/archway/e2e/testing"
	"github.com/archway-network/archway/x/genmsg"
	v1 "github.com/archway-network/archway/x/genmsg/v1"
)

func TestIntegration(t *testing.T) {
	recvAddr := sdk.AccAddress("recv")
	chain := e2eTesting.NewTestChain(t, 1, e2eTesting.WithDummyTestAddress(), e2eTesting.WithGenDefaultCoinBalance("100000000000000000000000000000000000"), e2eTesting.TestChainGenesisOption(func(cdc codec.Codec, genesis app.GenesisState) {
		testMsg := &banktypes.MsgSend{
			FromAddress: e2eTesting.TestAccountAddr.String(),
			ToAddress:   recvAddr.String(),
			Amount:      sdk.NewCoins(sdk.NewInt64Coin("stake", 1000)),
		}
		anyMsg, err := codectypes.NewAnyWithValue(testMsg)
		require.NoError(t, err)
		genesis[genmsg.ModuleName] = cdc.MustMarshalJSON(&v1.GenesisState{Messages: []*codectypes.Any{anyMsg}})
	}))
	bankQuery := banktypes.NewQueryClient(chain.Client())
	resp, err := bankQuery.Balance(sdk.WrapSDKContext(chain.GetContext()), &banktypes.QueryBalanceRequest{
		Address: recvAddr.String(),
		Denom:   "stake",
	})

	require.NoError(t, err)
	t.Log(resp)
}
