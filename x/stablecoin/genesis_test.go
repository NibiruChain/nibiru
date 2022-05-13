package stablecoin_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/stablecoin"
	"github.com/NibiruChain/nibiru/x/stablecoin/types"
	"github.com/NibiruChain/nibiru/x/testutil"
	"github.com/NibiruChain/nibiru/x/testutil/nullify"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params:               types.DefaultParams(),
		ModuleAccountBalance: sdk.NewCoin(common.CollDenom, sdk.ZeroInt()),
	}

	nibiruApp, ctx := testutil.NewNibiruApp(true)
	k := nibiruApp.StablecoinKeeper
	stablecoin.InitGenesis(ctx, k, genesisState)
	got := stablecoin.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)
}
