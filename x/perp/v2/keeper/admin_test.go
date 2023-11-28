package keeper_test

import (
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/NibiruChain/nibiru/x/common/testutil/genesis"
	"github.com/NibiruChain/nibiru/x/common/testutil/mock"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/x/perp/v2/keeper"
	"github.com/NibiruChain/nibiru/x/perp/v2/types"
	sudotypes "github.com/NibiruChain/nibiru/x/sudo/types"
	tftypes "github.com/NibiruChain/nibiru/x/tokenfactory/types"

	. "github.com/NibiruChain/nibiru/x/common/testutil/action"
	. "github.com/NibiruChain/nibiru/x/perp/v2/integration/action"
	. "github.com/NibiruChain/nibiru/x/perp/v2/integration/assertion"
)

func TestAdmin_WithdrawFromInsuranceFund(t *testing.T) {
	expectBalance := func(
		want sdkmath.Int, t *testing.T, nibiru *app.NibiruApp, ctx sdk.Context,
	) {
		insuranceFund := nibiru.AccountKeeper.GetModuleAddress(types.PerpEFModuleAccount)
		balances := nibiru.BankKeeper.GetAllBalances(ctx, insuranceFund)
		got := balances.AmountOf(types.TestingCollateralDenomNUSD)
		require.EqualValues(t, want.String(), got.String())
	}

	setup := func() (nibiru *app.NibiruApp, ctx sdk.Context) {
		testapp.EnsureNibiruPrefix()
		nibiru, ctx = testapp.NewNibiruTestAppAndContext()
		expectBalance(sdk.ZeroInt(), t, nibiru, ctx)
		nibiru.PerpKeeperV2.Collateral.Set(ctx, types.TestingCollateralDenomNUSD)
		return nibiru, ctx
	}

	fundModule := func(t *testing.T, amount sdkmath.Int, ctx sdk.Context, nibiru *app.NibiruApp) {
		coins := sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, amount))
		err := testapp.FundModuleAccount(
			nibiru.BankKeeper, ctx, types.PerpEFModuleAccount,
			coins,
		)
		require.NoError(t, err)
	}

	testCases := []testutil.FunctionTestCase{
		{
			Name: "withdraw all",
			Test: func() {
				nibiru, ctx := setup()
				admin := testutil.AccAddress()
				amountToFund := sdk.NewInt(420)
				fundModule(t, amountToFund, ctx, nibiru)

				amountToWithdraw := amountToFund
				err := nibiru.PerpKeeperV2.Admin.WithdrawFromInsuranceFund(
					ctx, amountToWithdraw, admin)
				require.NoError(t, err)

				require.EqualValues(t,
					amountToFund.String(),
					nibiru.BankKeeper.GetBalance(ctx, admin, types.TestingCollateralDenomNUSD).Amount.String(),
				)
				expectBalance(sdk.ZeroInt(), t, nibiru, ctx)
			},
		},
		{
			Name: "withdraw too much - err",
			Test: func() {
				nibiru, ctx := setup()
				admin := testutil.AccAddress()
				amountToFund := sdk.NewInt(420)
				fundModule(t, amountToFund, ctx, nibiru)

				amountToWithdraw := amountToFund.MulRaw(5)
				err := nibiru.PerpKeeperV2.Admin.WithdrawFromInsuranceFund(
					ctx, amountToWithdraw, admin)
				require.Error(t, err)
			},
		},
	}

	testutil.RunFunctionTests(t, testCases)
}

func TestCreateMarket(t *testing.T) {
	pair := asset.Registry.Pair(denoms.BTC, denoms.NUSD)
	amm := *mock.TestAMMDefault()
	app, ctx := testapp.NewNibiruTestAppAndContext()
	admin := app.PerpKeeperV2.Admin

	// Error because of invalid market
	market := types.DefaultMarket(pair).WithMaintenanceMarginRatio(sdk.NewDec(2))
	err := admin.CreateMarket(ctx, keeper.ArgsCreateMarket{
		Pair:            pair,
		PriceMultiplier: amm.PriceMultiplier,
		SqrtDepth:       amm.SqrtDepth,
		Market:          &market, // Invalid maintenance ratio
	})
	require.ErrorContains(t, err, "maintenance margin ratio ratio must be 0 <= ratio <= 1")

	// Error because of invalid amm
	err = admin.CreateMarket(ctx, keeper.ArgsCreateMarket{
		Pair:            pair,
		PriceMultiplier: sdk.NewDec(-1),
		SqrtDepth:       amm.SqrtDepth,
	})
	require.ErrorContains(t, err, "init price multiplier must be > 0")

	// Set it correctly
	err = admin.CreateMarket(ctx, keeper.ArgsCreateMarket{
		Pair:            pair,
		PriceMultiplier: amm.PriceMultiplier,
		SqrtDepth:       amm.SqrtDepth,
		EnableMarket:    true,
	})
	require.NoError(t, err)

	lastVersion, err := app.PerpKeeperV2.MarketLastVersion.Get(ctx, pair)
	require.NoError(t, err)
	require.Equal(t, uint64(1), lastVersion.Version)

	// Check that amm and market have version 1
	amm, err = app.PerpKeeperV2.GetAMM(ctx, pair)
	require.NoError(t, err)
	require.Equal(t, uint64(1), amm.Version)

	market, err = app.PerpKeeperV2.GetMarket(ctx, pair)
	require.NoError(t, err)
	require.Equal(t, uint64(1), market.Version)

	// Fail since it already exists and it is not disabled
	err = admin.CreateMarket(ctx, keeper.ArgsCreateMarket{
		Pair:            pair,
		PriceMultiplier: amm.PriceMultiplier,
		SqrtDepth:       amm.SqrtDepth,
	})
	require.ErrorContains(t, err, "already exists")

	// Close the market to test that we can create it again but with an increased version
	err = admin.CloseMarket(ctx, pair)
	require.NoError(t, err)

	err = admin.CreateMarket(ctx, keeper.ArgsCreateMarket{
		Pair:            pair,
		PriceMultiplier: amm.PriceMultiplier,
		SqrtDepth:       amm.SqrtDepth,
	})
	require.NoError(t, err)

	lastVersion, err = app.PerpKeeperV2.MarketLastVersion.Get(ctx, pair)
	require.NoError(t, err)
	require.Equal(t, uint64(2), lastVersion.Version)

	// Check that amm and market have version 2
	amm, err = app.PerpKeeperV2.GetAMM(ctx, pair)
	require.NoError(t, err)
	require.Equal(t, uint64(2), amm.Version)

	market, err = app.PerpKeeperV2.GetMarket(ctx, pair)
	require.NoError(t, err)
	require.Equal(t, uint64(2), market.Version)
}

func TestCloseMarket(t *testing.T) {
	pairBtcUsdc := asset.Registry.Pair(denoms.BTC, denoms.NUSD)
	startTime := time.Now()
	alice := testutil.AccAddress()

	tc := TestCases{
		TC("market can be disabled").
			Given(
				CreateCustomMarket(pairBtcUsdc, WithEnabled(true)),
				SetBlockTime(startTime),
				MarketShouldBeEqual(
					pairBtcUsdc,
					Market_EnableShouldBeEqualTo(true),
				),
			).
			When(
				CloseMarket(pairBtcUsdc),
			).
			Then(
				MarketShouldBeEqual(
					pairBtcUsdc,
					Market_EnableShouldBeEqualTo(false),
				),
			),
		TC("cannot open position on disabled market").
			Given(
				CreateCustomMarket(
					pairBtcUsdc,
					WithEnabled(true),
					WithPricePeg(sdk.OneDec()),
					WithSqrtDepth(sdk.NewDec(100_000)),
				),
				SetBlockNumber(1),
				SetBlockTime(startTime),

				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(1e6)))),
			).
			When(
				CloseMarket(pairBtcUsdc),
			).
			Then(
				MarketOrderFails(
					alice,
					pairBtcUsdc,
					types.Direction_LONG,
					sdk.NewInt(10_000),
					sdk.OneDec(),
					sdk.ZeroDec(),
					types.ErrMarketNotEnabled,
				),
			),
		TC("cannot close position on disabled market").When(
			CreateCustomMarket(
				pairBtcUsdc,
				WithPricePeg(sdk.OneDec()),
				WithSqrtDepth(sdk.NewDec(100_000)),
				WithEnabled(true),
			),
			SetBlockNumber(1),
			SetBlockTime(startTime),
			FundAccount(alice, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(10_200)))),
			MarketOrder(
				alice,
				pairBtcUsdc,
				types.Direction_LONG,
				sdk.NewInt(10_000),
				sdk.OneDec(),
				sdk.ZeroDec(),
			),
		).When(
			CloseMarket(pairBtcUsdc),
			CloseMarketShouldFail(pairBtcUsdc),
			CloseMarketShouldFail("random:pair"),
		).Then(
			ClosePositionFails(alice, pairBtcUsdc, types.ErrMarketNotEnabled),
		),
		TC("cannot partial close position on disabled market").When(
			CreateCustomMarket(
				pairBtcUsdc,
				WithPricePeg(sdk.OneDec()),
				WithSqrtDepth(sdk.NewDec(100_000)),
				WithEnabled(true),
			),
			SetBlockNumber(1),
			SetBlockTime(startTime),
			FundAccount(alice, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(10_200)))),
			MarketOrder(
				alice,
				pairBtcUsdc,
				types.Direction_LONG,
				sdk.NewInt(10_000),
				sdk.OneDec(),
				sdk.ZeroDec(),
			),
		).When(
			CloseMarket(pairBtcUsdc),
			AMMShouldBeEqual(pairBtcUsdc, AMM_SettlementPriceShoulBeEqual(sdk.MustNewDecFromStr("1.1"))),
		).Then(
			PartialCloseFails(alice, pairBtcUsdc, sdk.NewDec(5_000), types.ErrMarketNotEnabled),
		),
	}

	NewTestSuite(t).WithTestCases(tc...).Run()
}

func TestAdmin_ChangeCollateralDenom(t *testing.T) {
	adminSender := testutil.AccAddress()
	nonAdminSender := testutil.AccAddress()

	setup := func() (nibiru *app.NibiruApp, ctx sdk.Context) {
		nibiru, ctx = testapp.NewNibiruTestAppAndContext()
		nibiru.SudoKeeper.Sudoers.Set(ctx, sudotypes.Sudoers{
			Root:      "mock-root", // unused
			Contracts: []string{adminSender.String()},
		})
		return nibiru, ctx
	}

	for _, tc := range []struct {
		newDenom string
		sender   sdk.AccAddress
		wantErr  string
		name     string
	}{
		{name: "happy: normal denom", newDenom: "nusd", sender: adminSender, wantErr: ""},

		{name: "happy: token factory denom",
			newDenom: tftypes.TFDenom{
				Creator:  testutil.AccAddress().String(),
				Subdenom: "nusd",
			}.String(), sender: adminSender, wantErr: ""},

		{name: "happy: token factory denom",
			newDenom: tftypes.TFDenom{
				Creator:  testutil.AccAddress().String(),
				Subdenom: "nusd",
			}.String(), sender: adminSender, wantErr: "",
		},

		{name: "happy: IBC denom",
			newDenom: "ibc/46B44899322F3CD854D2D46DEEF881958467CDD4B3B10086DA49296BBED94BED", // JUNO on Osmosis
			sender:   adminSender, wantErr: "",
		},

		{name: "sad: invalid denom",
			newDenom: "", sender: adminSender, wantErr: types.ErrInvalidCollateral.Error(),
		},
		{name: "sad: sender not in sudoers",
			newDenom: "nusd", sender: nonAdminSender, wantErr: "insufficient permissions on smart contract",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bapp, ctx := setup()
			err := bapp.PerpKeeperV2.Admin.ChangeCollateralDenom(
				ctx, tc.newDenom, tc.sender,
			)

			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)

			newDenom, err := bapp.PerpKeeperV2.Collateral.Get(ctx)
			require.NoError(t, err)
			require.Equal(t, tc.newDenom, newDenom)
		})
	}
}

type TestSuiteSmartContracts struct {
	suite.Suite

	nibiru    *app.NibiruApp
	ctx       sdk.Context
	addrAdmin sdk.AccAddress

	ratesMap map[asset.Pair]sdk.Dec
}

func (s *TestSuiteSmartContracts) SetupSuite() {
	sender := testutil.AccAddress()
	s.addrAdmin = sender

	genesisState := genesis.NewTestGenesisState(app.MakeEncodingConfig())
	genesisState = genesis.AddOracleGenesis(genesisState)
	genesisState = genesis.AddPerpV2Genesis(genesisState)
	nibiru := testapp.NewNibiruTestApp(genesisState)
	ctx := nibiru.NewContext(false, tmproto.Header{
		Height:  1,
		ChainID: "nibiru-wasmnet-1",
		Time:    time.Now().UTC(),
	})
	coins := sdk.NewCoins(
		sdk.NewCoin(denoms.NIBI, sdk.NewInt(1_000_000)),
		sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(420_000*69)),
		sdk.NewCoin(denoms.USDT, sdk.NewInt(420_000*69)),
	)
	s.NoError(testapp.FundAccount(nibiru.BankKeeper, ctx, sender, coins))

	s.nibiru = nibiru
	s.ctx = ctx
	s.nibiru.PerpKeeperV2.Collateral.Set(s.ctx, types.TestingCollateralDenomNUSD)
}

func (s *TestSuiteSmartContracts) DoPegShiftTest(pair asset.Pair) error {
	sender := s.addrAdmin
	newPegMult := sdk.NewDec(420)
	err := s.nibiru.AppKeepers.PerpKeeperV2.Admin.ShiftPegMultiplier(
		s.ctx, pair, newPegMult, sender,
	)
	return err
}

func TestAdmin_PegShift(t *testing.T) {
	pair := asset.Registry.Pair(denoms.BTC, denoms.NUSD)
	amm := *mock.TestAMMDefault()
	app, ctx := testapp.NewNibiruTestAppAndContext()
	admin := app.PerpKeeperV2.Admin

	// Error because of invalid market
	market := types.DefaultMarket(pair).WithMaintenanceMarginRatio(sdk.NewDec(2))
	err := admin.CreateMarket(ctx, keeper.ArgsCreateMarket{
		Pair:            pair,
		PriceMultiplier: amm.PriceMultiplier,
		SqrtDepth:       amm.SqrtDepth,
		Market:          &market, // Invalid maintenance ratio
	})
	require.ErrorContains(t, err, "maintenance margin ratio ratio must be 0 <= ratio <= 1")

	// Error because of invalid amm
	err = admin.CreateMarket(ctx, keeper.ArgsCreateMarket{
		Pair:            pair,
		PriceMultiplier: sdk.NewDec(-1),
		SqrtDepth:       amm.SqrtDepth,
	})
	require.ErrorContains(t, err, "init price multiplier must be > 0")

	// Set it correctly
	err = admin.CreateMarket(ctx, keeper.ArgsCreateMarket{
		Pair:            pair,
		PriceMultiplier: amm.PriceMultiplier,
		SqrtDepth:       amm.SqrtDepth,
		EnableMarket:    true,
	})
	require.NoError(t, err)

	lastVersion, err := app.PerpKeeperV2.MarketLastVersion.Get(ctx, pair)
	require.NoError(t, err)
	require.Equal(t, uint64(1), lastVersion.Version)

	// Check that amm and market have version 1
	amm, err = app.PerpKeeperV2.GetAMM(ctx, pair)
	require.NoError(t, err)
	require.Equal(t, uint64(1), amm.Version)

	market, err = app.PerpKeeperV2.GetMarket(ctx, pair)
	require.NoError(t, err)
	require.Equal(t, uint64(1), market.Version)

	// Fail since it already exists and it is not disabled
	err = admin.CreateMarket(ctx, keeper.ArgsCreateMarket{
		Pair:            pair,
		PriceMultiplier: amm.PriceMultiplier,
		SqrtDepth:       amm.SqrtDepth,
	})
	require.ErrorContains(t, err, "already exists")

	// Close the market to test that we can create it again but with an increased version
	err = admin.CloseMarket(ctx, pair)
	require.NoError(t, err)

	err = admin.CreateMarket(ctx, keeper.ArgsCreateMarket{
		Pair:            pair,
		PriceMultiplier: amm.PriceMultiplier,
		SqrtDepth:       amm.SqrtDepth,
	})
	require.NoError(t, err)

	lastVersion, err = app.PerpKeeperV2.MarketLastVersion.Get(ctx, pair)
	require.NoError(t, err)
	require.Equal(t, uint64(2), lastVersion.Version)

	// Check that amm and market have version 2
	amm, err = app.PerpKeeperV2.GetAMM(ctx, pair)
	require.NoError(t, err)
	require.Equal(t, uint64(2), amm.Version)

	market, err = app.PerpKeeperV2.GetMarket(ctx, pair)
	require.NoError(t, err)
	require.Equal(t, uint64(2), market.Version)
}
