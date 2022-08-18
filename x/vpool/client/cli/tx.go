package cli

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/spf13/cobra"

	"github.com/NibiruChain/nibiru/x/vpool/types"
)

func GetTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Generalized automated market maker transaction subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		AddVPoolGenesisCmd(),
	)

	return txCmd
}
