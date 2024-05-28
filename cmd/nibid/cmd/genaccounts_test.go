package cmd_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/NibiruChain/nibiru/app"

	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltest "github.com/cosmos/cosmos-sdk/x/genutil/client/testutil"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common/testutil"

	nibid "github.com/NibiruChain/nibiru/cmd/nibid/cmd"
)

var testModuleBasicManager = module.NewBasicManager(genutil.AppModuleBasic{})

// Tests "add-genesis-account", a command that adds a genesis account to genesis.json
func TestAddGenesisAccountCmd(t *testing.T) {
	type TestCase struct {
		name        string
		addr        string
		denom       string
		expectError bool
	}

	executeTest := func(t *testing.T, testCase TestCase) {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			logger := log.NewNopLogger()
			cfg, err := genutiltest.CreateDefaultCometConfig(home)
			require.NoError(t, err)

			appCodec := app.MakeEncodingConfig().Codec
			err = genutiltest.ExecInitCmd(
				testModuleBasicManager, home, appCodec)
			require.NoError(t, err)

			serverCtx := server.NewContext(viper.New(), cfg, logger)
			clientCtx := client.Context{}.WithCodec(appCodec).WithHomeDir(home)

			ctx := context.Background()
			ctx = context.WithValue(ctx, client.ClientContextKey, &clientCtx)
			ctx = context.WithValue(ctx, server.ServerContextKey, serverCtx)

			cmd := nibid.AddGenesisAccountCmd(home)
			cmd.SetArgs([]string{
				tc.addr,
				tc.denom,
				fmt.Sprintf("--%s=home", flags.FlagHome),
			})

			if tc.expectError {
				require.Error(t, cmd.ExecuteContext(ctx))
			} else {
				require.NoError(t, cmd.ExecuteContext(ctx))
			}
		})
	}

	sampleAddr := testutil.AccAddress()

	testCases := []TestCase{
		{
			name:        "invalid address",
			addr:        "",
			denom:       "1000atom",
			expectError: true,
		},
		{
			name:        "valid address",
			addr:        sampleAddr.String(),
			denom:       "1000atom",
			expectError: false,
		},
		{
			name:        "multiple denoms",
			addr:        sampleAddr.String(),
			denom:       "1000atom, 2000stake",
			expectError: false,
		},
	}

	for _, testCase := range testCases {
		executeTest(t, testCase)
	}
}
