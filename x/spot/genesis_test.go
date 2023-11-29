package spot_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/x/spot"
	"github.com/NibiruChain/nibiru/x/spot/types"
)

func TestGenesis(t *testing.T) {
	testapp.EnsureNibiruPrefix()
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
		Pools: []types.Pool{
			{
				Id:      1,
				Address: "addr1",
				PoolParams: types.PoolParams{
					SwapFee:  sdk.MustNewDecFromStr("0.01"),
					ExitFee:  sdk.MustNewDecFromStr("0.01"),
					A:        sdk.ZeroInt(),
					PoolType: types.PoolType_BALANCER,
				},
				PoolAssets: []types.PoolAsset{
					{
						Token:  sdk.NewCoin("token1", sdk.NewInt(100)),
						Weight: sdk.OneInt(),
					},
					{
						Token:  sdk.NewCoin("token2", sdk.NewInt(100)),
						Weight: sdk.OneInt(),
					},
				},
				TotalWeight: sdk.NewInt(2),
				TotalShares: sdk.NewCoin("nibiru/pool/1", sdk.NewInt(100)),
			},
		},
	}

	app, ctx := testapp.NewNibiruTestAppAndContext()
	spot.InitGenesis(ctx, app.SpotKeeper, genesisState)
	got := spot.ExportGenesis(ctx, app.SpotKeeper)
	require.NotNil(t, got)

	testutil.Fill(&genesisState)
	testutil.Fill(got)

	require.Equal(t, genesisState, *got)
}
