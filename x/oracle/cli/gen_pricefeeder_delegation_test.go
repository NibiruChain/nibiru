package cli_test

import (
	"fmt"
	"testing"

	"github.com/cosmos/cosmos-sdk/client/flags"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/x/common/testutil"
	"github.com/NibiruChain/nibiru/v2/x/oracle/cli"
)

func TestAddGenesisPricefeederDelegation(t *testing.T) {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("nibi", "nibipub")
	config.SetBech32PrefixForValidator("nibivaloper", "nibivaloperpub")
	config.SetBech32PrefixForConsensusNode("nivivalcons", "nibivalconspub")
	tests := []struct {
		name        string
		validator   string
		pricefeeder string

		expectErr bool
	}{
		{
			name:        "valid",
			validator:   "nibivaloper1zaavvzxez0elundtn32qnk9lkm8kmcszuwx9jz",
			pricefeeder: "nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl",
			expectErr:   false,
		},
		{
			name:        "invalid pricefeeder",
			validator:   "nibivaloper1zaavvzxez0elundtn32qnk9lkm8kmcszuwx9jz",
			pricefeeder: "nibi1foobar",
			expectErr:   true,
		},
		{
			name:        "empty pricefeeder",
			validator:   "nibivaloper1zaavvzxez0elundtn32qnk9lkm8kmcszuwx9jz",
			pricefeeder: "",
			expectErr:   true,
		},
		{
			name:        "invalid validator",
			validator:   "nibivaloper1foobar",
			pricefeeder: "nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl",
			expectErr:   true,
		},
		{
			name:        "empty validator",
			validator:   "",
			pricefeeder: "nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl",
			expectErr:   true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			ctx := testutil.SetupClientCtx(t)
			cmd := cli.AddGenesisPricefeederDelegationCmd(t.TempDir())
			cmd.SetArgs([]string{
				fmt.Sprintf("--%s=%s", cli.FlagValidator, tc.validator),
				fmt.Sprintf("--%s=%s", cli.FlagPricefeeder, tc.pricefeeder),
				fmt.Sprintf("--%s=home", flags.FlagHome),
			})

			if tc.expectErr {
				require.Error(t, cmd.ExecuteContext(ctx))
			} else {
				require.NoError(t, cmd.ExecuteContext(ctx))
			}
		})
	}
}
