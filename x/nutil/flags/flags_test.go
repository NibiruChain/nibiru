package flags

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestAddTxFlagsToCmdHelpVerbose(t *testing.T) {
	testCases := []struct {
		name        string
		args        []string
		wantVerbose bool
	}{
		{
			name:        "help verbose without required args",
			args:        []string{"--help-verbose"},
			wantVerbose: true,
		},
		{
			name:        "short help remains terse",
			args:        []string{"-h"},
			wantVerbose: false,
		},
		{
			name:        "short help with help verbose",
			args:        []string{"-h", "--help-verbose"},
			wantVerbose: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			cmd := &cobra.Command{
				Use:  "test [arg]",
				Args: cobra.ExactArgs(1),
				RunE: func(cmd *cobra.Command, args []string) error {
					return nil
				},
			}
			AddTxFlagsToCmd(cmd)
			cmd.SetArgs(tc.args)
			cmd.SetOut(&out)
			cmd.SetErr(&out)

			require.NoError(t, cmd.Execute())
			helpOutput := out.String()
			if tc.wantVerbose {
				require.Contains(t, helpOutput, "--account-number")
				require.Contains(t, helpOutput, "--gas-prices")
				return
			}
			require.NotContains(t, helpOutput, "--account-number")
			require.NotContains(t, helpOutput, "--gas-prices")
		})
	}
}
