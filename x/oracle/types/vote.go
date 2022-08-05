package types

import (
	"fmt"
	"strings"

	"github.com/NibiruChain/nibiru/x/common"

	"gopkg.in/yaml.v2"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	ExchangeRateTuplesSeparator        = "|"
	ExchangeRateTupleStringPrefix      = '('
	ExchangeRateTupleStringSuffix      = ')'
	ExchangeRateTuplePairRateSeparator = ","
)

// NewAggregateExchangeRatePrevote returns AggregateExchangeRatePrevote object
func NewAggregateExchangeRatePrevote(hash AggregateVoteHash, voter sdk.ValAddress, submitBlock uint64) AggregateExchangeRatePrevote {
	return AggregateExchangeRatePrevote{
		Hash:        hash.String(),
		Voter:       voter.String(),
		SubmitBlock: submitBlock,
	}
}

// String implement stringify
func (v AggregateExchangeRatePrevote) String() string {
	out, _ := yaml.Marshal(v)
	return string(out)
}

// NewAggregateExchangeRateVote creates a AggregateExchangeRateVote instance
func NewAggregateExchangeRateVote(exchangeRateTuples ExchangeRateTuples, voter sdk.ValAddress) AggregateExchangeRateVote {
	return AggregateExchangeRateVote{
		ExchangeRateTuples: exchangeRateTuples,
		Voter:              voter.String(),
	}
}

// String implement stringify
func (v AggregateExchangeRateVote) String() string {
	out, _ := yaml.Marshal(v)
	return string(out)
}

// NewExchangeRateTuple creates a ExchangeRateTuple instance
func NewExchangeRateTuple(pair string, exchangeRate sdk.Dec) ExchangeRateTuple {
	return ExchangeRateTuple{
		pair,
		exchangeRate,
	}
}

// String implement stringify
func (m ExchangeRateTuple) String() string {
	out, _ := yaml.Marshal(m)
	return string(out)
}

// ToString converts the ExchangeRateTuple to the vote string.
func (m ExchangeRateTuple) ToString() (string, error) {
	_, err := common.NewAssetPair(m.Pair)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(
		"%c%s%s%s%c",
		ExchangeRateTupleStringPrefix,
		m.Pair,
		ExchangeRateTuplePairRateSeparator,
		m.ExchangeRate.String(),
		ExchangeRateTupleStringSuffix,
	), nil
}

// FromString populates ExchangeRateTuple from a string, fails if the string is of invalid format.
func (m *ExchangeRateTuple) FromString(s string) error {
	// strip parentheses
	if len(s) <= 2 {
		return fmt.Errorf("invalid string length")
	}
	if s[0] != ExchangeRateTupleStringPrefix || s[len(s)-1] != ExchangeRateTupleStringSuffix {
		return fmt.Errorf("invalid ExchangeRateTuple delimiters, start is expected with '(', end with ')', got: %s", s)
	}
	stripParentheses := s[1 : len(s)-1]
	split := strings.Split(stripParentheses, ExchangeRateTuplePairRateSeparator)
	if len(split) != 2 {
		return fmt.Errorf("invalid ExchangeRateTuple format")
	}

	_, err := common.NewAssetPair(split[0])
	if err != nil {
		return fmt.Errorf("invalid pair definition %s: %w", split[0], err)
	}

	dec, err := sdk.NewDecFromStr(split[1])
	if err != nil {
		return fmt.Errorf("invalid decimal %s: %w", split[1], err)
	}

	m.Pair = split[0]
	m.ExchangeRate = dec

	return nil
}

// ExchangeRateTuples - array of ExchangeRateTuple
type ExchangeRateTuples []ExchangeRateTuple

// String implements fmt.Stringer interface
func (tuples ExchangeRateTuples) String() string {
	out, _ := yaml.Marshal(tuples)
	return string(out)
}

func (tuples *ExchangeRateTuples) FromString(s string) error {
	stringTuples := strings.Split(s, ExchangeRateTuplesSeparator)
	*tuples = make([]ExchangeRateTuple, len(stringTuples))
	duplicates := make(map[string]struct{}, len(stringTuples))

	for i, stringTuple := range stringTuples {
		exchangeRate := new(ExchangeRateTuple)
		err := exchangeRate.FromString(stringTuple)
		if err != nil {
			return fmt.Errorf("invalid ExchangeRateTuple at index %d: %w", i, err)
		}

		// check duplicates
		if _, ok := duplicates[exchangeRate.Pair]; ok {
			return fmt.Errorf("found duplicate at index %d: %s", i, exchangeRate.Pair)
		} else {
			duplicates[exchangeRate.Pair] = struct{}{}
		}
		// insert exchange rate into the tuple
		(*tuples)[i] = *exchangeRate
	}

	return nil
}

func (tuples ExchangeRateTuples) ToString() (string, error) {
	tuplesStringSlice := make([]string, len(tuples))
	for i, r := range tuples {
		rStr, err := r.ToString()
		if err != nil {
			return "", fmt.Errorf("invalid ExchangeRateTuple at index %d: %w", i, err)
		}

		tuplesStringSlice[i] = rStr
	}

	return strings.Join(tuplesStringSlice, "|"), nil
}

// ParseExchangeRateTuples ExchangeRateTuple parser
func ParseExchangeRateTuples(tuplesStr string) (ExchangeRateTuples, error) {
	t := new(ExchangeRateTuples)
	if err := t.FromString(tuplesStr); err != nil {
		return nil, err
	}

	return *t, nil
}
