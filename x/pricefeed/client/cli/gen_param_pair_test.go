package cli_test

import (
	"context"
	"testing"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/simapp"
	"github.com/cosmos/cosmos-sdk/testutil"
	genutiltest "github.com/cosmos/cosmos-sdk/x/genutil/client/testutil"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/NibiruChain/nibiru/x/pricefeed/client/cli"
)

// Tests "add-genesis-pricefeed-pairs", a command that adds pairs to genesis.json
func TestAddPriceFeedParamPair(t *testing.T) {
	tests := []struct {
		name        string
		pairName    string
		expectError bool
	}{
		{
			name:        "pair name empty",
			pairName:    "",
			expectError: true,
		},
		{
			name:        "invalid pair name",
			pairName:    "token0:token1:token2:",
			expectError: true,
		},
		{
			name:        "token name absent",
			pairName:    "token0:",
			expectError: true,
		},
		{
			name:        "valid pricefeed pair",
			pairName:    "token0:token1",
			expectError: false,
		},
		{
			name:        "valid pricefeed pairs",
			pairName:    "token0:token1,token3:token4",
			expectError: false,
		},
	}

	for _, tc := range tests {
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

			cmd := cli.AddPriceFeedParamPairs(home)
			cmd.SetArgs([]string{tc.pairName})
			_, out := testutil.ApplyMockIO(cmd)
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
