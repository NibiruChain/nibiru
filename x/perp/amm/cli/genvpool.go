package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	flag "github.com/spf13/pflag"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/spf13/cobra"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/perp/amm/types"
)

const (
	FlagPair                   = "pair"
	FlagBaseAmt                = "base-amt"
	FlagQuoteAmt               = "quote-amt"
	FlagPegMultiplier          = "peg-multiplier"
	FlagTradeLim               = "trade-lim"
	FlagFluctLim               = "fluct-lim"
	FlagMaintenenceMarginRatio = "mmr"
	FlagMaxLeverage            = "max-leverage"
	FlagMaxOracleSpreadRatio   = "max-oracle-spread-ratio"
)

var flagsAddMarketGenesis = map[string]struct {
	flagName       string
	defaultValue   string
	usageDocString string
}{
	FlagPair:                   {"pair", "", "trading pair identifier of the form 'base:quote'. E.g., ueth:unusd"},
	FlagBaseAmt:                {"base-amt", "", "amount of base asset reserves"},
	FlagQuoteAmt:               {"quote-amt", "", "amount of quote asset reserves"},
	FlagPegMultiplier:          {"peg-multiplier", "", "the peg multiplier for the pool"},
	FlagTradeLim:               {"trade-lim", "0.1", "percentage applied to reserves in order not to over trade"},
	FlagFluctLim:               {"fluct-lim", "0.1", "percentage that a single open or close position can alter the reserves"},
	FlagMaintenenceMarginRatio: {"mmr", "0.0625", "maintenance margin ratio"},
	FlagMaxLeverage:            {"max-leverage", "10", "maximum leverage for opening a position"},
	FlagMaxOracleSpreadRatio:   {"max-oracle-spread-ratio", "0.1", "max oracle spread ratio"},
}

// AddMarketGenesisCmd returns add-market-genesis
func AddMarketGenesisCmd(defaultNodeHome string) *cobra.Command {
	usageExampleTail := strings.Join([]string{
		"pair", "base-asset-reserve", "quote-asset-reserve", "trade-limit-ratio",
		"fluctuation-limit-ratio", "max-oracle-spread-ratio", "maintenance-margin-ratio",
		"max-leverage",
	}, "] [")

	// getCmdFlagSet returns a flag set and list of required flags for the command.
	getCmdFlagSet := func() (fs *flag.FlagSet, reqFlags []string) {
		fs = flag.NewFlagSet("flags-add-genesis-pool", flag.ContinueOnError)

		for _, flagDefinitionArgs := range flagsAddMarketGenesis {
			args := flagDefinitionArgs
			fs.String(args.flagName, args.defaultValue, args.usageDocString)
		}
		return fs, []string{"pair", "base-amt", "quote-amt"}
	}
	cmd := &cobra.Command{
		Use:   fmt.Sprintf("add-genesis-perp-market [%s]", usageExampleTail),
		Short: "Add perp markets to genesis.json",
		Long:  `Add perp markets to genesis.json.`,
		Args:  cobra.ExactArgs(0),
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

			market, err := newMarketFromAddMarketGenesisFlags(cmd.Flags())
			if err != nil {
				return err
			}

			perpAmmGenState := types.GetGenesisStateFromAppState(clientCtx.Codec, appState)
			perpAmmGenState.Markets = append(perpAmmGenState.Markets, market)

			perpAmmGenStateBz, err := clientCtx.Codec.MarshalJSON(perpAmmGenState)
			if err != nil {
				return fmt.Errorf("failed to marshal market genesis state: %w", err)
			}

			appState[types.ModuleName] = perpAmmGenStateBz

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

	flagSet, reqFlags := getCmdFlagSet()
	cmd.Flags().AddFlagSet(flagSet)
	for _, reqFlag := range reqFlags {
		_ = cmd.MarkFlagRequired(reqFlag)
	}

	return cmd
}

func newMarketFromAddMarketGenesisFlags(flagSet *flag.FlagSet,
) (market types.Market, err error) {
	var flagErrors = []error{}
	pairStr, err := flagSet.GetString(FlagPair)
	flagErrors = append(flagErrors, err)

	baseAmtStr, err := flagSet.GetString(FlagBaseAmt)
	flagErrors = append(flagErrors, err)

	quoteAmtStr, err := flagSet.GetString(FlagQuoteAmt)
	flagErrors = append(flagErrors, err)

	pegMultiplierStr, err := flagSet.GetString(FlagPegMultiplier)
	flagErrors = append(flagErrors, err)

	tradeLimStr, err := flagSet.GetString(FlagTradeLim)
	flagErrors = append(flagErrors, err)

	fluctLimStr, err := flagSet.GetString(FlagFluctLim)
	flagErrors = append(flagErrors, err)

	mmrAsString, err := flagSet.GetString(FlagMaintenenceMarginRatio)
	flagErrors = append(flagErrors, err)

	maxLeverageStr, err := flagSet.GetString(FlagMaxLeverage)
	flagErrors = append(flagErrors, err)

	maxOracleSpreadStr, err := flagSet.GetString(FlagMaxOracleSpreadRatio)
	flagErrors = append(flagErrors, err)

	for _, err := range flagErrors { // for brevity's sake
		if err != nil {
			return market, err
		}
	}

	pair, err := asset.TryNewPair(pairStr)
	if err != nil {
		return
	}

	baseAsset, err := sdk.NewDecFromStr(baseAmtStr)
	if err != nil {
		return
	}
	quoteAsset, err := sdk.NewDecFromStr(quoteAmtStr)
	if err != nil {
		return
	}
	tradeLimit, err := sdk.NewDecFromStr(tradeLimStr)
	if err != nil {
		return
	}

	fluctuationLimitRatio, err := sdk.NewDecFromStr(fluctLimStr)
	if err != nil {
		return
	}

	maxOracleSpread, err := sdk.NewDecFromStr(maxOracleSpreadStr)
	if err != nil {
		return
	}

	maintenanceMarginRatio, err := sdk.NewDecFromStr(mmrAsString)
	if err != nil {
		return
	}

	maxLeverage, err := sdk.NewDecFromStr(maxLeverageStr)
	if err != nil {
		return types.Market{}, err
	}

	pegMultiplier, err := sdk.NewDecFromStr(pegMultiplierStr)
	if err != nil {
		return types.Market{}, err
	}

	market = types.Market{
		Pair:          pair,
		QuoteReserve:  quoteAsset,
		BaseReserve:   baseAsset,
		Bias:          sdk.ZeroDec(),
		PegMultiplier: pegMultiplier,
		Config: types.MarketConfig{
			TradeLimitRatio:        tradeLimit,
			FluctuationLimitRatio:  fluctuationLimitRatio,
			MaxOracleSpreadRatio:   maxOracleSpread,
			MaintenanceMarginRatio: maintenanceMarginRatio,
			MaxLeverage:            maxLeverage,
		},
	}
	market, err = market.InitLiqDepth()
	if err != nil {
		return
	}

	return market, market.Validate()
}
