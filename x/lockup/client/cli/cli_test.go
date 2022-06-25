package cli_test

import (
	"testing"
	"time"

	"github.com/NibiruChain/nibiru/x/lockup/types"

	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/lockup/client/cli"
	testutilcli "github.com/NibiruChain/nibiru/x/testutil/cli"
	"github.com/NibiruChain/nibiru/x/testutil/testapp"
)

type IntegrationTestSuite struct {
	suite.Suite

	cfg     testutilcli.Config
	network *testutilcli.Network

	userWithLock sdk.AccAddress
}

func (s *IntegrationTestSuite) SetupSuite() {
	if testing.Short() {
		s.T().Skip("skipping lockup CLI tests")
	}

	s.T().Log("setting up lockup CLI testing suite")

	app.SetPrefixes(app.AccountAddressPrefix)
	encCfg := app.MakeTestEncodingConfig()
	defaultAppGenesis := app.ModuleBasics.DefaultGenesis(encCfg.Marshaler)
	testAppGenesis := testapp.NewTestGenesisState(encCfg.Marshaler, defaultAppGenesis)
	s.cfg = testutilcli.BuildNetworkConfig(testAppGenesis)

	s.cfg.StartingTokens = sdk.NewCoins(
		sdk.NewInt64Coin("ATOM", 1_000_000),
		sdk.NewInt64Coin("OSMO", 1_000_000),
		sdk.NewInt64Coin("unibi", 1_000_000_000))

	s.network = testutilcli.NewNetwork(s.T(), s.cfg)
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

	_, err := testutilcli.ExecTx(s.network, cli.GetLockCoinsCmd(), s.userWithLock, lockup1Args)
	require.NoError(s.T(), err)

	// test query lock
	s.T().Log("testing query lock")
	queryLockArgs := []string{"0"}
	queryLockResponse := new(types.QueryLockResponse)
	require.NoError(s.T(), testutilcli.ExecQuery(s.network, cli.GetQueryLockCmd(), queryLockArgs, queryLockResponse))
	require.Equal(s.T(), 10*time.Second, queryLockResponse.Lock.Duration)
	require.Equal(s.T(), s.userWithLock.String(), queryLockResponse.Lock.Owner)
	require.Equal(s.T(), sdk.NewCoins(sdk.NewInt64Coin("ATOM", 1000)), queryLockResponse.Lock.Coins)

	// lock coins again
	lockup2Args := []string{
		"1000OSMO", // coins to lock
		"10s",      // duration
	}

	_, err = testutilcli.ExecTx(s.network, cli.GetLockCoinsCmd(), s.userWithLock, lockup2Args)
	require.NoError(s.T(), err)

	// test query multiple locks
	s.T().Log("testing query locks by address")
	queryLocksByAddressArgs := []string{s.userWithLock.String()}
	queryLocksByAddressResp := new(types.QueryLocksByAddressResponse)
	require.NoError(s.T(), testutilcli.ExecQuery(s.network, cli.GetQueryLocksByAddressCmd(), queryLocksByAddressArgs, queryLocksByAddressResp))
	require.Len(s.T(), queryLocksByAddressResp.Locks, 2)
	require.Equal(s.T(), sdk.NewCoins(sdk.NewInt64Coin("ATOM", 1000)), queryLocksByAddressResp.Locks[0].Coins)
	require.Equal(s.T(), sdk.NewCoins(sdk.NewInt64Coin("OSMO", 1000)), queryLocksByAddressResp.Locks[1].Coins)

	// test query locked coins
	s.T().Log("testing query locked coins")
	queryLockedCoinsArgs := []string{s.userWithLock.String()}
	queryLockedCoinsResp := new(types.QueryLockedCoinsResponse)
	require.NoError(s.T(), testutilcli.ExecQuery(s.network, cli.GetQueryLockedCoinsCmd(), queryLockedCoinsArgs, queryLockedCoinsResp))
	require.Equal(s.T(), sdk.NewCoins(sdk.NewInt64Coin("ATOM", 1000), sdk.NewInt64Coin("OSMO", 1000)), queryLockedCoinsResp.LockedCoins)

	// test initiate unlock
	s.T().Log("testing initiate unlock")
	initiateUnlockArgs := []string{"0"}
	_, err = testutilcli.ExecTx(s.network, cli.GetInitiateUnlockCmd(), s.userWithLock, initiateUnlockArgs)
	require.NoError(s.T(), err)
	// wait some seconds then unlock funds
	s.T().Logf("testing unlock funds")
	require.NoError(s.T(), s.network.WaitForDuration(10*time.Second))

	unlockArgs := []string{"0"}
	_, err = testutilcli.ExecTx(s.network, cli.GetUnlockCmd(), s.userWithLock, unlockArgs)
	require.NoError(s.T(), err)
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
