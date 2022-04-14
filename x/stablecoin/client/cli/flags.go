package cli

import (
	flag "github.com/spf13/pflag"
)

const (
	// Will be parsed to []string.
	MintDenoms = "swap-route-denoms"
)

func FlagSetSwapAmountOutRoutes() *flag.FlagSet {
	fs := flag.NewFlagSet("", flag.ContinueOnError)

	fs.StringArray(MintDenoms, []string{""}, "mint denoms")
	return fs
}
