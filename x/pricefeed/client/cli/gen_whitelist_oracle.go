package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/spf13/cobra"

	"github.com/NibiruChain/nibiru/x/pricefeed/types"
)

// AddWhitelistGenesisOracle returns add-genesis-oracle
func AddWhitelistGenesisOracle(defaultNodeHome string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-genesis-oracle [,[oracle-address]]",
		Short: "Add genesis oracle to genesis.json",
		Long:  `Adds the oracle address to genesis.json for pricefeed`,
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

			priceFeedOracle := types.GetGenesisStateFromAppState(clientCtx.Codec, appState)
			genesisOracles, err := parseAndVerifyOracles(args[0], priceFeedOracle.GenesisOracles)
			if err != nil {
				return err
			}

			priceFeedOracle.GenesisOracles = append(priceFeedOracle.GenesisOracles, genesisOracles...)
			priceFeedOracleStateBz, err := clientCtx.Codec.MarshalJSON(priceFeedOracle)
			if err != nil {
				return fmt.Errorf("failed to marshal price feed genesis state: %w", err)
			}

			appState[types.ModuleName] = priceFeedOracleStateBz
			appStateJSON, err := json.Marshal(appState)
			if err != nil {
				return fmt.Errorf("failed to marshal application genesis state: %w", err)
			}

			genDoc.AppState = appStateJSON
			return genutil.ExportGenesisFile(genDoc, genFile)
		},
	}

	cmd.Flags().String(flags.FlagHome, defaultNodeHome, "The application home directory")
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func parseAndVerifyOracles(oraclesStr string, genOracles []string) ([]string, error) {
	oracles := strings.TrimSpace(oraclesStr)
	if len(oracles) == 0 {
		return nil, fmt.Errorf("no oracle addresses found")
	}

	oracleAddresses := strings.Split(oracles, ",")
	exists := make(map[string]bool)
	for i, oracleAddress := range oracleAddresses {
		addr, err := sdk.AccAddressFromBech32(oracleAddress)
		if err != nil {
			return nil, err
		}

		oracleAddress = addr.String()
		if _, found := exists[oracleAddress]; found {
			return nil, fmt.Errorf("oracle address %s repeated", oracleAddress)
		}

		oracleAddresses[i] = oracleAddress
		exists[oracleAddress] = true
	}

	for _, genOracle := range genOracles {
		if _, found := exists[genOracle]; found {
			return nil, fmt.Errorf("oracle address %s already exists", genOracle)
		}
	}

	return oracleAddresses, nil
}
