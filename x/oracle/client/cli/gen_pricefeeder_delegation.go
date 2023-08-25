package cli

import (
	"encoding/json"
	"fmt"

	flag "github.com/spf13/pflag"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/spf13/cobra"

	"github.com/NibiruChain/nibiru/x/oracle/types"
)

const (
	FlagValidator   = "validator"
	FlagPricefeeder = "pricefeeder"
)

func AddGenesisPricefeederDelegationCmd(defaultNodeHome string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-genesis-pricefeeder-delegation",
		Short: "Add a pricefeeder delegation to genesis.json.",
		Long:  `Add a pricefeeder delegation to genesis.json.`,
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)
			serverCtx := server.GetServerContextFromCmd(cmd)
			config := serverCtx.Config
			config.SetRoot(clientCtx.HomeDir)
			genFile := config.GenesisFile()

			genState, genDoc, err := genutiltypes.GenesisStateFromGenFile(genFile)
			if err != nil {
				return err
			}

			valAddr, err := cmd.Flags().GetString(FlagValidator)
			if err != nil {
				return err
			}

			_, err = sdk.ValAddressFromBech32(valAddr)
			if err != nil {
				return err
			}

			pricefeederAddr, err := cmd.Flags().GetString(FlagPricefeeder)
			if err != nil {
				return err
			}

			_, err = sdk.AccAddressFromBech32(pricefeederAddr)
			if err != nil {
				return err
			}

			oracleGenState := types.GetGenesisStateFromAppState(clientCtx.Codec, genState)
			oracleGenState.FeederDelegations = append(oracleGenState.FeederDelegations, types.FeederDelegation{
				FeederAddress:    pricefeederAddr,
				ValidatorAddress: valAddr,
			})

			oracleGenStateBz, err := clientCtx.Codec.MarshalJSON(oracleGenState)
			if err != nil {
				return fmt.Errorf("failed to marshal market genesis state: %w", err)
			}

			genState[types.ModuleName] = oracleGenStateBz

			appStateJSON, err := json.Marshal(genState)
			if err != nil {
				return fmt.Errorf("failed to marshal application genesis state: %w", err)
			}

			genDoc.AppState = appStateJSON
			return genutil.ExportGenesisFile(genDoc, genFile)
		},
	}

	cmd.Flags().String(flags.FlagHome, defaultNodeHome, "The application home directory")

	flagSet := flag.NewFlagSet("flags-add-genesis-pricefeeder-delegation", flag.ContinueOnError)

	flagSet.String(FlagValidator, "", "validator address")
	_ = cmd.MarkFlagRequired(FlagValidator)

	flagSet.String(FlagPricefeeder, "", "pricefeeder address")
	_ = cmd.MarkFlagRequired(FlagPricefeeder)

	cmd.Flags().AddFlagSet(flagSet)

	return cmd
}
