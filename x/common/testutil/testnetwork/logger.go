package testnetwork

import (
	"testing"
)

// Logger is a network logger interface that exposes testnet-level Log() methods
// for an in-process testing network This is not to be confused with logging that
// may happen at an individual node or validator level
//
// Typically, a `testing.T` struct is used as the logger for both the "Network"
// and corresponding "Validators".
type Logger interface {
	Log(args ...interface{})
	Logf(format string, args ...interface{})
}

var _ Logger = (*testing.T)(nil)
