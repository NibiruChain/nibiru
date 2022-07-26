package cli_test

import (
	"fmt"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdktestutilcli "github.com/cosmos/cosmos-sdk/testutil/cli"
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
	fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(common.DenomGov, sdk.NewInt(10))).String()),
}

type IntegrationTestSuite struct {
	suite.Suite

	cfg     testutilcli.Config
	network *testutilcli.Network
	users   []sdk.AccAddress
}

// NewPricefeedGen returns an x/pricefeed GenesisState to specify the module parameters.
func NewPricefeedGen() *pftypes.GenesisState {
	const oracleAddress = "nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl"
	oracle := sdk.MustAccAddressFromBech32(oracleAddress)

	pairs := common.AssetPairs{common.PairBTCStable}
	return &pftypes.GenesisState{
		Params: pftypes.Params{Pairs: pairs},
		PostedPrices: []pftypes.PostedPrice{
			{
				PairID: common.PairBTCStable.String(),
				Oracle: oracle.String(),
				Price:  sdk.OneDec(),
				Expiry: time.Now().Add(1 * time.Hour),
			},
		},
		GenesisOracles: []string{oracle.String()},
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

	app.SetPrefixes(app.AccountAddressPrefix)
	encodingConfig := app.MakeTestEncodingConfig()
	defaultAppGenesis := app.NewDefaultGenesisState(encodingConfig.Marshaler)
	s.cfg = testutilcli.BuildNetworkConfig(defaultAppGenesis)

	genesisState := defaultAppGenesis

	// setup vpool
	vpoolGenesis := vpooltypes.DefaultGenesis()
	vpoolGenesis.Vpools = []*vpooltypes.Pool{
		{
			Pair:                  common.PairBTCStable,
			BaseAssetReserve:      sdk.NewDec(10_000_000),
			QuoteAssetReserve:     sdk.NewDec(60_000_000_000),
			TradeLimitRatio:       sdk.MustNewDecFromStr("0.8"),
			FluctuationLimitRatio: sdk.MustNewDecFromStr("0.2"),
			MaxOracleSpreadRatio:  sdk.MustNewDecFromStr("0.2"),
		},
		{
			Pair:                  common.PairETHStable,
			BaseAssetReserve:      sdk.NewDec(10_000_000),
			QuoteAssetReserve:     sdk.NewDec(60_000_000_000),
			TradeLimitRatio:       sdk.MustNewDecFromStr("0.8"),
			FluctuationLimitRatio: sdk.MustNewDecFromStr("0.2"),
			MaxOracleSpreadRatio:  sdk.MustNewDecFromStr("0.2"),
		},
	}
	genesisState[vpooltypes.ModuleName] = s.cfg.Codec.MustMarshalJSON(vpoolGenesis)

	// setup perp
	perpGenesis := perptypes.DefaultGenesis()
	perpGenesis.PairMetadata = []*perptypes.PairMetadata{
		{
			Pair: common.PairBTCStable,
			CumulativePremiumFractions: []sdk.Dec{
				sdk.ZeroDec(),
			},
		},
		{
			Pair: common.MustNewAssetPair("eth:unibi"),
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

	s.network = testutilcli.NewNetwork(s.T(), s.cfg)

	_, err := s.network.WaitForHeight(1)
	s.NoError(err)

	val := s.network.Validators[0]
	info, _, err := val.ClientCtx.Keyring.
		NewMnemonic("user1", keyring.English, sdk.FullFundraiserPath, "", hd.Secp256k1)
	s.NoError(err)
	user1 := sdk.AccAddress(info.GetPubKey().Address())

	s.users = []sdk.AccAddress{user1}

	_, err = testutilcli.FillWalletFromValidator(user1,
		sdk.NewCoins(
			sdk.NewInt64Coin(s.cfg.BondDenom, 20_000),
			sdk.NewInt64Coin(common.DenomGov, 100_000_000),
			sdk.NewInt64Coin(common.DenomColl, 100_000_000),
			sdk.NewInt64Coin(common.DenomStable, 50_000_000),
		),
		val,
		s.cfg.BondDenom,
	)
	s.NoError(err)
}

func (s *IntegrationTestSuite) TearDownSuite() {
	s.T().Log("tearing down integration test suite")
	s.network.Cleanup()
}

func (s *IntegrationTestSuite) TestOpenPositionsAndCloseCmd() {
	val := s.network.Validators[0]

	user := s.users[0]

	s.T().Log("A. check vpool balances")
	reserveAssets, err := testutilcli.QueryVpoolReserveAssets(val.ClientCtx, common.PairBTCStable)
	s.T().Logf("reserve assets: %+v", reserveAssets)
	s.NoError(err)
	s.EqualValues(sdk.NewDec(10_000_000), reserveAssets.BaseAssetReserve)
	s.EqualValues(sdk.NewDec(60_000_000_000), reserveAssets.QuoteAssetReserve)

	s.T().Log("A. check trader has no existing positions")
	_, err = testutilcli.QueryTraderPosition(val.ClientCtx, common.PairBTCStable, user)
	s.Error(err, "no position found")

	s.T().Log("B. open position")
	args := []string{
		"--from",
		user.String(),
		"buy",
		common.PairBTCStable.String(),
		/* leverage */ "1",
		/* quoteAmt */ "1000000", // 10^6 uNUSD
		/* baseAssetLimit */ "1",
	}
	_, err = sdktestutilcli.ExecTestCLICmd(val.ClientCtx, cli.OpenPositionCmd(), append(args, commonArgs...))
	s.NoError(err)

	s.T().Log("B. check vpool balance after open position")
	reserveAssets, err = testutilcli.QueryVpoolReserveAssets(val.ClientCtx, common.PairBTCStable)
	s.T().Logf("reserve assets: %+v", reserveAssets)
	s.NoError(err)
	s.EqualValues(sdk.MustNewDecFromStr("9999833.336111064815586407"), reserveAssets.BaseAssetReserve)
	s.EqualValues(sdk.NewDec(60_001_000_000), reserveAssets.QuoteAssetReserve)

	s.T().Log("B. check vpool balances")
	queryResp, err := testutilcli.QueryTraderPosition(val.ClientCtx, common.PairBTCStable, user)
	s.T().Logf("query response: %+v", queryResp)
	s.NoError(err)
	s.EqualValues(user.String(), queryResp.Position.TraderAddress)
	s.EqualValues(common.PairBTCStable, queryResp.Position.Pair)
	s.EqualValues(sdk.MustNewDecFromStr("166.663888935184413593"), queryResp.Position.Size_)
	s.EqualValues(sdk.NewDec(1_000_000), queryResp.Position.Margin)
	s.EqualValues(sdk.NewDec(1_000_000), queryResp.Position.OpenNotional)
	s.EqualValues(sdk.MustNewDecFromStr("999999.999999999999999359"), queryResp.PositionNotional)
	s.EqualValues(sdk.MustNewDecFromStr("-0.000000000000000641"), queryResp.UnrealizedPnl)
	s.EqualValues(sdk.NewDec(1), queryResp.MarginRatio)

	s.T().Log("C. open position with 2x leverage and zero baseAmtLimit")
	args = []string{
		"--from",
		user.String(),
		"buy",
		common.PairBTCStable.String(),
		/* leverage */ "2",
		/* quoteAmt */ "1000000", // 10^6 uNUSD
		/* baseAmtLimit */ "0",
	}
	_, err = sdktestutilcli.ExecTestCLICmd(val.ClientCtx, cli.OpenPositionCmd(), append(args, commonArgs...))
	s.NoError(err)

	s.T().Log("C. check trader position")
	queryResp, err = testutilcli.QueryTraderPosition(val.ClientCtx, common.PairBTCStable, user)
	s.T().Logf("query response: %+v", queryResp)
	s.NoError(err)
	s.EqualValues(user.String(), queryResp.Position.TraderAddress)
	s.EqualValues(common.PairBTCStable, queryResp.Position.Pair)
	s.EqualValues(sdk.MustNewDecFromStr("499.975001249937503125"), queryResp.Position.Size_)
	s.EqualValues(sdk.NewDec(2_000_000), queryResp.Position.Margin)
	s.EqualValues(sdk.NewDec(3_000_000), queryResp.Position.OpenNotional)
	s.EqualValues(sdk.MustNewDecFromStr("3000000.000000000000000938"), queryResp.PositionNotional)
	s.EqualValues(sdk.MustNewDecFromStr("0.000000000000000938"), queryResp.UnrealizedPnl)
	s.EqualValues(sdk.MustNewDecFromStr("0.666666666666666667"), queryResp.MarginRatio)

	s.T().Log("D. Open a reverse position smaller than the existing position")
	args = []string{
		"--from",
		user.String(),
		"sell",
		common.PairBTCStable.String(),
		/* leverage */ "1",
		/* quoteAmt */ "100", // 100 uNUSD
		/* baseAssetLimit */ "1",
	}
	res, err := sdktestutilcli.ExecTestCLICmd(val.ClientCtx, cli.OpenPositionCmd(), append(args, commonArgs...))
	s.NoError(err)
	s.NotContains(res.String(), "fail")

	s.T().Log("D. Check vpool after opening reverse position")
	reserveAssets, err = testutilcli.QueryVpoolReserveAssets(val.ClientCtx, common.PairBTCStable)
	s.T().Logf(" \n reserve assets: %+v \n", reserveAssets)
	s.NoError(err)
	s.EqualValues(sdk.MustNewDecFromStr("9999500.041663750215262154"), reserveAssets.BaseAssetReserve)
	s.EqualValues(sdk.NewDec(60_002_999_900), reserveAssets.QuoteAssetReserve)

	s.T().Log("D. Check trader position")
	queryResp, err = testutilcli.QueryTraderPosition(val.ClientCtx, common.PairBTCStable, user)
	s.T().Logf("query response: %+v", queryResp)
	s.NoError(err)
	s.EqualValues(user.String(), queryResp.Position.TraderAddress)
	s.EqualValues(common.PairBTCStable, queryResp.Position.Pair)
	s.EqualValues(sdk.MustNewDecFromStr("499.958336249784737846"), queryResp.Position.Size_)
	s.EqualValues(sdk.NewDec(2_000_000), queryResp.Position.Margin)
	s.EqualValues(sdk.NewDec(2_999_900), queryResp.Position.OpenNotional)
	s.EqualValues(sdk.MustNewDecFromStr("2999899.999999999999999506"), queryResp.PositionNotional)
	s.EqualValues(sdk.MustNewDecFromStr("-0.000000000000000494"), queryResp.UnrealizedPnl)
	s.EqualValues(sdk.MustNewDecFromStr("0.666688889629654322"), queryResp.MarginRatio)

	s.T().Log("E. Open a reverse position larger than the existing position")
	args = []string{
		"--from",
		user.String(),
		"sell",
		common.PairBTCStable.String(),
		/* leverage */ "1",
		/* quoteAmt */ "4000000", // 4*10^6 uNUSD
		/* baseAssetLimit */ "0",
	}
	res, err = sdktestutilcli.ExecTestCLICmd(val.ClientCtx, cli.OpenPositionCmd(), append(args, commonArgs...))
	s.NoError(err)
	s.NotContains(res.String(), "fail")

	s.T().Log("E. Check trader position")
	queryResp, err = testutilcli.QueryTraderPosition(val.ClientCtx, common.PairBTCStable, user)
	s.T().Logf("query response: %+v", queryResp)
	s.NoError(err)
	s.EqualValues(user.String(), queryResp.Position.TraderAddress)
	s.EqualValues(common.PairBTCStable, queryResp.Position.Pair)
	s.EqualValues(sdk.MustNewDecFromStr("-166.686111713005402945"), queryResp.Position.Size_)
	s.EqualValues(sdk.MustNewDecFromStr("1000100.000000000000000494"), queryResp.Position.OpenNotional)
	s.EqualValues(sdk.MustNewDecFromStr("1000100.000000000000000494"), queryResp.Position.Margin)
	s.EqualValues(sdk.MustNewDecFromStr("1000099.999999999999999651"), queryResp.PositionNotional)
	s.EqualValues(sdk.MustNewDecFromStr("0.000000000000000843"), queryResp.UnrealizedPnl)
	s.EqualValues(sdk.NewDec(1), queryResp.MarginRatio)

	s.T().Log("F. Close position")
	args = []string{
		"--from",
		user.String(),
		common.PairBTCStable.String(),
	}
	_, err = sdktestutilcli.ExecTestCLICmd(val.ClientCtx, cli.ClosePositionCmd(), append(args, commonArgs...))
	s.NoError(err)

	s.T().Log("F. check trader position")
	queryResp, err = testutilcli.QueryTraderPosition(val.ClientCtx, common.PairBTCStable, user)

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
	_, err := testutilcli.QueryTraderPosition(val.ClientCtx, common.PairETHStable, user)
	s.Error(err, "no position found")

	// close position should produce error
	args := []string{
		"--from",
		user.String(),
		common.PairETHStable.String(),
	}
	out, _ := sdktestutilcli.ExecTestCLICmd(val.ClientCtx, cli.ClosePositionCmd(), append(args, commonArgs...))
	s.Contains(out.String(), "no position found")
}

func (s *IntegrationTestSuite) TestGetPrices() {
	val := s.network.Validators[0]

	s.T().Log("check vpool balances")
	reserveAssets, err := testutilcli.QueryVpoolReserveAssets(val.ClientCtx, common.PairETHStable)
	s.NoError(err)
	s.EqualValues(sdk.MustNewDecFromStr("10000000"), reserveAssets.BaseAssetReserve)
	s.EqualValues(sdk.MustNewDecFromStr("60000000000"), reserveAssets.QuoteAssetReserve)

	s.T().Log("check prices")
	priceInfo, err := testutilcli.QueryBaseAssetPrice(val.ClientCtx, common.PairETHStable, "1", "100")
	s.T().Logf("priceInfo: %+v", priceInfo)
	s.EqualValues(sdk.MustNewDecFromStr("599994.000059999400006000"), priceInfo.PriceInQuoteDenom)
	s.NoError(err)
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
		common.PairBTCStable.String(),
		"10", // Leverage
		"1",  // Quote asset amount
		"0.0000001",
	}
	_, err := sdktestutilcli.ExecTestCLICmd(val.ClientCtx, cli.OpenPositionCmd(), append(args, commonArgs...))
	if err != nil {
		s.T().Logf("user1 open position err: %+v", err)
	}
	s.NoError(err)

	// Remove margin to trigger bad debt on user 1
	s.T().Log("removing margin on user 1....")
	args = []string{
		"--from",
		s.users[0].String(),
		common.PairBTCStable.String(),
		fmt.Sprintf("%s%s", "100", common.DenomStable), // Amount
	}
	out, err := sdktestutilcli.ExecTestCLICmd(val.ClientCtx, cli.RemoveMarginCmd(), append(args, commonArgs...))
	if err != nil {
		s.T().Logf("user1 remove margin err: %+v", err)
	}

	s.Contains(out.String(), perptypes.ErrFailedRemoveMarginCanCauseBadDebt.Error())
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
