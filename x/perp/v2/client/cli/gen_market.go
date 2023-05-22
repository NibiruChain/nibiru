package cli

import (
	"encoding/json"
	"fmt"
	"time"

	flag "github.com/spf13/pflag"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/spf13/cobra"

	"github.com/NibiruChain/nibiru/x/common/asset"
	epochstypes "github.com/NibiruChain/nibiru/x/epochs/types"
	"github.com/NibiruChain/nibiru/x/perp/v2/types"
)

const (
	FlagPair                   = "pair"
	FlagSqrtDepth              = "sqrt-depth"
	FlagPriceMultiplier        = "price-multiplier"
	FlagPriceFluctuationLimit  = "price-fluct-lim"
	FlagMaintenenceMarginRatio = "mmr"
	FlagMaxLeverage            = "max-leverage"
)

var addMarketGenesisFlags = map[string]struct {
	defaultValue   string
	usageDocString string
}{
	FlagPair:                   {"", "trading pair identifier of the form 'base:quote'. E.g., ueth:unusd"},
	FlagSqrtDepth:              {"", "sqrt k"},
	FlagPriceMultiplier:        {"", "the peg multiplier for the pool"},
	FlagPriceFluctuationLimit:  {"0.1", "percentage that a single open or close position can alter the reserves"},
	FlagMaintenenceMarginRatio: {"0.0625", "maintenance margin ratio"},
	FlagMaxLeverage:            {"10", "maximum leverage for opening a position"},
}

// getCmdFlagSet returns a flag set and list of required flags for the command.
func getCmdFlagSet() (fs *flag.FlagSet, reqFlags []string) {
	fs = flag.NewFlagSet("flags-add-genesis-pool", flag.ContinueOnError)

	for flagName, flagArgs := range addMarketGenesisFlags {
		fs.String(flagName, flagArgs.defaultValue, flagArgs.usageDocString)
	}
	return fs, []string{FlagPair, FlagSqrtDepth, FlagPriceMultiplier}
}

// AddMarketGenesisCmd returns add-market-genesis
func AddMarketGenesisCmd(defaultNodeHome string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-genesis-perp-market",
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

			market, amm, err := newMarketFromFlags(cmd.Flags())
			if err != nil {
				return err
			}

			perpGenState := types.GetGenesisStateFromAppState(clientCtx.Codec, appState)
			perpGenState.Markets = append(perpGenState.Markets, market)
			perpGenState.Amms = append(perpGenState.Amms, amm)

			perpGenStateBz, err := clientCtx.Codec.MarshalJSON(perpGenState)
			if err != nil {
				return fmt.Errorf("failed to marshal market genesis state: %w", err)
			}

			appState[types.ModuleName] = perpGenStateBz

			appStateJSON, err := json.Marshal(appState)
			if err != nil {
				return fmt.Errorf("failed to marshal application genesis state: %w", err)
			}

			genDoc.AppState = appStateJSON
			return genutil.ExportGenesisFile(genDoc, genFile)
		},
	}

	cmd.Flags().String(flags.FlagHome, defaultNodeHome, "The application home directory")

	flagSet, requiredFlags := getCmdFlagSet()
	cmd.Flags().AddFlagSet(flagSet)
	for _, reqFlag := range requiredFlags {
		_ = cmd.MarkFlagRequired(reqFlag)
	}

	return cmd
}

func newMarketFromFlags(flagSet *flag.FlagSet,
) (market types.Market, amm types.AMM, err error) {
	var flagErrors = []error{}

	pairStr, err := flagSet.GetString(FlagPair)
	flagErrors = append(flagErrors, err)

	sqrtDepthStr, err := flagSet.GetString(FlagSqrtDepth)
	flagErrors = append(flagErrors, err)

	priceMultiplierStr, err := flagSet.GetString(FlagPriceMultiplier)
	flagErrors = append(flagErrors, err)

	fluctLimStr, err := flagSet.GetString(FlagPriceFluctuationLimit)
	flagErrors = append(flagErrors, err)

	mmrAsString, err := flagSet.GetString(FlagMaintenenceMarginRatio)
	flagErrors = append(flagErrors, err)

	maxLeverageStr, err := flagSet.GetString(FlagMaxLeverage)
	flagErrors = append(flagErrors, err)

	for _, err := range flagErrors { // for brevity's sake
		if err != nil {
			return types.Market{}, types.AMM{}, err
		}
	}

	pair, err := asset.TryNewPair(pairStr)
	if err != nil {
		return
	}

	sqrtDepth, err := sdk.NewDecFromStr(sqrtDepthStr)
	if err != nil {
		return
	}

	fluctuationLimitRatio, err := sdk.NewDecFromStr(fluctLimStr)
	if err != nil {
		return
	}

	maintenanceMarginRatio, err := sdk.NewDecFromStr(mmrAsString)
	if err != nil {
		return
	}

	maxLeverage, err := sdk.NewDecFromStr(maxLeverageStr)
	if err != nil {
		return types.Market{}, types.AMM{}, err
	}

	priceMultiplier, err := sdk.NewDecFromStr(priceMultiplierStr)
	if err != nil {
		return types.Market{}, types.AMM{}, err
	}

	market = types.Market{
		Pair:                            pair,
		Enabled:                         true,
		PriceFluctuationLimitRatio:      fluctuationLimitRatio,
		MaintenanceMarginRatio:          maintenanceMarginRatio,
		MaxLeverage:                     maxLeverage,
		LatestCumulativePremiumFraction: sdk.ZeroDec(),
		ExchangeFeeRatio:                sdk.MustNewDecFromStr("0.0010"),
		EcosystemFundFeeRatio:           sdk.MustNewDecFromStr("0.0010"),
		LiquidationFeeRatio:             sdk.MustNewDecFromStr("0.0500"),
		PartialLiquidationRatio:         sdk.MustNewDecFromStr("0.5"),
		FundingRateEpochId:              epochstypes.ThirtyMinuteEpochID,
		TwapLookbackWindow:              time.Minute * 30,
		PrepaidBadDebt:                  sdk.NewInt64Coin(pair.QuoteDenom(), 0),
	}
	if err := market.Validate(); err != nil {
		return types.Market{}, types.AMM{}, err
	}

	amm = types.AMM{
		Pair:            pair,
		BaseReserve:     sqrtDepth,
		QuoteReserve:    sqrtDepth,
		SqrtDepth:       sqrtDepth,
		PriceMultiplier: priceMultiplier,
		TotalLong:       sdk.ZeroDec(),
		TotalShort:      sdk.ZeroDec(),
	}
	if err := amm.Validate(); err != nil {
		return types.Market{}, types.AMM{}, err
	}

	return market, amm, nil
}
