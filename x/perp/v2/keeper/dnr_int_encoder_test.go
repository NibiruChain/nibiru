package keeper

import (
	"math/big"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestIntEncoder(t *testing.T) {
	// we test our assumptions around int are correct.
	outOfBounds := new(big.Int).Lsh(big.NewInt(1), 256)       // 2^256
	maxBigInt := new(big.Int).Sub(outOfBounds, big.NewInt(1)) // 2^256 - 1
	require.Equal(t, maxBigInt.BitLen(), math.MaxBitLen)
	require.Greater(t, outOfBounds.BitLen(), math.MaxBitLen)

	require.NotPanics(t, func() {
		sdk.NewIntFromBigInt(maxBigInt)
	})
	require.Panics(t, func() {
		sdk.NewIntFromBigInt(outOfBounds)
	})

	require.Equal(t, maxIntKeyLen, len(maxBigInt.Bytes()))

	// test encoding ordering
	enc1 := IntKeyEncoder.Encode(sdk.NewInt(50_000))
	enc2 := IntKeyEncoder.Encode(sdk.NewInt(100_000))
	require.Less(t, enc1, enc2)

	// test decoding
	size, got1 := IntKeyEncoder.Decode(enc1)
	require.Equal(t, maxIntKeyLen, size)
	_, got2 := IntKeyEncoder.Decode(enc2)
	require.Equal(t, sdk.NewInt(50_000), got1)
	require.Equal(t, sdk.NewInt(100_000), got2)

	// require panics on negative values
	require.Panics(t, func() {
		IntKeyEncoder.Encode(sdk.NewInt(-1))
	})
	// require panics on invalid int
	require.Panics(t, func() {
		IntKeyEncoder.Encode(math.Int{})
	})

	// test value encoder
	value := sdk.NewInt(50_000)
	valueBytes := IntValueEncoder.Encode(value)
	gotValue := IntValueEncoder.Decode(valueBytes)
	require.Equal(t, value, gotValue)

	// panics on invalid math.Int
	require.Panics(t, func() {
		IntValueEncoder.Encode(math.Int{})
	})
}
