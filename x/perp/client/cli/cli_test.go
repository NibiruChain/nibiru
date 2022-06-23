package cli_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/app"
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
	fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(common.GovDenom, sdk.NewInt(10))).String()),
}

type IntegrationTestSuite struct {
	suite.Suite

	cfg     testutilcli.Config
	network *testutilcli.Network
	users   []sdk.AccAddress
}

const (
	oracleAddress = "nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl"
)

// NewPricefeedGen returns an x/pricefeed GenesisState to specify the module parameters.
func NewPricefeedGen() *pftypes.GenesisState {
	oracle, err := sdk.AccAddressFromBech32(oracleAddress)
	if err != nil {
		panic(err)
	}

	return &pftypes.GenesisState{
		Params: pftypes.Params{
			Pairs: []pftypes.Pair{
				{Token0: common.TestStablePool.Token0,
					Token1:  common.TestStablePool.Token1,
					Oracles: []sdk.AccAddress{oracle}, Active: true},
			},
		},
		PostedPrices: []pftypes.PostedPrice{
			{
				PairID:        common.TestStablePool.PairID(),
				OracleAddress: oracle,
				Price:         sdk.OneDec(),
				Expiry:        time.Now().Add(1 * time.Hour),
			},
		},
	}
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

	s.cfg = testutilcli.DefaultConfig()

	app.SetPrefixes(app.AccountAddressPrefix)

	genesisState := app.ModuleBasics.DefaultGenesis(s.cfg.Codec)

	// setup vpool
	vpoolGenesis := vpooltypes.DefaultGenesis()
	vpoolGenesis.Vpools = []*vpooltypes.Pool{
		{
			Pair:                  "ubtc:unibi",
			BaseAssetReserve:      sdk.NewDec(10_000_000),
			QuoteAssetReserve:     sdk.NewDec(60_000_000_000),
			TradeLimitRatio:       sdk.MustNewDecFromStr("0.8"),
			FluctuationLimitRatio: sdk.MustNewDecFromStr("0.2"),
			MaxOracleSpreadRatio:  sdk.MustNewDecFromStr("0.2"),
		},
		{
			Pair:                  "eth:unibi",
			BaseAssetReserve:      sdk.NewDec(10_000_000),
			QuoteAssetReserve:     sdk.NewDec(60_000_000_000),
			TradeLimitRatio:       sdk.MustNewDecFromStr("0.8"),
			FluctuationLimitRatio: sdk.MustNewDecFromStr("0.2"),
			MaxOracleSpreadRatio:  sdk.MustNewDecFromStr("0.2"),
		},
		{
			Pair:              common.TestStablePool.String(),
			BaseAssetReserve:  sdk.MustNewDecFromStr("100"),
			QuoteAssetReserve: sdk.MustNewDecFromStr("600"),

			// below sets any trade is allowed
			TradeLimitRatio:       sdk.NewDec(10_000_000), // 10000000 * 100%
			FluctuationLimitRatio: sdk.NewDec(10_000_000), // 1_000_000 is 1
			MaxOracleSpreadRatio:  sdk.NewDec(10_000_000),
		},
	}
	genesisState[vpooltypes.ModuleName] = s.cfg.Codec.MustMarshalJSON(vpoolGenesis)

	// setup perp
	perpGenesis := perptypes.DefaultGenesis()
	perpGenesis.PairMetadata = []*perptypes.PairMetadata{
		{
			Pair: "ubtc:unibi",
			CumulativePremiumFractions: []sdk.Dec{
				sdk.ZeroDec(),
			},
		},
		{
			Pair: "eth:unibi",
			CumulativePremiumFractions: []sdk.Dec{
				sdk.ZeroDec(),
			},
		},
		{
			Pair: common.TestStablePool.String(),
			CumulativePremiumFractions: []sdk.Dec{
				sdk.ZeroDec(),
			},
		},
	}
	genesisState[perptypes.ModuleName] = s.cfg.Codec.MustMarshalJSON(perpGenesis)

	// set up pricefeed
	pricefeedGenJson := s.cfg.Codec.MustMarshalJSON(NewPricefeedGen())
	genesisState[pftypes.ModuleName] = pricefeedGenJson

	s.cfg.GenesisState = genesisState

	s.network = testutilcli.New(s.T(), s.cfg)

	_, err := s.network.WaitForHeight(1)
	s.Require().NoError(err)

	val := s.network.Validators[0]
	info, _, err := val.ClientCtx.Keyring.
		NewMnemonic("user1", keyring.English, sdk.FullFundraiserPath, "", hd.Secp256k1)
	s.Require().NoError(err)
	user1 := sdk.AccAddress(info.GetPubKey().Address())

	s.users = []sdk.AccAddress{user1}

	_, err = testutilcli.FillWalletFromValidator(user1,
		sdk.NewCoins(
			sdk.NewInt64Coin(s.cfg.BondDenom, 20_000),
			sdk.NewInt64Coin(common.GovDenom, 100_000_000),
			sdk.NewInt64Coin(common.CollDenom, 100_000_000),
			sdk.NewInt64Coin(common.TestTokenDenom, 50_000_000),
			sdk.NewInt64Coin(common.StableDenom, 50_000_000),
		),
		val,
		s.cfg.BondDenom,
	)
	s.Require().NoError(err)
}

func (s *IntegrationTestSuite) TearDownSuite() {
	s.T().Log("tearing down integration test suite")
	s.network.Cleanup()
}

func (s *IntegrationTestSuite) TestOpenPositionsAndCloseCmd() {
	val := s.network.Validators[0]
	assetPair := common.AssetPair{
		Token0: "ubtc",
		Token1: "unibi",
	}

	user := s.users[0]

	s.T().Log("A. check vpool balances")
	reserveAssets, err := testutilcli.QueryVpoolReserveAssets(val.ClientCtx, assetPair)
	s.T().Logf("reserve assets: %+v", reserveAssets)
	s.Require().NoError(err)
	s.Assert().EqualValues(sdk.NewDec(10_000_000), reserveAssets.BaseAssetReserve)
	s.Assert().EqualValues(sdk.NewDec(60_000_000_000), reserveAssets.QuoteAssetReserve)

	s.T().Log("A. check trader has no existing positions")
	_, err = testutilcli.QueryTraderPosition(val.ClientCtx, assetPair, user)
	s.Assert().Error(err, "no position found")

	s.T().Log("B. open position")
	args := []string{
		"--from",
		user.String(),
		"buy",
		assetPair.String(),
		"1",       // Leverage
		"1000000", // 1 BTC
		"1",
	}
	_, err = clitestutil.ExecTestCLICmd(val.ClientCtx, cli.OpenPositionCmd(), append(args, commonArgs...))
	s.Require().NoError(err)

	s.T().Log("B. check vpool balance after open position")
	reserveAssets, err = testutilcli.QueryVpoolReserveAssets(val.ClientCtx, assetPair)
	s.T().Logf("reserve assets: %+v", reserveAssets)
	s.Require().NoError(err)
	s.Assert().EqualValues(sdk.MustNewDecFromStr("9999833.336111064815586407"), reserveAssets.BaseAssetReserve)
	s.Assert().EqualValues(sdk.NewDec(60_001_000_000), reserveAssets.QuoteAssetReserve)

	s.T().Log("B. check vpool balances")
	queryResp, err := testutilcli.QueryTraderPosition(val.ClientCtx, assetPair, user)
	s.T().Logf("query response: %+v", queryResp)
	s.Require().NoError(err)
	s.Assert().EqualValues(user.String(), queryResp.Position.TraderAddress)
	s.Assert().EqualValues(assetPair.String(), queryResp.Position.Pair)
	s.Assert().EqualValues(sdk.NewDec(1_000_000), queryResp.Position.Margin)
	s.Assert().EqualValues(sdk.NewDec(1_000_000), queryResp.Position.OpenNotional)

	s.T().Log("C. open position with 2x leverage and zero baseAmtLimit")
	args = []string{
		"--from",
		user.String(),
		"buy",
		assetPair.String(),
		/* leverage */ "2",
		/* quoteAmt */ "1000000", // 10^6 unusd
		/* baseAmtLimit */ "0",
	}
	_, err = clitestutil.ExecTestCLICmd(val.ClientCtx, cli.OpenPositionCmd(), append(args, commonArgs...))
	s.Require().NoError(err)

	s.T().Log("C. check trader position")
	queryResp, err = testutilcli.QueryTraderPosition(val.ClientCtx, assetPair, user)
	s.T().Logf("query response: %+v", queryResp)
	s.Require().NoError(err)
	s.Assert().EqualValues(user.String(), queryResp.Position.TraderAddress)
	s.Assert().EqualValues(assetPair.String(), queryResp.Position.Pair)
	s.Assert().EqualValues(sdk.NewDec(2_000_000), queryResp.Position.Margin)
	s.Assert().EqualValues(sdk.NewDec(3_000_000), queryResp.Position.OpenNotional)

	s.T().Log("D. Open a reverse position smaller than the existing position")
	args = []string{
		"--from",
		user.String(),
		"sell",
		assetPair.String(),
		"1",   // Leverage
		"100", // unusd
		"1",
	}
	res, err := clitestutil.ExecTestCLICmd(val.ClientCtx, cli.OpenPositionCmd(), append(args, commonArgs...))
	s.Require().NoError(err)
	s.Assert().NotContains(res.String(), "fail")

	s.T().Log("D. Check vpool after opening reverse position")
	reserveAssets, err = testutilcli.QueryVpoolReserveAssets(val.ClientCtx, assetPair)
	s.T().Logf(" \n reserve assets: %+v \n", reserveAssets)
	s.Require().NoError(err)
	s.Assert().EqualValues(sdk.MustNewDecFromStr("9999500.041663750215262154"), reserveAssets.BaseAssetReserve)
	s.Assert().EqualValues(sdk.NewDec(60_002_999_900), reserveAssets.QuoteAssetReserve)

	s.T().Log("D. Check trader position")
	queryResp, err = testutilcli.QueryTraderPosition(val.ClientCtx, assetPair, user)
	s.T().Logf("query response: %+v", queryResp)
	s.Require().NoError(err)
	s.Assert().EqualValues(user.String(), queryResp.Position.TraderAddress)
	s.Assert().EqualValues(assetPair.String(), queryResp.Position.Pair)
	s.Assert().EqualValues(sdk.NewDec(2_000_000), queryResp.Position.Margin)
	s.Assert().EqualValues(sdk.NewDec(2_999_900), queryResp.Position.OpenNotional)

	s.T().Log("E. Open a reverse position larger than the existing position")
	args = []string{
		"--from",
		user.String(),
		"sell",
		assetPair.String(),
		"1",          // Leverage
		"4000000",    // 4*10^6 unusd
		"2000000000", // TODO: just threw a large number here, figure out a more appropriate amount
	}
	res, err = clitestutil.ExecTestCLICmd(val.ClientCtx, cli.OpenPositionCmd(), append(args, commonArgs...))
	s.Require().NoError(err)
	s.Assert().NotContains(res.String(), "fail")

	s.T().Log("E. Check trader position")
	queryResp, err = testutilcli.QueryTraderPosition(val.ClientCtx, assetPair, user)
	s.T().Logf("query response: %+v", queryResp)
	s.Require().NoError(err)
	s.Assert().EqualValues(user.String(), queryResp.Position.TraderAddress)
	s.Assert().EqualValues(assetPair.String(), queryResp.Position.Pair)
	s.Assert().EqualValues(sdk.MustNewDecFromStr("1000100.000000000000000494"), queryResp.Position.OpenNotional)
	s.Assert().EqualValues(sdk.MustNewDecFromStr("-166.686111713005402945"), queryResp.Position.Size_)
	s.Assert().EqualValues(sdk.MustNewDecFromStr("1000100.000000000000000494"), queryResp.Position.Margin)

	s.T().Log("F. Close position")
	args = []string{
		"--from",
		user.String(),
		assetPair.String(),
	}
	_, err = clitestutil.ExecTestCLICmd(val.ClientCtx, cli.ClosePositionCmd(), append(args, commonArgs...))
	s.Require().NoError(err)

	s.T().Log("F. check trader position")
	queryResp, err = testutilcli.QueryTraderPosition(val.ClientCtx, assetPair, user)
	s.T().Logf("query response: %+v", queryResp)
	s.Require().NotNil(err)
	s.Assert().Contains(err.Error(), perptypes.ErrPositionNotFound.Error())
	s.Assert().Nil(queryResp.Position)
}

func (s *IntegrationTestSuite) TestPositionEmptyAndClose() {
	val := s.network.Validators[0]
	assetPair := common.AssetPair{
		Token0: "eth",
		Token1: "unibi",
	}

	user := s.users[0]

	// verify trader has no position (empty)
	_, err := testutilcli.QueryTraderPosition(val.ClientCtx, assetPair, user)
	s.Assert().Error(err, "no position found")

	// close position should produce error
	args := []string{
		"--from",
		user.String(),
		assetPair.String(),
	}
	out, _ := clitestutil.ExecTestCLICmd(val.ClientCtx, cli.ClosePositionCmd(), append(args, commonArgs...))
	s.Assert().Contains(out.String(), "no position found")
}

func (s *IntegrationTestSuite) TestGetPrices() {
	val := s.network.Validators[0]
	assetPair := common.AssetPair{
		Token0: "eth",
		Token1: "unibi",
	}

	s.T().Log("check vpool balances")
	reserveAssets, err := testutilcli.QueryVpoolReserveAssets(val.ClientCtx, assetPair)
	s.Require().NoError(err)
	s.Assert().EqualValues(sdk.MustNewDecFromStr("10000000"), reserveAssets.BaseAssetReserve)
	s.Assert().EqualValues(sdk.MustNewDecFromStr("60000000000"), reserveAssets.QuoteAssetReserve)

	s.T().Log("check prices")
	priceInfo, err := testutilcli.QueryBaseAssetPrice(val.ClientCtx, assetPair, "1", "100")
	s.T().Logf("priceInfo: %+v", priceInfo)
	s.Assert().EqualValues(sdk.MustNewDecFromStr("599994.000059999400006000"), priceInfo.PriceInQuoteDenom)
	s.Require().NoError(err)
}

func (s *IntegrationTestSuite) TestRemoveMargin() {
	// Set up the user accounts
	val := s.network.Validators[0]
	pair := common.TestStablePool

	// Open a position with first user
	s.T().Log("opening a position with user 1....")
	args := []string{
		"--from",
		s.users[0].String(),
		"buy",
		pair.String(),
		"10", // Leverage
		"1",  // Quote asset amount
		"0.0000001",
	}
	_, err := clitestutil.ExecTestCLICmd(val.ClientCtx, cli.OpenPositionCmd(), append(args, commonArgs...))
	if err != nil {
		s.T().Logf("user1 open position err: %+v", err)
	}
	s.Require().NoError(err)

	// Remove margin to trigger bad debt on user 1
	s.T().Log("removing margin on user 1....")
	args = []string{
		"--from",
		s.users[0].String(),
		pair.String(),
		fmt.Sprintf("%s%s", "100", common.TestStablePool.Token1), // Amount
	}
	out, err := clitestutil.ExecTestCLICmd(val.ClientCtx, cli.RemoveMarginCmd(), append(args, commonArgs...))
	if err != nil {
		s.T().Logf("user1 remove margin err: %+v", err)
	}

	s.Require().Contains(out.String(), perptypes.ErrFailedRemoveMarginCanCauseBadDebt.Error())
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
