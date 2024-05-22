package types

import (
	"fmt"
	"strings"

	"cosmossdk.io/math"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/set"

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

// NewAggregateExchangeRateVote creates a AggregateExchangeRateVote instance
func NewAggregateExchangeRateVote(exchangeRateTuples ExchangeRateTuples, voter sdk.ValAddress) AggregateExchangeRateVote {
	return AggregateExchangeRateVote{
		ExchangeRateTuples: exchangeRateTuples,
		Voter:              voter.String(),
	}
}

// NewExchangeRateTuple creates a ExchangeRateTuple instance
func NewExchangeRateTuple(pair asset.Pair, exchangeRate math.LegacyDec) ExchangeRateTuple {
	return ExchangeRateTuple{
		pair,
		exchangeRate,
	}
}

// ToString converts the ExchangeRateTuple to the vote string.
func (m ExchangeRateTuple) ToString() (string, error) {
	err := m.Pair.Validate()
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

// NewExchangeRateTupleFromString populates ExchangeRateTuple from a string, fails if the string is of invalid format.
func NewExchangeRateTupleFromString(s string) (ExchangeRateTuple, error) {
	// strip parentheses
	if len(s) <= 2 {
		return ExchangeRateTuple{}, fmt.Errorf("invalid string length: %v", len(s))
	}

	if s[0] != ExchangeRateTupleStringPrefix || s[len(s)-1] != ExchangeRateTupleStringSuffix {
		return ExchangeRateTuple{}, fmt.Errorf("invalid ExchangeRateTuple delimiters, start is expected with '(', end with ')', got: %s", s)
	}

	stripParentheses := s[1 : len(s)-1]
	split := strings.Split(stripParentheses, ExchangeRateTuplePairRateSeparator)
	if len(split) != 2 {
		return ExchangeRateTuple{}, fmt.Errorf("invalid ExchangeRateTuple format")
	}

	pair, err := asset.TryNewPair(split[0])
	if err != nil {
		return ExchangeRateTuple{}, fmt.Errorf("invalid pair definition %s: %w", split[0], err)
	}

	dec, err := math.LegacyNewDecFromStr(split[1])
	if err != nil {
		return ExchangeRateTuple{}, fmt.Errorf("invalid decimal %s: %w", split[1], err)
	}

	return ExchangeRateTuple{
		Pair:         pair,
		ExchangeRate: dec,
	}, nil
}

// ExchangeRateTuples - array of ExchangeRateTuple
type ExchangeRateTuples []ExchangeRateTuple

func (tuples ExchangeRateTuples) ToMap() (exchangeRateMap map[asset.Pair]math.LegacyDec) {
	exchangeRateMap = make(map[asset.Pair]math.LegacyDec)
	for _, tuple := range tuples {
		exchangeRateMap[tuple.Pair] = tuple.ExchangeRate
	}
	return exchangeRateMap
}

func NewExchangeRateTuplesFromString(s string) (ExchangeRateTuples, error) {
	stringTuples := strings.Split(s, ExchangeRateTuplesSeparator)

	tuples := make(ExchangeRateTuples, len(stringTuples))

	duplicates := make(set.Set[asset.Pair], len(stringTuples))

	for i, stringTuple := range stringTuples {
		exchangeRate, err := NewExchangeRateTupleFromString(stringTuple)
		if err != nil {
			return []ExchangeRateTuple{}, fmt.Errorf("invalid ExchangeRateTuple at index %d: %w", i, err)
		}

		// check duplicates
		if _, ok := duplicates[exchangeRate.Pair]; ok {
			return []ExchangeRateTuple{}, fmt.Errorf("found duplicate at index %d: %s", i, exchangeRate.Pair)
		} else {
			duplicates[exchangeRate.Pair] = struct{}{}
		}

		// insert exchange rate into the tuple
		tuples[i] = exchangeRate
	}

	return tuples, nil
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
	tuples, err := NewExchangeRateTuplesFromString(tuplesStr)
	if err != nil {
		return nil, err
	}

	return tuples, nil
}
