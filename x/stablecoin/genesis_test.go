package stablecoin_test

import (
	"testing"

	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/stablecoin"
	"github.com/NibiruChain/nibiru/x/stablecoin/types"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params:               types.DefaultParams(),
		ModuleAccountBalance: sdk.NewCoin(denoms.USDC, sdk.ZeroInt()),
	}

	nibiruApp, ctx := testapp.NewNibiruTestAppAndContext()
	k := nibiruApp.StablecoinKeeper
	stablecoin.InitGenesis(ctx, k, genesisState)
	got := stablecoin.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	testutil.Fill(&genesisState)
	testutil.Fill(got)
}
