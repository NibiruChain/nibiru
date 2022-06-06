// documentation: https://pkg.go.dev/github.com/cosmos/cosmos-sdk@v0.46.0-beta1/testutil/network
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
	banktestutil "github.com/cosmos/cosmos-sdk/x/bank/client/testutil"
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
	s.cfg.NumValidators = 2

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
		{
			Pair:              "test:unibi",
			BaseAssetReserve:  sdk.MustNewDecFromStr("100"),
			QuoteAssetReserve: sdk.MustNewDecFromStr("600"),

			// below sets any trade is allowed
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
		{
			Pair: "test:unibi",
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
	val2 := s.network.Validators[0]

	bip39Passphrase := "password"

	uid1 := "user1"
	info, _, err := val.ClientCtx.Keyring.
		NewMnemonic(uid1, keyring.English, sdk.FullFundraiserPath, bip39Passphrase, hd.Secp256k1)
	s.Require().NoError(err)
	user1 := sdk.AccAddress(info.GetPubKey().Address())
	s.T().Logf("user1 info: acc %+v | address %+v", user1, info.GetPubKey().Address())

	info2, _, err := val2.ClientCtx.Keyring.
		NewMnemonic("user2", keyring.English, sdk.FullFundraiserPath, bip39Passphrase, hd.Secp256k1)
	s.Require().NoError(err)
	user2 := sdk.AccAddress(info2.GetPubKey().Address())
	s.T().Logf("user2 info: acc %+v | address %+v", user2, info2.GetPubKey().Address())

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

func (s *IntegrationTestSuite) checkBalances(val *testutilcli.Validator, users []sdk.AccAddress) error {
	for i := 0; i < len(users); i++ {
		balance, err := banktestutil.QueryBalancesExec(
			val.ClientCtx,
			users[i],
		)
		s.T().Logf("user %+v (acc: %+v) balance: %+v", i, users[i], balance)

		if err != nil {
			s.T().Logf("balance err: %+v", err)
			return err
		}
	}

	return nil
}

func (s *IntegrationTestSuite) checkPositions(val *testutilcli.Validator, pair common.AssetPair, users []sdk.AccAddress) error {

	for i := 0; i < len(users); i++ {
		queryResp, err := testutilcli.QueryTraderPosition(val.ClientCtx, pair, users[i])
		s.T().Logf("user %+v (acc: %+v) query response: %+v", i, users[i], queryResp)

		if err != nil {
			s.T().Logf("query error: %+v", err)
			return err
		}
	}

	return nil
}

// remove margin, pull collateral out
func (s *IntegrationTestSuite) TestRemoveMarginOnUnderwaterPosition() {
	// Set up the user accounts
	val := s.network.Validators[0]
	pair := common.AssetPair{
		Token0: "test",
		Token1: "unibi",
	}
	pairStr := fmt.Sprintf("%s%s%s", "test", common.PairSeparator, "unibi")

	user1 := s.users[0]
	user2 := s.users[1]

	_, err := utils.FillWalletFromValidator(user1,
		sdk.NewCoins(
			sdk.NewInt64Coin(s.cfg.BondDenom, 10_000),
			sdk.NewInt64Coin(common.GovDenom, 50_000_000),
			sdk.NewInt64Coin(common.CollDenom, 50_000_000),
		),
		val,
		s.cfg.BondDenom,
	)
	s.Require().NoError(err)

	_, err = utils.FillWalletFromValidator(user2,
		sdk.NewCoins(
			sdk.NewInt64Coin(s.cfg.BondDenom, 10_000),
			sdk.NewInt64Coin(common.GovDenom, 50_000_000),
			sdk.NewInt64Coin(common.CollDenom, 50_000_000),
		),
		val,
		s.cfg.BondDenom,
	)
	s.Require().NoError(err)

	// Check vpool balances
	s.T().Log("checking vpool balances...")
	reserveAssets, err := testutilcli.QueryVpoolReserveAssets(val.ClientCtx, pair)
	s.T().Logf("reserve assets: %+v", reserveAssets)
	if err != nil {
		s.T().Logf("reserve assets err: %+v", err)
	}

	// Check wallets of users
	s.T().Log("checking user balances....")
	s.checkBalances(val, s.users)

	// Open a position with user 1
	s.T().Log("opening a position with user 1....")
	args := []string{
		"--from",
		user1.String(),
		"buy",
		pairStr,
		"1", // Leverage
		"1", // 1 BTC
		"10",
	}
	commonArgs := []string{
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastBlock),
		fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(10))).String()),
	}

	_, err = clitestutil.ExecTestCLICmd(val.ClientCtx, cli.OpenPositionCmd(), append(args, commonArgs...))
	if err != nil {
		s.T().Logf("user1 open position err: %+v", err)
	}

	return

	s.checkPositions(val, pair, []sdk.AccAddress{user1})
	s.checkBalances(val, s.users)

	// Open a huge position with user 2 to cause vpool to go underwater via price change
	args2 := []string{
		"--from",
		user2.String(),
		"buy",
		pairStr,
		"1", // Leverage
		"1", // 1 BTC
		"1",
	}
	_, err = clitestutil.ExecTestCLICmd(val.ClientCtx, cli.OpenPositionCmd(), append(args2, commonArgs...))
	if err != nil {
		s.T().Logf("user2 op en position err: %+v", err)
	}
	s.checkBalances(val, s.users)

	// Verify user 1 now has bad debt
	s.checkPositions(val, pair, s.users)

	// Remove margin from user 1
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
