package keeper_test

import (
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/keeper"
	"github.com/NibiruChain/nibiru/x/perp/types"
	"github.com/NibiruChain/nibiru/x/testutil"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
)

func TestQueryPosition_Ok(t *testing.T) {
	t.Log("initialize keeper")
	nibiruApp, ctx := testutil.NewNibiruApp(true)
	perpKeeper := &nibiruApp.PerpKeeper

	queryServer := keeper.NewQuerier(*perpKeeper)

	trader := sample.AccAddress()
	vpoolPair, err := common.NewTokenPairFromStr("btc:nusd")
	require.NoError(t, err)

	oldPosition := &types.Position{
		TraderAddress: trader,
		Pair:          vpoolPair.String(),
		Size_:         sdk.NewDec(10),
		OpenNotional:  sdk.NewDec(10),
		Margin:        sdk.NewDec(1),
	}

	perpKeeper.SetPosition(
		ctx, vpoolPair, trader, oldPosition)

	res, err := queryServer.TraderPosition(
		sdk.WrapSDKContext(ctx),
		&types.QueryTraderPositionRequest{
			Trader:    trader,
			TokenPair: vpoolPair.String(),
		},
	)
	fmt.Println("res:", res)
	require.NoError(t, err)

	assert.Equal(t, oldPosition.TraderAddress, res.Position.TraderAddress)
	assert.Equal(t, oldPosition.Pair, res.Position.Pair)
	assert.Equal(t, oldPosition.Size_, res.Position.Size_)
}
