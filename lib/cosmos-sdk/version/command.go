package version

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"

	cmtcli "github.com/NibiruChain/nibiru/v2/bft/libs/cli"
)

const flagLong = "long"

// NewVersionCommand returns a CLI command to interactively print the application binary version information.
func NewVersionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the application binary version information",
		RunE: func(cmd *cobra.Command, _ []string) error {
			verInfo := NewInfo()
			report := currentPathReport()

			if long, _ := cmd.Flags().GetBool(flagLong); !long {
				cmd.Println(verInfo.Version)
				return nil
			}

			output, _ := cmd.Flags().GetString(cmtcli.OutputFlag)
			bz, err := RenderLong(verInfo, report, output)
			if err != nil {
				return err
			}
			text := string(bz)
			cmd.Println(text)
			if report != nil && strings.ToLower(output) != "json" && !report.AllFirstRunning() {
				// A short hint for people. Scripts read binary_paths.
				cmd.PrintErrln("note: another nibiru or nibid comes first " +
					"on PATH; see binary_paths in this output")
			}
			return nil
		},
	}

	cmd.Flags().Bool(flagLong, false, "Print long version information")
	cmd.Flags().StringP(cmtcli.OutputFlag, "o", "text", "Output format (text|json)")

	return cmd
}

// RenderLong renders the long version output. The path report is one field,
// binary_paths, in both formats, so the whole text output parses as YAML and
// the JSON output parses as JSON. Existing fields keep their order, and
// binary_paths comes last in JSON.
func RenderLong(
	info Info, report *PathReport, output string,
) ([]byte, error) {
	info.BinaryPaths = report
	if strings.ToLower(output) == "json" {
		return json.Marshal(info)
	}
	return yaml.Marshal(&info)
}

// currentPathReport reports where the running binary lives and which other
// nibiru or nibid executables are on PATH. It returns nil when the running
// path is unknown.
func currentPathReport() *PathReport {
	exe, err := os.Executable()
	if err != nil {
		return nil
	}
	r := ReportPaths(exe, os.Getenv("PATH"), []string{"nibiru", "nibid"},
		OSPathProbe())
	return &r
}
