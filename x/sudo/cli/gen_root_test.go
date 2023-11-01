package cli_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
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
			testapp.EnsureNibiruPrefix()
			ctx := testutil.SetupClientCtx(t)
			cmd := cli.AddSudoRootAccountCmd(t.TempDir())
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
