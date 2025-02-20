// Alteration of [network/network_test.go](https://github.com/cosmos/cosmos-sdk/blob/v0.45.15/testutil/network/network_test.go)
package cli_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	sdkcodec "github.com/cosmos/cosmos-sdk/codec"
	sdktestutil "github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/app/codec"

	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/NibiruChain/nibiru/x/common/testutil/cli"
	"github.com/NibiruChain/nibiru/x/common/testutil/genesis"
)

func TestIntegrationTestSuite_RunAll(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

// Assert network cleanup
var _ suite.TearDownAllSuite = (*TestSuite)(nil)

type TestSuite struct {
	suite.Suite

	network *cli.Network
	cfg     *cli.Config
}

func (s *TestSuite) SetupSuite() {
	testutil.BeforeIntegrationSuite(s.T())

	encConfig := app.MakeEncodingConfig()
	cfg := new(cli.Config)
	*cfg = cli.BuildNetworkConfig(genesis.NewTestGenesisState(encConfig))
	network, err := cli.New(
		s.T(),
		s.T().TempDir(),
		*cfg,
	)
	s.Require().NoError(err)
	s.network = network

	cfg.AbsorbListenAddresses(network.Validators[0])
	s.cfg = cfg

	_, err = s.network.WaitForHeight(1)
	s.Require().NoError(err)
}

func (s *TestSuite) TearDownSuite() {
	s.T().Log("tearing down integration test suite")
	s.network.Cleanup()
}

func (s *TestSuite) TestNetwork_Liveness() {
	height, err := s.network.WaitForHeightWithTimeout(4, time.Minute)
	s.Require().NoError(err, "expected to reach 4 blocks; got %d", height)

	err = s.network.WaitForDuration(1 * time.Second)
	s.NoError(err)
}

func (s *TestSuite) TestNetwork_LatestHeight() {
	height, err := s.network.LatestHeight()
	s.NoError(err)
	s.Positive(height)

	sadNetwork := new(cli.Network)
	_, err = sadNetwork.LatestHeight()
	s.Error(err)
}

func (s *TestSuite) TestLogMnemonic() {
	kring, algo, nodeDirName := cli.NewKeyring(s.T())

	var cdc sdkcodec.Codec = codec.MakeEncodingConfig().Codec
	_, mnemonic, err := sdktestutil.GenerateCoinKey(algo, cdc)
	s.NoError(err)

	overwrite := true
	_, secret, err := sdktestutil.GenerateSaveCoinKey(
		kring, nodeDirName, mnemonic, overwrite, algo,
	)
	s.NoError(err)

	cli.LogMnemonic(&mockLogger{
		Logs: []string{},
	}, secret)
}

func (s *TestSuite) TestValidatorGetSecret() {
	val := s.network.Validators[0]
	secret := val.SecretMnemonic()
	secretSlice := val.SecretMnemonicSlice()
	s.Equal(secret, strings.Join(secretSlice, " "))

	kring, algo, nodeDirName := cli.NewKeyring(s.T())
	mnemonic := secret
	overwrite := true
	addrGenerated, secretGenerated, err := sdktestutil.GenerateSaveCoinKey(
		kring, nodeDirName, mnemonic, overwrite, algo,
	)
	s.NoError(err)
	s.Equal(secret, secretGenerated)
	s.Equal(val.Address, addrGenerated)
}

var _ cli.Logger = (*mockLogger)(nil)

type mockLogger struct {
	Logs []string
}

func (ml *mockLogger) Log(args ...interface{}) {
	ml.Logs = append(ml.Logs, fmt.Sprint(args...))
}

func (ml *mockLogger) Logf(format string, args ...interface{}) {
	ml.Logs = append(ml.Logs, fmt.Sprintf(format, args...))
}

func (s *TestSuite) TestNewAccount() {
	s.NotPanics(func() {
		addr := cli.NewAccount(s.network, "newacc")
		s.NoError(sdk.VerifyAddressFormat(addr))
	})
}
