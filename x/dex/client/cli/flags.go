package cli

import (
	flag "github.com/spf13/pflag"
)

const (
	// Will be parsed to string.
	FlagPoolFile = "pool-file"

	// Parsed into uint64
	FlagPoolId        = "pool-id"
	FlagPoolSharesOut = "pool-shares-out"
)

type createPoolInputs struct {
	Weights        string `json:"weights"`
	InitialDeposit string `json:"initial-deposit"`
	SwapFee        string `json:"swap-fee"`
	ExitFee        string `json:"exit-fee"`
}

func FlagSetCreatePool() *flag.FlagSet {
	fs := flag.NewFlagSet("", flag.ContinueOnError)

	fs.String(FlagPoolFile, "", "Pool json file path (if this path is given, other create pool flags should not be used).")
	return fs
}

func FlagSetExitPool() *flag.FlagSet {
	fs := flag.NewFlagSet("", flag.ContinueOnError)

	fs.String(FlagPoolId, "", "The pool id to withdraw from.")
	fs.String(FlagPoolSharesOut, "", "The amount of pool share tokens to burn.")
	return fs
}
