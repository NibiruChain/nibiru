// documentation: https://pkg.go.dev/github.com/cosmos/cosmos-sdk@v0.46.0-beta1/testutil/network
package cli_test

import (
	"fmt"
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
			TradeLimitRatio:       sdk.MustNewDecFromStr("10000000"),
			FluctuationLimitRatio: sdk.MustNewDecFromStr("10000000"),
			MaxOracleSpreadRatio:  sdk.MustNewDecFromStr("10000000"),
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
		"0.0000001",
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

	s.checkPositions(val, pair, []sdk.AccAddress{user1})
	s.checkBalances(val, s.users)

	// Open a huge position with user 2 to cause vpool to go underwater via price change
	args2 := []string{
		"--from",
		user2.String(),
		"buy",
		pairStr,
		"1",       // Leverage
		"1000000", // 1 BTC
		"0.0000001",
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
