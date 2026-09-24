package impl

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/pelletier/go-toml/v2"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/client"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/client/flags"
	clitestutil "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/testutil/cli"
)

func TestConfigCmdCreatesDefaultQueryMode(t *testing.T) {
	home := t.TempDir()

	output, err := executeConfigCmd(testClientContext(home))
	require.NoError(t, err)

	var config clientConfigOutput
	require.NoError(t, json.Unmarshal([]byte(output), &config))
	require.Equal(t, QueryModeGQL, config.QueryMode)

	values := readClientConfigValues(t, home)
	require.Equal(t, string(QueryModeGQL), values[QueryModeConfigKey])
}

func TestConfigCmdRepairsPersistedQueryMode(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "legacy file without query mode",
			input: "node = \"tcp://localhost:26657\"\n",
		},
		{
			name:  "invalid query mode",
			input: "query-mode = \"other\"\n",
		},
		{
			name:  "non-string query mode",
			input: "query-mode = 1\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			writeClientConfigText(t, home, tt.input)

			output, err := executeConfigCmd(testClientContext(home))
			require.NoError(t, err)

			var config clientConfigOutput
			require.NoError(t, json.Unmarshal([]byte(output), &config))
			require.Equal(t, QueryModeGQL, config.QueryMode)

			values := readClientConfigValues(t, home)
			require.Equal(t, string(QueryModeGQL), values[QueryModeConfigKey])
		})
	}
}

func TestConfigCmdSetsAndGetsQueryMode(t *testing.T) {
	home := t.TempDir()
	ctx := testClientContext(home)

	_, err := executeConfigCmd(ctx, QueryModeConfigKey, string(QueryModeDirect))
	require.NoError(t, err)

	output, err := executeConfigCmd(testClientContext(home), QueryModeConfigKey)
	require.NoError(t, err)
	require.Equal(t, string(QueryModeDirect)+"\n", output)

	values := readClientConfigValues(t, home)
	require.Equal(t, string(QueryModeDirect), values[QueryModeConfigKey])
}

func TestConfigCmdRejectsInvalidQueryModeWithoutWriting(t *testing.T) {
	home := t.TempDir()
	_, err := executeConfigCmd(testClientContext(home))
	require.NoError(t, err)

	configPath := filepath.Join(home, "config", "client.toml")
	before, err := os.ReadFile(configPath)
	require.NoError(t, err)

	_, err = executeConfigCmd(testClientContext(home), QueryModeConfigKey, "other")
	require.EqualError(t, err, `invalid query mode "other": expected gql or direct`)

	after, err := os.ReadFile(configPath)
	require.NoError(t, err)
	require.Equal(t, before, after)
}

func TestConfigCmdKeepsMalformedTOMLAndUnknownKeysDistinct(t *testing.T) {
	t.Run("malformed TOML returns an error", func(t *testing.T) {
		home := t.TempDir()
		writeClientConfigText(t, home, "query-mode =\n")

		_, err := executeConfigCmd(testClientContext(home))
		require.ErrorContains(t, err, "couldn't parse client config")
	})

	t.Run("unknown command key returns an error", func(t *testing.T) {
		home := t.TempDir()
		_, err := executeConfigCmd(testClientContext(home), "unknown-key")
		require.EqualError(t, err, `unknown configuration key: "unknown-key"`)
	})
}

func TestConfigCmdPreservesUnknownValues(t *testing.T) {
	home := t.TempDir()
	writeClientConfigText(t, home, `
node = "tcp://localhost:26657"
query-mode = "direct"
custom-setting = "keep-me"
`)

	_, err := executeConfigCmd(testClientContext(home), flags.FlagNode, "tcp://localhost:26658")
	require.NoError(t, err)

	values := readClientConfigValues(t, home)
	require.Equal(t, "keep-me", values["custom-setting"])
	require.Equal(t, "tcp://localhost:26658", values[flags.FlagNode])
	require.Equal(t, string(QueryModeDirect), values[QueryModeConfigKey])
}

func TestLoadClientConfigUsesGQLWhenRepairFails(t *testing.T) {
	home := t.TempDir()
	writeClientConfigText(t, home, "query-mode = \"other\"\n")

	_, mode, _, err := loadClientConfigWithWriter(testClientContext(home), func(clientConfigFile) error {
		return errors.New("write failed")
	})
	require.NoError(t, err)
	require.Equal(t, QueryModeGQL, mode)

	values := readClientConfigValues(t, home)
	require.Equal(t, "other", values[QueryModeConfigKey])
}

func TestConfigCmdPreservesNodeEnvironmentOverride(t *testing.T) {
	home := t.TempDir()
	writeClientConfigText(t, home, "node = \"tcp://localhost:26657\"\n")

	ctx := testClientContext(home)
	require.NoError(t, ctx.Viper.BindEnv(flags.FlagNode))
	t.Setenv("NODE", "tcp://localhost:26658")

	output, err := executeConfigCmd(ctx, flags.FlagNode)
	require.NoError(t, err)
	require.Equal(t, "tcp://localhost:26658\n", output)
}

func TestQueryModeFromCmd(t *testing.T) {
	cmd := ConfigCmd()
	cmd.SetContext(context.Background())
	require.Equal(t, QueryModeGQL, QueryModeFromCmd(cmd))

	SetQueryMode(cmd, QueryModeDirect)
	require.Equal(t, QueryModeDirect, QueryModeFromCmd(cmd))
}

func executeConfigCmd(ctx client.Context, args ...string) (string, error) {
	output, err := clitestutil.ExecTestCLICmd(ctx, ConfigCmd(), args)
	return output.String(), err
}

func testClientContext(home string) client.Context {
	return client.Context{}.
		WithHomeDir(home).
		WithViper("")
}

func writeClientConfigText(t *testing.T, home, contents string) {
	t.Helper()
	configPath := filepath.Join(home, "config")
	require.NoError(t, os.MkdirAll(configPath, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(configPath, "client.toml"), []byte(contents), 0o600))
}

func readClientConfigValues(t *testing.T, home string) map[string]any {
	t.Helper()
	contents, err := os.ReadFile(filepath.Join(home, "config", "client.toml"))
	require.NoError(t, err)

	values := make(map[string]any)
	require.NoError(t, toml.Unmarshal(contents, &values))
	return values
}
