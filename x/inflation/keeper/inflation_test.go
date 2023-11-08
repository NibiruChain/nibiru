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
		expectedCommunityPoolBalance     sdk.DecCoins
		rootAccount                      string
	}{
		{
			name:                             "pass",
			coinsToMint:                      sdk.NewCoin(denoms.NIBI, sdk.NewInt(1_000_000)),
			expectedStakingAmt:               sdk.NewCoin(denoms.NIBI, sdk.NewInt(278_000)),
			expectedStrategicAmt:             sdk.NewCoin(denoms.NIBI, sdk.NewInt(100_000)),
			expectedCommunityAmt:             sdk.NewCoin(denoms.NIBI, sdk.NewInt(622_000)),
			expectedStakingRewardsBalance:    sdk.NewCoin(denoms.NIBI, sdk.NewInt(278_000)),
			expectedStrategicReservesBalance: sdk.NewCoin(denoms.NIBI, sdk.NewInt(100_000)),
			expectedCommunityPoolBalance:     sdk.NewDecCoins(sdk.NewDecCoin(denoms.NIBI, sdk.NewInt(622_000))),
			rootAccount:                      "nibi1qyqf35fkhn73hjr70442fctpq8prpqr9ysj9sn",
		},
		{
			name:                             "pass - no coins minted ",
			coinsToMint:                      sdk.NewCoin(denoms.NIBI, sdk.ZeroInt()),
			expectedStakingAmt:               sdk.Coin{},
			expectedStrategicAmt:             sdk.Coin{},
			expectedCommunityAmt:             sdk.Coin{},
			expectedStakingRewardsBalance:    sdk.NewCoin(denoms.NIBI, sdk.ZeroInt()),
			expectedStrategicReservesBalance: sdk.NewCoin(denoms.NIBI, sdk.ZeroInt()),
			expectedCommunityPoolBalance:     nil,
			rootAccount:                      "nibi1qyqf35fkhn73hjr70442fctpq8prpqr9ysj9sn",
		},
		{
			name:                             "pass - no root account",
			coinsToMint:                      sdk.NewCoin(denoms.NIBI, sdk.NewInt(1_000_000)),
			expectedStakingAmt:               sdk.NewCoin(denoms.NIBI, sdk.NewInt(278_000)),
			expectedStrategicAmt:             sdk.NewCoin(denoms.NIBI, sdk.NewInt(100_000)),
			expectedCommunityAmt:             sdk.NewCoin(denoms.NIBI, sdk.NewInt(622_000)),
			expectedStakingRewardsBalance:    sdk.NewCoin(denoms.NIBI, sdk.NewInt(278_000)),
			expectedStrategicReservesBalance: sdk.NewCoin(denoms.NIBI, sdk.NewInt(100_000)),
			expectedCommunityPoolBalance:     sdk.NewDecCoins(sdk.NewDecCoin(denoms.NIBI, sdk.NewInt(622_000))),
			rootAccount:                      "",
		},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Case %s", tc.name), func(t *testing.T) {
			nibiruApp, ctx := testapp.NewNibiruTestAppAndContext()

			if tc.rootAccount != "" {
				t.Logf("setting root account to %s", tc.rootAccount)
				nibiruApp.SudoKeeper.Sudoers.Set(ctx, sudotypes.Sudoers{
					Root:      sdk.MustAccAddressFromBech32(tc.rootAccount).String(),
					Contracts: []string{},
				})
			}

			staking, strategic, community, err := nibiruApp.InflationKeeper.MintAndAllocateInflation(ctx, tc.coinsToMint, types.DefaultParams())
			require.NoError(t, err)
			assert.Equal(t, tc.expectedStakingAmt, staking)
			assert.Equal(t, tc.expectedStrategicAmt, strategic)
			assert.Equal(t, tc.expectedCommunityAmt, community)

			// Get balances
			var balanceStrategicReserve sdk.Coin
			if tc.rootAccount != "" {
				strategicAccount, err := nibiruApp.SudoKeeper.GetRoot(ctx)
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
			assert.Equal(t, tc.expectedStakingRewardsBalance, balanceStakingRewards)
			assert.Equal(t, tc.expectedStrategicReservesBalance, balanceStrategicReserve)
			assert.Equal(t, tc.expectedCommunityPoolBalance, balanceCommunityPool)
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
					EpochsPerPeriod:       0,
					InflationEnabled:      true,
					PolynomialFactors:     types.DefaultPolynomialFactors,
					InflationDistribution: types.DefaultInflationDistribution,
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
		_ = k.PolynomialFactors(ctx)
		_ = k.InflationDistribution(ctx)
		_ = k.InflationEnabled(ctx)
		_ = k.EpochsPerPeriod(ctx)
	})
}
