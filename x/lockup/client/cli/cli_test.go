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
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	tmcli "github.com/tendermint/tendermint/libs/cli"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/lockup/client/cli"
	testutilcli "github.com/NibiruChain/nibiru/x/testutil/cli"
)

type IntegrationTestSuite struct {
	suite.Suite

	cfg     testutilcli.Config
	network *testutilcli.Network

	userWithLock sdk.AccAddress
	txArgs       []string
	queryArgs    []string
}

func (s *IntegrationTestSuite) SetupSuite() {
	s.txArgs = []string{
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastBlock),
		fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(common.GovDenom, sdk.NewInt(10))).String()),
		fmt.Sprintf("--%s=userWithLock", flags.FlagFrom),
	}
	s.queryArgs = []string{
		fmt.Sprintf("--%s=json", tmcli.OutputFlag),
	}
	if testing.Short() {
		s.T().Skip("skipping lockup CLI tests")
	}

	s.T().Log("setting up lockup CLI testing suite")

	s.cfg = testutilcli.DefaultConfig()
	genesisState := app.ModuleBasics.DefaultGenesis(s.cfg.Codec)

	s.cfg.GenesisState = genesisState
	s.cfg.StartingTokens = sdk.NewCoins(
		sdk.NewInt64Coin("ATOM", 1_000_000),
		sdk.NewInt64Coin("OSMO", 1_000_000),
		sdk.NewInt64Coin("unibi", 1_000_000_000))

	app.SetPrefixes(app.AccountAddressPrefix)
	s.network = testutilcli.New(s.T(), s.cfg)
	_, err := s.network.WaitForHeight(1)
	require.NoError(s.T(), err)

	val := s.network.Validators[0]
	info, _, err := val.ClientCtx.Keyring.
		NewMnemonic("userWithLock", keyring.English, sdk.FullFundraiserPath, "", hd.Secp256k1)
	require.NoError(s.T(), err)

	s.userWithLock = info.GetAddress()

	// fill wallet
	_, err = testutilcli.FillWalletFromValidator(s.userWithLock,
		sdk.NewCoins(sdk.NewInt64Coin("ATOM", 20000), sdk.NewInt64Coin("OSMO", 20000), sdk.NewInt64Coin("unibi", 1_000_000)),
		val, s.cfg.BondDenom)
	require.NoError(s.T(), err)
}

func (s *IntegrationTestSuite) TearDownSuite() {
	s.network.Cleanup()
}

func (s *IntegrationTestSuite) TestLockupCLI() {
	// test coin lock
	s.T().Log("testing coin lock")
	lockup1Args := []string{
		"1000ATOM", // coins to lock
		"10s",      // duration
	}
	result, err := clitestutil.ExecTestCLICmd(
		s.network.Validators[0].ClientCtx,
		cli.GetLockCoinsCmd(),
		append(lockup1Args, s.txArgs...))

	require.NoError(s.T(), err)
	testutilcli.RequireTxOk(s.T(), s.network.Validators[0].ClientCtx.Codec, result.Bytes())
	// test query lock
	s.T().Log("testing query lock")
	queryLockArgs := []string{"0"}
	_, err = clitestutil.ExecTestCLICmd(s.network.Validators[0].ClientCtx, cli.GetQueryLockCmd(), append(queryLockArgs, s.queryArgs...))
	require.NoError(s.T(), err)
	// lock coins again
	lockup2Args := []string{
		"1000OSMO", // coins to lock
		"10s",      // duration
	}
	result, err = clitestutil.ExecTestCLICmd(
		s.network.Validators[0].ClientCtx,
		cli.GetLockCoinsCmd(),
		append(lockup2Args, s.txArgs...))

	require.NoError(s.T(), err)
	testutilcli.RequireTxOk(s.T(), s.network.Validators[0].ClientCtx.Codec, result.Bytes())
	// test query multiple locks
	s.T().Log("testing query locks by address")
	queryLocksByAddressArgs := []string{s.userWithLock.String()}
	result, err = clitestutil.ExecTestCLICmd(s.network.Validators[0].ClientCtx, cli.GetQueryLocksByAddressCmd(), append(queryLocksByAddressArgs, s.queryArgs...))
	require.NoError(s.T(), err)
	s.T().Logf("%s", result)
	// test query locked coins
	s.T().Log("testing query locked coins")
	queryLockedCoinsArgs := []string{s.userWithLock.String()}
	result, err = clitestutil.ExecTestCLICmd(s.network.Validators[0].ClientCtx, cli.GetQueryLockedCoinsCmd(), append(queryLockedCoinsArgs, s.queryArgs...))
	require.NoError(s.T(), err)
	s.T().Logf("%s", result)

	// test initiate unlock
	s.T().Log("testing initiate unlock")
	initiateUnlockArgs := []string{"0"}
	result, err = clitestutil.ExecTestCLICmd(s.network.Validators[0].ClientCtx, cli.GetInitiateUnlockCmd(), append(initiateUnlockArgs, s.txArgs...))
	require.NoError(s.T(), err)
	testutilcli.RequireTxOk(s.T(), s.network.Validators[0].ClientCtx.Codec, result.Bytes())
	s.T().Logf("%s", result)
	// wait some seconds then unlock funds
	s.T().Logf("testing unlock funds")
	require.NoError(s.T(), s.network.WaitForDuration(10*time.Second))

	unlockArgs := []string{"0"}
	result, err = clitestutil.ExecTestCLICmd(s.network.Validators[0].ClientCtx, cli.GetUnlockCmd(), append(unlockArgs, s.txArgs...))
	require.NoError(s.T(), err)
	testutilcli.RequireTxOk(s.T(), s.network.Validators[0].ClientCtx.Codec, result.Bytes())
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
