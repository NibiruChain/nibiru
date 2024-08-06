package eth_test

import (
	fmt "fmt"
	"math/big"
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/eth"
)

const maxInt64 = 9223372036854775807

type Suite struct {
	suite.Suite
}

func TestSuite_RunAll(t *testing.T) {
	suite.Run(t, new(Suite))
}

func (s *Suite) TestSafeNewIntFromBigInt() {
	tests := []struct {
		name      string
		input     *big.Int
		expectErr bool
	}{
		{
			name:      "Valid input within 256-bit limit",
			input:     big.NewInt(maxInt64), // Use max int64 as a valid test case
			expectErr: false,
		},
		{
			name:      "Invalid input exceeding 256-bit limit",
			input:     new(big.Int).Lsh(big.NewInt(1), 257), // Shift 1 left by 257 places, creating a 258-bit number
			expectErr: true,
		},
		{
			name:      "Nil input",
			input:     nil,
			expectErr: true,
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			result, err := eth.SafeNewIntFromBigInt(tc.input)
			if tc.expectErr {
				s.Error(err, fmt.Sprintf("result: %s", result))
			} else {
				s.NoError(err, fmt.Sprintf("result: %s", result))
				s.Equal(math.NewIntFromBigInt(tc.input), result, "The results should be equal")
			}
		})
	}
}

func (s *Suite) TestIsValidInt256() {
	tests := []struct {
		name        string
		input       *big.Int
		expectValid bool
	}{
		{
			name:        "Valid 256-bit number",
			input:       new(big.Int).Lsh(big.NewInt(1), 255), // Exactly 256-bit number
			expectValid: true,
		},
		{
			name:        "Invalid 257-bit number",
			input:       new(big.Int).Lsh(big.NewInt(1), 256), // 257-bit number
			expectValid: false,
		},
		{
			name:        "Nil input",
			input:       nil,
			expectValid: true,
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			result := eth.IsValidInt256(tc.input)
			s.Equal(tc.expectValid, result, "Validity check did not match expected")
		})
	}
}

func (s *Suite) TestSafeInt64() {
	tests := []struct {
		name      string
		input     uint64
		expectErr bool
	}{
		{
			name:      "Valid conversion",
			input:     maxInt64, // Maximum value for int64
			expectErr: false,
		},
		{
			name:      "Invalid conversion causes overflow",
			input:     uint64(maxInt64) + 1, // Exceeds maximum int64 value
			expectErr: true,
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			result, err := eth.SafeInt64(tc.input)
			if tc.expectErr {
				s.Error(err, "Expected an error due to overflow but did not get one")
			} else {
				s.NoError(err, "Did not expect an error but got one")
				s.Equal(int64(tc.input), result, "The results should match")
			}
		})
	}
}
