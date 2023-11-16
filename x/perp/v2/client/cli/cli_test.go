package cli_test

import (
	"fmt"
	"testing"

	"cosmossdk.io/errors"

	abcitypes "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankcli "github.com/cosmos/cosmos-sdk/x/bank/client/cli"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	testutilcli "github.com/NibiruChain/nibiru/x/common/testutil/cli"
	"github.com/NibiruChain/nibiru/x/common/testutil/genesis"
	oracletypes "github.com/NibiruChain/nibiru/x/oracle/types"
	"github.com/NibiruChain/nibiru/x/perp/v2/client/cli"
	"github.com/NibiruChain/nibiru/x/perp/v2/types"
)

type IntegrationTestSuite struct {
	suite.Suite

	cfg        testutilcli.Config
	network    *testutilcli.Network
	users      []sdk.AccAddress
	liquidator sdk.AccAddress
}

func (s *IntegrationTestSuite) SetupSuite() {
	testutil.BeforeIntegrationSuite(s.T())

	app.SetPrefixes(app.AccountAddressPrefix)

	// setup market
	encodingConfig := app.MakeEncodingConfig()
	genState := genesis.NewTestGenesisState(encodingConfig)
	genState = genesis.AddPerpV2Genesis(genState)
	genState = genesis.AddOracleGenesis(genState)

	oracleGenesis := oracletypes.DefaultGenesisState()
	oracleGenesis.Params.Whitelist = []asset.Pair{
		asset.Registry.Pair(denoms.BTC, denoms.NUSD),
		asset.Registry.Pair(denoms.ETH, denoms.NUSD),
		asset.Registry.Pair(denoms.ATOM, denoms.NUSD),
		asset.Registry.Pair(denoms.OSMO, denoms.NUSD),
	}
	oracleGenesis.Params.VotePeriod = 1_000
	oracleGenesis.ExchangeRates = []oracletypes.ExchangeRateTuple{
		{Pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD), ExchangeRate: sdk.NewDec(20_000)},
		{Pair: asset.Registry.Pair(denoms.ETH, denoms.NUSD), ExchangeRate: sdk.NewDec(2_000)},
		{Pair: asset.Registry.Pair(denoms.ATOM, denoms.NUSD), ExchangeRate: sdk.NewDec(6_000)},
		{Pair: asset.Registry.Pair(denoms.OSMO, denoms.NUSD), ExchangeRate: sdk.NewDec(6_000)},
	}
	genState[oracletypes.ModuleName] = encodingConfig.Marshaler.MustMarshalJSON(oracleGenesis)

	s.cfg = testutilcli.BuildNetworkConfig(genState)
	s.cfg.NumValidators = 1
	s.cfg.Mnemonics = []string{"satisfy december text daring wheat vanish save viable holiday rural vessel shuffle dice skate promote fade badge federal sail during lend fever balance give"}
	network, err := testutilcli.New(
		s.T(),
		s.T().TempDir(),
		s.cfg,
	)
	s.Require().NoError(err)
	s.network = network

	s.NoError(s.network.WaitForNextBlock())

	val := s.network.Validators[0]

	for i := 0; i < 8; i++ {
		newUser := testutilcli.NewAccount(s.network, fmt.Sprintf("user%d", i))
		s.users = append(s.users, newUser)
		s.Require().NoError(
			testutilcli.FillWalletFromValidator(newUser,
				sdk.NewCoins(
					sdk.NewInt64Coin(denoms.NIBI, 10e6),
					sdk.NewInt64Coin(denoms.USDC, 1e3*common.TO_MICRO),
					sdk.NewInt64Coin(types.TestingCollateralDenomNUSD, 5e3*common.TO_MICRO),
				),
				val,
				denoms.NIBI,
			),
		)
		s.NoError(s.network.WaitForNextBlock())
	}

	s.liquidator = sdk.MustAccAddressFromBech32("nibi1w89pf5yq8ntjg89048qmtaz929fdxup0a57d8m")
	s.NoError(
		testutilcli.FillWalletFromValidator(s.liquidator,
			sdk.NewCoins(sdk.NewInt64Coin(denoms.NIBI, 1000)),
			val,
			denoms.NIBI,
		),
	)
	s.NoError(s.network.WaitForNextBlock())
}

func (s *IntegrationTestSuite) TearDownSuite() {
	s.T().Log("tearing down integration test suite")
	s.network.Cleanup()
}

func (s *IntegrationTestSuite) TestMultiLiquidate() {
	s.T().Log("opening positions")
	_, err := s.network.ExecTxCmd(cli.MarketOrderCmd(), s.users[2], []string{
		"buy",
		asset.Registry.Pair(denoms.ATOM, denoms.NUSD).String(),
		"15",      // Leverage
		"9000000", // Quote asset amount
		"0",       // Base asset limit
	})
	s.Require().NoError(err)

	_, err = s.network.ExecTxCmd(cli.MarketOrderCmd(), s.users[3], []string{
		"buy",
		asset.Registry.Pair(denoms.OSMO, denoms.NUSD).String(),
		"15",      // Leverage
		"9000000", // Quote asset amount
		"0",       // Base asset limit
	})
	s.Require().NoError(err)

	s.T().Log("opening counter positions")
	_, err = s.network.ExecTxCmd(cli.MarketOrderCmd(), s.users[4], []string{
		"sell",
		asset.Registry.Pair(denoms.ATOM, denoms.NUSD).String(),
		"15",       // Leverage
		"90000000", // Quote asset amount
		"0",
	})
	s.Require().NoError(err)

	s.T().Logf("review positions")
	resp := new(types.QueryPositionsResponse)
	s.NoError(
		testutilcli.ExecQuery(
			s.network.Validators[0].ClientCtx,
			cli.CmdQueryPositions(),
			[]string{s.users[2].String()},
			resp,
		),
	)

	_, err = s.network.ExecTxCmd(cli.MarketOrderCmd(), s.users[5], []string{
		"sell",
		asset.Registry.Pair(denoms.OSMO, denoms.NUSD).String(),
		"15",       // Leverage
		"90000000", // Quote asset amount
		"0",
	})
	s.Require().NoError(err)

	s.T().Log("wait 10 blocks")
	height, err := s.network.LatestHeight()
	s.Require().NoError(err)
	_, err = s.network.WaitForHeight(height + 10)
	s.Require().NoError(err)

	s.T().Log("liquidating all users...")
	_, err = s.network.ExecTxCmd(cli.MultiLiquidateCmd(), s.liquidator, []string{
		fmt.Sprintf("%s:%s:%s", denoms.ATOM, denoms.NUSD, s.users[2].String()),
		fmt.Sprintf("%s:%s:%s", denoms.OSMO, denoms.NUSD, s.users[3].String()),
	})
	s.Require().NoError(err)
	err = s.network.WaitForNextBlock()
	s.Require().NoError(err)

	s.T().Log("check trader position")
	_, err = testutilcli.QueryPositionV2(s.network.Validators[0].ClientCtx, asset.Registry.Pair(denoms.ATOM, denoms.NUSD), s.users[2])
	s.Require().Error(err)

	_, err = testutilcli.QueryPositionV2(s.network.Validators[0].ClientCtx, asset.Registry.Pair(denoms.OSMO, denoms.NUSD), s.users[3])
	s.Require().Error(err)

	s.T().Log("closing positions - fail")
	_, err = s.network.ExecTxCmd(cli.ClosePositionCmd(), s.users[4], []string{
		"asset.Registry.Pair(denoms.ATOM, denoms.NUSD).String()",
	})
	s.Require().Error(err) // invalid pair

	_, err = s.network.ExecTxCmd(cli.ClosePositionCmd(), s.users[4], []string{
		"uluna:usdt",
	})
	s.Require().Error(err) // non whitelisted pair

	s.T().Log("closing positions")

	_, err = s.network.ExecTxCmd(cli.ClosePositionCmd(), s.users[4], []string{
		asset.Registry.Pair(denoms.ATOM, denoms.NUSD).String(),
	})
	s.Require().NoError(err)

	_, err = s.network.ExecTxCmd(cli.ClosePositionCmd(), s.users[5], []string{
		asset.Registry.Pair(denoms.OSMO, denoms.NUSD).String(),
	})
	s.Require().NoError(err)
}

// user[0] opens a long position
func (s *IntegrationTestSuite) TestMarketOrdersAndCloseCmd() {
	val := s.network.Validators[0]
	user := s.users[0]

	pair := asset.Registry.Pair(denoms.BTC, denoms.NUSD)
	exchangeRate, err := testutilcli.QueryOracleExchangeRate(
		val.ClientCtx, pair,
	)
	s.T().Logf("0. current exchange rate is: %+v", exchangeRate)
	s.NoError(err)

	s.T().Log("A. check market balances")
	ammMarketDuo, err := testutilcli.QueryMarketV2(val.ClientCtx, pair)
	s.Require().NoError(err)
	s.EqualValues(sdk.NewDec(10e6), ammMarketDuo.Amm.BaseReserve)
	s.EqualValues(sdk.NewDec(10e6), ammMarketDuo.Amm.QuoteReserve)

	s.T().Log("A. check trader has no existing positions")
	_, err = testutilcli.QueryPositionV2(
		val.ClientCtx, asset.Registry.Pair(denoms.BTC, denoms.NUSD), user,
	)
	s.Error(err)

	s.T().Log("B. open position")
	txResp, err := s.network.ExecTxCmd(cli.MarketOrderCmd(), user, []string{
		"buy",
		asset.Registry.Pair(denoms.BTC, denoms.NUSD).String(),
		/* leverage */ "1",
		/* quoteAmt */ "2000000", // 2*10^6 uNUSD
		/* baseAssetLimit */ "1",
	},
	)
	s.NoError(err)
	s.EqualValues(abcitypes.CodeTypeOK, txResp.Code)
	s.NoError(s.network.WaitForNextBlock())

	s.T().Log("B. check market balance after open position")
	ammMarketDuo, err = testutilcli.QueryMarketV2(val.ClientCtx, pair)
	s.Require().NoError(err)
	s.T().Logf("ammMarketDuo: %s", ammMarketDuo.String())
	s.EqualValues(sdk.MustNewDecFromStr("9999666.677777407419752675"), ammMarketDuo.Amm.BaseReserve)
	s.EqualValues(sdk.MustNewDecFromStr("10000333.333333333333333333"), ammMarketDuo.Amm.QuoteReserve)

	s.T().Log("B. check trader position")
	queryResp, err := testutilcli.QueryPositionV2(val.ClientCtx, asset.Registry.Pair(denoms.BTC, denoms.NUSD), user)
	s.NoError(err)
	s.T().Logf("query response: %+v", queryResp)
	s.EqualValues(user.String(), queryResp.Position.TraderAddress)
	s.EqualValues(asset.Registry.Pair(denoms.BTC, denoms.NUSD), queryResp.Position.Pair)
	s.EqualValues(sdk.MustNewDecFromStr("333.322222592580247325"), queryResp.Position.Size_)
	s.EqualValues(sdk.NewDec(2*common.TO_MICRO), queryResp.Position.Margin)
	s.EqualValues(sdk.NewDec(2*common.TO_MICRO), queryResp.Position.OpenNotional)
	s.EqualValues(sdk.MustNewDecFromStr("1999999.999999999999998000"), queryResp.PositionNotional)
	s.EqualValues(sdk.MustNewDecFromStr("-0.000000000000002000"), queryResp.UnrealizedPnl)
	s.EqualValues(sdk.OneDec(), queryResp.MarginRatio)

	s.T().Log("C. open position with 2x leverage and zero baseAmtLimit")
	txResp, err = s.network.ExecTxCmd(cli.MarketOrderCmd(), user, []string{
		"buy",
		asset.Registry.Pair(denoms.BTC, denoms.NUSD).String(),
		/* leverage */ "2",
		/* quoteAmt */ "2000000", // 2*10^6 uNUSD
		/* baseAmtLimit */ "0",
	})
	s.NoError(err)
	s.EqualValues(abcitypes.CodeTypeOK, txResp.Code)
	s.NoError(s.network.WaitForNextBlock())

	s.T().Log("C. check trader position")
	queryResp, err = testutilcli.QueryPositionV2(val.ClientCtx, asset.Registry.Pair(denoms.BTC, denoms.NUSD), user)
	s.NoError(err)
	s.T().Logf("query response: %+v", queryResp)
	s.EqualValues(user.String(), queryResp.Position.TraderAddress)
	s.EqualValues(asset.Registry.Pair(denoms.BTC, denoms.NUSD), queryResp.Position.Pair)
	s.EqualValues(sdk.MustNewDecFromStr("999.900009999000099990"), queryResp.Position.Size_)
	s.EqualValues(sdk.NewDec(4*common.TO_MICRO), queryResp.Position.Margin)
	s.EqualValues(sdk.NewDec(6*common.TO_MICRO), queryResp.Position.OpenNotional)
	s.EqualValues(sdk.MustNewDecFromStr("6000000.000000000000000000"), queryResp.PositionNotional)
	s.EqualValues(sdk.MustNewDecFromStr("0.000000000000000000"), queryResp.UnrealizedPnl)
	s.EqualValues(sdk.MustNewDecFromStr("0.666666666666666667"), queryResp.MarginRatio)

	s.T().Log("D. Open a reverse position smaller than the existing position")
	txResp, err = s.network.ExecTxCmd(cli.MarketOrderCmd(), user, []string{
		"sell",
		asset.Registry.Pair(denoms.BTC, denoms.NUSD).String(),
		/* leverage */ "1",
		/* quoteAmt */ "1000000", // 100 uNUSD
		/* baseAssetLimit */ "0",
	})
	s.NoError(err)
	s.EqualValues(abcitypes.CodeTypeOK, txResp.Code)
	s.NoError(s.network.WaitForNextBlock())

	s.T().Log("D. Check market after opening reverse position")

	ammMarketDuo, err = testutilcli.QueryMarketV2(val.ClientCtx, pair)
	s.Require().NoError(err)
	s.T().Logf("ammMarketDuo: %s", ammMarketDuo.String())
	s.EqualValues(sdk.MustNewDecFromStr("9999166.736105324556286976"), ammMarketDuo.Amm.BaseReserve)
	s.EqualValues(sdk.MustNewDecFromStr("10000833.333333333333333333"), ammMarketDuo.Amm.QuoteReserve)

	s.T().Log("D. Check trader position")
	queryResp, err = testutilcli.QueryPositionV2(val.ClientCtx, asset.Registry.Pair(denoms.BTC, denoms.NUSD), user)
	s.NoError(err)
	s.T().Logf("query response: %+v", queryResp)
	s.EqualValues(user.String(), queryResp.Position.TraderAddress)
	s.EqualValues(asset.Registry.Pair(denoms.BTC, denoms.NUSD), queryResp.Position.Pair)
	s.EqualValues(sdk.MustNewDecFromStr("833.263894675443713024"), queryResp.Position.Size_)
	s.EqualValues(sdk.NewDec(4*common.TO_MICRO), queryResp.Position.Margin)
	s.EqualValues(sdk.NewDec(5_000_000), queryResp.Position.OpenNotional)
	s.EqualValues(sdk.MustNewDecFromStr("4999999.999999999999998000"), queryResp.PositionNotional)
	s.EqualValues(sdk.MustNewDecFromStr("-0.000000000000002000"), queryResp.UnrealizedPnl)
	s.EqualValues(sdk.MustNewDecFromStr("0.800000000000000000"), queryResp.MarginRatio)

	s.T().Log("E. Open a reverse position larger than the existing position")
	txResp, err = s.network.ExecTxCmd(cli.MarketOrderCmd(), user, []string{
		"sell",
		asset.Registry.Pair(denoms.BTC, denoms.NUSD).String(),
		/* leverage */ "1",
		/* quoteAmt */ "8000000", // 8*10^6 uNUSD
		/* baseAssetLimit */ "0",
	})
	s.NoError(err)
	s.EqualValues(abcitypes.CodeTypeOK, txResp.Code)
	s.NoError(s.network.WaitForNextBlock())

	s.T().Log("E. Check trader position")
	queryResp, err = testutilcli.QueryPositionV2(val.ClientCtx, asset.Registry.Pair(denoms.BTC, denoms.NUSD), user)
	s.NoError(err)
	s.T().Logf("query response: %+v", queryResp)
	s.EqualValues(user.String(), queryResp.Position.TraderAddress)
	s.EqualValues(asset.Registry.Pair(denoms.BTC, denoms.NUSD), queryResp.Position.Pair)
	s.EqualValues(sdk.MustNewDecFromStr("-500.025001250062503125"), queryResp.Position.Size_)
	s.EqualValues(sdk.MustNewDecFromStr("3000000.000000000000002000"), queryResp.Position.OpenNotional)
	s.EqualValues(sdk.MustNewDecFromStr("3000000.000000000000002000"), queryResp.Position.Margin)
	s.EqualValues(sdk.MustNewDecFromStr("3000000.000000000000000000"), queryResp.PositionNotional)
	s.EqualValues(sdk.MustNewDecFromStr("0.000000000000002000"), queryResp.UnrealizedPnl)
	// there is a random delta due to twap margin ratio calculation and random block times in the in-process network
	s.InDelta(1, queryResp.MarginRatio.MustFloat64(), 0.008)

	s.T().Log("F. Close position")
	txResp, err = s.network.ExecTxCmd(cli.ClosePositionCmd(), user, []string{
		asset.Registry.Pair(denoms.BTC, denoms.NUSD).String(),
	})
	s.NoError(err)
	s.EqualValues(abcitypes.CodeTypeOK, txResp.Code)
	s.NoError(s.network.WaitForNextBlock())

	s.T().Log("F. check trader position")
	queryResp, err = testutilcli.QueryPositionV2(val.ClientCtx, asset.Registry.Pair(denoms.BTC, denoms.NUSD), user)
	s.Error(err)
	errors.IsOf(err, types.ErrPositionNotFound)
	s.T().Logf("query response: %+v", queryResp)
}

func (s *IntegrationTestSuite) TestPartialCloseCmd() {
	val := s.network.Validators[0]
	user := s.users[6]
	pair := asset.Registry.Pair(denoms.BTC, denoms.NUSD)
	var err error

	s.T().Log("Open position")
	txResp, err := s.network.ExecTxCmd(cli.MarketOrderCmd(), user, []string{
		"buy",
		asset.Registry.Pair(denoms.BTC, denoms.NUSD).String(),
		/* leverage */ "1",
		/* quoteAmt */ "12000000", // 12e6 uNUSD
		/* baseAssetLimit */ "0",
	},
	)
	s.NoError(err)
	s.EqualValues(abcitypes.CodeTypeOK, txResp.Code)
	s.NoError(s.network.WaitForNextBlock())

	s.T().Log("Check market balance after open position")
	ammMarketDuo, err := testutilcli.QueryMarketV2(val.ClientCtx, pair)
	s.Require().NoError(err)
	s.T().Logf("ammMarketDuo: %s", ammMarketDuo.String())
	s.EqualValues(sdk.MustNewDecFromStr("9998000.399920015996800640"), ammMarketDuo.Amm.BaseReserve)
	s.EqualValues(sdk.MustNewDecFromStr("10002000.000000000000000000"), ammMarketDuo.Amm.QuoteReserve)

	s.T().Log("Check trader position")
	queryResp, err := testutilcli.QueryPositionV2(val.ClientCtx, asset.Registry.Pair(denoms.BTC, denoms.NUSD), user)
	s.NoError(err)
	s.T().Logf("query response: %+v", queryResp)
	s.EqualValues(user.String(), queryResp.Position.TraderAddress)
	s.EqualValues(pair, queryResp.Position.Pair)
	s.EqualValues(sdk.MustNewDecFromStr("1999.600079984003199360"), queryResp.Position.Size_)
	s.EqualValues(sdk.NewDec(12e6), queryResp.Position.Margin)
	s.EqualValues(sdk.NewDec(12e6), queryResp.Position.OpenNotional)
	s.EqualValues(sdk.MustNewDecFromStr("12000000"), queryResp.PositionNotional)
	s.EqualValues(sdk.ZeroDec(), queryResp.UnrealizedPnl)
	s.EqualValues(sdk.OneDec(), queryResp.MarginRatio)

	s.T().Log("Partially close the position - fails")
	_, err = s.network.ExecTxCmd(cli.PartialCloseCmd(), user, []string{
		pair.String(),
		"",
	})
	s.Error(err) // invalid size amount
	_, err = s.network.ExecTxCmd(cli.PartialCloseCmd(), user, []string{
		"pair.String()",
		"500",
	})
	s.Error(err) // invalid pair
	_, err = s.network.ExecTxCmd(cli.PartialCloseCmd(), user, []string{
		"uluna:usdt",
		"500",
	})
	s.Error(err) // not whitelisted pair

	s.T().Log("Partially close the position")
	txResp, err = s.network.ExecTxCmd(cli.PartialCloseCmd(), user, []string{
		pair.String(),
		"500", // 500 uBTC
	})
	s.NoError(err)
	s.EqualValues(abcitypes.CodeTypeOK, txResp.Code)
	s.NoError(s.network.WaitForNextBlock())

	s.T().Log("Check market after partial close")
	ammMarketDuo, err = testutilcli.QueryMarketV2(val.ClientCtx, pair)
	s.Require().NoError(err)
	s.T().Logf("ammMarketDuo: %s", ammMarketDuo.String())
	s.EqualValues(sdk.MustNewDecFromStr("9998500.399920015996800640"), ammMarketDuo.Amm.BaseReserve)
	s.EqualValues(sdk.MustNewDecFromStr("10001499.824993752062459356"), ammMarketDuo.Amm.QuoteReserve)

	s.T().Log("Check trader position")
	queryResp, err = testutilcli.QueryPositionV2(val.ClientCtx, asset.Registry.Pair(denoms.BTC, denoms.NUSD), user)
	s.NoError(err)
	s.T().Logf("query response: %+v", queryResp)
	s.EqualValues(user.String(), queryResp.Position.TraderAddress)
	s.EqualValues(asset.Registry.Pair(denoms.BTC, denoms.NUSD), queryResp.Position.Pair)
	s.EqualValues(sdk.MustNewDecFromStr("1499.600079984003199360"), queryResp.Position.Size_)
	s.EqualValues(sdk.NewDec(12e6), queryResp.Position.Margin)
	s.EqualValues(sdk.MustNewDecFromStr("8998949.962512374756136000"), queryResp.Position.OpenNotional)
	s.EqualValues(sdk.MustNewDecFromStr("8998949.962512374756136000"), queryResp.PositionNotional)
	s.EqualValues(sdk.ZeroDec(), queryResp.UnrealizedPnl)
	s.EqualValues(sdk.MustNewDecFromStr("1.333488912594172945"), queryResp.MarginRatio)
}

func (s *IntegrationTestSuite) TestPositionEmptyAndClose() {
	val := s.network.Validators[0]
	user := s.users[0]

	// verify trader has no position (empty)
	_, err := testutilcli.QueryPositionV2(val.ClientCtx, asset.Registry.Pair(denoms.ETH, denoms.NUSD), user)
	s.Error(err, "no position found")

	// close position should produce error
	_, err = s.network.ExecTxCmd(cli.ClosePositionCmd(), user, []string{
		asset.Registry.Pair(denoms.ETH, denoms.NUSD).String(),
	})
	s.Contains(err.Error(), types.ErrPositionNotFound.Error())
}

// user[0] opens a position and removes margin to trigger bad debt
func (s *IntegrationTestSuite) TestRemoveMargin() {
	// Open a position with first user
	s.T().Log("opening a position with user 0")
	_, err := s.network.ExecTxCmd(cli.MarketOrderCmd(), s.users[0], []string{
		"buy",
		asset.Registry.Pair(denoms.BTC, denoms.NUSD).String(),
		"10",      // Leverage
		"1000000", // Quote asset amount
		"0",
	})
	s.NoError(err)
	s.NoError(s.network.WaitForNextBlock())

	// Remove margin to trigger bad debt on user 0
	s.T().Log("removing margin on user 0....")
	_, err = s.network.ExecTxCmd(cli.RemoveMarginCmd(), s.users[0], []string{
		asset.Registry.Pair(denoms.BTC, denoms.NUSD).String(),
		fmt.Sprintf("%s%s", "10000000", types.TestingCollateralDenomNUSD),
	})
	s.Contains(err.Error(), types.ErrBadDebt.Error())

	s.T().Log("removing margin on user 0....")
	_, err = s.network.ExecTxCmd(cli.RemoveMarginCmd(), s.users[0], []string{
		asset.Registry.Pair(denoms.BTC, denoms.NUSD).String(),
		fmt.Sprintf("%s%s", "1", types.TestingCollateralDenomNUSD),
	})
	s.NoError(err)
	s.NoError(s.network.WaitForNextBlock())
}

// user[1] opens a position and adds margin
func (s *IntegrationTestSuite) TestX_AddMargin() {
	// Open a new position
	s.T().Log("opening a position with user 1....")
	txResp, err := s.network.ExecTxCmd(cli.MarketOrderCmd(), s.users[1], []string{
		"buy",
		asset.Registry.Pair(denoms.ETH, denoms.NUSD).String(),
		"10",      // Leverage
		"1000000", // Quote asset amount
		"0.0000001",
	})
	s.Require().NoError(err, txResp)
	s.NoError(s.network.WaitForNextBlock())

	testCases := []struct {
		name           string
		args           []string
		expectedCode   uint32
		expectedMargin sdk.Dec
		expectFail     bool
	}{
		{
			name: "fail: not correct margin denom",
			args: []string{
				asset.Registry.Pair(denoms.BTC, denoms.NUSD).String(),
				fmt.Sprintf("10000%s", denoms.USDT),
			},
			expectFail:     false,
			expectedMargin: sdk.NewDec(1_000_000),
			expectedCode:   1,
		},
		{
			name: "fail: position not found",
			args: []string{
				asset.Registry.Pair(denoms.BTC, denoms.NUSD).String(),
				fmt.Sprintf("10000%s", types.TestingCollateralDenomNUSD),
			},
			expectedCode:   types.ErrPositionNotFound.ABCICode(),
			expectedMargin: sdk.NewDec(1_000_000),
			expectFail:     false,
		},
		{
			name: "PASS: add margin to correct position",
			args: []string{
				asset.Registry.Pair(denoms.ETH, denoms.NUSD).String(),
				fmt.Sprintf("10000%s", types.TestingCollateralDenomNUSD),
			},
			expectedCode:   0,
			expectedMargin: sdk.NewDec(1_010_000),
			expectFail:     false,
		},
		{
			name: "fail: invalid coin",
			args: []string{
				asset.Registry.Pair(denoms.BTC, denoms.NUSD).String(),
				"100",
			},
			expectFail: true,
		},
		{
			name: "fail: invalid pair",
			args: []string{
				"alisdhjal;dhao;sdh",
				fmt.Sprintf("10000%s", types.TestingCollateralDenomNUSD),
			},
			expectFail: true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.T().Run(tc.name, func(t *testing.T) {
			s.T().Log("adding margin on user 3....")
			canFail := true
			if tc.expectFail {
				txResp, err = s.network.ExecTxCmd(
					cli.AddMarginCmd(), s.users[1], tc.args,
					testutilcli.WithTxOptions(
						testutilcli.TxOptionChanges{CanFail: &canFail}),
				)
				s.Require().Error(err, txResp)
			} else {
				txResp, err := s.network.ExecTxCmd(
					cli.AddMarginCmd(), s.users[1], tc.args,
					testutilcli.WithTxOptions(
						testutilcli.TxOptionChanges{CanFail: &canFail}),
				)
				s.Require().NoError(err)
				s.Require().NoError(s.network.WaitForNextBlock())

				resp, err := testutilcli.QueryTx(s.network.Validators[0].ClientCtx, txResp.TxHash)
				s.Require().NoError(err)
				s.Require().EqualValues(tc.expectedCode, resp.Code)

				// query trader position
				queryResp, err := testutilcli.QueryPositionV2(s.network.Validators[0].ClientCtx, asset.Registry.Pair(denoms.ETH, denoms.NUSD), s.users[1])
				s.NoError(err)
				s.EqualValues(tc.expectedMargin, queryResp.Position.Margin)
			}
		})
	}
}

// user[1] opens a position and removes margin
func (s *IntegrationTestSuite) TestX_RemoveMargin() {
	// Open a new position
	s.T().Log("opening a position with user 1....")
	_, err := s.network.ExecTxCmd(cli.MarketOrderCmd(), s.users[2], []string{
		"buy",
		asset.Registry.Pair(denoms.ETH, denoms.NUSD).String(),
		"10",      // Leverage
		"1000000", // Quote asset amount
		"0.0000001",
	})
	s.Require().NoError(err)
	s.NoError(s.network.WaitForNextBlock())

	testCases := []struct {
		name           string
		args           []string
		expectedCode   uint32
		expectedMargin sdk.Dec
		expectFail     bool
	}{
		{
			name: "fail: not correct margin denom",
			args: []string{
				asset.Registry.Pair(denoms.BTC, denoms.NUSD).String(),
				fmt.Sprintf("10000%s", denoms.USDT),
			},
			expectFail:     false,
			expectedCode:   1,
			expectedMargin: sdk.NewDec(1_000_000),
		},
		{
			name: "fail: position not found",
			args: []string{
				asset.Registry.Pair(denoms.BTC, denoms.NUSD).String(),
				fmt.Sprintf("10000%s", types.TestingCollateralDenomNUSD),
			},
			expectedCode:   types.ErrPositionNotFound.ABCICode(),
			expectedMargin: sdk.NewDec(1_000_000),
			expectFail:     false,
		},
		{
			name: "PASS: remove margin to correct position",
			args: []string{
				asset.Registry.Pair(denoms.ETH, denoms.NUSD).String(),
				fmt.Sprintf("10000%s", types.TestingCollateralDenomNUSD),
			},
			expectedCode:   0,
			expectedMargin: sdk.NewDec(990_000),
			expectFail:     false,
		},
		{
			name: "fail: invalid coin",
			args: []string{
				asset.Registry.Pair(denoms.BTC, denoms.NUSD).String(),
				"100",
			},
			expectFail: true,
		},
		{
			name: "fail: invalid pair",
			args: []string{
				"alisdhjal;dhao;sdh",
				fmt.Sprintf("10000%s", types.TestingCollateralDenomNUSD),
			},
			expectFail: true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.T().Run(tc.name, func(t *testing.T) {
			s.T().Log("removing margin on user 3....")

			canFail := true
			txResp, err := s.network.ExecTxCmd(
				cli.RemoveMarginCmd(), s.users[2], tc.args,
				testutilcli.WithTxOptions(
					testutilcli.TxOptionChanges{CanFail: &canFail}),
			)
			if tc.expectFail {
				s.Require().Errorf(err, "txResp: %v", txResp)
			} else {
				s.Require().NoErrorf(err, "txResp: %v", txResp)
				s.Require().NoError(s.network.WaitForNextBlock())

				resp, err := testutilcli.QueryTx(s.network.Validators[0].ClientCtx, txResp.TxHash)
				s.Require().NoError(err)
				s.Require().EqualValues(tc.expectedCode, resp.Code)

				queryResp, err := testutilcli.QueryPositionV2(s.network.Validators[0].ClientCtx, asset.Registry.Pair(denoms.ETH, denoms.NUSD), s.users[2])
				s.NoError(err)
				s.EqualValues(tc.expectedMargin, queryResp.Position.Margin)
			}
		})
	}
}

func (s *IntegrationTestSuite) TestDonateToEcosystemFund() {
	s.T().Logf("donate to ecosystem fund")
	out, err := s.network.ExecTxCmd(
		cli.DonateToEcosystemFundCmd(),
		sdk.MustAccAddressFromBech32("nibi1w89pf5yq8ntjg89048qmtaz929fdxup0a57d8m"),
		[]string{"100" + types.TestingCollateralDenomNUSD},
	)
	s.NoError(err)
	s.Require().EqualValues(abcitypes.CodeTypeOK, out.Code)

	s.NoError(s.network.WaitForNextBlock())

	_, err = s.network.ExecTxCmd(
		cli.DonateToEcosystemFundCmd(),
		sdk.MustAccAddressFromBech32("nibi1w89pf5yq8ntjg89048qmtaz929fdxup0a57d8m"),
		[]string{"10"})
	s.Error(err)

	s.NoError(s.network.WaitForNextBlock())
	resp := new(sdk.Coin)
	moduleAccountAddrPerpEF := "nibi1trh2mamq64u4g042zfeevvjk4cukrthvppfnc7"
	s.NoError(
		testutilcli.ExecQuery(
			s.network.Validators[0].ClientCtx,
			bankcli.GetBalancesCmd(),
			[]string{moduleAccountAddrPerpEF, "--denom", types.TestingCollateralDenomNUSD},
			resp,
		),
	)
	s.Require().EqualValues(sdk.NewInt64Coin(types.TestingCollateralDenomNUSD, 100), *resp)
}

func (s *IntegrationTestSuite) TestQueryModuleAccount() {
	resp := new(types.QueryModuleAccountsResponse)
	s.NoError(
		s.network.ExecQuery(
			cli.NewQueryCmd(),
			[]string{"module-accounts"},
			resp,
		),
	)
	s.NotEmpty(resp.Accounts)
}

func (s *IntegrationTestSuite) TestQueryCollateralDenom() {
	resp := new(types.QueryCollateralResponse)
	s.NoError(
		s.network.ExecQuery(
			cli.NewQueryCmd(),
			[]string{"collateral"},
			resp,
		),
	)
	s.Equal(types.TestingCollateralDenomNUSD, resp.CollateralDenom,
		"resp: %s", resp.String())
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
