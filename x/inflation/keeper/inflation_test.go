package keeper_test

import (
	"fmt"
	"testing"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	types "github.com/NibiruChain/nibiru/x/inflation/types"
)

func TestMintAndAllocateInflation(t *testing.T) {
	testCases := []struct {
		name                    string
		mintCoin                sdk.Coin
		expStakingRewardAmt     sdk.Coin
		expStrategicReservesAmt sdk.Coin
		expCommunityPoolAmt     sdk.DecCoins
	}{
		{
			"pass",
			sdk.NewCoin(denoms.NIBI, sdk.NewInt(1_000_000)),
			sdk.NewCoin(denoms.NIBI, sdk.NewInt(278_000)),
			sdk.NewCoin(denoms.NIBI, sdk.NewInt(100_000)),
			sdk.NewDecCoins(sdk.NewDecCoin(denoms.NIBI, sdk.NewInt(622_000))),
		},
		{
			"pass - no coins minted ",
			sdk.NewCoin(denoms.NIBI, sdk.ZeroInt()),
			sdk.NewCoin(denoms.NIBI, sdk.ZeroInt()),
			sdk.NewCoin(denoms.NIBI, sdk.ZeroInt()),
			sdk.DecCoins(nil),
		},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Case %s", tc.name), func(t *testing.T) {
			nibiruApp, ctx := testapp.NewNibiruTestAppAndContext(true)

			_, _, _, err := nibiruApp.InflationKeeper.MintAndAllocateInflation(ctx, tc.mintCoin, types.DefaultParams())

			// Get balances
			balanceInflationModule := nibiruApp.BankKeeper.GetBalance(
				ctx,
				nibiruApp.AccountKeeper.GetModuleAddress(types.ModuleName),
				denoms.NIBI,
			)

			feeCollector := nibiruApp.AccountKeeper.GetModuleAddress(authtypes.FeeCollectorName)
			balanceStakingRewards := nibiruApp.BankKeeper.GetBalance(
				ctx,
				feeCollector,
				denoms.NIBI,
			)

			balanceCommunityPool := nibiruApp.DistrKeeper.GetFeePoolCommunityCoins(ctx)

			require.NoError(t, err, tc.name)
			assert.Equal(t, tc.expStakingRewardAmt, balanceStakingRewards)
			assert.Equal(t, tc.expStrategicReservesAmt, balanceInflationModule)
			assert.Equal(t, tc.expCommunityPoolAmt, balanceCommunityPool)
		})
	}
}

func TestGetCirculatingSupplyAndInflationRate(t *testing.T) {
	testCases := []struct {
		name             string
		supply           sdkmath.Int
		malleate         func(nibiruApp *app.NibiruApp, ctx sdk.Context)
		expInflationRate sdk.Dec
	}{
		{
			"no epochs per period",
			sdk.TokensFromConsensusPower(400_000_000, sdk.DefaultPowerReduction),
			func(nibiruApp *app.NibiruApp, ctx sdk.Context) {
				nibiruApp.InflationKeeper.SetParams(ctx, types.Params{
					EpochsPerPeriod:        0,
					InflationEnabled:       true,
					ExponentialCalculation: types.DefaultExponentialCalculation,
					InflationDistribution:  types.DefaultInflationDistribution,
				})
			},
			sdk.ZeroDec(),
		},
		{
			"high supply",
			sdk.TokensFromConsensusPower(800_000_000, sdk.DefaultPowerReduction),
			func(nibiruApp *app.NibiruApp, ctx sdk.Context) {},
			sdk.MustNewDecFromStr("50.674438476562500000"),
		},
		{
			"low supply",
			sdk.TokensFromConsensusPower(400_000_000, sdk.DefaultPowerReduction),
			func(nibiruApp *app.NibiruApp, ctx sdk.Context) {},
			sdk.MustNewDecFromStr("101.348876953125000000"),
		},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Case %s", tc.name), func(t *testing.T) {
			nibiruApp, ctx := testapp.NewNibiruTestAppAndContext(true)

			tc.malleate(nibiruApp, ctx)

			// Mint coins to increase supply
			coin := sdk.NewCoin(
				denoms.NIBI,
				tc.supply,
			)
			err := nibiruApp.InflationKeeper.MintCoins(ctx, coin)
			require.NoError(t, err)

			circulatingSupply := nibiruApp.InflationKeeper.GetCirculatingSupply(ctx, denoms.NIBI)
			require.EqualValues(t, tc.supply, circulatingSupply)

			inflationRate := nibiruApp.InflationKeeper.GetInflationRate(ctx, denoms.NIBI)
			require.Equal(t, tc.expInflationRate, inflationRate)
		})
	}
}
