package cmd_test

import (
	"context"
	"testing"

	"github.com/NibiruChain/nibiru/app"

	"cosmossdk.io/log"
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

			appCodec := app.MakeEncodingConfig().Codec
			err = genutiltest.ExecInitCmd(
				testModuleBasicManager, home, appCodec)
			require.NoError(t, err)

			serverCtx := server.NewContext(viper.New(), cfg, logger)
			clientCtx := (client.Context{}.
				WithCodec(appCodec).
				WithHomeDir(home).
				WithInterfaceRegistry(app.MakeEncodingConfig().InterfaceRegistry))

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
		{
			name: "valid message",
			json_message: `
			{
				"stargate": {
					"type_url": "/cosmos.staking.v1beta1.MsgUndelegate",
					"value": "Cj9uaWJpMTdwOXJ6d25uZnhjanAzMnVuOXVnN3loaHpndGtodmw5amZrc3p0Z3c1dWg2OXdhYzJwZ3N5bjcwbmoSMm5pYml2YWxvcGVyMXdqNWtma25qa3BjNmpkMzByeHRtOHRweGZqZjd4cWx3eDM4YzdwGgwKBXVuaWJpEgMxMTE="
				},
				"another": {
					"type_url": "/cosmos.staking.v1beta1.MsgDelegate",
					"value": {"delegator_address":"cosmos1eckjje8r8s48kv0pndgtwvehveedlzlnnshl3e", "validator_address":"cosmos1n6ndsc04xh2hqf506nhvhcggj0qwguf8ks06jj", "amount":{"denom":"unibi","amount":"42"} }
				}
			}`,
			expectError: false,
		},
		{
			name: "valid message",
			json_message: `
			{
				"another": {
					"type_url": "/cosmos.staking.v1beta1.MsgDelegate",
					"value": "{\"delegator_address\":\"cosmos1eckjje8r8s48kv0pndgtwvehveedlzlnnshl3e\", \"validator_address\":\"cosmos1n6ndsc04xh2hqf506nhvhcggj0qwguf8ks06jj\", \"amount\":{\"denom\":\"unibi\",\"amount\":\"42\"} }"
				}
			}`,
			expectError: false,
		},
		{
			name: "empty message",
			json_message: `
			{

			}`,
			expectError: false,
		},
		{
			name: "invalid json",
			json_message: `

			}`,
			expectError: true,
		},
	}

	for _, testCase := range testCases {
		executeTest(t, testCase)
	}
}
