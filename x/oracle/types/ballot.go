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

// NumValidVoters returns the number of voters who actually voted (i.e. did not abstain from voting for a pair).
func (b ExchangeRateBallots) NumValidVoters() uint64 {
	count := 0
	for _, ballot := range b {
		if ballot.ExchangeRate.IsPositive() {
			count++
		}
	}
	return uint64(count)
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
	sort.Sort(pb)
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
	n := 0
	for _, v := range pb {
		// ignore abstain votes in std dev calculation
		if v.ExchangeRate.IsPositive() {
			deviation := v.ExchangeRate.Sub(median)
			sum = sum.Add(deviation.Mul(deviation))
			n += 1
		}
	}

	variance := sum.QuoInt64(int64(n))

	floatNum, _ := strconv.ParseFloat(variance.String(), 64)
	floatNum = math.Sqrt(floatNum)
	standardDeviation, _ = sdk.NewDecFromStr(fmt.Sprintf("%f", floatNum))

	return
}

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

// ValidatorPerformance keeps track of a validator performance in the voting period.
type ValidatorPerformance struct {
	// Tendermint consensus voting power
	Power int64
	// RewardWeight: Weight of rewards the validator should receive in units of
	// consensus power.
	RewardWeight int64
	WinCount     int64 // Number of valid votes for which the validator will be rewarded
	AbstainCount int64 // Number of abstained votes for which there will be no reward or punishment
	MissCount    int64 // Number of invalid/punishable votes
	ValAddress   sdk.ValAddress
}

// NewValidatorPerformance generates a ValidatorPerformance instance.
func NewValidatorPerformance(power int64, recipient sdk.ValAddress) ValidatorPerformance {
	return ValidatorPerformance{
		Power:        power,
		RewardWeight: 0,
		WinCount:     0,
		AbstainCount: 0,
		MissCount:    0,
		ValAddress:   recipient,
	}
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
