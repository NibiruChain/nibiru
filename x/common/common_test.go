package common_test

import (
	"testing"

	"github.com/MatrixDao/matrix/x/common"
	"github.com/stretchr/testify/require"
)

func TestPoolNameFromDenoms(t *testing.T) {
	type TestCase struct {
		name     string
		denoms   []string
		poolName string
	}

	testCases := []TestCase{
		{
			name:     "ATOM:OSMO in correct order",
			denoms:   []string{"atom", "osmo"},
			poolName: "atom:osmo",
		},
		{
			name:     "ATOM:OSMO in wrong order",
			denoms:   []string{"osmo", "atom"},
			poolName: "atom:osmo",
		},
		{
			name:     "X:Y:Z in correct order",
			denoms:   []string{"x", "y", "z"},
			poolName: "x:y:z",
		},
		{
			name:     "X:Y:Z in wrong order",
			denoms:   []string{"z", "x", "y"},
			poolName: "x:y:z",
		},
	}

	executeTest := func(t *testing.T, testCase TestCase) {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			outPoolName := common.PoolNameFromDenoms(tc.denoms)
			require.Equal(t, tc.poolName, outPoolName)
		})
	}

	for _, testCase := range testCases {
		executeTest(t, testCase)
	}
}
