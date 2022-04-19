package keeper

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCombineKeys(t *testing.T) {
	tests := []struct {
		name     string
		keys     [][]byte
		expected []byte
	}{
		{
			name:     "three keys",
			keys:     [][]byte{{1, 2, 3}, {4, 5, 6}, {7, 8, 9}},
			expected: []byte{1, 2, 3, 255, 4, 5, 6, 255, 7, 8, 9},
		},
		{
			name:     "one key",
			keys:     [][]byte{{1, 2, 3}},
			expected: []byte{1, 2, 3},
		},
		{
			name:     "no keys",
			keys:     [][]byte{},
			expected: []byte{},
		},
		{
			name:     "max byte keys",
			keys:     [][]byte{{255}, {255}},
			expected: []byte{255, 255, 255},
		},
	}

	for _, testcase := range tests {
		tc := testcase
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, combineKeys(tc.keys...))
		})
	}
}

func TestLockStoreKey(t *testing.T) {
	tests := []struct {
		name     string
		lockId   uint64
		expected []byte
	}{
		{
			name:     "lockId is 1",
			lockId:   1,
			expected: []byte{2, 255, 0, 0, 0, 0, 0, 0, 0, 1},
		},
		{
			name:     "lockId is 0",
			lockId:   0,
			expected: []byte{2, 255, 0, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			name:     "lockId is max one byte",
			lockId:   255,
			expected: []byte{2, 255, 0, 0, 0, 0, 0, 0, 0, 255},
		},
		{
			name:     "lockId is greater than one byte",
			lockId:   256,
			expected: []byte{2, 255, 0, 0, 0, 0, 0, 0, 1, 0},
		},
		{
			name:     "lockId is max uint64",
			lockId:   18446744073709551615, // max uint64
			expected: []byte{2, 255, 255, 255, 255, 255, 255, 255, 255, 255},
		},
	}

	for _, testcase := range tests {
		tc := testcase
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, lockStoreKey(tc.lockId))
		})
	}
}
