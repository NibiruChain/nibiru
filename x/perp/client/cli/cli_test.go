package cli_test

import (
	"fmt"
	"testing"

	"github.com/NibiruChain/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankcli "github.com/cosmos/cosmos-sdk/x/bank/client/cli"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	abcitypes "github.com/tendermint/tendermint/abci/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/simapp"
	"github.com/NibiruChain/nibiru/x/common"
	oracletypes "github.com/NibiruChain/nibiru/x/oracle/types"
	"github.com/NibiruChain/nibiru/x/perp/client/cli"
	perptypes "github.com/NibiruChain/nibiru/x/perp/types"
	testutilcli "github.com/NibiruChain/nibiru/x/testutil/cli"
	vpooltypes "github.com/NibiruChain/nibiru/x/vpool/types"
)

type IntegrationTestSuite struct {
	suite.Suite

	cfg     testutilcli.Config
	network *testutilcli.Network
	users   []sdk.AccAddress
}

func (s *IntegrationTestSuite) SetupSuite() {
	/* 	Make test skip if -short is not used:
	All tests: `go test ./...`
	Unit tests only: `go test ./... -short`
	Integration tests only: `go test ./... -run Integration`
	https://stackoverflow.com/a/41407042/13305627 */
	if testing.Short() {
		s.T().Skip("skipping integration test suite")
	}

	s.T().Log("setting up integration test suite")

	app.SetPrefixes(app.AccountAddressPrefix)
	encodingConfig := app.MakeTestEncodingConfig()
	genesisState := simapp.NewTestGenesisStateFromDefault()

	// setup vpool
	vpoolGenesis := vpooltypes.DefaultGenesis()
	vpoolGenesis.Vpools = []vpooltypes.Vpool{
		{
			Pair:              common.Pair_BTC_NUSD,
			BaseAssetReserve:  sdk.NewDec(10 * common.Precision),
			QuoteAssetReserve: sdk.NewDec(60_000 * common.Precision),
			Config: vpooltypes.VpoolConfig{
				TradeLimitRatio:        sdk.MustNewDecFromStr("0.8"),
				FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.2"),
				MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.2"),
				MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
				MaxLeverage:            sdk.MustNewDecFromStr("15"),
			},
		},
		{
			Pair:              common.Pair_ETH_NUSD,
			BaseAssetReserve:  sdk.NewDec(10 * common.Precision),
			QuoteAssetReserve: sdk.NewDec(60_000 * common.Precision),
			Config: vpooltypes.VpoolConfig{
				TradeLimitRatio:        sdk.MustNewDecFromStr("0.8"),
				FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.2"),
				MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.2"),
				MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
				MaxLeverage:            sdk.MustNewDecFromStr("15"),
			},
		},
	}
	genesisState[vpooltypes.ModuleName] = encodingConfig.Marshaler.MustMarshalJSON(vpoolGenesis)

	// setup perp
	perpGenesis := perptypes.DefaultGenesis()
	perpGenesis.PairMetadata = []perptypes.PairMetadata{
		{
			Pair:                            common.Pair_BTC_NUSD,
			LatestCumulativePremiumFraction: sdk.NewDec(2),
		},
		{
			Pair:                            common.Pair_ETH_NUSD,
			LatestCumulativePremiumFraction: sdk.ZeroDec(),
		},
	}
	perpGenesis.Params.WhitelistedLiquidators = []string{"nibi1w89pf5yq8ntjg89048qmtaz929fdxup0a57d8m"} // address associated with mnemonic below
	genesisState[perptypes.ModuleName] = encodingConfig.Marshaler.MustMarshalJSON(perpGenesis)

	oracleGenesis := oracletypes.DefaultGenesisState()
	oracleGenesis.Params.Whitelist = []common.AssetPair{
		common.Pair_BTC_NUSD,
	}
	oracleGenesis.Params.VotePeriod = 1_000
	oracleGenesis.ExchangeRates = []oracletypes.ExchangeRateTuple{
		{Pair: common.Pair_BTC_NUSD, ExchangeRate: sdk.NewDec(20_000)},
		{Pair: common.Pair_ETH_NUSD, ExchangeRate: sdk.NewDec(2_000)},
	}
	genesisState[oracletypes.ModuleName] = encodingConfig.Marshaler.MustMarshalJSON(oracleGenesis)

	s.cfg = testutilcli.BuildNetworkConfig(genesisState)
	s.cfg.NumValidators = 1
	s.cfg.Mnemonics = []string{"satisfy december text daring wheat vanish save viable holiday rural vessel shuffle dice skate promote fade badge federal sail during lend fever balance give"}
	s.network = testutilcli.NewNetwork(s.T(), s.cfg)
	s.NoError(s.network.WaitForNextBlock())

	user1 := testutilcli.NewAccount(s.network, "user1")
	user2 := testutilcli.NewAccount(s.network, "user2")
	user3 := testutilcli.NewAccount(s.network, "user3")
	user4 := testutilcli.NewAccount(s.network, "user4")
	user5 := sdk.MustAccAddressFromBech32("nibi1w89pf5yq8ntjg89048qmtaz929fdxup0a57d8m")
	s.users = []sdk.AccAddress{user1, user2, user3, user4, user5}

	val := s.network.Validators[0]
	_, err := s.network.WaitForHeight(2)
	require.NoError(s.T(), err)

	s.NoError(
		testutilcli.FillWalletFromValidator(user1,
			sdk.NewCoins(
				sdk.NewInt64Coin(common.DenomNIBI, 10*common.Precision),
				sdk.NewInt64Coin(common.DenomUSDC, 10*common.Precision),
				sdk.NewInt64Coin(common.DenomNUSD, 50*common.Precision),
			),
			val,
			common.DenomNIBI,
		),
	)

	s.NoError(
		testutilcli.FillWalletFromValidator(user2,
			sdk.NewCoins(
				sdk.NewInt64Coin(common.DenomNIBI, 1000),
				sdk.NewInt64Coin(common.DenomUSDC, 1000),
				sdk.NewInt64Coin(common.DenomNUSD, 100000),
			),
			val,
			common.DenomNIBI,
		),
	)

	s.NoError(
		testutilcli.FillWalletFromValidator(user3,
			sdk.NewCoins(
				sdk.NewInt64Coin(common.DenomNIBI, 1000),
				sdk.NewInt64Coin(common.DenomUSDC, 1000),
				sdk.NewInt64Coin(common.DenomNUSD, 49*common.Precision),
			),
			val,
			common.DenomNIBI,
		),
	)

	s.NoError(
		testutilcli.FillWalletFromValidator(user4,
			sdk.NewCoins(
				sdk.NewInt64Coin(common.DenomNIBI, 1000),
				sdk.NewInt64Coin(common.DenomUSDC, 1000),
				sdk.NewInt64Coin(common.DenomNUSD, 100000),
			),
			val,
			common.DenomNIBI,
		),
	)

	s.NoError(
		testutilcli.FillWalletFromValidator(user5,
			sdk.NewCoins(sdk.NewInt64Coin(common.DenomNIBI, 1000)),
			val,
			common.DenomNIBI,
		),
	)
}

func (s *IntegrationTestSuite) TearDownSuite() {
	s.T().Log("tearing down integration test suite")
	s.network.Cleanup()
}

func (s *IntegrationTestSuite) TestOpenPositionsAndCloseCmd() {
	val := s.network.Validators[0]
	user := s.users[0]

	exchangeRate, err := testutilcli.QueryOracleExchangeRate(val.ClientCtx, common.Pair_BTC_NUSD)
	s.T().Logf("0. current exchange rate is: %+v", exchangeRate)
	s.NoError(err)

	s.T().Log("A. check vpool balances")
	reserveAssets, err := testutilcli.QueryVpoolReserveAssets(val.ClientCtx, common.Pair_BTC_NUSD)
	s.T().Logf("reserve assets: %+v", reserveAssets)
	s.NoError(err)
	s.EqualValues(sdk.NewDec(10*common.Precision), reserveAssets.BaseAssetReserve)
	s.EqualValues(sdk.NewDec(60_000*common.Precision), reserveAssets.QuoteAssetReserve)

	s.T().Log("A. check trader has no existing positions")
	_, err = testutilcli.QueryPosition(val.ClientCtx, common.Pair_BTC_NUSD, user)
	s.Error(err)

	s.T().Log("B. open position")
	txResp, err := testutilcli.ExecTx(s.network, cli.OpenPositionCmd(), user, []string{
		"buy",
		common.Pair_BTC_NUSD.String(),
		/* leverage */ "1",
		/* quoteAmt */ "1000000", // 10^6 uNUSD
		/* baseAssetLimit */ "1"},
	)
	s.NoError(err)
	s.EqualValues(abcitypes.CodeTypeOK, txResp.Code)

	s.T().Log("B. check vpool balance after open position")
	reserveAssets, err = testutilcli.QueryVpoolReserveAssets(val.ClientCtx, common.Pair_BTC_NUSD)
	s.T().Logf("reserve assets: %+v", reserveAssets)
	s.NoError(err)
	s.EqualValues(sdk.MustNewDecFromStr("9999833.336111064815586407"), reserveAssets.BaseAssetReserve)
	s.EqualValues(sdk.NewDec(60_001*common.Precision), reserveAssets.QuoteAssetReserve)

	s.T().Log("B. check trader position")
	queryResp, err := testutilcli.QueryPosition(val.ClientCtx, common.Pair_BTC_NUSD, user)
	s.NoError(err)
	s.T().Logf("query response: %+v", queryResp)
	s.EqualValues(user.String(), queryResp.Position.TraderAddress)
	s.EqualValues(common.Pair_BTC_NUSD, queryResp.Position.Pair)
	s.EqualValues(sdk.MustNewDecFromStr("166.663888935184413593"), queryResp.Position.Size_)
	s.EqualValues(sdk.NewDec(1*common.Precision), queryResp.Position.Margin)
	s.EqualValues(sdk.NewDec(1*common.Precision), queryResp.Position.OpenNotional)
	s.EqualValues(sdk.MustNewDecFromStr("999999.999999999999999359"), queryResp.PositionNotional)
	s.EqualValues(sdk.MustNewDecFromStr("-0.000000000000000641"), queryResp.UnrealizedPnl)
	s.EqualValues(sdk.NewDec(1), queryResp.MarginRatioMark)
	s.EqualValues(sdk.NewDec(1), queryResp.MarginRatioIndex)

	s.T().Log("C. open position with 2x leverage and zero baseAmtLimit")
	txResp, err = testutilcli.ExecTx(s.network, cli.OpenPositionCmd(), user, []string{
		"buy",
		common.Pair_BTC_NUSD.String(),
		/* leverage */ "2",
		/* quoteAmt */ "1000000", // 10^6 uNUSD
		/* baseAmtLimit */ "0",
	})
	s.NoError(err)
	s.EqualValues(abcitypes.CodeTypeOK, txResp.Code)

	s.T().Log("C. check trader position")
	queryResp, err = testutilcli.QueryPosition(val.ClientCtx, common.Pair_BTC_NUSD, user)
	s.NoError(err)
	s.T().Logf("query response: %+v", queryResp)
	s.EqualValues(user.String(), queryResp.Position.TraderAddress)
	s.EqualValues(common.Pair_BTC_NUSD, queryResp.Position.Pair)
	s.EqualValues(sdk.MustNewDecFromStr("499.975001249937503125"), queryResp.Position.Size_)
	s.EqualValues(sdk.NewDec(2*common.Precision), queryResp.Position.Margin)
	s.EqualValues(sdk.NewDec(3*common.Precision), queryResp.Position.OpenNotional)
	s.EqualValues(sdk.MustNewDecFromStr("3000000.000000000000000938"), queryResp.PositionNotional)
	s.EqualValues(sdk.MustNewDecFromStr("0.000000000000000938"), queryResp.UnrealizedPnl)
	s.EqualValues(sdk.MustNewDecFromStr("0.666666666666666667"), queryResp.MarginRatioMark)

	s.T().Log("D. Open a reverse position smaller than the existing position")
	txResp, err = testutilcli.ExecTx(s.network, cli.OpenPositionCmd(), user, []string{
		"sell",
		common.Pair_BTC_NUSD.String(),
		/* leverage */ "1",
		/* quoteAmt */ "100", // 100 uNUSD
		/* baseAssetLimit */ "1",
	})
	s.NoError(err)
	s.EqualValues(abcitypes.CodeTypeOK, txResp.Code)

	s.T().Log("D. Check vpool after opening reverse position")
	reserveAssets, err = testutilcli.QueryVpoolReserveAssets(val.ClientCtx, common.Pair_BTC_NUSD)
	s.NoError(err)
	s.T().Logf(" \n reserve assets: %+v \n", reserveAssets)
	s.EqualValues(sdk.MustNewDecFromStr("9999500.041663750215262154"), reserveAssets.BaseAssetReserve)
	s.EqualValues(sdk.NewDec(60_002_999_900), reserveAssets.QuoteAssetReserve)

	s.T().Log("D. Check trader position")
	queryResp, err = testutilcli.QueryPosition(val.ClientCtx, common.Pair_BTC_NUSD, user)
	s.NoError(err)
	s.T().Logf("query response: %+v", queryResp)
	s.EqualValues(user.String(), queryResp.Position.TraderAddress)
	s.EqualValues(common.Pair_BTC_NUSD, queryResp.Position.Pair)
	s.EqualValues(sdk.MustNewDecFromStr("499.958336249784737846"), queryResp.Position.Size_)
	s.EqualValues(sdk.NewDec(2*common.Precision), queryResp.Position.Margin)
	s.EqualValues(sdk.NewDec(2_999_900), queryResp.Position.OpenNotional)
	s.EqualValues(sdk.MustNewDecFromStr("2999899.999999999999999506"), queryResp.PositionNotional)
	s.EqualValues(sdk.MustNewDecFromStr("-0.000000000000000494"), queryResp.UnrealizedPnl)
	s.EqualValues(sdk.MustNewDecFromStr("0.666688889629654322"), queryResp.MarginRatioMark)

	s.T().Log("E. Open a reverse position larger than the existing position")
	txResp, err = testutilcli.ExecTx(s.network, cli.OpenPositionCmd(), user, []string{
		"sell",
		common.Pair_BTC_NUSD.String(),
		/* leverage */ "1",
		/* quoteAmt */ "4000000", // 4*10^6 uNUSD
		/* baseAssetLimit */ "0",
	})
	s.NoError(err)
	s.EqualValues(abcitypes.CodeTypeOK, txResp.Code)

	s.T().Log("E. Check trader position")
	queryResp, err = testutilcli.QueryPosition(val.ClientCtx, common.Pair_BTC_NUSD, user)
	s.NoError(err)
	s.T().Logf("query response: %+v", queryResp)
	s.EqualValues(user.String(), queryResp.Position.TraderAddress)
	s.EqualValues(common.Pair_BTC_NUSD, queryResp.Position.Pair)
	s.EqualValues(sdk.MustNewDecFromStr("-166.686111713005402945"), queryResp.Position.Size_)
	s.EqualValues(sdk.MustNewDecFromStr("1000100.000000000000000494"), queryResp.Position.OpenNotional)
	s.EqualValues(sdk.MustNewDecFromStr("1000100.000000000000000494"), queryResp.Position.Margin)
	s.EqualValues(sdk.MustNewDecFromStr("1000099.999999999999999651"), queryResp.PositionNotional)
	s.EqualValues(sdk.MustNewDecFromStr("0.000000000000000843"), queryResp.UnrealizedPnl)
	// there is a random delta due to twap margin ratio calculation and random block times in the in-process network
	s.InDelta(1, queryResp.MarginRatioMark.MustFloat64(), 0.008)

	s.T().Log("F. Close position")
	txResp, err = testutilcli.ExecTx(s.network, cli.ClosePositionCmd(), user, []string{
		common.Pair_BTC_NUSD.String(),
	})
	s.NoError(err)
	s.EqualValues(abcitypes.CodeTypeOK, txResp.Code)

	s.T().Log("F. check trader position")
	queryResp, err = testutilcli.QueryPosition(val.ClientCtx, common.Pair_BTC_NUSD, user)
	s.Error(err)
	s.T().Logf("query response: %+v", queryResp)

	status, ok := status.FromError(err)
	s.True(ok)
	s.EqualValues(codes.InvalidArgument, status.Code())
}

func (s *IntegrationTestSuite) TestPositionEmptyAndClose() {
	val := s.network.Validators[0]
	user := s.users[0]

	// verify trader has no position (empty)
	_, err := testutilcli.QueryPosition(val.ClientCtx, common.Pair_ETH_NUSD, user)
	s.Error(err, "no position found")

	// close position should produce error
	_, err = testutilcli.ExecTx(s.network, cli.ClosePositionCmd(), user, []string{
		common.Pair_ETH_NUSD.String(),
	})
	s.Contains(err.Error(), collections.ErrNotFound.Error())
}

func (s *IntegrationTestSuite) TestQueryCumulativePremiumFractions() {
	val := s.network.Validators[0]

	s.T().Log("get cumulative funding payments")
	queryResp, err := testutilcli.QueryCumulativePremiumFraction(val.ClientCtx, common.Pair_BTC_NUSD)
	s.NoError(err)
	s.EqualValues(sdk.NewDec(2), queryResp.CumulativePremiumFraction)
}

func (s *IntegrationTestSuite) TestRemoveMargin() {
	// Open a position with first user
	s.T().Log("opening a position with user 0")
	txResp, err := testutilcli.ExecTx(s.network, cli.OpenPositionCmd(), s.users[0], []string{
		"buy",
		common.Pair_BTC_NUSD.String(),
		"10", // Leverage
		"1",  // Quote asset amount
		"0.0000001",
	})
	s.NoError(err)
	s.EqualValues(abcitypes.CodeTypeOK, txResp.Code)

	// Remove margin to trigger bad debt on user 0
	s.T().Log("removing margin on user 0....")
	_, err = testutilcli.ExecTx(s.network, cli.RemoveMarginCmd(), s.users[0], []string{
		common.Pair_BTC_NUSD.String(),
		fmt.Sprintf("%s%s", "100", common.DenomNUSD),
	})
	s.Contains(err.Error(), perptypes.ErrFailedRemoveMarginCanCauseBadDebt.Error())
}

func (s *IntegrationTestSuite) TestX_AddMargin() {
	val := s.network.Validators[0]
	pair := common.Pair_ETH_NUSD

	// Open a new position
	s.T().Log("opening a position with user 3....")
	txResp, err := testutilcli.ExecTx(s.network, cli.OpenPositionCmd(), s.users[3], []string{
		"buy",
		pair.String(),
		"10",    // Leverage
		"10000", // Quote asset amount
		"0.0000001",
	})
	s.NoError(err)
	s.EqualValues(abcitypes.CodeTypeOK, txResp.Code)

	testCases := []struct {
		name           string
		args           []string
		expectedCode   uint32
		expectedMargin sdk.Dec
	}{
		{
			name: "PASS: add margin to correct position",
			args: []string{
				pair.String(),
				fmt.Sprintf("%s%s", "10000", pair.QuoteDenom()),
			},
			expectedCode:   0,
			expectedMargin: sdk.NewDec(20_000),
		},
		{
			name: "FAIL: position not found",
			args: []string{
				common.Pair_BTC_NUSD.String(),
				fmt.Sprintf("%s%s", "10000", pair.QuoteDenom()),
			},
			expectedCode: 1,
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			s.T().Log("adding margin on user 3....")
			txResp, err = testutilcli.ExecTx(s.network, cli.AddMarginCmd(), s.users[3], tc.args, testutilcli.WithTxCanFail(true))
			s.NoError(err)
			s.EqualValues(tc.expectedCode, txResp.Code)

			if tc.expectedCode == 0 {
				// query trader position
				queryResp, err := testutilcli.QueryPosition(val.ClientCtx, pair, s.users[3])
				s.NoError(err)
				s.EqualValues(tc.expectedMargin, queryResp.Position.Margin)
			}
		})
	}
}

func (s *IntegrationTestSuite) TestLiquidate() {
	s.T().Log("liquidate a position that does not exist")
	_, err := testutilcli.ExecTx(s.network, cli.LiquidateCmd(), s.users[4], []string{
		common.Pair_ETH_NUSD.String(),
		s.users[1].String(),
	})
	s.Contains(err.Error(), collections.ErrNotFound.Error())

	s.T().Log("opening a position with user 1....")
	txResp, err := testutilcli.ExecTx(s.network, cli.OpenPositionCmd(), s.users[1], []string{
		"buy",
		common.Pair_ETH_NUSD.String(),
		"15",    // Leverage
		"90000", // Quote asset amount
		"0",
	})
	s.NoError(err)
	s.EqualValues(abcitypes.CodeTypeOK, txResp.Code)

	s.T().Log("liquidate a position that is above maintenance margin mario")
	_, err = testutilcli.ExecTx(s.network, cli.LiquidateCmd(), s.users[4], []string{
		common.Pair_ETH_NUSD.String(),
		s.users[1].String(),
	})
	s.Contains(err.Error(), "margin ratio is too healthy to liquidate")

	s.T().Log("opening a position with user 2...")
	txResp, err = testutilcli.ExecTx(s.network, cli.OpenPositionCmd(), s.users[2], []string{
		"sell",
		common.Pair_ETH_NUSD.String(),
		"15",       // Leverage
		"45000000", // Quote asset amount
		"0",
	})
	s.NoError(err)
	s.EqualValues(abcitypes.CodeTypeOK, txResp.Code)

	s.T().Log("wait 10 blocks")
	height, err := s.network.LatestHeight()
	s.NoError(err)
	_, err = s.network.WaitForHeight(height + 10)
	s.NoError(err)

	s.T().Log("liquidating user 1...")
	txResp, err = testutilcli.ExecTx(s.network, cli.LiquidateCmd(), s.users[4], []string{
		common.Pair_ETH_NUSD.String(),
		s.users[1].String(),
	})
	s.NoError(err)
	s.EqualValues(abcitypes.CodeTypeOK, txResp.Code)
}

func (s *IntegrationTestSuite) TestDonateToEcosystemFund() {
	s.T().Logf("donate to ecosystem fund")
	out, err := testutilcli.ExecTx(s.network, cli.DonateToEcosystemFundCmd(), sdk.MustAccAddressFromBech32("nibi1w89pf5yq8ntjg89048qmtaz929fdxup0a57d8m"), []string{"100unusd"})
	s.NoError(err)
	s.Require().EqualValues(abcitypes.CodeTypeOK, out.Code)

	s.NoError(s.network.WaitForNextBlock())

	resp := new(sdk.Coin)
	s.NoError(
		testutilcli.ExecQuery(
			s.network.Validators[0].ClientCtx,
			bankcli.GetBalancesCmd(),
			[]string{"nibi1trh2mamq64u4g042zfeevvjk4cukrthvppfnc7", "--denom", "unusd"}, // nibi1trh2mamq64u4g042zfeevvjk4cukrthvppfnc7 is the perp_ef module account address
			resp,
		),
	)
	s.Require().EqualValues(sdk.NewInt64Coin("unusd", 100), *resp)
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
