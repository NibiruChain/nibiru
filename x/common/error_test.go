package common_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/NibiruChain/nibiru/v2/x/common"
	"github.com/NibiruChain/nibiru/v2/x/common/asset"
	"github.com/NibiruChain/nibiru/v2/x/common/denoms"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil"
)

func newErrors(strs ...string) []error {
	var errs []error
	for _, s := range strs {
		errs = append(errs, errors.New(s))
	}
	return errs
}

func TestCombineErrors(t *testing.T) {
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

func TestCombineErrorsGeneric(t *testing.T) {
	testCases := []struct {
		name string
		in   any
		out  error
		fail bool
	}{
		// cases: error
		{name: "type=err | single err unaffected", in: errors.New("err0"), out: errors.New("err0")},
		{name: "type=err | single nil remains nil", in: nil, out: nil},

		// cases: []error
		{name: "type=err | single nil remains nil", in: []error{nil}, out: nil},
		{name: "multiple nil becomes nil", in: []error{nil, nil, nil}, out: nil},
		{
			name: "multiple err coalesces - A",
			in:   newErrors("err0", "err1"),
			out:  errors.New("err0: err1"),
		},
		{
			name: "multiple err coalesces - B",
			in:   newErrors("err0", "err1", "err2", "foobar"),
			out:  errors.New(strings.Join([]string{"err0", "err1", "err2", "foobar"}, ": ")),
		},

		// cases: string
		{name: "type=string | empty string", in: "", out: errors.New("")},
		{name: "type=string | happy string", in: "happy", out: errors.New("happy")},

		// cases: []string
		{name: "type=[]string | empty string", in: []string{""}, out: errors.New("")},
		{name: "type=[]string | empty strings", in: []string{"", ""}, out: errors.New(": ")},
		{name: "type=[]string | mixed", in: []string{"", "abc", ""}, out: errors.New(": abc: ")},

		// cases: fmt.Stringer
		{name: "type=fmt.Stringer |", in: asset.Registry.Pair(denoms.USDC, denoms.NUSD), out: errors.New("uusdc:unusd")},

		// cases: []fmt.Stringer
		{
			name: "type=[]fmt.Stringer | happy",
			in:   []fmt.Stringer{asset.Registry.Pair(denoms.BTC, denoms.NUSD), asset.Registry.Pair(denoms.ETH, denoms.NUSD)},
			out:  errors.New("ubtc:unusd: ueth:unusd"),
		},
		{name: "type=[]fmt.Stringer | empty", in: []fmt.Stringer{}, out: nil},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			out, ok := common.CombineErrorsGeneric(tc.in)
			if tc.fail {
				assert.Falsef(t, ok, "out: %v", out)
			} else {
				assert.EqualValuesf(t, tc.out, out,
					"tc.errOut: %s\nerrOut: %s", tc.out, out)
				assert.True(t, ok)
			}
		})
	}
}

func TestToError(t *testing.T) {
	testCases := []testutil.FunctionTestCase{
		{
			Name: "string nonempty",
			Test: func() {
				description := "an error description"
				out, ok := common.ToError(description)
				assert.True(t, ok)
				assert.EqualValues(t, out.Error(), description)
			},
		},
		{
			Name: "error nonempty",
			Test: func() {
				description := "an error description"
				out, ok := common.ToError(errors.New(description))
				assert.True(t, ok)
				assert.EqualValues(t, out.Error(), description)
			},
		},
		{
			Name: "empty string creates blank error",
			Test: func() {
				description := ""
				out, ok := common.ToError("")
				assert.True(t, ok)
				assert.EqualValues(t, out.Error(), description)
			},
		},
		{
			Name: "fail - bad type",
			Test: func() {
				descriptionOfBadType := int64(2200)
				_, ok := common.ToError(descriptionOfBadType)
				assert.False(t, ok)
			},
		},
		{
			Name: "nil input returns nil",
			Test: func() {
				err, ok := common.ToError(nil)
				assert.True(t, ok)
				assert.Equal(t, nil, err)
			},
		},
		{
			Name: "slice of strings",
			Test: func() {
				err, ok := common.ToError([]string{"abc", "123"})
				assert.True(t, ok)
				assert.Equal(t, errors.New("abc: 123"), err)
			},
		},
		{
			Name: "slice of error",
			Test: func() {
				err, ok := common.ToError(newErrors("abc", "123"))
				assert.True(t, ok)
				assert.Equal(t, errors.New("abc: 123"), err)
			},
		},
	}

	testutil.RunFunctionTests(t, testCases)
}
