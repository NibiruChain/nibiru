package cli

import (
	"encoding/json"
	"fmt"
	"github.com/NibiruChain/nibiru/x/sudo/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/spf13/cobra"
)

func AddSudoRootAccountCmd(defaultNodeHome string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-sudo-root-account",
		Short: "Add sudo module root account to genesis.json.",
		Long:  `Add sudo module root account to genesis.json.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)
			serverCtx := server.GetServerContextFromCmd(cmd)
			config := serverCtx.Config

			config.SetRoot(clientCtx.HomeDir)

			genFile := config.GenesisFile()
			appState, genDoc, err := genutiltypes.GenesisStateFromGenFile(genFile)
			if err != nil {
				return err
			}

			rootAccount := args[0]
			addr, err := sdk.AccAddressFromBech32(rootAccount)
			if err != nil {
				return fmt.Errorf("failed to parse address: %w", err)
			}

			sudoGenState := types.GetGenesisStateFromAppState(clientCtx.Codec, appState)
			sudoGenState.Sudoers.Root = addr.String()

			sudoGenStateBz, err := clientCtx.Codec.MarshalJSON(sudoGenState)
			if err != nil {
				return fmt.Errorf("failed to marshal market genesis state: %w", err)
			}

			appState[types.ModuleName] = sudoGenStateBz

			appStateJSON, err := json.Marshal(appState)
			if err != nil {
				return fmt.Errorf("failed to marshal application genesis state: %w", err)
			}

			genDoc.AppState = appStateJSON
			err = genutil.ExportGenesisFile(genDoc, genFile)
			if err != nil {
				return err
			}

			err = clientCtx.PrintString(fmt.Sprintf("sudo module root account added to genesis.json: %s\n", rootAccount))
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().String(flags.FlagHome, defaultNodeHome, "The application home directory")

	return cmd
}
