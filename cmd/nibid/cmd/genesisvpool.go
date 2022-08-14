package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/server"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/spf13/cobra"

	"github.com/NibiruChain/nibiru/x/common"
	vpooltypes "github.com/NibiruChain/nibiru/x/vpool/types"
)

type (
	VPoolsJSON []VPoolJSON

	VPoolJSON struct {
		Pair                   string `json:"pair" yaml:"pair"`
		BaseAssetReserve       string `json:"base_asset_reserve" yaml:"base_asset_reserve"`
		QuoteAssetReserve      string `json:"quote_asset_reserve" yaml:"quote_asset_reserve"`
		TradeLimitRatio        string `json:"trade_limit_ratio" yaml:"trade_limit_ratio"`
		FluctuationLimitRatio  string `json:"fluctuation_limit_ratio" yaml:"fluctuation_limit_ratio"`
		MaxOracleSpreadRatio   string `json:"max_oracle_spread_ratio" yaml:"max_oracle_spread_ratio"`
		MaintenanceMarginRatio string `json:"maintenance_margin_ratio" yaml:"maintenance_margin_ratio"`
		MaxLeverage            string `json:"max_leverage" yaml:"max_leverage"`
	}
)

// AddVPoolGenesisCmd returns add-vpool-genesis
func AddVPoolGenesisCmd(defaultNodeHome string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-genesis-vpool [vpool-file]",
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
			vPools, err := parseVPoolsFile(clientCtx.LegacyAmino, args[0])
			if err != nil {
				return err
			}

			vPoolGenState := vpooltypes.GetGenesisStateFromAppState(clientCtx.Codec, appState)
			vPoolGenState.Vpools = append(vPoolGenState.Vpools, vPools...)

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
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
func parseVPoolsFile(cdc *codec.LegacyAmino, vPoolFile string) ([]*vpooltypes.Pool, error) {
	pools := VPoolsJSON{}

	poolsFile, err := ioutil.ReadFile(vPoolFile)
	if err != nil {
		return nil, err
	}

	if err := cdc.UnmarshalJSON(poolsFile, &pools); err != nil {
		return nil, err
	}

	var vPools []*vpooltypes.Pool
	for _, vPool := range pools {
		pair, err := common.NewAssetPair(vPool.Pair)
		if err != nil {
			return nil, err
		}

		vPools = append(vPools, vpooltypes.NewPool(
			pair,
			sdk.MustNewDecFromStr(vPool.BaseAssetReserve),
			sdk.MustNewDecFromStr(vPool.QuoteAssetReserve),
			sdk.MustNewDecFromStr(vPool.TradeLimitRatio),
			sdk.MustNewDecFromStr(vPool.FluctuationLimitRatio),
			sdk.MustNewDecFromStr(vPool.MaxOracleSpreadRatio),
			sdk.MustNewDecFromStr(vPool.MaintenanceMarginRatio),
			sdk.MustNewDecFromStr(vPool.MaxLeverage)))
	}

	return vPools, nil
}
