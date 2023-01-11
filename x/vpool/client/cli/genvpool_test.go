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
		baseAmt       string
		quoteAmt      string
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
			baseAmt:       "1",
			quoteAmt:      "1",
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
			baseAmt:       "1",
			quoteAmt:      "1",
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
			baseAmt:       "1",
			quoteAmt:      "1",
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
			baseAmt:       "",
			quoteAmt:      "1",
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
			baseAmt:       "100",
			quoteAmt:      "100",
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
			baseAmt:       "100",
			quoteAmt:      "100",
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

			cmd := cli.AddVpoolGenesisCmd("home")
			cmd.SetArgs([]string{
				fmt.Sprintf("--%s=%s", cli.FlagPair, tc.pairName),
				fmt.Sprintf("--%s=%s", cli.FlagBaseAmt, tc.baseAmt),
				fmt.Sprintf("--%s=%s", cli.FlagQuoteAmt, tc.quoteAmt),
				fmt.Sprintf("--%s=%s", cli.FlagTradeLim, tc.tradeLimit),
				fmt.Sprintf("--%s=%s", cli.FlagFluctLim, tc.flucLimit),
				fmt.Sprintf("--%s=%s", cli.FlagMaxOracleSpreadRatio, tc.maxOracle),
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
