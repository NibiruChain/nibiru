package nutil_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/x/nutil"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil"
)

func TestAddress(t *testing.T) {
	require.NotPanics(t, func() {
		_, addrs := testutil.PrivKeyAddressPairs(5)
		strs := nutil.AddrsToStrings(addrs...)
		addrsOut := nutil.StringsToAddrs(strs...)
		require.EqualValues(t, addrs, addrsOut)
	})
}

func TestStringValueEncoder(t *testing.T) {
	encoder := nutil.StringValueEncoder
	tests := []struct {
		given string
	}{
		{"hello"},
		{"12345"},
		{""},
		{testutil.NewAccAddress().String()},
	}

	for _, tc := range tests {
		t.Run(tc.given, func(t *testing.T) {
			want := tc.given
			encoded, err := encoder.Encode(tc.given)
			require.NoError(t, err)
			got := encoder.Decode(encoded)
			assert.Equal(t, want, got)
			assert.Equal(t, want, encoder.Stringify(got))
		})
	}
}
