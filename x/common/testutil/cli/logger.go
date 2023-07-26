package cli

import (
	"testing"

	"github.com/spf13/cobra"
)

// Logger is a network logger interface that exposes testnet-level Log() methods for an in-process testing network
// This is not to be confused with logging that may happen at an individual node or validator level
type Logger interface {
	Log(args ...interface{})
	Logf(format string, args ...interface{})
}

var (
	_ Logger = (*testing.T)(nil)
	_ Logger = (*CLILogger)(nil)
)

// CLILogger wraps a cobra.Command and provides command logging methods.
type CLILogger struct {
	cmd *cobra.Command
}

// Log logs given args.
func (s CLILogger) Log(args ...interface{}) {
	s.cmd.Println(args...)
}

// Logf logs given args according to a format specifier.
func (s CLILogger) Logf(format string, args ...interface{}) {
	s.cmd.Printf(format, args...)
}

// NewCLILogger creates a new CLILogger.
func NewCLILogger(cmd *cobra.Command) CLILogger {
	return CLILogger{cmd}
}
