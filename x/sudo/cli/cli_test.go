package cli_test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/cosmos/gogoproto/jsonpb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/set"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	testutilcli "github.com/NibiruChain/nibiru/x/common/testutil/cli"
	"github.com/NibiruChain/nibiru/x/common/testutil/genesis"
	"github.com/NibiruChain/nibiru/x/sudo/cli"
	"github.com/NibiruChain/nibiru/x/sudo/pb"

	"github.com/cosmos/cosmos-sdk/crypto"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdktestutil "github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ———————————————————————————————————————————————————————————————————
// MsgEditSudoersPlus
// ———————————————————————————————————————————————————————————————————

// MsgEditSudoersPlus is a wrapper struct to extend the default MsgEditSudoers
// type with convenience functions
type MsgEditSudoersPlus struct {
	pb.MsgEditSudoers
}

// ToJson converts the message into a json string and saves it in a temporary
// file, returning the json bytes and file name if done successfully.
func (msg MsgEditSudoersPlus) ToJson(t *testing.T) (fileJsonBz []byte, fileName string) {
	require.NoError(t, msg.ValidateBasic())

	// msgJsonStr showcases a valid example for the cmd args json file.
	msgJsonStr := fmt.Sprintf(`
	{
		"action": "%v",
		"contracts": ["%s"],
		"sender": "%v"
	}
	`, msg.Action, strings.Join(msg.Contracts, `", "`), msg.Sender)

	t.Log("check the unmarshal json → proto")
	tempMsg := new(pb.MsgEditSudoers)
	err := jsonpb.UnmarshalString(msgJsonStr, tempMsg)
	assert.NoErrorf(t, err, "DEBUG tempMsg: %v\njsonStr: %v", tempMsg, msgJsonStr)

	t.Log("save example json to a file")
	jsonFile := sdktestutil.WriteToNewTempFile(
		t, msgJsonStr,
	)

	fileName = jsonFile.Name()
	fileJsonBz, err = os.ReadFile(fileName)
	assert.NoError(t, err)
	return fileJsonBz, fileName
}

func (msg MsgEditSudoersPlus) Exec(
	network *testutilcli.Network,
	fileName string,
	from sdk.AccAddress,
) (*sdk.TxResponse, error) {
	args := []string{
		fileName,
	}
	return testutilcli.ExecTx(network, cli.CmdEditSudoers(), from, args)
}

type IntegrationSuite struct {
	suite.Suite
	cfg     testutilcli.Config
	network *testutilcli.Network
	root    Account
}

type Account struct {
	privKey    cryptotypes.PrivKey
	addr       sdk.AccAddress
	passphrase string
}

func TestSuite_IntegrationSuite_RunAll(t *testing.T) {
	suite.Run(t, new(IntegrationSuite))
}

// ———————————————————————————————————————————————————————————————————
// IntegrationSuite - Setup
// ———————————————————————————————————————————————————————————————————

func (s *IntegrationSuite) SetupSuite() {
	app.SetPrefixes(app.AccountAddressPrefix)

	genState := genesis.NewTestGenesisState(app.MakeEncodingConfig())
	genState, rootPrivKey, rootAddr := genesis.AddSudoGenesis(genState)
	s.root = Account{
		privKey:    rootPrivKey,
		addr:       rootAddr,
		passphrase: "secure-password",
	}
	homeDir := s.T().TempDir()
	s.cfg = testutilcli.BuildNetworkConfig(genState)
	network, err := testutilcli.New(s.T(), homeDir, s.cfg)
	s.Require().NoError(err)

	s.network = network
	s.FundRoot(s.root)
	s.AddRootToKeyring(s.root)
}

func (s *IntegrationSuite) FundRoot(root Account) {
	val := s.network.Validators[0]
	funds := sdk.NewCoins(
		sdk.NewInt64Coin(denoms.NIBI, 420*common.TO_MICRO),
	)
	feeDenom := denoms.NIBI
	s.NoError(testutilcli.FillWalletFromValidator(
		root.addr, funds, val, feeDenom,
	))
}

func (s *IntegrationSuite) AddRootToKeyring(root Account) {
	s.T().Log("add the x/sudo root account to the clientCtx.Keyring")
	// Encrypt the x/sudo root account's private key to get its "armor"
	passphrase := root.passphrase
	privKey := root.privKey
	armor := crypto.EncryptArmorPrivKey(privKey, passphrase, privKey.Type())
	// Import this account to the keyring
	val := s.network.Validators[0]
	s.NoError(
		val.ClientCtx.Keyring.ImportPrivKey("root", armor, passphrase),
	)
}

// ———————————————————————————————————————————————————————————————————
// IntegrationSuite - Tests
// ———————————————————————————————————————————————————————————————————

func (s *IntegrationSuite) TestCmdEditSudoers() {
	val := s.network.Validators[0]

	_, contractAddrs := testutil.PrivKeyAddressPairs(3)
	var contracts []string
	for _, addr := range contractAddrs {
		contracts = append(contracts, addr.String())
	}

	var sender sdk.AccAddress = s.root.addr

	pbMsg := pb.MsgEditSudoers{
		Action:    "add_contracts",
		Contracts: []string{contracts[0], contracts[1], contracts[2]},
		Sender:    sender.String(),
	}

	msg := MsgEditSudoersPlus{pbMsg}
	jsonBz, fileName := msg.ToJson(s.T())

	s.T().Log("sending from the wrong address should fail.")
	wrongSender := testutil.AccAddress()
	msg.Sender = wrongSender.String()
	out, err := msg.Exec(s.network, fileName, wrongSender)
	s.Assert().Errorf(err, "out: %s\n", out)
	s.Contains(err.Error(), "key not found", "msg: %s\nout: %s", jsonBz, out)

	s.T().Log("happy - add_contracts exec tx")
	msg.Sender = sender.String()
	out, err = msg.Exec(s.network, fileName, sender)
	s.NoErrorf(err, "msg: %s\nout: %s", jsonBz, out)

	state, err := testutilcli.QuerySudoers(val.ClientCtx)
	s.NoError(err)

	gotRoot := state.Sudoers.Root
	s.Equal(s.root.addr.String(), gotRoot)

	gotContracts := set.New(state.Sudoers.Contracts...)
	s.Equal(len(contracts), gotContracts.Len())
	for _, contract := range contracts {
		s.True(gotContracts.Has(contract))
	}

	pbMsg = pb.MsgEditSudoers{
		Action:    "remove_contracts",
		Contracts: []string{contracts[1]},
		Sender:    sender.String(),
	}

	msg = MsgEditSudoersPlus{pbMsg}
	jsonBz, fileName = msg.ToJson(s.T())

	s.T().Log("happy - remove_contracts exec tx")
	out, err = msg.Exec(s.network, fileName, sender)
	s.NoErrorf(err, "msg: %s\nout: %s", jsonBz, out)

	state, err = testutilcli.QuerySudoers(val.ClientCtx)
	s.NoError(err)

	gotRoot = state.Sudoers.Root
	s.Equal(s.root.addr.String(), gotRoot)

	wantContracts := []string{contracts[0], contracts[2]}
	gotContracts = set.New(state.Sudoers.Contracts...)
	s.Equal(len(wantContracts), gotContracts.Len())
	for _, contract := range wantContracts {
		s.True(gotContracts.Has(contract))
	}
}

// TestMarshal_EditSudoers verifies that the expected proto.Message for
// the EditSudoders fn marshals and unmarshals properly from JSON.
// This unmarshaling is used in the main body of the CmdEditSudoers command.
func (s *IntegrationSuite) TestMarshal_EditSudoers() {
	t := s.T()

	t.Log("create valid example json for the message")
	_, addrs := testutil.PrivKeyAddressPairs(4)
	var contracts []string
	sender := addrs[0]
	for _, addr := range addrs[1:] {
		contracts = append(contracts, addr.String())
	}
	msg := pb.MsgEditSudoers{
		Action:    "add_contracts",
		Contracts: contracts,
		Sender:    sender.String(),
	}
	require.NoError(t, msg.ValidateBasic())

	msgPlus := MsgEditSudoersPlus{msg}
	fileJsonBz, _ := msgPlus.ToJson(t)

	t.Log("check unmarshal file → proto")
	cdc := genesis.TEST_ENCODING_CONFIG.Marshaler
	newMsg := new(pb.MsgEditSudoers)
	err := cdc.UnmarshalJSON(fileJsonBz, newMsg)
	assert.NoErrorf(t, err, "fileJsonBz: #%v", fileJsonBz)
	require.NoError(t, newMsg.ValidateBasic(), newMsg.String())
}

func (s *IntegrationSuite) TearDownSuite() {
	s.T().Log("tearing down integration test suite")
	s.network.Cleanup()
}
