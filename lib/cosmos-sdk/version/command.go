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

			var (
				bz  []byte
				err error
			)

			output, _ := cmd.Flags().GetString(cmtcli.OutputFlag)
			switch strings.ToLower(output) {
			case "json":
				// The path report is added after the existing fields.
				verInfo.BinaryPaths = report
				bz, err = json.Marshal(verInfo)

			default:
				bz, err = yaml.Marshal(&verInfo)
			}

			if err != nil {
				return err
			}

			text := string(bz)
			if report != nil && strings.ToLower(output) != "json" {
				// Text output keeps the YAML body as is and appends the path
				// report as plain lines.
				text += "\n" + report.String() + "\n"
			}
			cmd.Println(text)
			return nil
		},
	}

	cmd.Flags().Bool(flagLong, false, "Print long version information")
	cmd.Flags().StringP(cmtcli.OutputFlag, "o", "text", "Output format (text|json)")

	return cmd
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
