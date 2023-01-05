package types

import (
	"fmt"
	"math"
	"sort"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// NOTE: we don't need to implement proto interface on this file
//       these are not used in store or rpc response

// BallotVoteForTally is a convenience wrapper to reduce redundant lookup cost
type BallotVoteForTally struct {
	Pair         string
	ExchangeRate sdk.Dec
	Voter        sdk.ValAddress
	Power        int64
}

// NewBallotVoteForTally returns a new VoteForTally instance
func NewBallotVoteForTally(rate sdk.Dec, pair string, voter sdk.ValAddress, power int64) BallotVoteForTally {
	return BallotVoteForTally{
		ExchangeRate: rate,
		Pair:         pair,
		Voter:        voter,
		Power:        power,
	}
}

// ExchangeRateBallot is a convenience wrapper around a ExchangeRateVote slice
type ExchangeRateBallot []BallotVoteForTally

// ToMap return organized exchange rate map by validator
func (pb ExchangeRateBallot) ToMap() map[string]sdk.Dec {
	exchangeRateMap := make(map[string]sdk.Dec)
	for _, vote := range pb {
		if vote.ExchangeRate.IsPositive() {
			exchangeRateMap[string(vote.Voter)] = vote.ExchangeRate
		}
	}

	return exchangeRateMap
}

// ToCrossRate return cross_rate(base/exchange_rate) ballot
func (pb ExchangeRateBallot) ToCrossRate(bases map[string]sdk.Dec) (cb ExchangeRateBallot) {
	for i := range pb {
		vote := pb[i]

		if exchangeRateRT, ok := bases[string(vote.Voter)]; ok && vote.ExchangeRate.IsPositive() {
			vote.ExchangeRate = exchangeRateRT.Quo(vote.ExchangeRate)
		} else {
			// If we can't get reference exchange rate, we just convert the vote as abstain vote
			vote.ExchangeRate = sdk.ZeroDec()
			vote.Power = 0
		}

		cb = append(cb, vote)
	}

	return
}

// Power returns the total amount of voting power in the ballot
func (pb ExchangeRateBallot) Power() int64 {
	totalPower := int64(0)
	for _, vote := range pb {
		totalPower += vote.Power
	}

	return totalPower
}

// WeightedMedianWithAssertion returns the median weighted by the power of the ExchangeRateVote.
//
// It fails if the ballot is not sorted.
func (pb ExchangeRateBallot) WeightedMedianWithAssertion() sdk.Dec {
	if !sort.IsSorted(pb) {
		panic("ballot must be sorted")
	}

	totalPower := pb.Power()
	if pb.Len() > 0 {
		pivot := int64(0)
		for _, v := range pb {
			votePower := v.Power

			pivot += votePower
			if pivot >= (totalPower / 2) {
				return v.ExchangeRate
			}
		}
	}
	return sdk.ZeroDec()
}

// StandardDeviation returns the standard deviation by the power of the ExchangeRateVote.
func (pb ExchangeRateBallot) StandardDeviation(median sdk.Dec) (standardDeviation sdk.Dec) {
	if len(pb) == 0 {
		return sdk.ZeroDec()
	}

	defer func() {
		if e := recover(); e != nil {
			standardDeviation = sdk.ZeroDec()
		}
	}()

	sum := sdk.ZeroDec()
	for _, v := range pb {
		deviation := v.ExchangeRate.Sub(median)
		sum = sum.Add(deviation.Mul(deviation))
	}

	variance := sum.QuoInt64(int64(len(pb)))

	floatNum, _ := strconv.ParseFloat(variance.String(), 64)
	floatNum = math.Sqrt(floatNum)
	standardDeviation, _ = sdk.NewDecFromStr(fmt.Sprintf("%f", floatNum))

	return
}

// Len implements sort.Interface
func (pb ExchangeRateBallot) Len() int {
	return len(pb)
}

// Less reports whether the element with
// index i should sort before the element with index j.
func (pb ExchangeRateBallot) Less(i, j int) bool {
	return pb[i].ExchangeRate.LT(pb[j].ExchangeRate)
}

// Swap implements sort.Interface.
func (pb ExchangeRateBallot) Swap(i, j int) {
	pb[i], pb[j] = pb[j], pb[i]
}

// ValidatorPerformance keeps track of a validator performance in the voting period.
type ValidatorPerformance struct {
	Power      int64
	Weight     int64
	WinCount   int64
	ValAddress sdk.ValAddress
}

// NewValidatorPerformance generates a ValidatorPerformance instance.
func NewValidatorPerformance(power int64, recipient sdk.ValAddress) ValidatorPerformance {
	return ValidatorPerformance{
		Power:      power,
		Weight:     0,
		WinCount:   0,
		ValAddress: recipient,
	}
}

// GetValidatorWeightSum returns the sum of the weight of all the validators included in the map
func GetValidatorWeightSum(validatorList map[string]ValidatorPerformance) int64 {
	ballotPowerSum := int64(0)
	for _, winner := range validatorList {
		ballotPowerSum += winner.Weight
	}

	return ballotPowerSum
}
