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
	}

	genesisState[perptypes.ModuleName] = s.cfg.Codec.MustMarshalJSON(perpGenesis)
	s.cfg.GenesisState = genesisState

	app.SetPrefixes(app.AccountAddressPrefix)

	s.network = testutilcli.New(s.T(), s.cfg)

	_, err := s.network.WaitForHeight(1)
	s.Require().NoError(err)
}

func (s *IntegrationTestSuite) TearDownSuite() {
	s.T().Log("tearing down integration test suite")
	s.network.Cleanup()
}

func (s *IntegrationTestSuite) TestOpenPositionCmd() {
	val := s.network.Validators[0]
	pair := fmt.Sprintf("%s%s%s", "ubtc", common.PairSeparator, "unibi")

	info, _, err := val.ClientCtx.Keyring.
		NewMnemonic("user1", keyring.English, sdk.FullFundraiserPath, "", hd.Secp256k1)
	s.Require().NoError(err)

	user := sdk.AccAddress(info.GetPubKey().Address())

	_, err = utils.FillWalletFromValidator(user,
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
	reserveAssets, err := testutilcli.QueryVpoolReserveAssets(val.ClientCtx, common.TokenPair(pair))
	s.Require().NoError(err)
	s.Require().Equal(sdk.MustNewDecFromStr("10000000"), reserveAssets.BaseAssetReserve)
	s.Require().Equal(sdk.MustNewDecFromStr("60000000000"), reserveAssets.QuoteAssetReserve)

	args := []string{
		"--from",
		user.String(),
		"buy",
		fmt.Sprintf("%s%s%s", "ubtc", common.PairSeparator, "unibi"),
		"1",       // Leverage
		"1000000", // 1 BTC
		"1",
	}
	commonArgs := []string{
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastBlock),
		fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(10))).String()),
	}

	_, err = testutilcli.QueryTraderPosition(val.ClientCtx, common.TokenPair(pair), user)
	s.Require().True(strings.Contains(err.Error(), "no position found"))

	_, err = clitestutil.ExecTestCLICmd(val.ClientCtx, cli.OpenPositionCmd(), append(args, commonArgs...))
	s.Require().NoError(err)

	// Check vpool after opening position
	reserveAssets, err = testutilcli.QueryVpoolReserveAssets(val.ClientCtx, "ubtc:unibi")
	s.Require().NoError(err)
	s.Require().Equal(sdk.MustNewDecFromStr("9999833.336111064815586407"), reserveAssets.BaseAssetReserve)
	s.Require().Equal(sdk.MustNewDecFromStr("60001000000"), reserveAssets.QuoteAssetReserve)

	// Check position
	queryResp, err := testutilcli.QueryTraderPosition(val.ClientCtx, common.TokenPair(pair), user)
	s.Require().NoError(err)
	s.Require().Equal(user, queryResp.Position.TraderAddress)
	s.Require().Equal(pair, queryResp.Position.Pair)
	s.Require().Equal(sdk.MustNewDecFromStr("1000000"), queryResp.Position.Margin)
	s.Require().Equal(sdk.MustNewDecFromStr("1000000"), queryResp.Position.OpenNotional)
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
