package cmd_test

import (
	"context"
	"testing"

	"github.com/NibiruChain/nibiru/app"

	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/server"
	genutiltest "github.com/cosmos/cosmos-sdk/x/genutil/client/testutil"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	nibid "github.com/NibiruChain/nibiru/cmd/nibid/cmd"
)

func TestBase64Decode(t *testing.T) {
	type TestCase struct {
		name         string
		json_message string
		expectError  bool
	}

	executeTest := func(t *testing.T, testCase TestCase) {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			logger := log.NewNopLogger()
			cfg, err := genutiltest.CreateDefaultTendermintConfig(home)
			require.NoError(t, err)

			appCodec := app.MakeEncodingConfig().Marshaler
			err = genutiltest.ExecInitCmd(
				testModuleBasicManager, home, appCodec)
			require.NoError(t, err)

			serverCtx := server.NewContext(viper.New(), cfg, logger)
			clientCtx := client.Context{}.WithCodec(appCodec).WithHomeDir(home)

			ctx := context.Background()
			ctx = context.WithValue(ctx, client.ClientContextKey, &clientCtx)
			ctx = context.WithValue(ctx, server.ServerContextKey, serverCtx)

			cmd := nibid.DecodeBase64Cmd(home)
			cmd.SetArgs([]string{
				tc.json_message,
			})

			if tc.expectError {
				require.Error(t, cmd.ExecuteContext(ctx))
			} else {
				require.NoError(t, cmd.ExecuteContext(ctx))
			}
		})
	}

	testCases := []TestCase{
		// {
		// 	name:         "empty message",
		// 	json_message: "",
		// 	expectError:  true,
		// },
		{
			name: "valid message",
			json_message: `
			{
				"stargate": {
				  "type_url": "/cosmos.staking.v1beta1.MsgUndelegate",
				  "value": "Cj9uaWJpMTdwOXJ6d25uZnhjanAzMnVuOXVnN3loaHpndGtodmw5amZrc3p0Z3c1dWg2OXdhYzJwZ3N5bjcwbmoSMm5pYml2YWxvcGVyMXdqNWtma25qa3BjNmpkMzByeHRtOHRweGZqZjd4cWx3eDM4YzdwGgwKBXVuaWJpEgMxMTE="
				}
			  }`,
			expectError: false,
		},
	}

	for _, testCase := range testCases {
		executeTest(t, testCase)
	}
}
