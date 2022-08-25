package cli_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/simapp"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltest "github.com/cosmos/cosmos-sdk/x/genutil/client/testutil"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/NibiruChain/nibiru/x/vpool/client/cli"
)

var testModuleBasicManager = module.NewBasicManager(genutil.AppModuleBasic{})

// Tests "add-genesis-vpool", a command that adds a vpool to genesis.json
func TestAddGenesisVpoolCmd(t *testing.T) {
	tests := []struct {
		name          string
		pairName      string
		baseAsset     string
		quoteAsset    string
		tradeLimit    string
		flucLimit     string
		maxOracle     string
		maintainRatio string
		maxLeverage   string
		expectError   bool
	}{
		{
			name:          "pair name empty",
			pairName:      "",
			baseAsset:     "1",
			quoteAsset:    "1",
			tradeLimit:    "1",
			flucLimit:     "1",
			maxOracle:     "1",
			maintainRatio: "1",
			maxLeverage:   "1",
			expectError:   true,
		},
		{
			name:          "invalid pair name",
			pairName:      "token0:token1:token2",
			baseAsset:     "1",
			quoteAsset:    "1",
			tradeLimit:    "1",
			flucLimit:     "1",
			maxOracle:     "1",
			maintainRatio: "1",
			maxLeverage:   "1",
			expectError:   true,
		},
		{
			name:          "invalid trade limit input",
			pairName:      "token0:token1",
			baseAsset:     "1",
			quoteAsset:    "1",
			tradeLimit:    "test",
			flucLimit:     "1",
			maxOracle:     "1",
			maintainRatio: "1",
			maxLeverage:   "1",
			expectError:   true,
		},
		{
			name:          "empty base asset input",
			pairName:      "token0:token1",
			baseAsset:     "",
			quoteAsset:    "1",
			tradeLimit:    "1",
			flucLimit:     "1",
			maxOracle:     "1",
			maintainRatio: "1",
			maxLeverage:   "1",
			expectError:   true,
		},
		{
			name:          "max leverage cannot be zero",
			pairName:      "token0:token1",
			baseAsset:     "100",
			quoteAsset:    "100",
			tradeLimit:    "0.1",
			flucLimit:     "0.1",
			maxOracle:     "0.1",
			maintainRatio: "0.1",
			maxLeverage:   "0",
			expectError:   true,
		},
		{
			name:          "valid vpool pair",
			pairName:      "token0:token1",
			baseAsset:     "100",
			quoteAsset:    "100",
			tradeLimit:    "0.1",
			flucLimit:     "0.1",
			maxOracle:     "0.1",
			maintainRatio: "0.1",
			maxLeverage:   "10",
			expectError:   false,
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
			err = genutiltest.ExecInitCmd(
				testModuleBasicManager, home, appCodec)
			require.NoError(t, err)

			serverCtx := server.NewContext(viper.New(), cfg, logger)
			clientCtx := client.Context{}.WithCodec(appCodec).WithHomeDir(home)

			ctx := context.Background()
			ctx = context.WithValue(ctx, client.ClientContextKey, &clientCtx)
			ctx = context.WithValue(ctx, server.ServerContextKey, serverCtx)

			cmd := cli.AddVPoolGenesisCmd("home")
			cmd.SetArgs([]string{
				tc.pairName,
				tc.baseAsset,
				tc.quoteAsset,
				tc.tradeLimit,
				tc.flucLimit,
				tc.maxOracle,
				tc.maintainRatio,
				tc.maxLeverage,
				fmt.Sprintf("--%s=home", flags.FlagHome)})

			if tc.expectError {
				require.Error(t, cmd.ExecuteContext(ctx))
			} else {
				require.NoError(t, cmd.ExecuteContext(ctx))
			}
		})
	}
}
