package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/NibiruChain/nibiru/x/common/asset"

	"github.com/pkg/errors"

	"github.com/NibiruChain/nibiru/v2/x/oracle/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/spf13/cobra"
)

// GetTxCmd returns the transaction commands for this module
func GetTxCmd() *cobra.Command {
	oracleTxCmd := &cobra.Command{
		Use:                        "oracle",
		Short:                      "Oracle transaction subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	oracleTxCmd.AddCommand(
		GetCmdDelegateFeederPermission(),
		GetCmdAggregateExchangeRatePrevote(),
		GetCmdAggregateExchangeRateVote(),
		GetCmdEditOracleParams(),
	)

	return oracleTxCmd
}

// GetCmdDelegateFeederPermission will create a feeder permission delegation tx and sign it with the given key.
func GetCmdDelegateFeederPermission() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-feeder [feeder]",
		Args:  cobra.ExactArgs(1),
		Short: "Delegate the permission to vote for the oracle to an address",
		Long: strings.TrimSpace(`
Delegate the permission to submit exchange rate votes for the oracle to an address.

Delegation can keep your validator operator key offline and use a separate replaceable key online.

$ nibid tx oracle set-feeder nibi1...

where "nibi1..." is the address you want to delegate your voting rights to.
`),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			// Get from address
			voter := clientCtx.GetFromAddress()

			// The address the right is being delegated from
			validator := sdk.ValAddress(voter)

			feederStr := args[0]
			feeder, err := sdk.AccAddressFromBech32(feederStr)
			if err != nil {
				return err
			}

			msg := types.NewMsgDelegateFeedConsent(validator, feeder)
			if err = msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// GetCmdAggregateExchangeRatePrevote will create a aggregateExchangeRatePrevote tx and sign it with the given key.
func GetCmdAggregateExchangeRatePrevote() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "aggregate-prevote [salt] [exchange-rates] [validator]",
		Args:  cobra.RangeArgs(2, 3),
		Short: "Submit an oracle aggregate prevote for the exchange rates of Nibiru",
		Long: strings.TrimSpace(`
Submit an oracle aggregate prevote for the exchange rates of a pair.
The purpose of aggregate prevote is to hide aggregate exchange rate vote with hash which is formatted 
as hex string in SHA256("{salt}:({pair},{exchange_rate})|...|({pair},{exchange_rate}):{voter}")

# Aggregate Prevote
$ nibid tx oracle aggregate-prevote 1234 (40000.0,BTC:USD)|(1.243,NIBI:USD)

where "BTC:USD,NIBI:USD" is the pair, and "40000.0, 1.243" is the exchange rates expressed in decimal value.

If voting from a voting delegate, set "validator" to the address of the validator to vote on behalf of:
$ nibid tx oracle aggregate-prevote 1234 1234 (40000.0,BTC:USD)|(1.243,NIBI:USD) nibivaloper1...
`),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			salt := args[0]
			exchangeRatesStr := args[1]
			_, err = types.ParseExchangeRateTuples(exchangeRatesStr)
			if err != nil {
				return fmt.Errorf("given exchange_rates {%s} is not a valid format; exchange_rate should be formatted as DecCoins; %s", exchangeRatesStr, err.Error())
			}

			// Get from address
			feeder := clientCtx.GetFromAddress()

			// By default, the feeder is voting on behalf of itself
			validator := sdk.ValAddress(feeder)

			// Override validator if validator is given
			if len(args) == 3 {
				parsedVal, err := sdk.ValAddressFromBech32(args[2])
				if err != nil {
					return errors.Wrap(err, "validator address is invalid")
				}

				validator = parsedVal
			}

			hash := types.GetAggregateVoteHash(salt, exchangeRatesStr, validator)

			msg := types.NewMsgAggregateExchangeRatePrevote(hash, feeder, validator)
			if err = msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// GetCmdAggregateExchangeRateVote will create a aggregateExchangeRateVote tx and sign it with the given key.
func GetCmdAggregateExchangeRateVote() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "aggregate-vote [salt] [exchange-rates] [validator]",
		Args:  cobra.RangeArgs(2, 3),
		Short: "Submit an oracle aggregate vote for the exchange_rates of Nibiru",
		Long: strings.TrimSpace(`
Submit an aggregate vote for the exchange_rates of the proposed pairs. Companion to a prevote submitted in the previous vote period. 

$ nibid tx oracle aggregate-vote 1234 (40000.0,BTC:USD)|(1.243,NIBI:USD)

where "BTC:USD, NIBI:USD" is the pairs, and "40000.0,1.243" is the exchange rates as decimal string.

"salt" should match the salt used to generate the SHA256 hex in the aggregated pre-vote. 

If voting from a voting delegate, set "validator" to the address of the validator to vote on behalf of:
$ nibid tx oracle aggregate-vote 1234 (40000.0,BTC:USD)|(1.243,NIBI:USD) nibivaloper1....
`),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			salt := args[0]
			exchangeRatesStr := args[1]
			_, err = types.ParseExchangeRateTuples(exchangeRatesStr)
			if err != nil {
				return fmt.Errorf("given exchange_rate {%s} is not a valid format; exchange rate should be formatted as DecCoin; %s", exchangeRatesStr, err.Error())
			}

			// Get from address
			feeder := clientCtx.GetFromAddress()

			// By default, the feeder is voting on behalf of itself
			validator := sdk.ValAddress(feeder)

			// Override validator if validator is given
			if len(args) == 3 {
				parsedVal, err := sdk.ValAddressFromBech32(args[2])
				if err != nil {
					return errors.Wrap(err, "validator address is invalid")
				}
				validator = parsedVal
			}

			msg := types.NewMsgAggregateExchangeRateVote(salt, exchangeRatesStr, feeder, validator)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

func GetCmdEditOracleParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit-params --vote-period [vote-period] --vote-threshold [vote-threshold] --reward-band [reward-band] --slash-fraction [slash-fraction] --slash-window [slash-window] --min-valid-per-window [min-valid-per-window] --whitelist [whitelist]",
		Args:  cobra.ExactArgs(0),
		Short: "Edit the oracle module parameters",
		Long: strings.TrimSpace(`
Edit the oracle module parameters.

Requires sudo permissions.

--vote-period: the period of oracle vote
--vote-threshold: the threshold of oracle vote
--reward-band: the reward band of oracle vote
--slash-fraction: the slash fraction of oracle vote
--slash-window: the slash window of oracle vote
--min-valid-per-window: the min valid per window of oracle vote
--twap-lookback-window: the twap lookback window of oracle vote in seconds
--min-voters: the min voters of oracle vote
--validator-fee-ratio: the validator fee ratio of oracle vote
--expiration-blocks: the expiration blocks of oracle vote
--whitelist: the whitelist of oracle vote

$ nibid tx oracle edit-params --vote-period 10 --vote-threshold 0.5 --reward-band 0.1 --slash-fraction 0.01 --slash-window 100 --min-valid-per-window 0.6 --whitelist BTC:USD,NIBI:USD
`),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgEditOracleParams{
				Sender: clientCtx.GetFromAddress().String(),
				Params: &types.OracleParamsMsg{},
			}

			if votePeriod, _ := cmd.Flags().GetUint64("vote-period"); votePeriod != 0 {
				msg.Params.VotePeriod = votePeriod
			}

			if voteThreshold, _ := cmd.Flags().GetString("vote-threshold"); voteThreshold != "" {
				voteThresholdDec, err := sdk.NewDecFromStr(voteThreshold)
				if err != nil {
					return err
				}

				msg.Params.VoteThreshold = &voteThresholdDec
			}

			if rewardBand, _ := cmd.Flags().GetString("reward-band"); rewardBand != "" {
				rewardBandDec, err := sdk.NewDecFromStr(rewardBand)
				if err != nil {
					return err
				}

				msg.Params.RewardBand = &rewardBandDec
			}

			if slashFraction, _ := cmd.Flags().GetString("slash-fraction"); slashFraction != "" {
				slashFractionDec, err := sdk.NewDecFromStr(slashFraction)
				if err != nil {
					return err
				}

				msg.Params.SlashFraction = &slashFractionDec
			}

			if slashWindow, _ := cmd.Flags().GetUint64("slash-window"); slashWindow != 0 {
				msg.Params.SlashWindow = slashWindow
			}

			if minValidPerWindow, _ := cmd.Flags().GetString("min-valid-per-window"); minValidPerWindow != "" {
				minValidPerWindowDec, err := sdk.NewDecFromStr(minValidPerWindow)
				if err != nil {
					return err
				}

				msg.Params.MinValidPerWindow = &minValidPerWindowDec
			}

			if twapLookbackWindow, _ := cmd.Flags().GetUint64("twap-lookback-window"); twapLookbackWindow != 0 {
				duration := time.Duration(twapLookbackWindow) * time.Second
				msg.Params.TwapLookbackWindow = &duration
			}

			if minVoters, _ := cmd.Flags().GetUint64("min-voters"); minVoters != 0 {
				msg.Params.MinVoters = minVoters
			}

			if validatorFeeRatio, _ := cmd.Flags().GetString("validator-fee-ratio"); validatorFeeRatio != "" {
				validatorFeeRatioDec, err := sdk.NewDecFromStr(validatorFeeRatio)
				if err != nil {
					return err
				}

				msg.Params.ValidatorFeeRatio = &validatorFeeRatioDec
			}

			if expirationBlocks, _ := cmd.Flags().GetUint64("expiration-blocks"); expirationBlocks != 0 {
				msg.Params.ExpirationBlocks = expirationBlocks
			}

			if whitelist, _ := cmd.Flags().GetString("whitelist"); whitelist != "" {
				whitelistArr := strings.Split(whitelist, ",")
				realWhitelist := make([]asset.Pair, len(whitelistArr))
				for i, pair := range whitelistArr {
					p, err := asset.TryNewPair(pair)
					if err != nil {
						return fmt.Errorf("invalid pair %s", p)
					}

					realWhitelist[i] = p
				}

				msg.Params.Whitelist = realWhitelist
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	cmd.Flags().Uint64("vote-period", 0, "the period of oracle vote")
	cmd.Flags().String("vote-threshold", "", "the threshold of oracle vote")
	cmd.Flags().String("reward-band", "", "the reward band of oracle vote")
	cmd.Flags().String("slash-fraction", "", "the slash fraction of oracle vote")
	cmd.Flags().Uint64("slash-window", 0, "the slash window of oracle vote")
	cmd.Flags().String("min-valid-per-window", "", "the min valid per window of oracle vote")
	cmd.Flags().Uint64("twap-lookback-window", 0, "the twap lookback window of oracle vote")
	cmd.Flags().Uint64("min-voters", 0, "the min voters of oracle vote")
	cmd.Flags().String("validator-fee-ratio", "", "the validator fee ratio of oracle vote")
	cmd.Flags().Uint64("expiration-blocks", 0, "the expiration blocks of oracle vote")
	cmd.Flags().String("whitelist", "", "the whitelist of oracle vote")

	return cmd
}
