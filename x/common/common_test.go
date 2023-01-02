package common_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
)

type FunctionTestCase struct {
	name string
	test func()
}

func RunFunctionTests(t *testing.T, testCases []FunctionTestCase) {
	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			tc.test()
		})
	}
}

func TestNewAssetPair_Constructor(t *testing.T) {
	tests := []struct {
		name      string
		tokenPair string
		err       error
	}{
		{
			"only one token",
			common.DenomNIBI,
			common.ErrInvalidTokenPair,
		},
		{
			"more than 2 tokens",
			fmt.Sprintf("%s%s%s%s%s", common.DenomNIBI, common.PairSeparator, common.DenomNUSD,
				common.PairSeparator, common.DenomUSDC),
			common.ErrInvalidTokenPair,
		},
		{
			"different separator",
			fmt.Sprintf("%s%s%s", common.DenomNIBI, "%", common.DenomNUSD),
			common.ErrInvalidTokenPair,
		},
		{
			"correct pair",
			fmt.Sprintf("%s%s%s", common.DenomNIBI, common.PairSeparator, common.DenomNUSD),
			nil,
		},
		{
			"empty token identifier",
			fmt.Sprintf("%s%s%s", "", common.PairSeparator, "eth"),
			fmt.Errorf("empty token identifiers are not allowed"),
		},
		{
			"invalid denom 1",
			fmt.Sprintf("-invalid1%svalid", common.PairSeparator),
			fmt.Errorf("invalid denom"),
		},
		{
			"invalid denom 2",
			fmt.Sprintf("valid%s-invalid2", common.PairSeparator),
			fmt.Errorf("invalid denom"),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, err := common.NewAssetPair(tc.tokenPair)
			if tc.err != nil {
				require.ErrorContains(t, err, tc.err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestAsset_GetQuoteBaseToken(t *testing.T) {
	pair, err := common.NewAssetPair("uatom:unibi")
	require.NoError(t, err)

	require.Equal(t, "uatom", pair.BaseDenom())
	require.Equal(t, "unibi", pair.QuoteDenom())
}

func TestAssetPair_Marshaling(t *testing.T) {
	testCases := []FunctionTestCase{
		{
			name: "verbose equal suite",
			test: func() {
				pair := common.MustNewAssetPair("abc:xyz")
				matchingOther := common.MustNewAssetPair("abc:xyz")
				mismatchToken1 := common.MustNewAssetPair("abc:abc")
				inversePair := common.MustNewAssetPair("xyz:abc")

				require.NoError(t, (&pair).VerboseEqual(&matchingOther))
				require.True(t, (&pair).Equal(&matchingOther))

				require.Error(t, (&pair).VerboseEqual(&inversePair))
				require.False(t, (&pair).Equal(&inversePair))

				require.Error(t, (&pair).VerboseEqual(&mismatchToken1))
				require.True(t, !(&pair).Equal(&mismatchToken1))

				require.Error(t, (&pair).VerboseEqual(pair.String()))
				require.False(t, (&pair).Equal(&mismatchToken1))
			},
		},
		{
			name: "panics suite",
			test: func() {
				require.Panics(t, func() {
					common.MustNewAssetPair("aaa:bbb:ccc")
				})
			},
		},
	}

	RunFunctionTests(t, testCases)
}

func TestCombineErrors(t *testing.T) {
	newErrors := func(strs ...string) []error {
		var errs []error
		for _, s := range strs {
			errs = append(errs, errors.New(s))
		}
		return errs
	}

	testCases := []struct {
		name   string
		errs   []error
		errOut error
	}{
		{name: "single nil remains nil", errs: []error{nil}, errOut: nil},
		{name: "multiple nil becomes nil", errs: []error{nil, nil, nil}, errOut: nil},
		{name: "single err unaffected", errs: newErrors("err0"), errOut: errors.New("err0")},
		{
			name:   "multiple err coalesces - A",
			errs:   newErrors("err0", "err1"),
			errOut: errors.New("err0: err1"),
		},
		{
			name:   "multiple err coalesces - B",
			errs:   newErrors("err0", "err1", "err2", "foobar"),
			errOut: errors.New(strings.Join([]string{"err0", "err1", "err2", "foobar"}, ": ")),
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			errOut := common.CombineErrors(tc.errs...)
			assert.EqualValuesf(t, tc.errOut, errOut,
				"tc.errOut: %s\nerrOut: %s", tc.errOut, errOut)
		})
	}
}

func TestCombineErrorsFromStrings(t *testing.T) {
	// REALUTODO
	testCases := []FunctionTestCase{
		{name: "", test: func() {}},
	}

	RunFunctionTests(t, testCases)
}

func TestToError(t *testing.T) {
	testCases := []FunctionTestCase{
		{
			name: "string nonempty",
			test: func() {
				description := "an error description"
				out := common.ToError(description)
				assert.EqualValues(t, out.Error(), description)
			},
		},
		{
			name: "error nonempty",
			test: func() {
				description := "an error description"
				out := common.ToError(errors.New(description))
				assert.EqualValues(t, out.Error(), description)
			},
		},
		{
			name: "empty string creates blank error",
			test: func() {
				description := ""
				out := common.ToError("")
				assert.EqualValues(t, out.Error(), description)
			},
		},
		{
			name: "fail - bad type",
			test: func() {
				descriptionOfBadType := int64(2200)
				assert.Panics(t, func() {
					_ = common.ToError(descriptionOfBadType)
				})
			},
		},
		{
			name: "nil input returns nil",
			test: func() {
				assert.Equal(t, nil, common.ToError(nil))
			},
		},
	}

	RunFunctionTests(t, testCases)
}
