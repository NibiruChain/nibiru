// documentation: https://pkg.go.dev/github.com/cosmos/cosmos-sdk@v0.46.0-beta1/testutil/network
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
	banktestutil "github.com/cosmos/cosmos-sdk/x/bank/client/testutil"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/client/cli"
	perptypes "github.com/NibiruChain/nibiru/x/perp/types"
	pftypes "github.com/NibiruChain/nibiru/x/pricefeed/types"
	utils "github.com/NibiruChain/nibiru/x/testutil"
	testutilcli "github.com/NibiruChain/nibiru/x/testutil/cli"
	vpooltypes "github.com/NibiruChain/nibiru/x/vpool/types"
)

const (
	oracleAddress = "nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl"
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

// NewPricefeedGen returns an x/pricefeed GenesisState to specify the module parameters.
func NewPricefeedGen() *pftypes.GenesisState {
	oracle, err := sdk.AccAddressFromBech32(oracleAddress)
	if err != nil {
		panic(err)
	}

	return &pftypes.GenesisState{
		Params: pftypes.Params{
			Pairs: []pftypes.Pair{
				// TODO: not used - delete after wrapping up remove margin test
				// {Token0: common.GovStablePool.Token0,
				// 	Token1:  common.GovStablePool.Token1,
				// 	Oracles: []sdk.AccAddress{oracle}, Active: true},
				// {Token0: common.CollStablePool.Token0,
				// 	Token1:  common.CollStablePool.Token1,
				// 	Oracles: []sdk.AccAddress{oracle}, Active: true},
				{Token0: common.TestStablePool.Token0,
					Token1:  common.TestStablePool.Token1,
					Oracles: []sdk.AccAddress{oracle}, Active: true},
			},
		},
		PostedPrices: []pftypes.PostedPrice{
			// TODO: not used - delete after wrapping up remove margin test
			// {
			// 	PairID:        common.GovStablePool.PairID(),
			// 	OracleAddress: oracle,
			// 	Price:         sdk.NewDec(10),
			// 	Expiry:        time.Now().Add(1 * time.Hour),
			// },
			// {
			// 	PairID:        common.CollStablePool.PairID(),
			// 	OracleAddress: oracle,
			// 	Price:         sdk.OneDec(),
			// 	Expiry:        time.Now().Add(1 * time.Hour),
			// },
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

	s.cfg = utils.DefaultConfig()
	s.cfg.NumValidators = 1

	app.SetPrefixes(app.AccountAddressPrefix)
	genesisState := app.ModuleBasics.DefaultGenesis(s.cfg.Codec)

	// setup vpool
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
			Pair:              common.TestStablePool.String(),
			BaseAssetReserve:  sdk.MustNewDecFromStr("100"),
			QuoteAssetReserve: sdk.MustNewDecFromStr("600"),

			// below sets any trade is allowed
			TradeLimitRatio:       sdk.MustNewDecFromStr("10000000"), // 10000000 * 100%
			FluctuationLimitRatio: sdk.MustNewDecFromStr("10000000"),
			MaxOracleSpreadRatio:  sdk.MustNewDecFromStr("10000000"),
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

	bip39Passphrase := "password"

	uid1 := "user1"
	info, _, err := val.ClientCtx.Keyring.
		NewMnemonic(uid1, keyring.English, sdk.FullFundraiserPath, bip39Passphrase, hd.Secp256k1)
	s.Require().NoError(err)
	user1 := sdk.AccAddress(info.GetPubKey().Address())
	// s.T().Logf("user1 info: acc %+v | address %+v", user1, info.GetPubKey())

	info2, _, err := val.ClientCtx.Keyring.
		NewMnemonic("user2", keyring.English, sdk.FullFundraiserPath, bip39Passphrase, hd.Secp256k1)
	s.Require().NoError(err)
	user2 := sdk.AccAddress(info2.GetPubKey().Address())
	// s.T().Logf("user2 info: acc %+v | address %+v", user2, info2.GetPubKey())

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
	s.T().Log("Checking trader balances.... \n \n")

	for i := 0; i < len(users); i++ {
		balance, err := banktestutil.QueryBalancesExec(
			val.ClientCtx,
			users[i],
		)
		s.T().Logf("user %+v (acc: %+v) balance: \n %+v \n", i, users[i], balance)

		if err != nil {
			s.T().Logf("balance err: %+v", err)
			return err
		}
	}

	return nil
}

func (s *IntegrationTestSuite) checkPositions(val *testutilcli.Validator, pair common.AssetPair, users []sdk.AccAddress) error {
	s.T().Log("Checking trader positions.... \n \n")

	for i := 0; i < len(users); i++ {
		queryResp, err := testutilcli.QueryTraderPosition(val.ClientCtx, pair, users[i])
		s.T().Logf("user %+v (acc: %+v) position: \n %+v \n", i, users[i], queryResp)

		if err != nil {
			s.T().Logf("query error: %+v", err)
			return err
		}
	}

	return nil
}

func (s *IntegrationTestSuite) checkReserveAssets(val *testutilcli.Validator, pair common.AssetPair) error {
	s.T().Log("Checking vpool reserve assets....")

	reserveAssets, err := testutilcli.QueryVpoolReserveAssets(val.ClientCtx, pair)
	s.T().Logf("reserve assets: %+v", reserveAssets)
	if err != nil {
		s.T().Logf("reserve assets err: %+v", err)
	}

	return nil
}

func (s *IntegrationTestSuite) checkStatus(val *testutilcli.Validator, pair common.AssetPair, users []sdk.AccAddress) {
	err := s.checkReserveAssets(val, pair)
	if err != nil {
		s.T().Logf("query reserve assets err: %+v", err)
	}

	err = s.checkBalances(val, s.users)
	if err != nil {
		s.T().Logf("query balances err: %+v", err)
	}

	err = s.checkPositions(val, pair, users)
	if err != nil {
		s.T().Logf("query positions err: %+v", err)
	}

	// add a break to the logs for easier readability
	s.T().Log("\n \n")
}

func (s *IntegrationTestSuite) TestRemoveMarginOnUnderwaterPosition() {
	// Set up the user accounts
	val := s.network.Validators[0]
	pair := common.TestStablePool

	_, err := utils.FillWalletFromValidator(s.users[0],
		sdk.NewCoins(
			sdk.NewInt64Coin(s.cfg.BondDenom, 10_000),
			sdk.NewInt64Coin(common.GovDenom, 50_000_000),
			sdk.NewInt64Coin(common.CollDenom, 50_000_000),
			sdk.NewInt64Coin(common.TestTokenDenom, 50_000_000),
			sdk.NewInt64Coin(common.StableDenom, 50_000_000),
		),
		val,
		s.cfg.BondDenom,
	)
	s.Require().NoError(err)

	_, err = utils.FillWalletFromValidator(s.users[1],
		sdk.NewCoins(
			sdk.NewInt64Coin(s.cfg.BondDenom, 10_000),
			sdk.NewInt64Coin(common.GovDenom, 50_000_000),
			sdk.NewInt64Coin(common.CollDenom, 50_000_000),
			sdk.NewInt64Coin(common.TestTokenDenom, 50_000_000),
			sdk.NewInt64Coin(common.StableDenom, 50_000_000),
		),
		val,
		s.cfg.BondDenom,
	)
	s.Require().NoError(err)

	// Check status: vpool reserve assets, balances, positions
	s.checkStatus(val, pair, s.users)

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
	commonArgs := []string{
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastBlock),
		fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(10))).String()),
	}

	_, err = clitestutil.ExecTestCLICmd(val.ClientCtx, cli.OpenPositionCmd(), append(args, commonArgs...))
	if err != nil {
		s.T().Logf("user1 open position err: %+v", err)
	}
	s.Require().NoError(err)

	// Check status: vpool reserve assets, balances, positions
	s.checkStatus(val, pair, s.users)

	// Remove margin to trigger bad debt on user 1
	s.T().Log("removing margin on user 1....")
	args = []string{
		"--from",
		s.users[0].String(),
		pair.String(),
		fmt.Sprintf("%s%s", "100", common.TestStablePool.Token1), // Amount
	}
	// TODO: errors don't bubble up to the "err" variable here
	_, err = clitestutil.ExecTestCLICmd(val.ClientCtx, cli.RemoveMarginCmd(), append(args, commonArgs...))
	if err != nil {
		s.T().Logf("user1 remove margin err: %+v", err)
	}

	// Check status: vpool reserve assets, balances, positions
	s.checkStatus(val, pair, s.users)

	// Open a huge position with second user
	args = []string{
		"--from",
		s.users[1].String(),
		"buy",
		pair.String(),
		"1",    // Leverage
		"6000", // Amount
		"0.0000000001",
	}
	_, err = clitestutil.ExecTestCLICmd(val.ClientCtx, cli.OpenPositionCmd(), append(args, commonArgs...))
	if err != nil {
		s.T().Logf("user2 open position err: %+v", err)
	}
	s.Require().NoError(err)

	// Check status: vpool reserve assets, balances, positions
	s.checkStatus(val, pair, s.users)

	// Add margin to trigger bad debt
	// TODO: not working on triggering bad debt
	s.T().Log("adding margin on user 1....")
	args = []string{
		"--from",
		s.users[1].String(),
		pair.String(),
		fmt.Sprintf("%s%s", "1", common.TestStablePool.Token1), // amount / margin
	}
	_, err = clitestutil.ExecTestCLICmd(val.ClientCtx, cli.AddMarginCmd(), append(args, commonArgs...))
	if err != nil {
		s.T().Logf("user 1 add margin err: %+v", err)
	}
	s.Require().NoError(err)

	// Check status: vpool reserve assets, balances, positions
	s.checkStatus(val, pair, s.users)

	// Liquidate on user 1 to trigger bad debt
	// TODO: not working on triggering bad debt
	// TODO: error doesn't bubble up here so if a liquidation fails there is no "err" or way to know
	s.T().Log("liquidating on user 1....")
	args = []string{
		"--from",
		s.users[1].String(),
		pair.String(),
		s.users[0].String(), // user to liquidate
	}
	_, err = clitestutil.ExecTestCLICmd(val.ClientCtx, cli.LiquidateCmd(), append(args, commonArgs...))
	if err != nil {
		s.T().Logf("liquidating on user 1 err: %+v", err)
	}
	s.Require().NoError(err)

	// Check status: vpool reserve assets, balances, positions
	s.checkStatus(val, pair, s.users)
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
