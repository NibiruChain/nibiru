package keeper_test

import (
	"fmt"
	"testing"

	"cosmossdk.io/math"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/x/inflation/types"
	sudotypes "github.com/NibiruChain/nibiru/x/sudo/types"
)

func init() {
	testapp.EnsureNibiruPrefix()
}

func TestMintAndAllocateInflation(t *testing.T) {
	testCases := []struct {
		name                             string
		coinsToMint                      sdk.Coin
		expectedStakingAmt               sdk.Coin
		expectedStrategicAmt             sdk.Coin
		expectedCommunityAmt             sdk.Coin
		expectedStakingRewardsBalance    sdk.Coin
		expectedStrategicReservesBalance sdk.Coin
		expectedCommunityPoolBalance     math.LegacyDecCoins
		rootAccount                      string
	}{
		{
			name:                             "pass",
			coinsToMint:                      sdk.NewCoin(denoms.NIBI, math.NewInt(1_000_000)),
			expectedStakingAmt:               sdk.NewCoin(denoms.NIBI, math.NewInt(281_250)),
			expectedStrategicAmt:             sdk.NewCoin(denoms.NIBI, math.NewInt(363_925)),
			expectedCommunityAmt:             sdk.NewCoin(denoms.NIBI, math.NewInt(354_825)),
			expectedStakingRewardsBalance:    sdk.NewCoin(denoms.NIBI, math.NewInt(281_250)),
			expectedStrategicReservesBalance: sdk.NewCoin(denoms.NIBI, math.NewInt(363_925)),
			expectedCommunityPoolBalance:     sdk.NewDecCoins(sdk.NewDecCoin(denoms.NIBI, math.NewInt(354_825))),
			rootAccount:                      "nibi1qyqf35fkhn73hjr70442fctpq8prpqr9ysj9sn",
		},
		{
			name:                             "pass - no coins minted ",
			coinsToMint:                      sdk.NewCoin(denoms.NIBI, math.ZeroInt()),
			expectedStakingAmt:               sdk.Coin{},
			expectedStrategicAmt:             sdk.Coin{},
			expectedCommunityAmt:             sdk.Coin{},
			expectedStakingRewardsBalance:    sdk.NewCoin(denoms.NIBI, math.ZeroInt()),
			expectedStrategicReservesBalance: sdk.NewCoin(denoms.NIBI, math.ZeroInt()),
			expectedCommunityPoolBalance:     nil,
			rootAccount:                      "nibi1qyqf35fkhn73hjr70442fctpq8prpqr9ysj9sn",
		},
		{
			name:                             "pass - no root account",
			coinsToMint:                      sdk.NewCoin(denoms.NIBI, math.NewInt(1_000_000)),
			expectedStakingAmt:               sdk.NewCoin(denoms.NIBI, math.NewInt(281_250)),
			expectedStrategicAmt:             sdk.NewCoin(denoms.NIBI, math.NewInt(363_925)),
			expectedCommunityAmt:             sdk.NewCoin(denoms.NIBI, math.NewInt(354_825)),
			expectedStakingRewardsBalance:    sdk.NewCoin(denoms.NIBI, math.NewInt(281_250)),
			expectedStrategicReservesBalance: sdk.NewCoin(denoms.NIBI, math.NewInt(363_925)),
			expectedCommunityPoolBalance:     sdk.NewDecCoins(sdk.NewDecCoin(denoms.NIBI, math.NewInt(354_825))),
			rootAccount:                      "",
		},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Case %s", tc.name), func(t *testing.T) {
			nibiruApp, ctx := testapp.NewNibiruTestAppAndContext()

			t.Logf("setting root account to %s", tc.rootAccount)
			nibiruApp.SudoKeeper.Sudoers.Set(ctx, sudotypes.Sudoers{
				Root:      tc.rootAccount,
				Contracts: []string{},
			})

			staking, strategic, community, err := nibiruApp.InflationKeeper.MintAndAllocateInflation(ctx, tc.coinsToMint, types.DefaultParams())
			if tc.rootAccount != "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				return
			}
			assert.Equal(t, tc.expectedStakingAmt, staking)
			assert.Equal(t, tc.expectedStrategicAmt, strategic)
			assert.Equal(t, tc.expectedCommunityAmt, community)

			// Get balances
			var balanceStrategicReserve sdk.Coin
			if tc.rootAccount != "" {
				strategicAccount, err := nibiruApp.SudoKeeper.GetRootAddr(ctx)
				require.NoError(t, err)
				balanceStrategicReserve = nibiruApp.BankKeeper.GetBalance(
					ctx,
					strategicAccount,
					denoms.NIBI,
				)
			} else {
				// if no root account is specified, then the strategic reserve remains in the x/inflation module account
				balanceStrategicReserve = nibiruApp.BankKeeper.GetBalance(ctx, nibiruApp.AccountKeeper.GetModuleAddress(types.ModuleName), denoms.NIBI)
			}

			balanceStakingRewards := nibiruApp.BankKeeper.GetBalance(
				ctx,
				nibiruApp.AccountKeeper.GetModuleAddress(authtypes.FeeCollectorName),
				denoms.NIBI,
			)

			balanceCommunityPool := nibiruApp.DistrKeeper.GetFeePoolCommunityCoins(ctx)

			require.NoError(t, err, tc.name)
			assert.Equal(t,
				tc.expectedStakingRewardsBalance.String(),
				balanceStakingRewards.String())
			assert.Equal(t,
				tc.expectedStrategicReservesBalance.String(),
				balanceStrategicReserve.String())
			assert.Equal(t,
				tc.expectedCommunityPoolBalance.String(),
				balanceCommunityPool.String())
		})
	}
}

func TestGetCirculatingSupplyAndInflationRate(t *testing.T) {
	testCases := []struct {
		name             string
		supply           sdkmath.Int
		malleate         func(nibiruApp *app.NibiruApp, ctx sdk.Context)
		expInflationRate math.LegacyDec
	}{
		{
			"no epochs per period",
			sdk.TokensFromConsensusPower(400_000_000, sdk.DefaultPowerReduction),
			func(nibiruApp *app.NibiruApp, ctx sdk.Context) {
				nibiruApp.InflationKeeper.Params.Set(ctx, types.Params{
					EpochsPerPeriod:       0,
					InflationEnabled:      true,
					PolynomialFactors:     types.DefaultPolynomialFactors,
					InflationDistribution: types.DefaultInflationDistribution,
				})
			},
			math.LegacyZeroDec(),
		},
		{
			"high supply",
			sdk.TokensFromConsensusPower(800_000_000, sdk.DefaultPowerReduction),
			func(nibiruApp *app.NibiruApp, ctx sdk.Context) {
				params := nibiruApp.InflationKeeper.GetParams(ctx)
				params.InflationEnabled = true
				nibiruApp.InflationKeeper.Params.Set(ctx, params)
			},
			math.LegacyMustNewDecFromStr("26.741197359810099000"),
		},
		{
			"low supply",
			sdk.TokensFromConsensusPower(400_000_000, sdk.DefaultPowerReduction),
			func(nibiruApp *app.NibiruApp, ctx sdk.Context) {
				params := nibiruApp.InflationKeeper.GetParams(ctx)
				params.InflationEnabled = true
				nibiruApp.InflationKeeper.Params.Set(ctx, params)
			},
			math.LegacyMustNewDecFromStr("53.482394719620198000"),
		},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Case %s", tc.name), func(t *testing.T) {
			nibiruApp, ctx := testapp.NewNibiruTestAppAndContext()

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

func TestGetters(t *testing.T) {
	nibiruApp, ctx := testapp.NewNibiruTestAppAndContext()
	k := nibiruApp.InflationKeeper
	require.NotPanics(t, func() {
		_ = k.GetPolynomialFactors(ctx)
		_ = k.GetPeriodsPerYear(ctx)
		_ = k.GetInflationDistribution(ctx)
		_ = k.GetInflationEnabled(ctx)
		_ = k.GetEpochsPerPeriod(ctx)
	})
}
