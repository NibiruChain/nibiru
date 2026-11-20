package main_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	nibid "github.com/NibiruChain/nibiru/v2/x/cli"
)

// nibidMention matches a literal "nibid" that is not part of the ~/.nibid
// directory or another word.
var nibidMention = regexp.MustCompile(`(^|[^\w./~-])nibid\b`)

// nibidLines collects every line of help text (Use, Short, Long, Example,
// and flag usages) of the command tree that mentions "nibid".
func nibidLines(cmd *cobra.Command, path string, out *[]string) {
	texts := []string{cmd.Use, cmd.Short, cmd.Long, cmd.Example,
		cmd.LocalFlags().FlagUsages()}
	for _, text := range texts {
		for _, line := range strings.Split(text, "\n") {
			// The default daemon home directory name is "nibid" on purpose.
			line = strings.ReplaceAll(line, `(default "nibid")`, "")
			if nibidMention.MatchString(line) {
				*out = append(*out, path+": "+strings.TrimSpace(line))
			}
		}
	}
	for _, c := range cmd.Commands() {
		nibidLines(c, path+" "+c.Name(), out)
	}
}

// Under the nibiru name, help text never names the nibid binary. The ~/.nibid
// directory is allowed.
func TestHelpUnderNibiruNameHasNoNibid(t *testing.T) {
	rootCmd, _ := nibid.NewRootCmdNamed("nibiru")
	var lines []string
	nibidLines(rootCmd, "nibiru", &lines)
	require.Empty(t, lines, strings.Join(lines, "\n"))
}

// Under the nibid name, the same commands print nibid.
func TestHelpUnderNibidNameFollowsName(t *testing.T) {
	rootCmd, _ := nibid.NewRootCmdNamed("nibid")
	for _, args := range [][]string{{"debug"}, {"node"}} {
		cmd, _, err := rootCmd.Find(args)
		require.NoError(t, err)
		text := cmd.Long + cmd.Example
		require.Contains(t, text, "nibid ", args)
		require.NotRegexp(t, `nibiru (debug|start)`, text, args)
	}
}
