package cli_test

import (
	"fmt"
	"testing"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/NibiruChain/nibiru/x/perp/v2/client/cli"
)

// Tests "add-genesis-perp-market", a command that adds a market to genesis.json
func TestAddMarketGenesisCmd(t *testing.T) {
	tests := []struct {
		name            string
		pairName        string
		sqrtDepth       string
		priceMultiplier string
		maintainRatio   string
		maxLeverage     string
		maxFundingRate  string
		oraclePair      string
		expectError     bool
	}{
		{
			name:            "pair name empty",
			pairName:        "",
			sqrtDepth:       "1",
			priceMultiplier: "1",
			maintainRatio:   "1",
			maxLeverage:     "1",
			maxFundingRate:  "1",
			oraclePair:      "token0:token1",
			expectError:     true,
		},
		{
			name:            "invalid pair name",
			pairName:        "token0:token1:token2",
			sqrtDepth:       "1",
			priceMultiplier: "1",
			maintainRatio:   "1",
			maxLeverage:     "1",
			maxFundingRate:  "1",
			oraclePair:      "token0:token1",
			expectError:     true,
		},
		{
			name:            "empty sqrt depth input",
			pairName:        "token0:token1",
			sqrtDepth:       "",
			priceMultiplier: "1",
			maintainRatio:   "1",
			maxLeverage:     "1",
			maxFundingRate:  "1",
			oraclePair:      "token0:token1",
			expectError:     true,
		},
		{
			name:            "max leverage cannot be zero",
			pairName:        "token0:token1",
			sqrtDepth:       "100",
			priceMultiplier: "1",
			maintainRatio:   "0.1",
			maxLeverage:     "0",
			maxFundingRate:  "0",
			oraclePair:      "token0:token1",
			expectError:     true,
		},
		{
			name:            "price multiplier cannot be zero",
			pairName:        "token0:token1",
			sqrtDepth:       "100",
			priceMultiplier: "0",
			maintainRatio:   "0.1",
			maxLeverage:     "1",
			maxFundingRate:  "1",
			oraclePair:      "token0:token1",
			expectError:     true,
		},
		{
			name:            "price multiplier cannot be negative",
			pairName:        "token0:token1",
			sqrtDepth:       "100",
			priceMultiplier: "-1",
			maintainRatio:   "0.1",
			maxLeverage:     "1",
			maxFundingRate:  "1",
			oraclePair:      "token0:token1",
			expectError:     true,
		},
		{
			name:            "negative max funding rate",
			pairName:        "token0:token1",
			sqrtDepth:       "100",
			priceMultiplier: "1",
			maintainRatio:   "0.1",
			maxLeverage:     "10",
			maxFundingRate:  "-1",
			oraclePair:      "token0:token1",
			expectError:     true,
		},
		{
			name:            "negative max funding rate",
			pairName:        "token0:token1",
			sqrtDepth:       "100",
			priceMultiplier: "1",
			maintainRatio:   "0.1",
			maxLeverage:     "10",
			maxFundingRate:  "-1",
			oraclePair:      "invalidPair",
			expectError:     true,
		},
		{
			name:            "valid market pair",
			pairName:        "token0:token1",
			sqrtDepth:       "100",
			priceMultiplier: "1",
			maintainRatio:   "0.1",
			maxLeverage:     "10",
			maxFundingRate:  "10",
			oraclePair:      "token0:token1",
			expectError:     false,
		},
	}

	ctx := testutil.SetupClientCtx(t)
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
				fmt.Sprintf("--%s=%s", cli.FlagMaxFundingrate, tc.maxFundingRate),
				fmt.Sprintf("--%s=%s", cli.FlagOraclePair, tc.oraclePair),
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
