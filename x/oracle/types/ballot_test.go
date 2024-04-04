package types_test

import (
	sdkmath "cosmossdk.io/math"
	"fmt"
	"math"
	"sort"
	"strconv"
	"testing"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"

	"github.com/stretchr/testify/require"

	"github.com/cometbft/cometbft/crypto/secp256k1"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/oracle/types"
)

func TestExchangeRateVotesToMap(t *testing.T) {
	tests := struct {
		votes   []types.ExchangeRateVote
		isValid []bool
	}{
		[]types.ExchangeRateVote{
			{
				Voter:        sdk.ValAddress(secp256k1.GenPrivKey().PubKey().Address()),
				Pair:         asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				ExchangeRate: sdkmath.LegacyNewDec(1600),
				Power:        100,
			},
			{
				Voter:        sdk.ValAddress(secp256k1.GenPrivKey().PubKey().Address()),
				Pair:         asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				ExchangeRate: sdkmath.LegacyZeroDec(),
				Power:        100,
			},
			{
				Voter:        sdk.ValAddress(secp256k1.GenPrivKey().PubKey().Address()),
				Pair:         asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				ExchangeRate: sdkmath.LegacyNewDec(1500),
				Power:        100,
			},
		},
		[]bool{true, false, true},
	}

	pb := types.ExchangeRateVotes(tests.votes)
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
	require.NotPanics(t, func() {
		types.ExchangeRateVotes(tests.votes).NumValidVoters()
	})
}

func TestToCrossRate(t *testing.T) {
	data := []struct {
		base     sdkmath.LegacyDec
		quote    sdkmath.LegacyDec
		expected sdkmath.LegacyDec
	}{
		{
			base:     sdkmath.LegacyNewDec(1600),
			quote:    sdkmath.LegacyNewDec(100),
			expected: sdkmath.LegacyNewDec(16),
		},
		{
			base:     sdkmath.LegacyZeroDec(),
			quote:    sdkmath.LegacyNewDec(100),
			expected: sdkmath.LegacyNewDec(16),
		},
		{
			base:     sdkmath.LegacyNewDec(1600),
			quote:    sdkmath.LegacyZeroDec(),
			expected: sdkmath.LegacyNewDec(16),
		},
	}

	pbBase := types.ExchangeRateVotes{}
	pbQuote := types.ExchangeRateVotes{}
	cb := types.ExchangeRateVotes{}
	for _, data := range data {
		valAddr := sdk.ValAddress(secp256k1.GenPrivKey().PubKey().Address())
		if !data.base.IsZero() {
			pbBase = append(pbBase, types.NewExchangeRateVote(data.base, asset.Registry.Pair(denoms.BTC, denoms.NUSD), valAddr, 100))
		}

		pbQuote = append(pbQuote, types.NewExchangeRateVote(data.quote, asset.Registry.Pair(denoms.BTC, denoms.NUSD), valAddr, 100))

		if !data.base.IsZero() && !data.quote.IsZero() {
			cb = append(cb, types.NewExchangeRateVote(data.base.Quo(data.quote), asset.Registry.Pair(denoms.BTC, denoms.NUSD), valAddr, 100))
		} else {
			cb = append(cb, types.NewExchangeRateVote(sdkmath.LegacyZeroDec(), asset.Registry.Pair(denoms.BTC, denoms.NUSD), valAddr, 0))
		}
	}

	basePairPrices := pbBase.ToMap()
	require.Equal(t, cb, pbQuote.ToCrossRate(basePairPrices))

	sort.Sort(cb)
}

func TestSqrt(t *testing.T) {
	num := sdkmath.LegacyNewDecWithPrec(144, 4)
	floatNum, err := strconv.ParseFloat(num.String(), 64)
	require.NoError(t, err)

	floatNum = math.Sqrt(floatNum)
	num, err = sdkmath.LegacyNewDecFromStr(fmt.Sprintf("%f", floatNum))
	require.NoError(t, err)

	require.Equal(t, sdkmath.LegacyNewDecWithPrec(12, 2), num)
}

func TestPBPower(t *testing.T) {
	ctx := sdk.NewContext(nil, tmproto.Header{}, false, nil)
	_, valAccAddrs, sk := types.GenerateRandomTestCase()
	pb := types.ExchangeRateVotes{}
	totalPower := int64(0)

	for i := 0; i < len(sk.Validators()); i++ {
		val, _ := sk.Validator(ctx, valAccAddrs[i])
		power := val.GetConsensusPower(sdk.DefaultPowerReduction)
		vote := types.NewExchangeRateVote(
			sdkmath.LegacyZeroDec(),
			asset.Registry.Pair(denoms.ETH, denoms.NUSD),
			valAccAddrs[i],
			power,
		)

		pb = append(pb, vote)

		require.NotEqual(t, int64(0), vote.Power)

		totalPower += vote.Power
	}

	require.Equal(t, totalPower, pb.Power())

	// Mix in a fake validator, the total power should not have changed.
	pubKey := secp256k1.GenPrivKey().PubKey()
	faceValAddr := sdk.ValAddress(pubKey.Address())
	fakeVote := types.NewExchangeRateVote(
		sdkmath.LegacyOneDec(),
		asset.Registry.Pair(denoms.ETH, denoms.NUSD),
		faceValAddr,
		0,
	)

	pb = append(pb, fakeVote)
	require.Equal(t, totalPower, pb.Power())
}

func TestPBWeightedMedian(t *testing.T) {
	tests := []struct {
		inputs      []int64
		weights     []int64
		isValidator []bool
		median      sdkmath.LegacyDec
	}{
		{
			// Supermajority one number
			[]int64{1, 2, 10, 100000},
			[]int64{1, 1, 100, 1},
			[]bool{true, true, true, true},
			sdkmath.LegacyNewDec(10),
		},
		{
			// Adding fake validator doesn't change outcome
			[]int64{1, 2, 10, 100000, 10000000000},
			[]int64{1, 1, 100, 1, 10000},
			[]bool{true, true, true, true, false},
			sdkmath.LegacyNewDec(10),
		},
		{
			// Tie votes
			[]int64{1, 2, 3, 4},
			[]int64{1, 100, 100, 1},
			[]bool{true, true, true, true},
			sdkmath.LegacyNewDec(2),
		},
		{
			// No votes
			[]int64{},
			[]int64{},
			[]bool{true, true, true, true},
			sdkmath.LegacyZeroDec(),
		},
		{
			// not sorted
			[]int64{2, 1, 10, 100000},
			[]int64{1, 1, 100, 1},
			[]bool{true, true, true, true},
			sdkmath.LegacyNewDec(10),
		},
	}

	for _, tc := range tests {
		pb := types.ExchangeRateVotes{}
		for i, input := range tc.inputs {
			valAddr := sdk.ValAddress(secp256k1.GenPrivKey().PubKey().Address())

			power := tc.weights[i]
			if !tc.isValidator[i] {
				power = 0
			}

			vote := types.NewExchangeRateVote(
				sdkmath.LegacyNewDec(int64(input)),
				asset.Registry.Pair(denoms.ETH, denoms.NUSD),
				valAddr,
				power,
			)

			pb = append(pb, vote)
		}

		require.Equal(t, tc.median, pb.WeightedMedian())
		require.Equal(t, tc.median, pb.WeightedMedianWithAssertion())
	}
}

func TestPBStandardDeviation(t *testing.T) {
	tests := []struct {
		inputs            []float64
		weights           []int64
		isValidator       []bool
		standardDeviation sdkmath.LegacyDec
	}{
		{
			// Supermajority one number
			[]float64{1.0, 2.0, 10.0, 100000.0},
			[]int64{1, 1, 100, 1},
			[]bool{true, true, true, true},
			sdkmath.LegacyMustNewDecFromStr("49995.000362536000000000"),
		},
		{
			// Adding fake validator doesn't change outcome
			[]float64{1.0, 2.0, 10.0, 100000.0, 10000000000},
			[]int64{1, 1, 100, 1, 10000},
			[]bool{true, true, true, true, false},
			sdkmath.LegacyMustNewDecFromStr("4472135950.751005519000000000"),
		},
		{
			// Tie votes
			[]float64{1.0, 2.0, 3.0, 4.0},
			[]int64{1, 100, 100, 1},
			[]bool{true, true, true, true},
			sdkmath.LegacyMustNewDecFromStr("1.224744871000000000"),
		},
		{
			// No votes
			[]float64{},
			[]int64{},
			[]bool{true, true, true, true},
			sdkmath.LegacyNewDecWithPrec(0, 0),
		},
		{
			// Abstain votes are ignored
			[]float64{1.0, 2.0, 10.0, 100000.0, -99999999999.0, 0},
			[]int64{1, 1, 100, 1, 1, 1},
			[]bool{true, true, true, true, true, true},
			sdkmath.LegacyMustNewDecFromStr("49995.000362536000000000"),
		},
	}

	base := math.Pow10(types.OracleDecPrecision)
	for _, tc := range tests {
		pb := types.ExchangeRateVotes{}
		for i, input := range tc.inputs {
			valAddr := sdk.ValAddress(secp256k1.GenPrivKey().PubKey().Address())

			power := tc.weights[i]
			if !tc.isValidator[i] {
				power = 0
			}

			vote := types.NewExchangeRateVote(
				sdkmath.LegacyNewDecWithPrec(int64(input*base), int64(types.OracleDecPrecision)),
				asset.Registry.Pair(denoms.ETH, denoms.NUSD),
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
	exchangeRate, err := sdkmath.LegacyNewDecFromStr("100000000000000000000000000000000000000000000000000000000.0")
	require.NoError(t, err)

	pb := types.ExchangeRateVotes{types.NewExchangeRateVote(
		sdkmath.LegacyZeroDec(),
		asset.Registry.Pair(denoms.ETH, denoms.NUSD),
		valAddr,
		2,
	), types.NewExchangeRateVote(
		exchangeRate,
		asset.Registry.Pair(denoms.ETH, denoms.NUSD),
		valAddr,
		1,
	)}

	require.Equal(t, sdkmath.LegacyZeroDec(), pb.StandardDeviation(pb.WeightedMedianWithAssertion()))
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

func TestValidatorPerformances(t *testing.T) {
	power := int64(42)
	valNames := []string{"val0", "val1", "val2", "val3"}
	perfList := []types.ValidatorPerformance{
		types.NewValidatorPerformance(power, sdk.ValAddress([]byte(valNames[0]))),
		types.NewValidatorPerformance(power, sdk.ValAddress([]byte(valNames[1]))),
		types.NewValidatorPerformance(power, sdk.ValAddress([]byte(valNames[2]))),
		types.NewValidatorPerformance(power, sdk.ValAddress([]byte(valNames[3]))),
	}
	perfs := make(types.ValidatorPerformances)
	for idx, perf := range perfList {
		perfs[valNames[idx]] = perf
	}

	require.NotPanics(t, func() {
		out := perfs.String()
		require.NotEmpty(t, out)

		out = perfs[valNames[0]].String()
		require.NotEmpty(t, out)
	})
}
