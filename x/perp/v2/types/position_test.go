package types

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common/asset"
)

func TestZeroPosition(t *testing.T) {
	// Initialization
	ctx := sdk.Context{}
	tokenPair := asset.NewPair("ubtc", "unusd")
	traderAddr := sdk.AccAddress{}

	position := ZeroPosition(ctx, tokenPair, traderAddr)

	// Test the conditions
	require.NotNil(t, position)
	// Continue testing individual attributes of position as required
}

func TestPositionsAreEqual(t *testing.T) {
	accAddress := "cosmos1zaavvzxez0elundtn32qnk9lkm8kmcszzsv80v"
	accOtherAddress := "cosmos1g7vzqfthhf4l4vs6skyjj27vqhe97m5gp33hxy"

	expected := Position{
		TraderAddress:                   accAddress,
		Pair:                            "ubtc:unusd",
		Size_:                           sdk.OneDec(),
		Margin:                          sdk.OneDec(),
		OpenNotional:                    sdk.OneDec(),
		LatestCumulativePremiumFraction: sdk.OneDec(),
		LastUpdatedBlockNumber:          0,
	}

	err := PositionsAreEqual(&expected, expected.copy())
	require.NoError(t, err)

	testCases := []struct {
		modifier      func(*Position)
		requiredError string
	}{
		{
			modifier:      func(p *Position) { p.WithPair(asset.NewPair("ueth", "unusd")) },
			requiredError: "expected position pair",
		},
		{
			modifier:      func(p *Position) { p.WithTraderAddress(accOtherAddress) },
			requiredError: "expected position trader address",
		},
		{
			modifier:      func(p *Position) { p.WithMargin(sdk.NewDec(42)) },
			requiredError: "expected position margin",
		},
		{
			modifier:      func(p *Position) { p.WithOpenNotional(sdk.NewDec(42)) },
			requiredError: "expected position open notional",
		},
		{
			modifier:      func(p *Position) { p.WithSize_(sdk.NewDec(42)) },
			requiredError: "expected position size",
		},
		{
			modifier:      func(p *Position) { p.WithLastUpdatedBlockNumber(42) },
			requiredError: "expected position block number",
		},
		{
			modifier:      func(p *Position) { p.WithLatestCumulativePremiumFraction(sdk.NewDec(42)) },
			requiredError: "expected position latest cumulative premium fraction",
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.requiredError, func(t *testing.T) {
			newPosition := expected.copy()

			tc.modifier(newPosition)

			err := PositionsAreEqual(&expected, newPosition)
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.requiredError)
		})
	}
}

func TestPositionValidate(t *testing.T) {
	accAddress := "cosmos1zaavvzxez0elundtn32qnk9lkm8kmcszzsv80v"

	validPosition := Position{
		TraderAddress:                   accAddress,
		Pair:                            "ubtc:unusd",
		Size_:                           sdk.OneDec(),
		Margin:                          sdk.OneDec(),
		OpenNotional:                    sdk.OneDec(),
		LatestCumulativePremiumFraction: sdk.OneDec(),
		LastUpdatedBlockNumber:          0,
	}

	testCases := []struct {
		modifier      func(*Position)
		requiredError string
	}{
		{
			modifier:      func(p *Position) { p.WithTraderAddress("invalid") },
			requiredError: "decoding bech32 failed",
		},
		{
			modifier:      func(p *Position) { p.WithPair("invalid") },
			requiredError: "invalid: invalid token pair",
		},
		{
			modifier:      func(p *Position) { p.WithSize_(sdk.ZeroDec()) },
			requiredError: "zero size",
		},
		{
			modifier:      func(p *Position) { p.WithMargin(sdk.ZeroDec()) },
			requiredError: "margin <= 0",
		},
		{
			modifier:      func(p *Position) { p.WithOpenNotional(sdk.ZeroDec()) },
			requiredError: "open notional <= 0",
		},
		{
			modifier:      func(p *Position) { p.WithLastUpdatedBlockNumber(-1) },
			requiredError: "invalid block number",
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.requiredError, func(t *testing.T) {
			newPosition := validPosition.copy()

			tc.modifier(newPosition)

			err := newPosition.Validate()
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.requiredError)
		})
	}
}
