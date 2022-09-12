package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/spf13/cobra"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/pricefeed/types"
)

// AddPriceFeedParamPairs returns add-genesis-pricefeed-pairs
func AddPriceFeedParamPairs(defaultNodeHome string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-genesis-pricefeed-pairs [,[pair]]",
		Short: "Add pair to genesis.json",
		Long:  `Adds the pricefeed pairs to genesis.json`,
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

			priceFeed := types.GetGenesisStateFromAppState(clientCtx.Codec, appState)
			pairs, err := parseAndValidatePairs(args[0], priceFeed.Params.Pairs)
			if err != nil {
				return err
			}

			priceFeed.Params.Pairs = append(priceFeed.Params.Pairs, pairs...)
			priceFeedParamsStateBz, err := clientCtx.Codec.MarshalJSON(priceFeed)
			if err != nil {
				return fmt.Errorf("failed to marshal price feed genesis state: %w", err)
			}

			appState[types.ModuleName] = priceFeedParamsStateBz
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

func parseAndValidatePairs(pairsStr string, genPairs common.AssetPairs) (common.AssetPairs, error) {
	pairs := strings.TrimSpace(pairsStr)
	if len(pairs) == 0 {
		return nil, fmt.Errorf("no pairs found")
	}

	pairList := strings.Split(pairs, ",")
	newPairs := common.AssetPairs{}
	exists := make(map[string]bool)
	for _, pair := range pairList {
		newPair, err := common.NewAssetPair(pair)
		if err != nil {
			return nil, err
		}

		if genPairs.Contains(newPair) {
			return nil, fmt.Errorf("pair %s already exists in genesis", pair)
		}

		if _, found := exists[pair]; found {
			return nil, fmt.Errorf("pair %s repeated", pair)
		}

		newPairs = append(newPairs, newPair)
		exists[pair] = true
	}

	return newPairs, nil
}
