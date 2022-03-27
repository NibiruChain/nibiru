package keeper_test

import (
	"testing"

	"github.com/MatrixDao/matrix/x/stablecoin/keeper"
	// "github.com/MatrixDao/matrix/x/testutil/sample"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
)

func TestMint_NeededCollAmtGivenGov(t *testing.T) {
	tests := []struct {
		name              string
		govAmt            sdk.Int
		priceGov          sdk.Dec
		priceColl         sdk.Dec
		collRatio         sdk.Dec
		neededCollAmt     sdk.Int
		mintableStableAmt sdk.Int
		err               error
	}{
		{
			name:              "",
			govAmt:            sdk.ZeroInt(),
			priceGov:          sdk.NewDec(1),
			priceColl:         sdk.NewDec(1),
			collRatio:         sdk.NewDec(1),
			neededCollAmt:     sdk.NewInt(1),
			mintableStableAmt: sdk.NewInt(1),
			err:               nil,
		}, {
			name:              "",
			govAmt:            sdk.ZeroInt(),
			priceGov:          sdk.NewDec(1),
			priceColl:         sdk.NewDec(1),
			collRatio:         sdk.NewDec(1),
			neededCollAmt:     sdk.NewInt(1),
			mintableStableAmt: sdk.NewInt(1),
			err:               nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			neededCollAmt, mintableStableAmt := keeper.NeededCollAmtGivenGov(
				test.govAmt, test.priceGov, test.priceColl, test.collRatio)
			require.Equal(t, neededCollAmt, test.neededCollAmt)
			require.Equal(t, mintableStableAmt, test.mintableStableAmt)
		})
	}
}

func TestMint_NeededGovAmtGivenColl(t *testing.T) {
	tests := []struct {
		name              string
		collAmt           sdk.Int
		priceGov          sdk.Dec
		priceColl         sdk.Dec
		collRatio         sdk.Dec
		neededGovAmt      sdk.Int
		mintableStableAmt sdk.Int
		err               error
	}{
		{
			name:              "",
			collAmt:           sdk.ZeroInt(),
			priceGov:          sdk.NewDec(1),
			priceColl:         sdk.NewDec(1),
			collRatio:         sdk.NewDec(1),
			neededGovAmt:      sdk.NewInt(1),
			mintableStableAmt: sdk.NewInt(1),
			err:               sdkerrors.ErrInvalidAddress,
		}, {
			name:              "",
			collAmt:           sdk.ZeroInt(),
			priceGov:          sdk.NewDec(1),
			priceColl:         sdk.NewDec(1),
			collRatio:         sdk.NewDec(1),
			neededGovAmt:      sdk.NewInt(1),
			mintableStableAmt: sdk.NewInt(1),
			err:               nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			neededGovAmt, mintableStableAmt := keeper.NeededGovAmtGivenColl(
				test.collAmt, test.priceGov, test.priceColl, test.collRatio)
			require.Equal(t, neededGovAmt, test.neededGovAmt)
			require.Equal(t, mintableStableAmt, test.mintableStableAmt)
		})
	}

}
