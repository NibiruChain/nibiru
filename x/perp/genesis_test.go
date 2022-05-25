package perp_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp"
	"github.com/NibiruChain/nibiru/x/perp/types"
	"github.com/NibiruChain/nibiru/x/testutil"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params:               types.DefaultParams(),
		ModuleAccountBalance: sdk.NewCoin(common.GovDenom, sdk.ZeroInt()),
		PairMetadata: []*types.PairMetadata{
			{
				Pair: "ubtc:unibi",
				CumulativePremiumFractions: []sdk.Dec{
					sdk.MustNewDecFromStr("2.0"),
				},
			},
			{
				Pair:                       "ueth:unibi",
				CumulativePremiumFractions: nil,
			},
		},
	}

	nibiruApp, ctx := testutil.NewNibiruApp(true)
	perp.InitGenesis(ctx, nibiruApp.PerpKeeper, genesisState)

	exportedGenesisState := perp.ExportGenesis(ctx, nibiruApp.PerpKeeper)
	require.NotNil(t, exportedGenesisState)

	require.Equal(t, genesisState.GetParams(), exportedGenesisState.GetParams())
	require.Equal(t, genesisState.ModuleAccountBalance, exportedGenesisState.ModuleAccountBalance)

	require.Len(t, exportedGenesisState.PairMetadata, 2)

	for _, pm := range exportedGenesisState.PairMetadata {
		require.Contains(t, genesisState.PairMetadata, pm)
	}
}
