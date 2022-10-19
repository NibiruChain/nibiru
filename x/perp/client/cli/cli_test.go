package cli_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/NibiruChain/nibiru/collections"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdktestutilcli "github.com/cosmos/cosmos-sdk/testutil/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankcli "github.com/cosmos/cosmos-sdk/x/bank/client/cli"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/simapp"
	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/client/cli"
	perptypes "github.com/NibiruChain/nibiru/x/perp/types"
	pftypes "github.com/NibiruChain/nibiru/x/pricefeed/types"
	testutilcli "github.com/NibiruChain/nibiru/x/testutil/cli"
	vpooltypes "github.com/NibiruChain/nibiru/x/vpool/types"
)

var commonArgs = []string{
	fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
	fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastBlock),
	fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(common.DenomNIBI, sdk.NewInt(10))).String()),
}

type IntegrationTestSuite struct {
	suite.Suite

	cfg     testutilcli.Config
	network *testutilcli.Network
	users   []sdk.AccAddress
}

// NewPricefeedGen returns an x/pricefeed GenesisState to specify the module parameters.
func NewPricefeedGen() *pftypes.GenesisState {
	pairs := common.AssetPairs{common.Pair_BTC_NUSD, common.Pair_ETH_NUSD}
	defaultGenesis := simapp.PricefeedGenesis()
	defaultGenesis.Params.Pairs = append(defaultGenesis.Params.Pairs, pairs...)
	defaultGenesis.PostedPrices = append(defaultGenesis.PostedPrices, []pftypes.PostedPrice{
		{
			PairID: common.Pair_BTC_NUSD.String(),
			Oracle: simapp.GenOracleAddress,
			Price:  sdk.OneDec(),
			Expiry: time.Now().Add(1 * time.Hour),
		},
		{
			PairID: common.Pair_ETH_NUSD.String(),
			Oracle: simapp.GenOracleAddress,
			Price:  sdk.OneDec(),
			Expiry: time.Now().Add(1 * time.Hour),
		},
	}...)

	return &defaultGenesis
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
	vpoolGenesis.Vpools = []vpooltypes.VPool{
		{
			Pair:                   common.Pair_BTC_NUSD,
			BaseAssetReserve:       sdk.NewDec(10_000_000),
			QuoteAssetReserve:      sdk.NewDec(60_000_000_000),
			TradeLimitRatio:        sdk.MustNewDecFromStr("0.8"),
			FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.2"),
			MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.2"),
			MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
			MaxLeverage:            sdk.MustNewDecFromStr("15"),
		},
		{
			Pair:                   common.Pair_ETH_NUSD,
			BaseAssetReserve:       sdk.NewDec(10_000_000),
			QuoteAssetReserve:      sdk.NewDec(60_000_000_000),
			TradeLimitRatio:        sdk.MustNewDecFromStr("0.8"),
			FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.2"),
			MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.2"),
			MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
			MaxLeverage:            sdk.MustNewDecFromStr("15"),
		},
	}
	genesisState[vpooltypes.ModuleName] = encodingConfig.Marshaler.MustMarshalJSON(vpoolGenesis)

	// setup perp
	perpGenesis := perptypes.DefaultGenesis()
	perpGenesis.PairMetadata = []perptypes.PairMetadata{
		{
			Pair: common.Pair_BTC_NUSD,
			CumulativePremiumFractions: []sdk.Dec{
				sdk.ZeroDec(),
				sdk.OneDec(),
				sdk.NewDec(2),
			},
		},
		{
			Pair: common.Pair_ETH_NUSD,
			CumulativePremiumFractions: []sdk.Dec{
				sdk.ZeroDec(),
			},
		},
	}
	perpGenesis.Params.WhitelistedLiquidators = []string{"nibi1w89pf5yq8ntjg89048qmtaz929fdxup0a57d8m"} // address associated with mnemonic below
	genesisState[perptypes.ModuleName] = encodingConfig.Marshaler.MustMarshalJSON(perpGenesis)

	// set up pricefeed
	genesisState[pftypes.ModuleName] = encodingConfig.Marshaler.MustMarshalJSON(NewPricefeedGen())

	s.cfg = testutilcli.BuildNetworkConfig(genesisState)
	s.cfg.Mnemonics = []string{"satisfy december text daring wheat vanish save viable holiday rural vessel shuffle dice skate promote fade badge federal sail during lend fever balance give"}
	s.network = testutilcli.NewNetwork(s.T(), s.cfg)
	_, err := s.network.WaitForHeight(1)
	s.NoError(err)

	val := s.network.Validators[0]
	info, _, err := val.ClientCtx.Keyring.
		NewMnemonic("user1", keyring.English, sdk.FullFundraiserPath, "", hd.Secp256k1)
	s.NoError(err)
	user1 := sdk.AccAddress(info.GetPubKey().Address())

	info, _, err = val.ClientCtx.Keyring.
		NewMnemonic("user2", keyring.English, sdk.FullFundraiserPath, "", hd.Secp256k1)
	s.NoError(err)
	user2 := sdk.AccAddress(info.GetPubKey().Address())

	info, _, err = val.ClientCtx.Keyring.
		NewMnemonic("user3", keyring.English, sdk.FullFundraiserPath, "", hd.Secp256k1)
	s.NoError(err)
	user3 := sdk.AccAddress(info.GetPubKey().Address())

	info, _, err = val.ClientCtx.Keyring.
		NewMnemonic("user4", keyring.English, sdk.FullFundraiserPath, "", hd.Secp256k1)
	s.NoError(err)
	user4 := sdk.AccAddress(info.GetPubKey().Address())

	s.users = []sdk.AccAddress{user1, user2, user3, user4}

	_, err = testutilcli.FillWalletFromValidator(user1,
		sdk.NewCoins(
			sdk.NewInt64Coin(common.DenomNIBI, 10_000_000),
			sdk.NewInt64Coin(common.DenomUSDC, 10_000_000),
			sdk.NewInt64Coin(common.DenomNUSD, 50_000_000),
		),
		val,
		common.DenomNIBI,
	)
	s.Require().NoError(err)

	_, err = testutilcli.FillWalletFromValidator(user2,
		sdk.NewCoins(
			sdk.NewInt64Coin(common.DenomNIBI, 1000),
			sdk.NewInt64Coin(common.DenomUSDC, 1000),
			sdk.NewInt64Coin(common.DenomNUSD, 100000),
		),
		val,
		common.DenomNIBI,
	)
	s.Require().NoError(err)

	_, err = testutilcli.FillWalletFromValidator(user3,
		sdk.NewCoins(
			sdk.NewInt64Coin(common.DenomNIBI, 1000),
			sdk.NewInt64Coin(common.DenomUSDC, 1000),
			sdk.NewInt64Coin(common.DenomNUSD, 49_000_000),
		),
		val,
		common.DenomNIBI,
	)
	s.Require().NoError(err)

	_, err = testutilcli.FillWalletFromValidator(user4,
		sdk.NewCoins(
			sdk.NewInt64Coin(common.DenomNIBI, 1000),
			sdk.NewInt64Coin(common.DenomUSDC, 1000),
			sdk.NewInt64Coin(common.DenomNUSD, 100000),
		),
		val,
		common.DenomNIBI,
	)
	s.Require().NoError(err)

	_, err = testutilcli.FillWalletFromValidator(
		sdk.MustAccAddressFromBech32("nibi1w89pf5yq8ntjg89048qmtaz929fdxup0a57d8m"),
		sdk.NewCoins(sdk.NewInt64Coin(common.DenomNIBI, 1000)),
		val, common.DenomNIBI)
	s.Require().NoError(err)
}

func (s *IntegrationTestSuite) TearDownSuite() {
	s.T().Log("tearing down integration test suite")
	s.network.Cleanup()
}

func (s *IntegrationTestSuite) TestOpenPositionsAndCloseCmd() {
	val := s.network.Validators[0]

	user := s.users[0]

	s.T().Log("A. check vpool balances")
	reserveAssets, err := testutilcli.QueryVpoolReserveAssets(val.ClientCtx, common.Pair_BTC_NUSD)
	s.T().Logf("reserve assets: %+v", reserveAssets)
	s.NoError(err)
	s.EqualValues(sdk.NewDec(10_000_000), reserveAssets.BaseAssetReserve)
	s.EqualValues(sdk.NewDec(60_000_000_000), reserveAssets.QuoteAssetReserve)

	s.T().Log("A. check trader has no existing positions")
	_, err = testutilcli.QueryPosition(val.ClientCtx, common.Pair_BTC_NUSD, user)
	s.Error(err, "no position found")

	s.T().Log("B. open position")
	args := []string{
		"--from",
		user.String(),
		"buy",
		common.Pair_BTC_NUSD.String(),
		/* leverage */ "1",
		/* quoteAmt */ "1000000", // 10^6 uNUSD
		/* baseAssetLimit */ "1",
	}
	_, err = sdktestutilcli.ExecTestCLICmd(val.ClientCtx, cli.OpenPositionCmd(), append(args, commonArgs...))
	s.NoError(err)

	s.T().Log("B. check vpool balance after open position")
	reserveAssets, err = testutilcli.QueryVpoolReserveAssets(val.ClientCtx, common.Pair_BTC_NUSD)
	s.T().Logf("reserve assets: %+v", reserveAssets)
	s.NoError(err)
	s.EqualValues(sdk.MustNewDecFromStr("9999833.336111064815586407"), reserveAssets.BaseAssetReserve)
	s.EqualValues(sdk.NewDec(60_001_000_000), reserveAssets.QuoteAssetReserve)

	s.T().Log("B. check vpool balances")
	queryResp, err := testutilcli.QueryPosition(val.ClientCtx, common.Pair_BTC_NUSD, user)
	s.T().Logf("query response: %+v", queryResp)
	s.NoError(err)
	s.EqualValues(user.String(), queryResp.Position.TraderAddress)
	s.EqualValues(common.Pair_BTC_NUSD, queryResp.Position.Pair)
	s.EqualValues(sdk.MustNewDecFromStr("166.663888935184413593"), queryResp.Position.Size_)
	s.EqualValues(sdk.NewDec(1_000_000), queryResp.Position.Margin)
	s.EqualValues(sdk.NewDec(1_000_000), queryResp.Position.OpenNotional)
	s.EqualValues(sdk.MustNewDecFromStr("999999.999999999999999359"), queryResp.PositionNotional)
	s.EqualValues(sdk.MustNewDecFromStr("-0.000000000000000641"), queryResp.UnrealizedPnl)
	s.EqualValues(sdk.NewDec(1), queryResp.MarginRatioMark)
	s.EqualValues(sdk.NewDec(1), queryResp.MarginRatioIndex)

	s.T().Log("C. open position with 2x leverage and zero baseAmtLimit")
	args = []string{
		"--from",
		user.String(),
		"buy",
		common.Pair_BTC_NUSD.String(),
		/* leverage */ "2",
		/* quoteAmt */ "1000000", // 10^6 uNUSD
		/* baseAmtLimit */ "0",
	}
	_, err = sdktestutilcli.ExecTestCLICmd(val.ClientCtx, cli.OpenPositionCmd(), append(args, commonArgs...))
	s.NoError(err)

	s.T().Log("C. check trader position")
	queryResp, err = testutilcli.QueryPosition(val.ClientCtx, common.Pair_BTC_NUSD, user)
	s.T().Logf("query response: %+v", queryResp)
	s.NoError(err)
	s.EqualValues(user.String(), queryResp.Position.TraderAddress)
	s.EqualValues(common.Pair_BTC_NUSD, queryResp.Position.Pair)
	s.EqualValues(sdk.MustNewDecFromStr("499.975001249937503125"), queryResp.Position.Size_)
	s.EqualValues(sdk.NewDec(2_000_000), queryResp.Position.Margin)
	s.EqualValues(sdk.NewDec(3_000_000), queryResp.Position.OpenNotional)
	s.EqualValues(sdk.MustNewDecFromStr("3000000.000000000000000938"), queryResp.PositionNotional)
	s.EqualValues(sdk.MustNewDecFromStr("0.000000000000000938"), queryResp.UnrealizedPnl)
	s.EqualValues(sdk.MustNewDecFromStr("0.666666666666666667"), queryResp.MarginRatioMark)

	s.T().Log("D. Open a reverse position smaller than the existing position")
	args = []string{
		"--from",
		user.String(),
		"sell",
		common.Pair_BTC_NUSD.String(),
		/* leverage */ "1",
		/* quoteAmt */ "100", // 100 uNUSD
		/* baseAssetLimit */ "1",
	}
	res, err := sdktestutilcli.ExecTestCLICmd(val.ClientCtx, cli.OpenPositionCmd(), append(args, commonArgs...))
	s.NoError(err)
	s.NotContains(res.String(), "fail")

	s.T().Log("D. Check vpool after opening reverse position")
	reserveAssets, err = testutilcli.QueryVpoolReserveAssets(val.ClientCtx, common.Pair_BTC_NUSD)
	s.T().Logf(" \n reserve assets: %+v \n", reserveAssets)
	s.NoError(err)
	s.EqualValues(sdk.MustNewDecFromStr("9999500.041663750215262154"), reserveAssets.BaseAssetReserve)
	s.EqualValues(sdk.NewDec(60_002_999_900), reserveAssets.QuoteAssetReserve)

	s.T().Log("D. Check trader position")
	queryResp, err = testutilcli.QueryPosition(val.ClientCtx, common.Pair_BTC_NUSD, user)
	s.T().Logf("query response: %+v", queryResp)
	s.NoError(err)
	s.EqualValues(user.String(), queryResp.Position.TraderAddress)
	s.EqualValues(common.Pair_BTC_NUSD, queryResp.Position.Pair)
	s.EqualValues(sdk.MustNewDecFromStr("499.958336249784737846"), queryResp.Position.Size_)
	s.EqualValues(sdk.NewDec(2_000_000), queryResp.Position.Margin)
	s.EqualValues(sdk.NewDec(2_999_900), queryResp.Position.OpenNotional)
	s.EqualValues(sdk.MustNewDecFromStr("2999899.999999999999999506"), queryResp.PositionNotional)
	s.EqualValues(sdk.MustNewDecFromStr("-0.000000000000000494"), queryResp.UnrealizedPnl)
	s.EqualValues(sdk.MustNewDecFromStr("0.666688889629654322"), queryResp.MarginRatioMark)

	s.T().Log("E. Open a reverse position larger than the existing position")
	args = []string{
		"--from",
		user.String(),
		"sell",
		common.Pair_BTC_NUSD.String(),
		/* leverage */ "1",
		/* quoteAmt */ "4000000", // 4*10^6 uNUSD
		/* baseAssetLimit */ "0",
	}
	res, err = sdktestutilcli.ExecTestCLICmd(val.ClientCtx, cli.OpenPositionCmd(), append(args, commonArgs...))
	s.NoError(err)
	s.NotContains(res.String(), "fail")

	s.T().Log("E. Check trader position")
	queryResp, err = testutilcli.QueryPosition(val.ClientCtx, common.Pair_BTC_NUSD, user)
	s.T().Logf("query response: %+v", queryResp)
	s.NoError(err)
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
	args = []string{
		"--from",
		user.String(),
		common.Pair_BTC_NUSD.String(),
	}
	_, err = sdktestutilcli.ExecTestCLICmd(val.ClientCtx, cli.ClosePositionCmd(), append(args, commonArgs...))
	s.NoError(err)

	s.T().Log("F. check trader position")
	queryResp, err = testutilcli.QueryPosition(val.ClientCtx, common.Pair_BTC_NUSD, user)

	s.T().Logf("query response: %+v", queryResp)
	s.Error(err)

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
	args := []string{
		"--from",
		user.String(),
		common.Pair_ETH_NUSD.String(),
	}
	out, _ := sdktestutilcli.ExecTestCLICmd(val.ClientCtx, cli.ClosePositionCmd(), append(args, commonArgs...))
	s.Contains(out.String(), collections.ErrNotFound.Error())
}

func (s *IntegrationTestSuite) TestQueryCumulativePremiumFractions() {
	val := s.network.Validators[0]

	s.T().Log("get cumulative funding payments")
	queryResp, err := testutilcli.QueryFundingRates(val.ClientCtx, common.Pair_BTC_NUSD)
	s.NoError(err)
	s.EqualValues([]sdk.Dec{sdk.ZeroDec(), sdk.OneDec(), sdk.NewDec(2)}, queryResp.CumulativeFundingRates)
}

func (s *IntegrationTestSuite) TestRemoveMargin() {
	// Set up the user accounts
	val := s.network.Validators[0]

	// Open a position with first user
	s.T().Log("opening a position with user 1....")
	args := []string{
		"--from",
		s.users[0].String(),
		"buy",
		common.Pair_BTC_NUSD.String(),
		"10", // Leverage
		"1",  // Quote asset amount
		"0.0000001",
	}
	_, err := sdktestutilcli.ExecTestCLICmd(val.ClientCtx, cli.OpenPositionCmd(), append(args, commonArgs...))
	s.NoError(err)

	// Remove margin to trigger bad debt on user 1
	s.T().Log("removing margin on user 1....")
	args = []string{
		"--from",
		s.users[0].String(),
		common.Pair_BTC_NUSD.String(),
		fmt.Sprintf("%s%s", "100", common.DenomNUSD), // Amount
	}
	out, err := sdktestutilcli.ExecTestCLICmd(val.ClientCtx, cli.RemoveMarginCmd(), append(args, commonArgs...))
	s.NoError(err)
	s.Contains(out.String(), perptypes.ErrFailedRemoveMarginCanCauseBadDebt.Error())
}

func (s *IntegrationTestSuite) TestX_AddMargin() {
	val := s.network.Validators[0]
	pair := common.Pair_ETH_NUSD

	// Open a new position
	s.T().Log("opening a position with user 3....")
	args := []string{
		"--from",
		s.users[3].String(),
		"buy",
		pair.String(),
		"10",    // Leverage
		"10000", // Quote asset amount
		"0.0000001",
	}

	_, err := sdktestutilcli.ExecTestCLICmd(val.ClientCtx, cli.OpenPositionCmd(), append(args, commonArgs...))
	s.Require().NoError(err)

	testCases := []struct {
		name           string
		args           []string
		expectedCode   uint32
		expectedMargin sdk.Dec
	}{
		{
			name: "PASS: add margin to correct position",
			args: []string{
				"--from",
				s.users[3].String(),
				pair.String(),
				fmt.Sprintf("%s%s", "10000", pair.Token1),
			},
			expectedCode:   0,
			expectedMargin: sdk.NewDec(20_000),
		},
		{
			name: "FAIL: position not found",
			args: []string{
				"--from",
				s.users[3].String(),
				common.Pair_BTC_NUSD.String(),
				fmt.Sprintf("%s%s", "10000", pair.Token1),
			},
			expectedCode: 1,
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			s.T().Log("adding margin on user 3....")
			out, err := sdktestutilcli.ExecTestCLICmd(val.ClientCtx, cli.AddMarginCmd(), append(tc.args, commonArgs...))
			s.Require().NoError(err)

			var tx sdk.TxResponse
			val.ClientCtx.Codec.MustUnmarshalJSON(out.Bytes(), &tx)

			s.EqualValues(tc.expectedCode, tx.Code)

			if tc.expectedCode == 0 {
				// query trader position
				queryResp, err := testutilcli.QueryPosition(val.ClientCtx, pair, s.users[3])
				s.NoError(err)

				s.EqualValues(tc.expectedMargin, queryResp.Position.Margin)
				s.T().Logf(queryResp.Position.String())
			}
		})
	}
}

func (s *IntegrationTestSuite) TestLiquidate() {
	// Set up the user accounts
	val := s.network.Validators[0]

	args := []string{
		"--from",
		"nibi1w89pf5yq8ntjg89048qmtaz929fdxup0a57d8m",
		common.Pair_ETH_NUSD.String(),
		s.users[1].String(),
	}

	// liquidate a position that does not exist
	out, err := sdktestutilcli.ExecTestCLICmd(val.ClientCtx, cli.LiquidateCmd(), append(args, commonArgs...))
	s.Contains(out.String(), collections.ErrNotFound.Error())
	if err != nil {
		s.T().Logf("user liquidate error: %+v", err)
	}

	positionArgs := []string{
		"--from",
		s.users[1].String(),
		"buy",
		common.Pair_ETH_NUSD.String(),
		"15",    // Leverage
		"90000", // Quote asset amount
		"0",
	}

	s.T().Log("opening a position with user 2....")
	_, err = sdktestutilcli.ExecTestCLICmd(val.ClientCtx, cli.OpenPositionCmd(), append(positionArgs, commonArgs...))
	s.NoError(err)

	// error : margin is higher than required maintenance margin ratio"
	out, err = sdktestutilcli.ExecTestCLICmd(val.ClientCtx, cli.LiquidateCmd(), append(args, commonArgs...))
	s.Contains(out.String(), "margin is higher than required maintenance margin ratio")
	if err != nil {
		s.T().Logf("user liquidate error: %+v", err)
	}

	positionArgs = []string{
		"--from",
		s.users[2].String(),
		"sell",
		common.Pair_ETH_NUSD.String(),
		"15",       // Leverage
		"45000000", // Quote asset amount
		"0",
	}

	s.T().Log("opening a position with user 3....")
	_, err = sdktestutilcli.ExecTestCLICmd(val.ClientCtx, cli.OpenPositionCmd(), append(positionArgs, commonArgs...))
	s.NoError(err)

	height, err := s.network.LatestHeight()
	s.Require().NoError(err)

	_, err = s.network.WaitForHeight(height + 10)
	s.Require().NoError(err)

	// liquidate
	args = []string{
		"--from",
		"nibi1w89pf5yq8ntjg89048qmtaz929fdxup0a57d8m",
		common.Pair_ETH_NUSD.String(),
		s.users[1].String(),
	}

	s.T().Log("liquidating user 2....")
	out, err = sdktestutilcli.ExecTestCLICmd(val.ClientCtx, cli.LiquidateCmd(), append(args, commonArgs...))
	s.NotContains(out.String(), "fail", out.String())
	s.NoError(err)
}

func (s *IntegrationTestSuite) TestDonateToEcosystemFund() {
	// Set up the user accounts
	val := s.network.Validators[0]

	args := []string{
		"--from",
		"nibi1w89pf5yq8ntjg89048qmtaz929fdxup0a57d8m",
		"100unusd",
	}

	s.T().Logf("donate to ecosystem fund")
	_, err := sdktestutilcli.ExecTestCLICmd(val.ClientCtx, cli.DonateToEcosystemFundCmd(), append(args, commonArgs...))
	s.Require().NoError(err)
	s.Require().NoError(s.network.WaitForNextBlock())

	resp := new(sdk.Coin)
	s.Require().NoError(
		testutilcli.ExecQuery(
			s.network,
			bankcli.GetBalancesCmd(),
			[]string{"nibi1trh2mamq64u4g042zfeevvjk4cukrthvppfnc7", "--denom", "unusd"},
			resp,
		),
	)
	s.Require().EqualValues(sdk.NewInt64Coin("unusd", 100), *resp)
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
