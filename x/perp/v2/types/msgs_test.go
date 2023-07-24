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
		{
			"Test MsgRemoveMargin: Invalid margin",
			&MsgRemoveMargin{
				Sender: validSender,
				Pair:   validPair,
				Margin: sdk.NewCoin("denom", sdk.OneInt()),
			},
			true,
			"invalid margin denom",
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
		{
			"Test MsgAddMargin: Invalid margin",
			&MsgAddMargin{
				Sender: validSender,
				Pair:   validPair,
				Margin: sdk.NewCoin("denom", sdk.OneInt()),
			},
			true,
			"invalid margin denom",
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
