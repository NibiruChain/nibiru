package cli_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/simapp"
	sdktestutil "github.com/cosmos/cosmos-sdk/testutil"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltest "github.com/cosmos/cosmos-sdk/x/genutil/client/testutil"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/NibiruChain/nibiru/x/pricefeed/client/cli"
	"github.com/NibiruChain/nibiru/x/testutil"
)

var testModuleBasicManager = module.NewBasicManager(genutil.AppModuleBasic{})

// Tests "add-genesis-oracle", a command that adds a oracle address to genesis.json
func TestAddGenesisWhitelistOracleCmd(t *testing.T) {
	testCases := []struct {
		name        string
		oracles     string
		expectError bool
	}{
		{
			name:        "add single oracle",
			oracles:     testutil.AccAddress().String(),
			expectError: false,
		},
		{
			name:        "add multiple oracles",
			oracles:     fmt.Sprintf("%s,%s", testutil.AccAddress().String(), testutil.AccAddress().String()),
			expectError: false,
		},
		{
			name:        "repeated oracle addresses",
			oracles:     fmt.Sprintf("%[1]s,%[1]s", testutil.AccAddress().String()),
			expectError: true,
		},
		{
			name:        "empty oracle address",
			oracles:     "",
			expectError: true,
		},
		{
			name:        "invalid oracle address",
			oracles:     "test,test",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			logger := log.NewNopLogger()
			cfg, err := genutiltest.CreateDefaultTendermintConfig(home)
			require.NoError(t, err)

			appCodec := simapp.MakeTestEncodingConfig().Marshaler
			err = genutiltest.ExecInitCmd(testModuleBasicManager, home, appCodec)
			require.NoError(t, err)

			serverCtx := server.NewContext(viper.New(), cfg, logger)

			cmd := cli.AddWhitelistGenesisOracle(home)
			cmd.SetArgs([]string{tc.oracles})
			_, out := sdktestutil.ApplyMockIO(cmd)
			clientCtx := client.Context{}.WithCodec(appCodec).WithHomeDir(home).WithOutput(out)

			ctx := context.Background()
			ctx = context.WithValue(ctx, client.ClientContextKey, &clientCtx)
			ctx = context.WithValue(ctx, server.ServerContextKey, serverCtx)

			if tc.expectError {
				require.Error(t, cmd.ExecuteContext(ctx))
			} else {
				require.NoError(t, cmd.ExecuteContext(ctx))
			}
		})
	}
}
