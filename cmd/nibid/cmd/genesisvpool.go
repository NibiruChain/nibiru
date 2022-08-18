package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/NibiruChain/nibiru/x/common"
	vpooltypes "github.com/NibiruChain/nibiru/x/vpool/types"
)

var (
	FlagBaseAssetReserve       = "base-asset-reserve"
	FlagQuoteAssetReserve      = "quote-asset-reserve"
	FlagTradeLimitRatio        = "trade-limit-ratio"
	FlagFluctuationLimitRatio  = "fluctuation-limit-ratio"
	FlagMaxOracleSpreadRatio   = "maxOracle-spread-ratio"
	FlagMaintenanceMarginRatio = "maintenance-margin-ratio"
	FlagMaxLeverage            = "max-leverage"
)

// AddVPoolGenesisCmd returns add-vpool-genesis
func AddVPoolGenesisCmd(defaultNodeHome string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-genesis-vpool [pair] [flag]",
		Short: "Add vPools to genesis.json",
		Long:  `Add vPools to genesis.json.`,
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

			vPool, err := parseVpoolParams(args[0], cmd.Flags())
			if err != nil {
				return err
			}

			vPoolGenState := vpooltypes.GetGenesisStateFromAppState(clientCtx.Codec, appState)
			vPoolGenState.Vpools = append(vPoolGenState.Vpools, vPool)

			vPoolGenStateBz, err := clientCtx.Codec.MarshalJSON(vPoolGenState)
			if err != nil {
				return fmt.Errorf("failed to marshal bank genesis state: %w", err)
			}

			appState[vpooltypes.ModuleName] = vPoolGenStateBz

			appStateJSON, err := json.Marshal(appState)
			if err != nil {
				return fmt.Errorf("failed to marshal application genesis state: %w", err)
			}

			genDoc.AppState = appStateJSON
			return genutil.ExportGenesisFile(genDoc, genFile)
		},
	}

	cmd.Flags().String(flags.FlagHome, defaultNodeHome, "The application home directory")
	cmd.Flags().String(FlagBaseAssetReserve, "", "Base Asset Reserve")
	cmd.Flags().String(FlagQuoteAssetReserve, "", "Quote Asset Reserve")
	cmd.Flags().String(FlagTradeLimitRatio, "", "Trade limit ratio")
	cmd.Flags().String(FlagFluctuationLimitRatio, "", "Fluctuation limit ratio")
	cmd.Flags().String(FlagMaxOracleSpreadRatio, "", "Max Oracle Spread ratio")
	cmd.Flags().String(FlagMaintenanceMarginRatio, "", "Maintenance Margin Ratio")
	cmd.Flags().String(FlagMaxLeverage, "", "Max Leverage")
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func parseVpoolParams(pair string, flags *pflag.FlagSet) (*vpooltypes.Pool, error) {
	vPair, err := common.NewAssetPair(pair)
	if err != nil {
		return nil, err
	}

	baseAsset, err := flags.GetString(FlagBaseAssetReserve)
	if err != nil {
		return nil, err
	}
	quoteAsset, err := flags.GetString(FlagQuoteAssetReserve)
	if err != nil {
		return nil, err
	}
	tradeLimit, err := flags.GetString(FlagTradeLimitRatio)
	if err != nil {
		return nil, err
	}
	fluctuationLimitRatio, err := flags.GetString(FlagFluctuationLimitRatio)
	if err != nil {
		return nil, err
	}
	maxOracleSpread, err := flags.GetString(FlagMaxOracleSpreadRatio)
	if err != nil {
		return nil, err
	}
	maintenanceMarginRatio, err := flags.GetString(FlagMaintenanceMarginRatio)
	if err != nil {
		return nil, err
	}
	maxLeverage, err := flags.GetString(FlagMaxLeverage)
	if err != nil {
		return nil, err
	}

	vPool := vpooltypes.NewPool(
		vPair,
		sdk.MustNewDecFromStr(tradeLimit),
		sdk.MustNewDecFromStr(baseAsset),
		sdk.MustNewDecFromStr(quoteAsset),
		sdk.MustNewDecFromStr(fluctuationLimitRatio),
		sdk.MustNewDecFromStr(maxOracleSpread),
		sdk.MustNewDecFromStr(maintenanceMarginRatio),
		sdk.MustNewDecFromStr(maxLeverage),
	)

	return vPool, nil
}
