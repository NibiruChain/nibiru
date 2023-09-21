package types_test

import (
	"testing"

	"github.com/NibiruChain/nibiru/x/tokenfactory/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDenomStr_Validate(t *testing.T) {

	testCases := []struct {
		denom   types.DenomStr
		wantErr string
	}{
		{"tf/creator123/subdenom", ""},
		{"tf//subdenom", "empty creator"},
		{"tf/creator123/", "empty subdenom"},
		{"creator123/subdenom", "invalid number of sections"},
		{"tf/creator123/subdenom/extra", "invalid number of sections"},
		{"/creator123/subdenom", "missing denom prefix"},
	}

	for _, tc := range testCases {
		t.Run(string(tc.denom), func(t *testing.T) {
			tfDenom, err := tc.denom.ToStruct()

			if tc.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tfDenom.Denom(), tc.denom)
			assert.Equal(t, tfDenom.String(), string(tc.denom))

			assert.NoError(t, tfDenom.Validate())
			assert.NotPanics(t, func() {
				_ = tfDenom.DefaultBankMetadata()
			})
		})
	}

}
