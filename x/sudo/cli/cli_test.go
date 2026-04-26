package cli_test

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/cosmos/gogoproto/jsonpb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/v2/x/sudo"

	"github.com/cosmos/cosmos-sdk/codec"
	sdktestutil "github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/x/nutil"
	"github.com/NibiruChain/nibiru/v2/x/nutil/set"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil/localnet"
	"github.com/NibiruChain/nibiru/v2/x/sudo/cli"
)

// ———————————————————————————————————————————————————————————————————
// MsgEditSudoersPlus
// ———————————————————————————————————————————————————————————————————

// MsgEditSudoersPlus is a wrapper struct to extend the default MsgEditSudoers
// type with convenience functions
type MsgEditSudoersPlus struct {
	sudo.MsgEditSudoers
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
	tempMsg := new(sudo.MsgEditSudoers)
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

var _ suite.TearDownAllSuite = (*TestSuite)(nil)

type TestSuite struct {
	suite.Suite

	localnetCLI localnet.CLI
	root        sdk.AccAddress
}

func TestSudoIntegration(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

// ———————————————————————————————————————————————————————————————————
// IntegrationSuite - Setup
// ———————————————————————————————————————————————————————————————————

func (s *TestSuite) SetupSuite() {
	if err := nutil.EnsureLocalBlockchain(); err != nil {
		s.T().Skipf("skipping localnet-backed sudo CLI tests: %v", err)
	}

	localnetCLI, err := localnet.NewCLI()
	s.Require().NoError(err)
	s.localnetCLI = localnetCLI

	// `contrib/scripts/localnet.sh` patches genesis so the recovered
	// `validator` account is the x/sudo root; `nutil.LocalnetValAddr` matches
	// that fixed localnet account.
	s.root = nutil.LocalnetValAddr

	state := s.querySudoers()
	s.Require().Equal(
		s.root.String(),
		state.Sudoers.Root,
		"localnet sudo root must be the validator account",
	)
}

// ———————————————————————————————————————————————————————————————————
// IntegrationSuite - Tests
// ———————————————————————————————————————————————————————————————————

func (s *TestSuite) TestCmdEditSudoers() {
	initialState := s.querySudoers()
	initialContracts := set.New(initialState.Sudoers.Contracts...)
	contracts := s.newContracts(3, initialContracts)
	sender := s.root

	pbMsg := sudo.MsgEditSudoers{
		Action:    "add_contracts",
		Contracts: contracts,
		Sender:    sender.String(),
	}

	msg := MsgEditSudoersPlus{pbMsg}
	_, fileName := msg.ToJson(s.T())

	s.T().Log("happy - add_contracts exec tx")
	s.Require().NoError(s.execLocalTx("edit-sudoers", fileName))

	state := s.querySudoers()
	gotRoot := state.Sudoers.Root
	s.Equal(s.root.String(), gotRoot)

	gotContracts := set.New(state.Sudoers.Contracts...)
	for _, contract := range contracts {
		s.True(gotContracts.Has(contract))
	}

	pbMsg = sudo.MsgEditSudoers{
		Action:    "remove_contracts",
		Contracts: contracts,
		Sender:    sender.String(),
	}

	msg = MsgEditSudoersPlus{pbMsg}
	_, fileName = msg.ToJson(s.T())

	s.T().Log("happy - remove_contracts exec tx")
	s.Require().NoError(s.execLocalTx("edit-sudoers", fileName))

	state = s.querySudoers()
	gotRoot = state.Sudoers.Root
	s.Equal(s.root.String(), gotRoot)
	gotContracts = set.New(state.Sudoers.Contracts...)
	s.Require().Equal(initialContracts.Len(), gotContracts.Len())
	for _, contract := range initialState.Sudoers.Contracts {
		s.True(gotContracts.Has(contract))
	}
}

// TestMarshal_EditSudoers verifies that the expected proto.Message for
// the EditSudoders fn marshals and unmarshals properly from JSON.
// This unmarshaling is used in the main body of the CmdEditSudoers command.
func (s *Suite) TestMarshal_EditSudoers() {
	t := s.T()

	t.Log("create valid example json for the message")
	_, addrs := testutil.PrivKeyAddressPairs(4)
	var contracts []string
	sender := addrs[0]
	for _, addr := range addrs[1:] {
		contracts = append(contracts, addr.String())
	}
	msg := sudo.MsgEditSudoers{
		Action:    "add_contracts",
		Contracts: contracts,
		Sender:    sender.String(),
	}
	require.NoError(t, msg.ValidateBasic())

	msgPlus := MsgEditSudoersPlus{msg}
	fileJsonBz, _ := msgPlus.ToJson(t)

	t.Log("check unmarshal file → proto")
	cdc := app.MakeEncodingConfig().Codec
	newMsg := new(sudo.MsgEditSudoers)
	err := cdc.UnmarshalJSON(fileJsonBz, newMsg)
	assert.NoErrorf(t, err, "fileJsonBz: #%v", fileJsonBz)
	require.NoError(t, newMsg.ValidateBasic(), newMsg.String())
}

func (s *TestSuite) TearDownSuite() {
	s.T().Log("leaving localnet state in place")
}

func (s *TestSuite) querySudoers() *sudo.QuerySudoersResponse {
	resp := new(sudo.QuerySudoersResponse)
	s.Require().NoError(s.execLocalQuery(resp, "state"))
	return resp
}

func (s *TestSuite) execLocalQuery(
	result codec.ProtoMarshaler,
	args ...string,
) error {
	cmd := cli.GetQueryCmd()
	s.T().Log(s.localnetCLI.RenderQueryCmd(cmd, args))
	return s.localnetCLI.ExecQueryCmd(cmd, args, result)
}

func (s *TestSuite) execLocalTx(args ...string) error {
	cmd := cli.GetTxCmd()
	s.T().Log(s.localnetCLI.RenderTxCmd(cmd, args))
	_, err := s.localnetCLI.ExecTxCmd(cmd, args)
	return err
}

func (s *TestSuite) newContracts(
	n int,
	existing set.Set[string],
) []string {
	contracts := make([]string, 0, n)
	for len(contracts) < n {
		addr := testutil.AccAddress().String()
		if existing.Has(addr) {
			continue
		}
		alreadySelected := false
		for _, contract := range contracts {
			if contract == addr {
				alreadySelected = true
				break
			}
		}
		if alreadySelected {
			continue
		}
		contracts = append(contracts, addr)
	}
	return contracts
}

// ———————————————————————————————————————————————————————————————————
// CLI Tests using TestVars helper
// ———————————————————————————————————————————————————————————————————

func (s *Suite) TestCliCmdEditSudoers() {
	testVars := SetupTestVars(s.T())
	tempDir := s.T().TempDir()

	// Create temporary JSON files for testing
	addContractsJSON := s.createTempJSONFile(tempDir, map[string]any{
		"action":    "add_contracts",
		"contracts": []string{testutil.AccAddress().String(), testutil.AccAddress().String()},
	})
	removeContractsJSON := s.createTempJSONFile(tempDir, map[string]any{
		"action":    "remove_contracts",
		"contracts": []string{testutil.AccAddress().String()},
	})
	invalidActionJSON := s.createTempJSONFile(tempDir, map[string]any{
		"action":    "invalid_action",
		"contracts": []string{testutil.AccAddress().String()},
	})
	invalidAddressJSON := s.createTempJSONFile(tempDir, map[string]any{
		"action":    "add_contracts",
		"contracts": []string{"invalid-address"},
	})

	testCases := []TestCase{
		{
			name: "happy: edit sudoers add contracts",
			args: []string{
				"edit-sudoers",
				addContractsJSON,
			},
			extraArgs: []string{fmt.Sprintf("--from=%s", testVars.TestAcc.Address)},
			wantErr:   "",
		},
		{
			name: "happy: edit sudoers remove contracts",
			args: []string{
				"edit-sudoers",
				removeContractsJSON,
			},
			extraArgs: []string{fmt.Sprintf("--from=%s", testVars.TestAcc.Address)},
			wantErr:   "",
		},
		{
			name: "sad: invalid action type",
			args: []string{
				"edit-sudoers",
				invalidActionJSON,
			},
			extraArgs: []string{fmt.Sprintf("--from=%s", testVars.TestAcc.Address)},
			wantErr:   "invalid action type",
		},
		{
			name: "sad: invalid contract address",
			args: []string{
				"edit-sudoers",
				invalidAddressJSON,
			},
			extraArgs: []string{fmt.Sprintf("--from=%s", testVars.TestAcc.Address)},
			wantErr:   "decoding bech32 failed",
		},
		{
			name: "sad: file not found",
			args: []string{
				"edit-sudoers",
				"nonexistent.json",
			},
			extraArgs: []string{fmt.Sprintf("--from=%s", testVars.TestAcc.Address)},
			wantErr:   "no such file or directory",
		},
		{
			name: "sad: missing args",
			args: []string{
				"edit-sudoers",
			},
			extraArgs: []string{fmt.Sprintf("--from=%s", testVars.TestAcc.Address)},
			wantErr:   "accepts 1 arg(s), received 0",
		},
		{
			name: "sad: too many args",
			args: []string{
				"edit-sudoers",
				addContractsJSON,
				"extra-arg",
			},
			extraArgs: []string{fmt.Sprintf("--from=%s", testVars.TestAcc.Address)},
			wantErr:   "accepts 1 arg(s), received 2",
		},
	}

	for _, tc := range testCases {
		tc.RunTxCmd(s, testVars)
	}
}

func (s *Suite) TestCliCmdChangeRoot() {
	testVars := SetupTestVars(s.T())
	newRootAddr := testutil.AccAddress().String()

	testCases := []TestCase{
		{
			name: "happy: change root",
			args: []string{
				"change-root",
				newRootAddr,
			},
			extraArgs: []string{fmt.Sprintf("--from=%s", testVars.TestAcc.Address)},
			wantErr:   "",
		},
		{
			name: "sad: invalid new root address",
			args: []string{
				"change-root",
				"invalid-address",
			},
			extraArgs: []string{fmt.Sprintf("--from=%s", testVars.TestAcc.Address)},
			wantErr:   "decoding bech32 failed",
		},
		{
			name: "sad: missing args",
			args: []string{
				"change-root",
			},
			extraArgs: []string{fmt.Sprintf("--from=%s", testVars.TestAcc.Address)},
			wantErr:   "accepts 1 arg(s), received 0",
		},
		{
			name: "sad: too many args",
			args: []string{
				"change-root",
				newRootAddr,
				"extra-arg",
			},
			extraArgs: []string{fmt.Sprintf("--from=%s", testVars.TestAcc.Address)},
			wantErr:   "accepts 1 arg(s), received 2",
		},
	}

	for _, tc := range testCases {
		tc.RunTxCmd(s, testVars)
	}
}

func (s *Suite) TestCliCmdEditZeroGasActors() {
	testVars := SetupTestVars(s.T())

	// Generate test addresses
	validSender1 := testutil.AccAddress().String()
	validSender2 := testutil.AccAddress().String()
	validContract1 := "0x1234567890123456789012345678901234567890"
	validContract2 := "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd"

	// Happy path: Valid JSON with both senders and contracts
	validJSON := fmt.Sprintf(`{"senders":["%s","%s"],"contracts":["%s","%s"]}`,
		validSender1, validSender2, validContract1, validContract2)

	// Happy path: Valid JSON including always_zero_gas_contracts
	validJSONWithAlwaysZeroGas := fmt.Sprintf(`{"senders":[],"contracts":[],"always_zero_gas_contracts":["%s","%s"]}`,
		validContract1, validContract2)

	// Sad path: Not JSON string
	notJSON := "this is not a json string"

	// Sad path: Valid JSON but wrong structure (array instead of object)
	wrongStructureJSON := `["sender1", "sender2"]`

	// Sad path: Valid JSON structure but invalid sender address
	invalidSenderJSON := fmt.Sprintf(`{"senders":["invalid-address"],"contracts":["%s"]}`, validContract1)

	testCases := []TestCase{
		{
			name: "happy: edit zero gas actors with valid JSON",
			args: []string{
				"edit-zero-gas",
				validJSON,
			},
			extraArgs: []string{fmt.Sprintf("--from=%s", testVars.TestAcc.Address)},
			wantErr:   "",
		},
		{
			name: "happy: edit zero gas actors with always_zero_gas_contracts",
			args: []string{
				"edit-zero-gas",
				validJSONWithAlwaysZeroGas,
			},
			extraArgs: []string{fmt.Sprintf("--from=%s", testVars.TestAcc.Address)},
			wantErr:   "",
		},
		{
			name: "sad: not a JSON string",
			args: []string{
				"edit-zero-gas",
				notJSON,
			},
			extraArgs: []string{fmt.Sprintf("--from=%s", testVars.TestAcc.Address)},
			wantErr:   "is not a JSON string",
		},
		{
			name: "sad: JSON with wrong structure",
			args: []string{
				"edit-zero-gas",
				wrongStructureJSON,
			},
			extraArgs: []string{fmt.Sprintf("--from=%s", testVars.TestAcc.Address)},
			wantErr:   "failed to unpack actors json string",
		},
		{
			name: "sad: JSON with invalid sender address",
			args: []string{
				"edit-zero-gas",
				invalidSenderJSON,
			},
			extraArgs: []string{fmt.Sprintf("--from=%s", testVars.TestAcc.Address)},
			wantErr:   "ZeroGasActors stateless validation error",
		},
	}

	for _, tc := range testCases {
		tc.RunTxCmd(s, testVars)
	}
}

func (s *Suite) TestCliCmdQuerySudoers() {
	testVars := SetupTestVars(s.T())

	testCases := []TestCase{
		{
			name: "happy: query sudoers state",
			args: []string{
				"state",
			},
			wantErr: "",
		},
		{
			name: "sad: too many args",
			args: []string{
				"state",
				"extra-arg",
			},
			wantErr: `unknown command "extra-arg"`,
		},
	}

	for _, tc := range testCases {
		tc.RunQueryCmd(s, testVars)
	}
}

// Helper function to create temporary JSON files for testing
func (s *Suite) createTempJSONFile(tempDir string, data any) string {
	jsonData, err := json.Marshal(data)
	s.Require().NoError(err)

	// Create a temporary file
	tmpFile, err := os.CreateTemp(tempDir, "sudo_test_*.json")
	s.Require().NoError(err)
	defer tmpFile.Close()

	// Write JSON data to the file
	_, err = tmpFile.Write(jsonData)
	s.Require().NoError(err)

	// Return the file path
	return tmpFile.Name()
}
