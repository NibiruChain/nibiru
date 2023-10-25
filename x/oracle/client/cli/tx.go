package cli

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/NibiruChain/nibiru/x/oracle/types"

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
		Args:  cobra.ExactArgs(7),
		Short: "Edit the oracle module parameters",
		Long: strings.TrimSpace(`
Edit the oracle module parameters.

$ nibid tx oracle edit-params --vote-period 10 --vote-threshold 0.5 --reward-band 0.1 --slash-fraction 0.01 --slash-window 100 --min-valid-per-window 0.6 --whitelist BTC:USD,NIBI:USD
`),
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
