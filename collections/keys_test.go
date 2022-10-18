package collections

import (
	"bytes"
	"sort"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	tmtime "github.com/tendermint/tendermint/types/time"
)

func TestUint64(t *testing.T) {
	t.Run("bijectivity", func(t *testing.T) {
		assertBijective[uint64](t, Uint64KeyEncoder, uint64(0x0123456789ABCDEF))
	})

	t.Run("empty", func(t *testing.T) {
		var k uint64
		require.Equal(t, []byte{0, 0, 0, 0, 0, 0, 0, 0}, Uint64KeyEncoder.Encode(k))
	})
}

func TestStringKey(t *testing.T) {
	t.Run("bijective", func(t *testing.T) {
		assertBijective[string](t, StringKeyEncoder, "test")
	})

	t.Run("panics", func(t *testing.T) {
		// invalid string key
		require.Panics(t, func() {
			invalid := []byte{0x1, 0x0, 0x3}
			StringKeyEncoder.Encode(string(invalid))
		})
		// invalid bytes do not end with 0x0
		require.Panics(t, func() {
			StringKeyEncoder.Decode([]byte{0x1, 0x2})
		})
		// invalid size
		require.Panics(t, func() {
			StringKeyEncoder.Decode([]byte{0x1})
		})
	})

	t.Run("proper ordering", func(t *testing.T) {
		stringKeys := []string{
			"a", "aa", "b", "c", "dd",
			"1", "2", "3", "55", string([]byte{1}),
		}

		strings := make([]string, len(stringKeys))
		bytesStringKeys := make([][]byte, len(stringKeys))
		for i, sk := range stringKeys {
			strings[i] = sk
			bytesStringKeys[i] = StringKeyEncoder.Encode(sk)
		}

		sort.Strings(strings)
		sort.Slice(bytesStringKeys, func(i, j int) bool {
			return bytes.Compare(bytesStringKeys[i], bytesStringKeys[j]) < 0
		})

		for i, b := range bytesStringKeys {
			expected := strings[i]
			got := string(b[:len(b)-1]) // removes null termination
			require.Equal(t, expected, got)
		}
	})
}

func TestAccAddressKey(t *testing.T) {
	t.Run("bijective", func(t *testing.T) {
		assertBijective[sdk.AccAddress](t, AccAddressKeyEncoder, sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()))
	})
}

func TestTimeKey(t *testing.T) {
	t.Run("bijective", func(t *testing.T) {
		key := tmtime.Now()
		assertBijective[time.Time](t, TimeKeyEncoder, key)
	})
}

func TestValAddressKey(t *testing.T) {
	t.Run("bijective", func(t *testing.T) {
		assertBijective(t, ValAddressKeyEncoder, sdk.ValAddress(secp256k1.GenPrivKey().PubKey().Address()))
	})
}
