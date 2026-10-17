package cli_test

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	rpcclientmock "github.com/cometbft/cometbft/rpc/client/mock"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/client"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/client/flags"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/crypto/keyring"
	svrcmd "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/server/cmd"
	clitestutil "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/testutil/cli"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	testutilmod "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/module/testutil"
	govtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/upgrade"
	upgradecli "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/upgrade/client/cli"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/upgrade/plan"
	upgradetypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/upgrade/types"
)

func TestSubmitLegacyUpgradeDockerMetadata(t *testing.T) {
	binary := []byte("binary fixture")
	path := filepath.Join(t.TempDir(), "nibid")
	require.NoError(t, os.WriteFile(path, binary, 0o600))
	url := fmt.Sprintf("file://%s?checksum=sha256:%x", path, sha256.Sum256(binary))
	encCfg := testutilmod.MakeTestEncodingConfig(upgrade.AppModuleBasic{})
	govtypes.RegisterInterfaces(encCfg.InterfaceRegistry)
	clientCtx := client.Context{}.
		WithTxConfig(encCfg.TxConfig).
		WithCodec(encCfg.Codec).
		WithFrom(sdk.AccAddress(make([]byte, 20)).String()).
		WithFromAddress(sdk.AccAddress(make([]byte, 20))).
		WithGenerateOnly(true).
		WithChainID("test-chain")

	for _, tc := range []struct {
		name       string
		image      string
		binaryURL  string
		noValidate bool
		err        string
	}{
		{"valid Docker", "registry.invalid/nibiru:2.21.0", url, false, ""},
		{"archive only", "", url, false, ""},
		{"invalid Docker", "https://ghcr.io/nibiruchain/nibiru:2.21.0", url, false, "invalid docker image reference"},
		{"missing archive checksum", "ghcr.io/nibiruchain/nibiru:2.21.0", "file://" + path, false, "missing checksum"},
		{"explicit bypass", "invalid image", "file://" + path, true, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			info := plan.Info{Binaries: plan.BinaryDownloadURLMap{"any": tc.binaryURL}}
			if tc.image != "" {
				info.Binaries["docker"] = tc.image
			}
			infoJSON, err := json.Marshal(info)
			require.NoError(t, err)
			cmd := upgradecli.NewCmdSubmitLegacyUpgradeProposal()
			flags.AddTxFlagsToCmd(cmd)
			args := []string{
				"test-upgrade", "--upgrade-height=100", "--title=Upgrade", "--description=Upgrade",
				"--daemon-name=nibid", "--deposit=1stake", "--upgrade-info=" + string(infoJSON),
			}
			if tc.noValidate {
				args = append(args, "--no-validate")
			}
			out, err := clitestutil.ExecTestCLICmd(clientCtx, cmd, args)
			if tc.err != "" {
				require.ErrorContains(t, err, tc.err)
				return
			}
			require.NoError(t, err)
			tx, err := encCfg.TxConfig.TxJSONDecoder()(out.Bytes())
			require.NoError(t, err)
			require.Len(t, tx.GetMsgs(), 1)
			proposal := tx.GetMsgs()[0].(*govtypes.MsgSubmitProposal).GetContent().(*upgradetypes.SoftwareUpgradeProposal)
			require.Equal(t, string(infoJSON), proposal.Plan.Info)
			require.Equal(t, "test-upgrade", proposal.Plan.Name)
		})
	}
}

func TestModuleVersionsCLI(t *testing.T) {
	cmd := upgradecli.GetModuleVersionsCmd()
	cmd.SetOut(io.Discard)
	require.NotNil(t, cmd)

	encCfg := testutilmod.MakeTestEncodingConfig(upgrade.AppModuleBasic{})
	kr := keyring.NewInMemory(encCfg.Codec)
	baseCtx := client.Context{}.
		WithKeyring(kr).
		WithTxConfig(encCfg.TxConfig).
		WithCodec(encCfg.Codec).
		WithClient(clitestutil.MockTendermintRPC{Client: rpcclientmock.Client{}}).
		WithAccountRetriever(client.MockAccountRetriever{}).
		WithOutput(io.Discard).
		WithChainID("test-chain")

	testCases := []struct {
		msg          string
		args         []string
		expCmdOuptut string
	}{
		{
			msg:          "test full query with json output",
			args:         []string{fmt.Sprintf("--%s=1", flags.FlagHeight), fmt.Sprintf("--%s=json", flags.FlagOutput)},
			expCmdOuptut: `--height=1 --output=json`,
		},
		{
			msg:          "test full query with text output",
			args:         []string{fmt.Sprintf("--%s=1", flags.FlagHeight), fmt.Sprintf("--%s=text", flags.FlagOutput)},
			expCmdOuptut: `--height=1 --output=text`,
		},
		{
			msg:          "test single module",
			args:         []string{"bank", fmt.Sprintf("--%s=1", flags.FlagHeight)},
			expCmdOuptut: `bank --height=1`,
		},
		{
			msg:          "test non-existent module",
			args:         []string{"abcdefg", fmt.Sprintf("--%s=1", flags.FlagHeight)},
			expCmdOuptut: `abcdefg --height=1`,
		},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(tc.msg, func(t *testing.T) {
			ctx := svrcmd.CreateExecuteContext(context.Background())

			cmd.SetOut(io.Discard)
			require.NotNil(t, cmd)

			cmd.SetContext(ctx)
			cmd.SetArgs(tc.args)

			require.NoError(t, client.SetCmdClientContextHandler(baseCtx, cmd))

			if len(tc.args) != 0 {
				require.Contains(t, fmt.Sprint(cmd), tc.expCmdOuptut)
			}
		})
	}
}
