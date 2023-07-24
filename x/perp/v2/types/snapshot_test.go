package types

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common/asset"
)

func TestReserveSnapshot_Validate(t *testing.T) {
	validPair := asset.MustNewPair("ubtc:unusd")
	invalidPair := asset.NewPair("//ubtc", "unusd")

	validDec := sdk.NewDec(1)

	validTimestamp := time.Now().UnixMilli()

	testCases := []struct {
		name          string
		snapshot      ReserveSnapshot
		expectErr     bool
		expectedError string
	}{
		{
			name: "Valid snapshot",
			snapshot: ReserveSnapshot{
				Amm: AMM{
					Pair:            validPair,
					BaseReserve:     validDec,
					QuoteReserve:    validDec,
					PriceMultiplier: validDec,
				},
				TimestampMs: validTimestamp,
			},
			expectErr: false,
		},
		{
			name: "Invalid pair",
			snapshot: ReserveSnapshot{
				Amm: AMM{
					Pair:            invalidPair,
					BaseReserve:     validDec,
					QuoteReserve:    validDec,
					PriceMultiplier: validDec,
				},
				TimestampMs: validTimestamp,
			},
			expectErr: true,
		},
		{
			name: "Missing quote reserve",
			snapshot: ReserveSnapshot{
				Amm: AMM{
					Pair:            validPair,
					BaseReserve:     validDec,
					PriceMultiplier: validDec,
				},
				TimestampMs: validTimestamp,
			},
			expectErr: true,
		},
		{
			name: "Negative base reserve",
			snapshot: ReserveSnapshot{
				Amm: AMM{
					Pair:            validPair,
					BaseReserve:     sdk.NewDec(-1),
					QuoteReserve:    sdk.NewDec(1),
					PriceMultiplier: validDec,
				},
				TimestampMs: validTimestamp,
			},
			expectErr: true,
		},
		{
			name: "Negative quote reserve",
			snapshot: ReserveSnapshot{
				Amm: AMM{
					Pair:            validPair,
					BaseReserve:     sdk.NewDec(1),
					QuoteReserve:    sdk.NewDec(-1),
					PriceMultiplier: validDec,
				},
				TimestampMs: validTimestamp,
			},
			expectErr: true,
		},
		{
			name: "Peg multiplier",
			snapshot: ReserveSnapshot{
				Amm: AMM{
					Pair:            validPair,
					BaseReserve:     validDec,
					QuoteReserve:    validDec,
					PriceMultiplier: sdk.NewDec(-1),
				},
				TimestampMs: validTimestamp,
			},
			expectErr: true,
		},
		{
			name: "low timestamp",
			snapshot: ReserveSnapshot{
				Amm: AMM{
					Pair:            validPair,
					BaseReserve:     validDec,
					QuoteReserve:    validDec,
					PriceMultiplier: validDec,
				},
				TimestampMs: -62135596800001,
			},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.snapshot.Validate()
			if tc.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
