package types

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
)

func TestSnapshotValidate(t *testing.T) {
	tests := []struct {
		name      string
		snapshot  ReserveSnapshot
		shouldErr bool
	}{
		{
			name: "happy path",
			snapshot: ReserveSnapshot{
				Pair:          asset.Registry.Pair(denoms.ETH, denoms.NUSD),
				BaseReserve:   sdk.OneDec(),
				QuoteReserve:  sdk.OneDec(),
				PegMultiplier: sdk.OneDec(),
				TimestampMs:   time.Now().UnixMilli(),
			},
			shouldErr: false,
		},
		{
			name: "invalid pair",
			snapshot: ReserveSnapshot{
				Pair:          asset.NewPair("$invalid", "valid"),
				BaseReserve:   sdk.OneDec(),
				QuoteReserve:  sdk.OneDec(),
				PegMultiplier: sdk.OneDec(),
				TimestampMs:   time.Now().UnixMilli(),
			},
			shouldErr: true,
		},
		{
			name: "base asset negative",
			snapshot: ReserveSnapshot{
				Pair:          asset.Registry.Pair(denoms.ETH, denoms.NUSD),
				BaseReserve:   sdk.NewDec(-1),
				QuoteReserve:  sdk.OneDec(),
				PegMultiplier: sdk.OneDec(),
				TimestampMs:   time.Now().UnixMilli(),
			},
			shouldErr: true,
		},
		{
			name: "quote asset negative",
			snapshot: ReserveSnapshot{
				Pair:          asset.Registry.Pair(denoms.ETH, denoms.NUSD),
				BaseReserve:   sdk.ZeroDec(),
				QuoteReserve:  sdk.NewDec(-1),
				PegMultiplier: sdk.OneDec(),
				TimestampMs:   time.Now().UnixMilli(),
			},
			shouldErr: true,
		},
		{
			name: "timestamp lower than smallest UTC ('0001-01-01 00:00:00 +0000 UTC')",
			// see time.UnixMilli(-62135596800000).UTC())
			snapshot: ReserveSnapshot{
				Pair:          asset.Registry.Pair(denoms.ETH, denoms.NUSD),
				BaseReserve:   sdk.ZeroDec(),
				QuoteReserve:  sdk.ZeroDec(),
				PegMultiplier: sdk.OneDec(),
				TimestampMs:   -62135596800000 - 1,
			},
			shouldErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := tc.snapshot.Validate()
			if tc.shouldErr {
				require.Error(t, err)
			} else {
				require.Nil(t, err)
			}
		})
	}
}
