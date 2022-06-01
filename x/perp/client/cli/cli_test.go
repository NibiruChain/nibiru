package cli_test

import (
	"fmt"
	"strings"
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

func (s *IntegrationTestSuite) TestOpenAndClosePositionCmd() {
	val := s.network.Validators[0]
	pair := common.AssetPair{
		Token0: "ubtc",
		Token1: "unibi",
	}
	pairStr := fmt.Sprintf("%s%s%s", "ubtc", common.PairSeparator, "unibi")

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

	// Check vpool balances
	reserveAssets, err := testutilcli.QueryVpoolReserveAssets(val.ClientCtx, pair)
	s.T().Logf("reserve assets: %+v", reserveAssets)
	s.Require().NoError(err)
	s.Require().Equal(sdk.MustNewDecFromStr("10000000"), reserveAssets.BaseAssetReserve)
	s.Require().Equal(sdk.MustNewDecFromStr("60000000000"), reserveAssets.QuoteAssetReserve)

	args := []string{
		"--from",
		user.String(),
		"buy",
		pairStr,
		"1",       // Leverage
		"1000000", // 1 BTC
		"1",
	}
	commonArgs := []string{
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastBlock),
		fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(10))).String()),
	}

	_, err = testutilcli.QueryTraderPosition(val.ClientCtx, pair, user)
	s.Require().True(strings.Contains(err.Error(), "no position found"))

	// Open position
	_, err = clitestutil.ExecTestCLICmd(val.ClientCtx, cli.OpenPositionCmd(), append(args, commonArgs...))
	s.Require().NoError(err)

	// Check vpool after opening position
	reserveAssets, err = testutilcli.QueryVpoolReserveAssets(val.ClientCtx, pair)
	s.T().Logf("reserve assets: %+v", reserveAssets)
	s.Require().NoError(err)
	s.Require().Equal(sdk.MustNewDecFromStr("9999833.336111064815586407"), reserveAssets.BaseAssetReserve)
	s.Require().Equal(sdk.MustNewDecFromStr("60001000000"), reserveAssets.QuoteAssetReserve)

	// Check position
	queryResp, err := testutilcli.QueryTraderPosition(val.ClientCtx, pair, user)
	s.T().Logf("query response: %+v", queryResp)
	s.Require().NoError(err)
	s.Require().Equal(user.String(), queryResp.Position.TraderAddress)
	s.Require().Equal(pair.String(), queryResp.Position.Pair)
	s.Require().Equal(sdk.MustNewDecFromStr("1000000"), queryResp.Position.Margin)
	s.Require().Equal(sdk.MustNewDecFromStr("1000000"), queryResp.Position.OpenNotional)

	// Open a reverse position smaller than the existing position
	args = []string{
		"--from",
		user.String(),
		"sell",
		pairStr,
		"1",   // Leverage
		"100", // BTC
		"1",
	}
	res, err := clitestutil.ExecTestCLICmd(val.ClientCtx, cli.OpenPositionCmd(), append(args, commonArgs...))
	s.Require().NoError(err)
	s.Require().NotContains(res.String(), "fail")

	// Check vpool after opening reverse position
	reserveAssets, err = testutilcli.QueryVpoolReserveAssets(val.ClientCtx, pair)
	s.T().Logf(" \n reserve assets: %+v \n", reserveAssets)
	s.Require().NoError(err)
	s.Require().Equal(sdk.MustNewDecFromStr("9999833.352777175968362487"), reserveAssets.BaseAssetReserve)
	s.Require().Equal(sdk.MustNewDecFromStr("60000999900.000000000000000000"), reserveAssets.QuoteAssetReserve)

	// Check position
	queryResp, err = testutilcli.QueryTraderPosition(val.ClientCtx, pair, user)
	s.T().Logf("query response: %+v", queryResp)
	s.Require().NoError(err)
	s.Require().NoError(err)
	s.Require().Equal(user.String(), queryResp.Position.TraderAddress)
	s.Require().Equal(pair.String(), queryResp.Position.Pair)
	s.Require().Equal(sdk.MustNewDecFromStr("1000000"), queryResp.Position.Margin)
	s.Require().Equal(sdk.MustNewDecFromStr("999900"), queryResp.Position.OpenNotional)

	// Close positions
	args = []string{
		"--from",
		user.String(),
		pairStr,
	}
	_, err = clitestutil.ExecTestCLICmd(val.ClientCtx, cli.ClosePositionCmd(), append(args, commonArgs...))
	s.Require().NoError(err)

	// After closing position should be zero
	queryResp, err = testutilcli.QueryTraderPosition(val.ClientCtx, pair, user)
	s.T().Logf("query response: %+v", queryResp)
	s.Require().NoError(err)
	s.Require().Equal(sdk.MustNewDecFromStr("0"), queryResp.Position.Margin)
	s.Require().Equal(sdk.MustNewDecFromStr("0"), queryResp.Position.OpenNotional)
	s.Require().Equal(sdk.MustNewDecFromStr("0"), queryResp.Position.Size_)
}

func (s *IntegrationTestSuite) TestPositionEmptyAndClose() {
	val := s.network.Validators[0]
	pair := common.AssetPair{
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
	_, err = testutilcli.QueryTraderPosition(val.ClientCtx, pair, user)
	s.Require().True(strings.Contains(err.Error(), "no position found"))

	// close position should produce error
	args := []string{
		"--from",
		user.String(),
		fmt.Sprintf("%s%s%s", "eth", common.PairSeparator, "unibi"),
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
