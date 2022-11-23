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

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/vpool/types"
)

// AddVpoolGenesisCmd returns add-vpool-genesis
func AddVpoolGenesisCmd(defaultNodeHome string) *cobra.Command {
	usageExampleTail := strings.Join([]string{
		"pair", "base-asset-reserve", "quote-asset-reserve", "trade-limit-ratio",
		"fluctuation-limit-ratio", "max-oracle-spread-ratio", "maintenance-margin-ratio",
		"max-leverage",
	}, "] [")
	cmd := &cobra.Command{
		Use:   fmt.Sprintf("add-genesis-vpool [%s]", usageExampleTail),
		Short: "Add vPools to genesis.json",
		Long:  `Add vPools to genesis.json.`,
		Args:  cobra.ExactArgs(8),
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

			vPool, err := parseVpoolParams(args)
			if err != nil {
				return err
			}

			vPoolGenState := types.GetGenesisStateFromAppState(clientCtx.Codec, appState)
			vPoolGenState.Vpools = append(vPoolGenState.Vpools, vPool)

			vPoolGenStateBz, err := clientCtx.Codec.MarshalJSON(vPoolGenState)
			if err != nil {
				return fmt.Errorf("failed to marshal vpool genesis state: %w", err)
			}

			appState[types.ModuleName] = vPoolGenStateBz

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

func parseVpoolParams(args []string) (types.Vpool, error) {
	vPair, err := common.NewAssetPair(args[0])
	if err != nil {
		return types.Vpool{}, err
	}

	baseAsset, err := sdk.NewDecFromStr(args[1])
	if err != nil {
		return types.Vpool{}, err
	}
	quoteAsset, err := sdk.NewDecFromStr(args[2])
	if err != nil {
		return types.Vpool{}, err
	}
	tradeLimit, err := sdk.NewDecFromStr(args[3])
	if err != nil {
		return types.Vpool{}, err
	}

	fluctuationLimitRatio, err := sdk.NewDecFromStr(args[4])
	if err != nil {
		return types.Vpool{}, err
	}

	maxOracleSpread, err := sdk.NewDecFromStr(args[5])
	if err != nil {
		return types.Vpool{}, err
	}

	maintenanceMarginRatio, err := sdk.NewDecFromStr(args[6])
	if err != nil {
		return types.Vpool{}, err
	}

	maxLeverage, err := sdk.NewDecFromStr(args[7])
	if err != nil {
		return types.Vpool{}, err
	}

	vPool := types.Vpool{
		Pair:              vPair,
		QuoteAssetReserve: quoteAsset,
		BaseAssetReserve:  baseAsset,
		Config: types.VpoolConfig{
			TradeLimitRatio:        tradeLimit,
			FluctuationLimitRatio:  fluctuationLimitRatio,
			MaxOracleSpreadRatio:   maxOracleSpread,
			MaintenanceMarginRatio: maintenanceMarginRatio,
			MaxLeverage:            maxLeverage,
		},
	}

	return vPool, vPool.Validate()
}
