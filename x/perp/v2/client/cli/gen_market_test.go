package cli_test

import (
	"context"
	"fmt"
	"testing"

	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"

	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltest "github.com/cosmos/cosmos-sdk/x/genutil/client/testutil"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/perp/v2/client/cli"
)

var testModuleBasicManager = module.NewBasicManager(genutil.AppModuleBasic{})

// setupServer configures the client context and sets up a context at which we
// can execute genesis CLI commands.
func setupServer(t *testing.T) context.Context {
	home := t.TempDir()
	logger := log.NewNopLogger()
	cfg, err := genutiltest.CreateDefaultTendermintConfig(home)
	require.NoError(t, err)

	appCodec := moduletestutil.MakeTestEncodingConfig().Codec
	err = genutiltest.ExecInitCmd(
		testModuleBasicManager, home, appCodec)
	require.NoError(t, err)

	serverCtx := server.NewContext(viper.New(), cfg, logger)
	clientCtx := client.Context{}.WithCodec(appCodec).WithHomeDir(home)

	ctx := context.Background()
	ctx = context.WithValue(ctx, client.ClientContextKey, &clientCtx)
	ctx = context.WithValue(ctx, server.ServerContextKey, serverCtx)
	return ctx
}

// Tests "add-genesis-perp-market", a command that adds a market to genesis.json
func TestAddMarketGenesisCmd(t *testing.T) {
	tests := []struct {
		name            string
		pairName        string
		sqrtDepth       string
		priceMultiplier string
		maintainRatio   string
		maxLeverage     string
		expectError     bool
	}{
		{
			name:            "pair name empty",
			pairName:        "",
			sqrtDepth:       "1",
			priceMultiplier: "1",
			maintainRatio:   "1",
			maxLeverage:     "1",
			expectError:     true,
		},
		{
			name:            "invalid pair name",
			pairName:        "token0:token1:token2",
			sqrtDepth:       "1",
			priceMultiplier: "1",
			maintainRatio:   "1",
			maxLeverage:     "1",
			expectError:     true,
		},
		{
			name:            "empty sqrt depth input",
			pairName:        "token0:token1",
			sqrtDepth:       "",
			priceMultiplier: "1",
			maintainRatio:   "1",
			maxLeverage:     "1",
			expectError:     true,
		},
		{
			name:            "max leverage cannot be zero",
			pairName:        "token0:token1",
			sqrtDepth:       "100",
			priceMultiplier: "1",
			maintainRatio:   "0.1",
			maxLeverage:     "0",
			expectError:     true,
		},
		{
			name:            "price multiplier cannot be zero",
			pairName:        "token0:token1",
			sqrtDepth:       "100",
			priceMultiplier: "0",
			maintainRatio:   "0.1",
			maxLeverage:     "1",
			expectError:     true,
		},
		{
			name:            "price multiplier cannot be negative",
			pairName:        "token0:token1",
			sqrtDepth:       "100",
			priceMultiplier: "-1",
			maintainRatio:   "0.1",
			maxLeverage:     "1",
			expectError:     true,
		},
		{
			name:            "valid market pair",
			pairName:        "token0:token1",
			sqrtDepth:       "100",
			priceMultiplier: "1",
			maintainRatio:   "0.1",
			maxLeverage:     "10",
			expectError:     false,
		},
	}

	ctx := setupServer(t)
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			cmd := cli.AddMarketGenesisCmd("home")
			cmd.SetArgs([]string{
				fmt.Sprintf("--%s=%s", cli.FlagPair, tc.pairName),
				fmt.Sprintf("--%s=%s", cli.FlagSqrtDepth, tc.sqrtDepth),
				fmt.Sprintf("--%s=%s", cli.FlagPriceMultiplier, tc.priceMultiplier),
				fmt.Sprintf("--%s=%s", cli.FlagMaintenenceMarginRatio, tc.maintainRatio),
				fmt.Sprintf("--%s=%s", cli.FlagMaxLeverage, tc.maxLeverage),
				fmt.Sprintf("--%s=home", flags.FlagHome),
			})

			if tc.expectError {
				require.Error(t, cmd.ExecuteContext(ctx))
			} else {
				require.NoError(t, cmd.ExecuteContext(ctx))
			}
		})
	}
}
