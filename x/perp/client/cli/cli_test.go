package cli_test

import (
	"fmt"
	"testing"

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
	utils "github.com/NibiruChain/nibiru/x/testutil"
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

	s.cfg = utils.DefaultConfig()

	genesisState := app.ModuleBasics.DefaultGenesis(s.cfg.Codec)

	vpoolGenesis := vpooltypes.DefaultGenesis()
	vpoolGenesis.Vpools = []*vpooltypes.Pool{
		{
			Pair:                  "ubtc:unibi",
			BaseAssetReserve:      sdk.MustNewDecFromStr("10000000"),
			QuoteAssetReserve:     sdk.MustNewDecFromStr("60000000000"),
			TradeLimitRatio:       sdk.MustNewDecFromStr("0.8"),
			FluctuationLimitRatio: sdk.MustNewDecFromStr("0.2"),
			MaxOracleSpreadRatio:  sdk.MustNewDecFromStr("0.2"),
		},
		{
			Pair:                  "eth:unibi",
			BaseAssetReserve:      sdk.MustNewDecFromStr("10000000"),
			QuoteAssetReserve:     sdk.MustNewDecFromStr("60000000000"),
			TradeLimitRatio:       sdk.MustNewDecFromStr("0.8"),
			FluctuationLimitRatio: sdk.MustNewDecFromStr("0.2"),
			MaxOracleSpreadRatio:  sdk.MustNewDecFromStr("0.2"),
		},
	}
	genesisState[vpooltypes.ModuleName] = s.cfg.Codec.MustMarshalJSON(vpoolGenesis)

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
	}

	genesisState[perptypes.ModuleName] = s.cfg.Codec.MustMarshalJSON(perpGenesis)
	s.cfg.GenesisState = genesisState

	app.SetPrefixes(app.AccountAddressPrefix)

	s.network = testutilcli.New(s.T(), s.cfg)

	_, err := s.network.WaitForHeight(1)
	s.Require().NoError(err)

	val := s.network.Validators[0]
	info, _, err := val.ClientCtx.Keyring.
		NewMnemonic("user1", keyring.English, sdk.FullFundraiserPath, "", hd.Secp256k1)
	s.Require().NoError(err)
	user1 := sdk.AccAddress(info.GetPubKey().Address())

	info2, _, err := val.ClientCtx.Keyring.
		NewMnemonic("user2", keyring.English, sdk.FullFundraiserPath, "", hd.Secp256k1)
	s.Require().NoError(err)
	user2 := sdk.AccAddress(info2.GetPubKey().Address())

	// TODO: figure out why using user2 gives a "key <addr> not found" error
	s.users = []sdk.AccAddress{user1, user2}
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

	_, err := utils.FillWalletFromValidator(user,
		sdk.NewCoins(
			sdk.NewInt64Coin(s.cfg.BondDenom, 20_000),
			sdk.NewInt64Coin(common.GovDenom, 100_000_000),
			sdk.NewInt64Coin(common.CollDenom, 100_000_000),
		),
		val,
		s.cfg.BondDenom,
	)
	s.Require().NoError(err)

	// A. Check vpool balances
	reserveAssets, err := testutilcli.QueryVpoolReserveAssets(val.ClientCtx, assetPair)
	s.T().Logf("reserve assets: %+v", reserveAssets)
	s.Require().NoError(err)
	s.Assert().EqualValues(sdk.NewDec(10_000_000), reserveAssets.BaseAssetReserve)
	s.Assert().EqualValues(sdk.NewDec(60_000_000_000), reserveAssets.QuoteAssetReserve)

	args := []string{
		"--from",
		user.String(),
		"buy",
		assetPair.String(),
		"1",       // Leverage
		"1000000", // 1 BTC
		"1",
	}
	commonArgs := []string{
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastBlock),
		fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(10))).String()),
	}

	_, err = testutilcli.QueryTraderPosition(val.ClientCtx, assetPair, user)
	s.Assert().Error(err, "no position found")

	// A. Open position
	_, err = clitestutil.ExecTestCLICmd(val.ClientCtx, cli.OpenPositionCmd(), append(args, commonArgs...))
	s.Require().NoError(err)

	// A. Check vpool after opening position
	reserveAssets, err = testutilcli.QueryVpoolReserveAssets(val.ClientCtx, assetPair)
	s.T().Logf("reserve assets: %+v", reserveAssets)
	s.Require().NoError(err)
	s.Assert().EqualValues(sdk.MustNewDecFromStr("9999833.336111064815586407"), reserveAssets.BaseAssetReserve)
	s.Assert().EqualValues(sdk.NewDec(60_001_000_000), reserveAssets.QuoteAssetReserve)

	// A. Check position
	queryResp, err := testutilcli.QueryTraderPosition(val.ClientCtx, assetPair, user)
	s.T().Logf("query response: %+v", queryResp)
	s.Require().NoError(err)
	s.Assert().EqualValues(user.String(), queryResp.Position.TraderAddress)
	s.Assert().EqualValues(assetPair.String(), queryResp.Position.Pair)
	s.Assert().EqualValues(sdk.NewDec(1_000_000), queryResp.Position.Margin)
	s.Assert().EqualValues(sdk.NewDec(1_000_000), queryResp.Position.OpenNotional)

	// B. Open a reverse position smaller than the existing position
	args = []string{
		"--from",
		user.String(),
		"sell",
		assetPair.String(),
		"1",   // Leverage
		"100", // BTC
		"1",
	}
	res, err := clitestutil.ExecTestCLICmd(val.ClientCtx, cli.OpenPositionCmd(), append(args, commonArgs...))
	s.Require().NoError(err)
	s.Assert().NotContains(res.String(), "fail")

	// B. Check vpool after opening reverse position
	reserveAssets, err = testutilcli.QueryVpoolReserveAssets(val.ClientCtx, assetPair)
	s.T().Logf(" \n reserve assets: %+v \n", reserveAssets)
	s.Require().NoError(err)
	s.Assert().EqualValues(sdk.MustNewDecFromStr("9999833.352777175968362487"), reserveAssets.BaseAssetReserve)
	s.Assert().EqualValues(sdk.MustNewDecFromStr("60000999900.000000000000000000"), reserveAssets.QuoteAssetReserve)

	// B. Check position
	queryResp, err = testutilcli.QueryTraderPosition(val.ClientCtx, assetPair, user)
	s.T().Logf("query response: %+v", queryResp)
	s.Require().NoError(err)
	s.Assert().EqualValues(user.String(), queryResp.Position.TraderAddress)
	s.Assert().EqualValues(assetPair.String(), queryResp.Position.Pair)
	s.Assert().EqualValues(sdk.NewDec(1_000_000), queryResp.Position.Margin)
	s.Assert().EqualValues(sdk.NewDec(999_900), queryResp.Position.OpenNotional)

	// C. Open a reverse position larger than the existing position
	args = []string{
		"--from",
		user.String(),
		"sell",
		assetPair.String(),
		"1",          // Leverage
		"2000000",    // 2*10^6 unusd
		"2000000000", // TODO: just threw a large number here, figure out a more appropriate amount
	}
	res, err = clitestutil.ExecTestCLICmd(val.ClientCtx, cli.OpenPositionCmd(), append(args, commonArgs...))
	s.Require().NoError(err)
	s.Assert().NotContains(res.String(), "fail")

	// C. Check position
	queryResp, err = testutilcli.QueryTraderPosition(val.ClientCtx, assetPair, user)
	s.T().Logf("query response: %+v", queryResp)
	s.Require().NoError(err)
	s.Assert().EqualValues(user.String(), queryResp.Position.TraderAddress)
	s.Assert().EqualValues(assetPair.String(), queryResp.Position.Pair)
	s.Assert().EqualValues(sdk.MustNewDecFromStr("1000099.999999999999998565"), queryResp.Position.OpenNotional)
	s.Assert().EqualValues(sdk.MustNewDecFromStr("-166.686111713005402945"), queryResp.Position.Size_)
	s.Assert().EqualValues(sdk.MustNewDecFromStr("1000099.999999999999998565"), queryResp.Position.Margin)

	// D. Close positions
	args = []string{
		"--from",
		user.String(),
		assetPair.String(),
	}
	_, err = clitestutil.ExecTestCLICmd(val.ClientCtx, cli.ClosePositionCmd(), append(args, commonArgs...))
	s.Require().NoError(err)

	// D. After closing position should be zero
	queryResp, err = testutilcli.QueryTraderPosition(val.ClientCtx, assetPair, user)
	s.T().Logf("query response: %+v", queryResp)
	s.Require().NoError(err)
	s.Assert().EqualValues(sdk.ZeroDec(), queryResp.Position.Margin)
	s.Assert().EqualValues(sdk.ZeroDec(), queryResp.Position.OpenNotional)
	s.Assert().EqualValues(sdk.ZeroDec(), queryResp.Position.Size_)
}

func (s *IntegrationTestSuite) TestPositionEmptyAndClose() {
	val := s.network.Validators[0]
	assetPair := common.AssetPair{
		Token0: "eth",
		Token1: "unibi",
	}

	user := s.users[0]

	_, err := utils.FillWalletFromValidator(user,
		sdk.NewCoins(
			sdk.NewInt64Coin(s.cfg.BondDenom, 20_000),
			sdk.NewInt64Coin(common.GovDenom, 100_000_000),
			sdk.NewInt64Coin(common.CollDenom, 100_000_000),
		),
		val,
		s.cfg.BondDenom,
	)
	s.Require().NoError(err)

	// verify trader has no position (empty)
	_, err = testutilcli.QueryTraderPosition(val.ClientCtx, assetPair, user)
	s.Assert().Error(err, "no position found")

	// close position should produce error
	args := []string{
		"--from",
		user.String(),
		assetPair.String(),
	}
	commonArgs := []string{
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastBlock),
		fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(10))).String()),
	}
	// TODO: fix that this err doesn't get propagated back up to show up here
	res, err := clitestutil.ExecTestCLICmd(val.ClientCtx, cli.ClosePositionCmd(), append(args, commonArgs...))
	s.T().Logf("res: %+v", res)
	s.T().Logf("err: %+v", err)
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
