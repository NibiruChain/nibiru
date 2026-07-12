package types

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
)

func Test_validateAuxFuncs(t *testing.T) {
	type args struct {
		i interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"wrong type", args{10.5}, true},
		{"empty sdk.Dec", args{sdk.Dec{}}, true},
		{"negative", args{sdkmath.LegacyNewDec(-1)}, true},
		{"one dec", args{sdkmath.LegacyNewDec(1)}, false},
		{"two dec", args{sdkmath.LegacyNewDec(2)}, true},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.wantErr, validateCommunityTax(tt.args.i) != nil)
		})
	}
}
