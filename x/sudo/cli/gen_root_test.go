package cli_test

import (
	"context"
	"testing"

	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/types/module"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltest "github.com/cosmos/cosmos-sdk/x/genutil/client/testutil"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/sudo/cli"
)

func TestAddSudoRootAccountCmd(t *testing.T) {
	tests := []struct {
		name    string
		account string

		expectErr bool
	}{
		{
			name:      "valid",
			account:   "nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl",
			expectErr: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			logger := log.NewNopLogger()

			home := t.TempDir()
			cfg, err := genutiltest.CreateDefaultTendermintConfig(home)
			require.NoError(t, err)

			testModuleBasicManager := module.NewBasicManager(genutil.AppModuleBasic{})
			appCodec := moduletestutil.MakeTestEncodingConfig().Codec
			err = genutiltest.ExecInitCmd(testModuleBasicManager, home, appCodec)
			require.NoError(t, err)

			serverCtx := server.NewContext(viper.New(), cfg, logger)
			clientCtx := client.Context{}.WithCodec(appCodec).WithHomeDir(home)

			ctx := context.Background()
			ctx = context.WithValue(ctx, client.ClientContextKey, &clientCtx)
			ctx = context.WithValue(ctx, server.ServerContextKey, serverCtx)

			cmd := cli.AddSudoRootAccountCmd(home)
			cmd.SetArgs([]string{
				tc.account,
			})

			if tc.expectErr {
				require.Error(t, cmd.ExecuteContext(ctx))
			} else {
				require.NoError(t, cmd.ExecuteContext(ctx))
			}
		})
	}
}
