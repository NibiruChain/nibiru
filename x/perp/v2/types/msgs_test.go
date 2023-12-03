package types

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common/asset"
)

func TestMsgValidateBasic(t *testing.T) {
	validSender := "cosmos1zaavvzxez0elundtn32qnk9lkm8kmcszzsv80v"
	validPair := asset.MustNewPair("ubtc:unusd")
	invalidPair := asset.NewPair("//ubtc", "unusd")

	testCases := []struct {
		name          string
		msg           sdk.Msg
		expectErr     bool
		expectedError string
	}{
		// MsgRemoveMargin test cases
		{
			"Test MsgRemoveMargin: Valid input",
			&MsgRemoveMargin{
				Sender: validSender,
				Pair:   validPair,
				Margin: sdk.NewCoin("unusd", sdk.NewInt(10)),
			},
			false,
			"",
		},
		{
			"Test MsgRemoveMargin: Invalid pair",
			&MsgRemoveMargin{
				Sender: validSender,
				Pair:   invalidPair,
				Margin: sdk.NewCoin("denom", sdk.NewInt(10)),
			},
			true,
			"invalid base asset",
		},
		{
			"Test MsgRemoveMargin: Invalid sender",
			&MsgRemoveMargin{
				Sender: "invalid",
				Pair:   validPair,
				Margin: sdk.NewCoin("denom", sdk.NewInt(10)),
			},
			true,
			"decoding bech32 failed",
		},
		{
			"Test MsgRemoveMargin: Negative margin",
			&MsgRemoveMargin{
				Sender: validSender,
				Pair:   validPair,
				Margin: sdk.NewCoin("denom", sdk.ZeroInt()),
			},
			true,
			"margin must be positive",
		},

		// MsgAddMargin test cases
		{
			"Test MsgAddMargin: Valid input",
			&MsgAddMargin{
				Sender: validSender,
				Pair:   validPair,
				Margin: sdk.NewCoin("unusd", sdk.NewInt(10)),
			},
			false,
			"",
		},
		{
			"Test MsgAddMargin: Invalid sender",
			&MsgAddMargin{
				Sender: "invalid",
				Pair:   validPair,
				Margin: sdk.NewCoin("denom", sdk.NewInt(10)),
			},
			true,
			"decoding bech32 failed",
		},
		{
			"Test MsgAddMargin: Invalid pair",
			&MsgAddMargin{
				Sender: validSender,
				Pair:   invalidPair,
				Margin: sdk.NewCoin("denom", sdk.NewInt(10)),
			},
			true,
			"invalid base asset",
		},
		{
			"Test MsgAddMargin: Negative margin",
			&MsgAddMargin{
				Sender: validSender,
				Pair:   validPair,
				Margin: sdk.NewCoin("denom", sdk.ZeroInt()),
			},
			true,
			"margin must be positive",
		},
		// MsgMarketOrder test cases
		{
			"Test MsgMarketOrder: Valid input",
			&MsgMarketOrder{
				Sender:               validSender,
				Pair:                 validPair,
				Side:                 Direction_SHORT,
				Leverage:             sdk.NewDec(10),
				BaseAssetAmountLimit: sdk.NewInt(10),
				QuoteAssetAmount:     sdk.NewInt(10),
			},
			false,
			"",
		},
		{
			"Test MsgMarketOrder: Invalid Side",
			&MsgMarketOrder{
				Sender:               validSender,
				Pair:                 validPair,
				Side:                 Direction_DIRECTION_UNSPECIFIED,
				Leverage:             sdk.NewDec(10),
				BaseAssetAmountLimit: sdk.NewInt(10),
				QuoteAssetAmount:     sdk.NewInt(10),
			},
			true,
			"invalid side",
		},
		{
			"Test MsgMarketOrder: Invalid Pair",
			&MsgMarketOrder{
				Sender:               validSender,
				Pair:                 invalidPair,
				Side:                 Direction_SHORT,
				Leverage:             sdk.NewDec(10),
				BaseAssetAmountLimit: sdk.NewInt(10),
				QuoteAssetAmount:     sdk.NewInt(10),
			},
			true,
			"invalid base asset",
		},
		{
			"Test MsgMarketOrder: Invalid Sender",
			&MsgMarketOrder{
				Sender:               "invalid",
				Pair:                 validPair,
				Side:                 Direction_SHORT,
				Leverage:             sdk.NewDec(10),
				BaseAssetAmountLimit: sdk.NewInt(10),
				QuoteAssetAmount:     sdk.NewInt(10),
			},
			true,
			"decoding bech32 failed",
		},
		{
			"Test MsgMarketOrder: Negative Leverage",
			&MsgMarketOrder{
				Sender:               validSender,
				Pair:                 validPair,
				Side:                 Direction_SHORT,
				Leverage:             sdk.NewDec(-10),
				BaseAssetAmountLimit: sdk.NewInt(10),
				QuoteAssetAmount:     sdk.NewInt(10),
			},
			true,
			"leverage must always be greater than zero",
		},
		{
			"Test MsgMarketOrder: Negative BaseAssetAmountLimit",
			&MsgMarketOrder{
				Sender:               validSender,
				Pair:                 validPair,
				Side:                 Direction_SHORT,
				Leverage:             sdk.NewDec(10),
				BaseAssetAmountLimit: sdk.NewInt(-10),
				QuoteAssetAmount:     sdk.NewInt(10),
			},
			true,
			"base asset amount limit must not be negative",
		},
		{
			"Test MsgMarketOrder: Negative QuoteAssetAmount",
			&MsgMarketOrder{
				Sender:               validSender,
				Pair:                 validPair,
				Side:                 Direction_SHORT,
				Leverage:             sdk.NewDec(10),
				BaseAssetAmountLimit: sdk.NewInt(10),
				QuoteAssetAmount:     sdk.NewInt(-10),
			},
			true,
			"quote asset amount must be always greater than zero",
		},
		// MsgClosePosition test cases
		{
			"Test MsgClosePosition: Valid input",
			&MsgClosePosition{
				Sender: validSender,
				Pair:   validPair,
			},
			false,
			"",
		},
		{
			"Test MsgClosePosition: Invalid pair",
			&MsgClosePosition{
				Sender: validSender,
				Pair:   invalidPair,
			},
			true,
			"invalid base asset",
		},
		{
			"Test MsgClosePosition: Invalid sender",
			&MsgClosePosition{
				Sender: "invalid",
				Pair:   validPair,
			},
			true,
			"decoding bech32 failed",
		},
		// MsgSettlePosition test cases
		{
			"Test MsgSettlePosition: Valid input",
			&MsgSettlePosition{
				Sender:  validSender,
				Pair:    validPair,
				Version: 1,
			},
			false,
			"",
		},
		{
			"Test MsgSettlePosition: Invalid pair",
			&MsgSettlePosition{
				Sender:  validSender,
				Pair:    invalidPair,
				Version: 1,
			},
			true,
			"invalid base asset",
		},
		{
			"Test MsgSettlePosition: Invalid sender",
			&MsgSettlePosition{
				Sender: "invalid",
				Pair:   validPair,
			},
			true,
			"decoding bech32 failed",
		},
		// MsgPartialClose test cases
		{
			"Test MsgPartialClose: Valid input",
			&MsgPartialClose{
				Sender: validSender,
				Pair:   validPair,
				Size_:  sdk.NewDec(10),
			},
			false,
			"",
		},
		{
			"Test MsgPartialClose: Invalid pair",
			&MsgPartialClose{
				Sender: validSender,
				Pair:   invalidPair,
				Size_:  sdk.NewDec(10),
			},
			true,
			"invalid base asset",
		},
		{
			"Test MsgPartialClose: Invalid sender",
			&MsgPartialClose{
				Sender: "invalid",
				Pair:   validPair,
				Size_:  sdk.NewDec(10),
			},
			true,
			"decoding bech32 failed",
		},
		{
			"Test MsgPartialClose: Invalid size",
			&MsgPartialClose{
				Sender: validSender,
				Pair:   validPair,
				Size_:  sdk.NewDec(0),
			},
			true,
			"invalid size amount",
		},

		// MsgDonateToEcosystemFund test cases
		{
			"Test MsgDonateToEcosystemFund: Valid input",
			&MsgDonateToEcosystemFund{
				Sender:   validSender,
				Donation: sdk.NewCoin("unusd", sdk.NewInt(10)),
			},
			false,
			"",
		},
		{
			"Test MsgDonateToEcosystemFund: Invalid sender",
			&MsgDonateToEcosystemFund{
				Sender:   "invalid",
				Donation: sdk.NewCoin("unusd", sdk.NewInt(10)),
			},
			true,
			"decoding bech32 failed",
		},
		{
			"Test MsgDonateToEcosystemFund: Invalid donation amount",
			&MsgDonateToEcosystemFund{
				Sender: validSender,
			},
			true,
			"invalid donation amount",
		},
		// MsgMultiLiquidate
		{
			"Test MsgMultiLiquidate: Valid input",
			&MsgMultiLiquidate{
				Sender: validSender,
				Liquidations: []*MsgMultiLiquidate_Liquidation{
					{Trader: validSender, Pair: validPair},
					{Trader: validSender, Pair: validPair},
				},
			},
			false,
			"",
		},
		{
			"Test MsgMultiLiquidate: Invalid Sender",
			&MsgMultiLiquidate{
				Sender: "invalid",
				Liquidations: []*MsgMultiLiquidate_Liquidation{
					{Trader: validSender, Pair: validPair},
					{Trader: validSender, Pair: validPair},
				},
			},
			true,
			"decoding bech32 failed",
		},
		{
			"Test MsgMultiLiquidate: Invalid Trader",
			&MsgMultiLiquidate{
				Sender: validSender,
				Liquidations: []*MsgMultiLiquidate_Liquidation{
					{Trader: "invalid", Pair: validPair},
					{Trader: validSender, Pair: validPair},
				},
			},
			true,
			"invalid liquidation at index 0: decoding bech32 failed",
		},
		{
			"Test MsgMultiLiquidate: Invalid Pair",
			&MsgMultiLiquidate{
				Sender: validSender,
				Liquidations: []*MsgMultiLiquidate_Liquidation{
					{Trader: validSender, Pair: invalidPair},
					{Trader: validSender, Pair: validPair},
				},
			},
			true,
			"invalid liquidation at index 0: invalid base asset",
		},
		// MsgShiftPegMultiplier
		{
			name: "MsgShiftPegMultiplier: Invalid pair",
			msg: &MsgShiftPegMultiplier{
				Sender:     validSender,
				Pair:       asset.Pair("not_a_pair"),
				NewPegMult: sdk.NewDec(420),
			},
			expectErr:     true,
			expectedError: asset.ErrInvalidTokenPair.Error(),
		},
		{
			name: "MsgShiftPegMultiplier: nonpositive peg multiplier",
			msg: &MsgShiftPegMultiplier{
				Sender:     validSender,
				Pair:       asset.Pair("valid:pair"),
				NewPegMult: sdk.NewDec(-420),
			},
			expectErr:     true,
			expectedError: ErrNonPositivePegMultiplier.Error(),
		},
		// MsgDonateToEcosystemFund test cases
		{
			name: "MsgShiftSwapInvariant: Invalid pair",
			msg: &MsgShiftSwapInvariant{
				Sender:           validSender,
				Pair:             asset.Pair("not_a_pair"),
				NewSwapInvariant: sdk.NewInt(420),
			},
			expectErr:     true,
			expectedError: asset.ErrInvalidTokenPair.Error(),
		},
		{
			name: "MsgShiftSwapInvariant: nonpositive swap invariant",
			msg: &MsgShiftSwapInvariant{
				Sender:           validSender,
				Pair:             asset.Pair("valid:pair"),
				NewSwapInvariant: sdk.NewInt(-420),
			},
			expectErr:     true,
			expectedError: ErrNonPositiveSwapInvariant.Error(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.msg.ValidateBasic()
			if tc.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMsg_GetSigners(t *testing.T) {
	validSender := "cosmos1zaavvzxez0elundtn32qnk9lkm8kmcszzsv80v"
	invalidSender := "invalid_address"

	msgValidSenderList := []sdk.Msg{
		&MsgAddMargin{Sender: validSender},
		&MsgRemoveMargin{Sender: validSender},
		&MsgMarketOrder{Sender: validSender},
		&MsgClosePosition{Sender: validSender},
		&MsgSettlePosition{Sender: validSender},
		&MsgPartialClose{Sender: validSender},
		&MsgDonateToEcosystemFund{Sender: validSender},
		&MsgMultiLiquidate{Sender: validSender},
		&MsgShiftPegMultiplier{Sender: validSender},
		&MsgShiftSwapInvariant{Sender: validSender},
	}
	msgInvalidSenderList := []sdk.Msg{
		&MsgAddMargin{Sender: invalidSender},
		&MsgRemoveMargin{Sender: invalidSender},
		&MsgMarketOrder{Sender: invalidSender},
		&MsgClosePosition{Sender: invalidSender},
		&MsgSettlePosition{Sender: invalidSender},
		&MsgPartialClose{Sender: invalidSender},
		&MsgDonateToEcosystemFund{Sender: invalidSender},
		&MsgMultiLiquidate{Sender: invalidSender},
		&MsgShiftPegMultiplier{Sender: invalidSender},
		&MsgShiftSwapInvariant{Sender: invalidSender},
	}

	for _, msg := range msgValidSenderList {
		defer func() {
			r := recover()
			if (r != nil) != false {
				t.Errorf("GetSigners() recover = %v, expectPanic = %v", r, false)
			}
		}()
		signerAddr, _ := sdk.AccAddressFromBech32(validSender)
		require.Equal(t, []sdk.AccAddress{signerAddr}, msg.GetSigners())
	}
	for _, msg := range msgInvalidSenderList {
		defer func() {
			r := recover()
			if (r != nil) != true {
				t.Errorf("GetSigners() recover = %v, expectPanic = %v", r, true)
			}
		}()
		msg.GetSigners()
	}
}

type RouteTyper interface {
	sdk.Msg
	Route() string
	Type() string
}

func TestMsg_RouteAndType(t *testing.T) {
	testCases := []struct {
		name          string
		msg           RouteTyper
		expectedRoute string
		expectedType  string
	}{
		{
			name:          "MsgAddMargin",
			msg:           &MsgAddMargin{},
			expectedRoute: "perp",
			expectedType:  "add_margin_msg",
		},
		{
			name:          "MsgRemoveMargin",
			msg:           &MsgRemoveMargin{},
			expectedRoute: "perp",
			expectedType:  "remove_margin_msg",
		},
		{
			name:          "MsgMarketOrder",
			msg:           &MsgMarketOrder{},
			expectedRoute: "perp",
			expectedType:  "market_order_msg",
		},
		{
			name:          "MsgClosePosition",
			msg:           &MsgClosePosition{},
			expectedRoute: "perp",
			expectedType:  "close_position_msg",
		},
		{
			name:          "MsgSettlePosition",
			msg:           &MsgSettlePosition{},
			expectedRoute: "perp",
			expectedType:  "settle_position_msg",
		},
		{
			name:          "MsgPartialClose",
			msg:           &MsgPartialClose{},
			expectedRoute: "perp",
			expectedType:  "partial_close_msg",
		},
		{
			name:          "MsgDonateToEcosystemFund",
			msg:           &MsgDonateToEcosystemFund{},
			expectedRoute: "perp",
			expectedType:  "donate_to_ef_msg",
		},
		{
			name:          "MsgMultiLiquidate",
			msg:           &MsgMultiLiquidate{},
			expectedRoute: "perp",
			expectedType:  "multi_liquidate_msg",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expectedRoute, tc.msg.Route())
			require.Equal(t, tc.expectedType, tc.msg.Type())
		})
	}
}

type SignBytesGetter interface {
	sdk.Msg
	GetSignBytes() []byte
}

func TestMsg_GetSignBytes(t *testing.T) {
	testCases := []struct {
		name string
		msg  SignBytesGetter
	}{
		{
			name: "MsgAddMargin",
			msg:  &MsgAddMargin{},
		},
		{
			name: "MsgRemoveMargin",
			msg:  &MsgRemoveMargin{},
		},
		{
			name: "MsgMarketOrder",
			msg:  &MsgMarketOrder{},
		},
		{
			name: "MsgClosePosition",
			msg:  &MsgClosePosition{},
		},
		{
			name: "MsgSettlePosition",
			msg:  &MsgSettlePosition{},
		},
		{
			name: "MsgPartialClose",
			msg:  &MsgPartialClose{},
		},
		{
			name: "MsgDonateToEcosystemFund",
			msg:  &MsgDonateToEcosystemFund{},
		},
		{
			name: "MsgMultiLiquidate",
			msg:  &MsgMultiLiquidate{},
		},
		{
			name: "MsgShiftPegMultiplier",
			msg:  &MsgShiftPegMultiplier{},
		},
		{
			name: "MsgShiftSwapInvariant",
			msg:  &MsgShiftSwapInvariant{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bytes := tc.msg.GetSignBytes()
			require.NotNil(t, bytes)
			require.NotEmpty(t, bytes)

			// The same message should always return the same bytes
			require.Equal(t, bytes, tc.msg.GetSignBytes())
		})
	}
}
