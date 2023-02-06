package types

import (
	"fmt"
	"math"
	"sort"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
)

// NOTE: we don't need to implement proto interface on this file
//       these are not used in store or rpc response

// ExchangeRateBallot is a convenience wrapper to reduce redundant lookup cost
type ExchangeRateBallot struct {
	Pair         asset.Pair
	ExchangeRate sdk.Dec // aka price
	Voter        sdk.ValAddress
	Power        int64 // how much tendermint consensus power this vote should have
}

// NewExchangeRateBallot returns a new ExchangeRateBallot instance
func NewExchangeRateBallot(rate sdk.Dec, pair asset.Pair, voter sdk.ValAddress, power int64) ExchangeRateBallot {
	return ExchangeRateBallot{
		ExchangeRate: rate,
		Pair:         pair,
		Voter:        voter,
		Power:        power,
	}
}

// ExchangeRateBallots is a convenience wrapper around a ExchangeRateVote slice
type ExchangeRateBallots []ExchangeRateBallot

// ToMap return organized exchange rate map by validator
func (pb ExchangeRateBallots) ToMap() map[string]sdk.Dec {
	validatorExchangeRateMap := make(map[string]sdk.Dec)
	for _, vote := range pb {
		if vote.ExchangeRate.IsPositive() {
			validatorExchangeRateMap[string(vote.Voter)] = vote.ExchangeRate
		}
	}

	return validatorExchangeRateMap
}

// ToCrossRate return cross_rate(base/exchange_rate) ballot
func (pb ExchangeRateBallots) ToCrossRate(bases map[string]sdk.Dec) (cb ExchangeRateBallots) {
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
func (b ExchangeRateBallots) Power() int64 {
	totalPower := int64(0)
	for _, vote := range b {
		totalPower += vote.Power
	}

	return totalPower
}

// WeightedMedian returns the median weighted by the power of the ExchangeRateVote.
// CONTRACT: ballot must be sorted
func (pb ExchangeRateBallots) WeightedMedian() sdk.Dec {
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

// WeightedMedianWithAssertion returns the median weighted by the power of the ExchangeRateVote.
func (pb ExchangeRateBallots) WeightedMedianWithAssertion() sdk.Dec {
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
func (pb ExchangeRateBallots) StandardDeviation(median sdk.Dec) (standardDeviation sdk.Dec) {
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

var _ (sort.Interface) = ExchangeRateBallots{}

// Len implements sort.Interface
func (pb ExchangeRateBallots) Len() int {
	return len(pb)
}

// Less reports whether the element with
// index i should sort before the element with index j.
func (pb ExchangeRateBallots) Less(i, j int) bool {
	return pb[i].ExchangeRate.LT(pb[j].ExchangeRate)
}

// Swap implements sort.Interface.
func (pb ExchangeRateBallots) Swap(i, j int) {
	pb[i], pb[j] = pb[j], pb[i]
}

// NumValidators returns the number of validators that voted in the ballot, excluding abstentions.
func (pb ExchangeRateBallots) NumValidators() int {
	count := 0
	for _, vote := range pb {
		if vote.ExchangeRate.IsPositive() {
			count++
		}
	}
	return count
}

type VoteMap map[asset.Pair]ExchangeRateBallots

// ValidatorPerformance keeps track of a validator performance in the voting period.
type ValidatorPerformance struct {
	Power        int64 // tendermint consensus power
	RewardWeight int64 // how much of the rewards this validator should receive, units of consensus power
	WinCount     int64
	ValAddress   sdk.ValAddress
}

type ValidatorPerformances map[string]ValidatorPerformance

// GetTotalRewardWeight returns the sum of the reward weight of all the validators included in the map
func (vp ValidatorPerformances) GetTotalRewardWeight() int64 {
	totalRewardWeight := int64(0)
	for _, validator := range vp {
		totalRewardWeight += validator.RewardWeight
	}

	return totalRewardWeight
}
