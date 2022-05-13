package pricefeed_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/pricefeed"
	"github.com/NibiruChain/nibiru/x/pricefeed/types"
	"github.com/NibiruChain/nibiru/x/testutil/nullify"
	"github.com/NibiruChain/nibiru/x/testutil/testkeeper"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
	}

	k, ctx := testkeeper.PricefeedKeeper(t)
	pricefeed.InitGenesis(ctx, k, genesisState)
	got := pricefeed.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)
}
