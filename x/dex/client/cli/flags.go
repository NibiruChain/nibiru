package cli

import (
	flag "github.com/spf13/pflag"
)

const (
	// Will be parsed to string.
	FlagPoolFile = "pool-file"

	// Will be parsed to uint64.
	FlagPoolId = "pool-id"

	// Will be parsed to []sdk.Coin.
	FlagTokensIn = "tokens-in"

	FlagPoolSharesOut = "pool-shares-out"
)

type createPoolInputs struct {
	Weights        string `json:"weights"`
	InitialDeposit string `json:"initial-deposit"`
	SwapFee        string `json:"swap-fee"`
	ExitFee        string `json:"exit-fee"`
}

func FlagSetCreatePool() *flag.FlagSet {
	fs := flag.NewFlagSet("create-pool", flag.PanicOnError)

	fs.String(FlagPoolFile, "", "Pool json file path")
	return fs
}

func FlagSetJoinPool() *flag.FlagSet {
	fs := flag.NewFlagSet("join-pool", flag.PanicOnError)

	fs.Uint64(FlagPoolId, 0, "The id of pool")
	fs.StringArray(FlagTokensIn, []string{""}, "Amount of each denom to send into the pool (specify multiple denoms with: --tokens-in=1ust --tokens-in=1usdm)")
}

func FlagSetExitPool() *flag.FlagSet {
	fs := flag.NewFlagSet("", flag.ContinueOnError)

	fs.String(FlagPoolId, "", "The pool id to withdraw from.")
	fs.String(FlagPoolSharesOut, "", "The amount of pool share tokens to burn.")
	return fs
}
