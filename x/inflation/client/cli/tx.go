package cli

import (
	"strings"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"

	"github.com/NibiruChain/nibiru/v2/x/inflation/types"
)

// GetTxCmd returns the transaction commands for this module
func GetTxCmd() *cobra.Command {
	inflationTxCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Inflation module subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	inflationTxCmd.AddCommand(
		CmdToggleInflation(),
		CmdEditInflationParams(),
	)

	return inflationTxCmd
}

func CmdToggleInflation() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "toggle-inflation [true | false]",
		Args:  cobra.ExactArgs(1),
		Short: "Toggle inflation on or off",
		Long: strings.TrimSpace(`
Toggle inflation on or off.

Requires sudo permissions.

$ nibid tx inflation toggle-inflation true
`),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgToggleInflation{
				Sender: clientCtx.GetFromAddress().String(),
				Enable: args[0] == "true",
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

func CmdEditInflationParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit-params --staking-proportion [staking-proportion] --community-pool-proportion [community-pool-proportion] --strategic-reserves-proportion [strategic-reserves-proportion] --polynomial-factors [polynomial-factors] --epochs-per-period [epochs-per-period] --periods-per-year [periods-per-year] --max-period [max-period]",
		Args:  cobra.ExactArgs(0),
		Short: "Edit the inflation module parameters",
		Long: strings.TrimSpace(`
Edit the inflation module parameters.

Requires sudo permissions.

--staking-proportion: the proportion of minted tokens to be distributed to stakers
--community-pool-proportion: the proportion of minted tokens to be distributed to the community pool
--strategic-reserves-proportion: the proportion of minted tokens to be distributed to validators

--polynomial-factors: the polynomial factors of the inflation distribution curve
--epochs-per-period: the number of epochs per period
--periods-per-year: the number of periods per year
--max-period: the maximum number of periods

$ nibid tx oracle edit-params --staking-proportion 0.6 --community-pool-proportion 0.2 --strategic-reserves-proportion 0.2 --polynomial-factors 0.1,0.2,0.3,0.4,0.5,0.6 --epochs-per-period 100 --periods-per-year 100 --max-period 100
`),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgEditInflationParams{
				Sender: clientCtx.GetFromAddress().String(),
			}

			var stakingProportionDec sdk.Dec
			if stakingProportion, _ := cmd.Flags().GetString("staking-proportion"); stakingProportion != "" {
				stakingProportionDec = math.LegacyMustNewDecFromStr(stakingProportion)
				msg.InflationDistribution.StakingRewards = stakingProportionDec
			}

			var communityPoolProportionDec sdk.Dec
			if communityPoolProportion, _ := cmd.Flags().GetString("community-pool-proportion"); communityPoolProportion != "" {
				communityPoolProportionDec = math.LegacyMustNewDecFromStr(communityPoolProportion)
				msg.InflationDistribution.CommunityPool = communityPoolProportionDec
			}

			var strategicReservesProportionDec sdk.Dec
			if strategicReservesProportion, _ := cmd.Flags().GetString("strategic-reserves-proportion"); strategicReservesProportion != "" {
				strategicReservesProportionDec = math.LegacyMustNewDecFromStr(strategicReservesProportion)
				msg.InflationDistribution.StrategicReserves = strategicReservesProportionDec
			}

			if !stakingProportionDec.IsNil() && !communityPoolProportionDec.IsNil() && !strategicReservesProportionDec.IsNil() {
				msg.InflationDistribution = &types.InflationDistribution{
					StakingRewards:    stakingProportionDec,
					CommunityPool:     communityPoolProportionDec,
					StrategicReserves: strategicReservesProportionDec,
				}
			}

			if polynomialFactors, _ := cmd.Flags().GetString("polynomial-factors"); polynomialFactors != "" {
				polynomialFactorsArr := strings.Split(polynomialFactors, ",")
				realPolynomialFactors := make([]sdk.Dec, len(polynomialFactorsArr))
				for i, factor := range polynomialFactorsArr {
					factorDec := math.LegacyMustNewDecFromStr(factor)
					realPolynomialFactors[i] = factorDec
				}
				msg.PolynomialFactors = realPolynomialFactors
			}

			if epochsPerPeriod, _ := cmd.Flags().GetUint64("epochs-per-period"); epochsPerPeriod != 0 {
				epochsPerPeriodInt := sdk.NewIntFromUint64(epochsPerPeriod)
				msg.EpochsPerPeriod = &epochsPerPeriodInt
			}

			if periodsPerYear, _ := cmd.Flags().GetUint64("periods-per-year"); periodsPerYear != 0 {
				periodsPerYearInt := sdk.NewIntFromUint64(periodsPerYear)
				msg.PeriodsPerYear = &periodsPerYearInt
			}

			if maxPeriod, _ := cmd.Flags().GetUint64("max-period"); maxPeriod != 0 {
				maxPeriodInt := sdk.NewIntFromUint64(maxPeriod)
				msg.MaxPeriod = &maxPeriodInt
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	cmd.Flags().String("staking-proportion", "", "the proportion of minted tokens to be distributed to stakers")
	cmd.Flags().String("community-pool-proportion", "", "the proportion of minted tokens to be distributed to the community pool")
	cmd.Flags().String("strategic-reserves-proportion", "", "the proportion of minted tokens to be distributed to validators")
	cmd.Flags().String("polynomial-factors", "", "the polynomial factors of the inflation distribution curve")
	cmd.Flags().Uint64("epochs-per-period", 0, "the number of epochs per period")
	cmd.Flags().Uint64("periods-per-year", 0, "the number of periods per year")
	cmd.Flags().Uint64("max-period", 0, "the maximum number of periods")

	return cmd
}
