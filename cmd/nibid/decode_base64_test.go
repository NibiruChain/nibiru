package main_test

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/NibiruChain/nibiru/v2/app"

	"github.com/cometbft/cometbft/libs/log"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/client"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/server"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/module"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/genutil"
	genutiltest "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/genutil/client/testutil"

	nibid "github.com/NibiruChain/nibiru/v2/cmd/nibid"
)

func TestBase64Decode(t *testing.T) {
	type TestCase struct {
		name        string
		jsonMessage string
		expectError bool
		assertOut   func(t *testing.T, out string)
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
				module.NewBasicManager(genutil.AppModuleBasic{}), home, appCodec)
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
			var stdout bytes.Buffer
			cmd.SetOut(&stdout)
			cmd.SetArgs([]string{
				tc.jsonMessage,
			})

			if tc.expectError {
				require.Error(t, cmd.ExecuteContext(ctx))
			} else {
				require.NoError(t, cmd.ExecuteContext(ctx))
				if tc.assertOut != nil {
					tc.assertOut(t, stdout.String())
				}
			}
		})
	}

	testCases := []TestCase{
		{
			name: "valid message",
			jsonMessage: `
			{
				"stargate": {
				  "type_url": "/cosmos.staking.v1beta1.MsgUndelegate",
				  "value": "Cj9uaWJpMTdwOXJ6d25uZnhjanAzMnVuOXVnN3loaHpndGtodmw5amZrc3p0Z3c1dWg2OXdhYzJwZ3N5bjcwbmoSMm5pYml2YWxvcGVyMXdqNWtma25qa3BjNmpkMzByeHRtOHRweGZqZjd4cWx3eDM4YzdwGgwKBXVuaWJpEgMxMTE="
				}
			  }`,
			assertOut: func(t *testing.T, out string) {
				var decoded []map[string]any
				require.NoError(t, json.Unmarshal([]byte(out), &decoded))
				require.Len(t, decoded, 1)
				require.Equal(t,
					"/cosmos.staking.v1beta1.MsgUndelegate",
					decoded[0]["type_url"],
				)

				value, ok := decoded[0]["value"].(map[string]any)
				require.True(t, ok)
				require.Equal(t,
					"nibi17p9rzwnnfxcjp32un9ug7yhhzgtkhvl9jfksztgw5uh69wac2pgsyn70nj",
					value["delegator_address"],
				)
				require.Equal(t,
					map[string]any{"amount": "111", "denom": "unibi"},
					value["amount"],
				)
			},
			expectError: false,
		},
		{
			name: "valid message",
			jsonMessage: `
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
			jsonMessage: `
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
			jsonMessage: `
			{

			}`,
			expectError: false,
		},
		{
			name: "invalid json",
			jsonMessage: `

			}`,
			expectError: true,
		},
	}

	for _, testCase := range testCases {
		executeTest(t, testCase)
	}
}
