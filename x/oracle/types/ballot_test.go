package types_test

import (
	"fmt"
	"math"
	"sort"
	"strconv"

	"github.com/NibiruChain/nibiru/x/common"

	"testing"

	"github.com/stretchr/testify/require"

	"github.com/tendermint/tendermint/crypto/secp256k1"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/oracle/types"
)

func TestToMap(t *testing.T) {
	tests := struct {
		votes   []types.ExchangeRateBallot
		isValid []bool
	}{

		[]types.ExchangeRateBallot{
			{

				Voter:        sdk.ValAddress(secp256k1.GenPrivKey().PubKey().Address()),
				Pair:         common.AssetRegistry.Pair(common.DenomBTC, common.DenomNUSD),
				ExchangeRate: sdk.NewDec(1600),
				Power:        100,
			},
			{

				Voter:        sdk.ValAddress(secp256k1.GenPrivKey().PubKey().Address()),
				Pair:         common.AssetRegistry.Pair(common.DenomBTC, common.DenomNUSD),
				ExchangeRate: sdk.ZeroDec(),
				Power:        100,
			},
			{

				Voter:        sdk.ValAddress(secp256k1.GenPrivKey().PubKey().Address()),
				Pair:         common.AssetRegistry.Pair(common.DenomBTC, common.DenomNUSD),
				ExchangeRate: sdk.NewDec(1500),
				Power:        100,
			},
		},
		[]bool{true, false, true},
	}

	pb := types.ExchangeRateBallots(tests.votes)
	mapData := pb.ToMap()
	for i, vote := range tests.votes {
		exchangeRate, ok := mapData[string(vote.Voter)]
		if tests.isValid[i] {
			require.True(t, ok)
			require.Equal(t, exchangeRate, vote.ExchangeRate)
		} else {
			require.False(t, ok)
		}
	}
}

func TestToCrossRate(t *testing.T) {
	data := []struct {
		base     sdk.Dec
		quote    sdk.Dec
		expected sdk.Dec
	}{
		{
			base:     sdk.NewDec(1600),
			quote:    sdk.NewDec(100),
			expected: sdk.NewDec(16),
		},
		{
			base:     sdk.NewDec(0),
			quote:    sdk.NewDec(100),
			expected: sdk.NewDec(16),
		},
		{
			base:     sdk.NewDec(1600),
			quote:    sdk.NewDec(0),
			expected: sdk.NewDec(16),
		},
	}

	pbBase := types.ExchangeRateBallots{}
	pbQuote := types.ExchangeRateBallots{}
	cb := types.ExchangeRateBallots{}
	for _, data := range data {
		valAddr := sdk.ValAddress(secp256k1.GenPrivKey().PubKey().Address())
		if !data.base.IsZero() {
			pbBase = append(pbBase, types.NewExchangeRateBallot(data.base, common.AssetRegistry.Pair(common.DenomBTC, common.DenomNUSD), valAddr, 100))
		}

		pbQuote = append(pbQuote, types.NewExchangeRateBallot(data.quote, common.AssetRegistry.Pair(common.DenomBTC, common.DenomNUSD), valAddr, 100))

		if !data.base.IsZero() && !data.quote.IsZero() {
			cb = append(cb, types.NewExchangeRateBallot(data.base.Quo(data.quote), common.AssetRegistry.Pair(common.DenomBTC, common.DenomNUSD), valAddr, 100))
		} else {
			cb = append(cb, types.NewExchangeRateBallot(sdk.ZeroDec(), common.AssetRegistry.Pair(common.DenomBTC, common.DenomNUSD), valAddr, 0))
		}
	}

	baseMapBallot := pbBase.ToMap()
	require.Equal(t, cb, pbQuote.ToCrossRate(baseMapBallot))

	sort.Sort(cb)
}

func TestSqrt(t *testing.T) {
	num := sdk.NewDecWithPrec(144, 4)
	floatNum, err := strconv.ParseFloat(num.String(), 64)
	require.NoError(t, err)

	floatNum = math.Sqrt(floatNum)
	num, err = sdk.NewDecFromStr(fmt.Sprintf("%f", floatNum))
	require.NoError(t, err)

	require.Equal(t, sdk.NewDecWithPrec(12, 2), num)
}

func TestPBPower(t *testing.T) {
	ctx := sdk.NewContext(nil, tmproto.Header{}, false, nil)
	_, valAccAddrs, sk := types.GenerateRandomTestCase()
	pb := types.ExchangeRateBallots{}
	ballotPower := int64(0)

	for i := 0; i < len(sk.Validators()); i++ {
		power := sk.Validator(ctx, valAccAddrs[i]).GetConsensusPower(sdk.DefaultPowerReduction)
		vote := types.NewExchangeRateBallot(
			sdk.ZeroDec(),
			common.AssetRegistry.Pair(common.DenomETH, common.DenomNUSD),
			valAccAddrs[i],
			power,
		)

		pb = append(pb, vote)

		require.NotEqual(t, int64(0), vote.Power)

		ballotPower += vote.Power
	}

	require.Equal(t, ballotPower, pb.Power())

	// Mix in a fake validator, the total power should not have changed.
	pubKey := secp256k1.GenPrivKey().PubKey()
	faceValAddr := sdk.ValAddress(pubKey.Address())
	fakeVote := types.NewExchangeRateBallot(
		sdk.OneDec(),
		common.AssetRegistry.Pair(common.DenomETH, common.DenomNUSD),
		faceValAddr,
		0,
	)

	pb = append(pb, fakeVote)
	require.Equal(t, ballotPower, pb.Power())
}

func TestPBWeightedMedian(t *testing.T) {
	tests := []struct {
		inputs      []int64
		weights     []int64
		isValidator []bool
		median      sdk.Dec
		panic       bool
	}{
		{
			// Supermajority one number
			[]int64{1, 2, 10, 100000},
			[]int64{1, 1, 100, 1},
			[]bool{true, true, true, true},
			sdk.NewDec(10),
			false,
		},
		{
			// Adding fake validator doesn't change outcome
			[]int64{1, 2, 10, 100000, 10000000000},
			[]int64{1, 1, 100, 1, 10000},
			[]bool{true, true, true, true, false},
			sdk.NewDec(10),
			false,
		},
		{
			// Tie votes
			[]int64{1, 2, 3, 4},
			[]int64{1, 100, 100, 1},
			[]bool{true, true, true, true},
			sdk.NewDec(2),
			false,
		},
		{
			// No votes
			[]int64{},
			[]int64{},
			[]bool{true, true, true, true},
			sdk.NewDec(0),
			false,
		},
		{
			// not sorted panic
			[]int64{2, 1, 10, 100000},
			[]int64{1, 1, 100, 1},
			[]bool{true, true, true, true},
			sdk.NewDec(10),
			true,
		},
	}

	for _, tc := range tests {
		pb := types.ExchangeRateBallots{}
		for i, input := range tc.inputs {
			valAddr := sdk.ValAddress(secp256k1.GenPrivKey().PubKey().Address())

			power := tc.weights[i]
			if !tc.isValidator[i] {
				power = 0
			}

			vote := types.NewExchangeRateBallot(
				sdk.NewDec(int64(input)),
				common.AssetRegistry.Pair(common.DenomETH, common.DenomNUSD),
				valAddr,
				power,
			)

			pb = append(pb, vote)
		}

		if tc.panic {
			require.Panics(t, func() { pb.WeightedMedianWithAssertion() })
		} else {
			require.Equal(t, tc.median, pb.WeightedMedian())
			require.Equal(t, tc.median, pb.WeightedMedianWithAssertion())
		}
	}
}

func TestPBStandardDeviation(t *testing.T) {
	tests := []struct {
		inputs            []float64
		weights           []int64
		isValidator       []bool
		standardDeviation sdk.Dec
	}{
		{
			// Supermajority one number
			[]float64{1.0, 2.0, 10.0, 100000.0},
			[]int64{1, 1, 100, 1},
			[]bool{true, true, true, true},
			sdk.NewDecWithPrec(4999500036300, types.OracleDecPrecision),
		},
		{
			// Adding fake validator doesn't change outcome
			[]float64{1.0, 2.0, 10.0, 100000.0, 10000000000},
			[]int64{1, 1, 100, 1, 10000},
			[]bool{true, true, true, true, false},
			sdk.NewDecWithPrec(447213595075100600, types.OracleDecPrecision),
		},
		{
			// Tie votes
			[]float64{1.0, 2.0, 3.0, 4.0},
			[]int64{1, 100, 100, 1},
			[]bool{true, true, true, true},
			sdk.NewDecWithPrec(122474500, types.OracleDecPrecision),
		},
		{
			// No votes
			[]float64{},
			[]int64{},
			[]bool{true, true, true, true},
			sdk.NewDecWithPrec(0, 0),
		},
	}

	base := math.Pow10(types.OracleDecPrecision)
	for _, tc := range tests {
		pb := types.ExchangeRateBallots{}
		for i, input := range tc.inputs {
			valAddr := sdk.ValAddress(secp256k1.GenPrivKey().PubKey().Address())

			power := tc.weights[i]
			if !tc.isValidator[i] {
				power = 0
			}

			vote := types.NewExchangeRateBallot(
				sdk.NewDecWithPrec(int64(input*base), int64(types.OracleDecPrecision)),
				common.AssetRegistry.Pair(common.DenomETH, common.DenomNUSD),
				valAddr,
				power,
			)

			pb = append(pb, vote)
		}

		require.Equal(t, tc.standardDeviation, pb.StandardDeviation(pb.WeightedMedianWithAssertion()))
	}
}

func TestPBStandardDeviationOverflow(t *testing.T) {
	valAddr := sdk.ValAddress(secp256k1.GenPrivKey().PubKey().Address())
	exchangeRate, err := sdk.NewDecFromStr("100000000000000000000000000000000000000000000000000000000.0")
	require.NoError(t, err)

	pb := types.ExchangeRateBallots{types.NewExchangeRateBallot(
		sdk.ZeroDec(),
		common.AssetRegistry.Pair(common.DenomETH, common.DenomNUSD),
		valAddr,
		2,
	), types.NewExchangeRateBallot(
		exchangeRate,
		common.AssetRegistry.Pair(common.DenomETH, common.DenomNUSD),
		valAddr,
		1,
	)}

	require.Equal(t, sdk.ZeroDec(), pb.StandardDeviation(pb.WeightedMedianWithAssertion()))
}

func TestNewClaim(t *testing.T) {
	power := int64(10)
	weight := int64(11)
	winCount := int64(1)
	addr := sdk.ValAddress(secp256k1.GenPrivKey().PubKey().Address().Bytes())
	claim := types.ValidatorPerformance{
		Power:        power,
		RewardWeight: weight,
		WinCount:     winCount,
		ValAddress:   addr,
	}
	require.Equal(t, types.ValidatorPerformance{
		Power:        power,
		RewardWeight: weight,
		WinCount:     winCount,
		ValAddress:   addr,
	}, claim)
}
