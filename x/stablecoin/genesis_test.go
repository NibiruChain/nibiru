package stablecoin_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/stablecoin"
	"github.com/NibiruChain/nibiru/x/stablecoin/types"
	"github.com/NibiruChain/nibiru/x/testutil/nullify"
	"github.com/NibiruChain/nibiru/x/testutil/testapp"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params:               types.DefaultParams(),
		ModuleAccountBalance: sdk.NewCoin(common.DenomColl, sdk.ZeroInt()),
	}

	nibiruApp, ctx := testapp.NewNibiruAppAndContext(true)
	k := nibiruApp.StablecoinKeeper
	stablecoin.InitGenesis(ctx, k, genesisState)
	got := stablecoin.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)
}
