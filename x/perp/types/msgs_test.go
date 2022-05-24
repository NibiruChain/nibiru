package types

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/testutil/sample"
)

func TestMsgOpenPosition_ValidateBasic(t *testing.T) {
	type test struct {
		msg     *MsgOpenPosition
		wantErr bool
	}

	cases := map[string]test{
		"ok": {
			msg: &MsgOpenPosition{
				Sender:               sample.AccAddress(),
				TokenPair:            "NIBI:USDN",
				Side:                 Side_BUY,
				QuoteAssetAmount:     sdk.NewInt(100),
				Leverage:             sdk.NewDec(10),
				BaseAssetAmountLimit: sdk.NewInt(100),
			},
			wantErr: false,
		},

		"invalid side": {
			msg: &MsgOpenPosition{
				Sender:               sample.AccAddress(),
				TokenPair:            "NIBI:USDN",
				Side:                 2,
				QuoteAssetAmount:     sdk.NewInt(100),
				Leverage:             sdk.NewDec(10),
				BaseAssetAmountLimit: sdk.NewInt(100),
			},
			wantErr: true,
		},
		"invalid address": {
			msg: &MsgOpenPosition{
				Sender:               []byte(""),
				TokenPair:            "NIBI:USDN",
				Side:                 Side_SELL,
				QuoteAssetAmount:     sdk.NewInt(100),
				Leverage:             sdk.NewDec(10),
				BaseAssetAmountLimit: sdk.NewInt(100),
			},
			wantErr: true,
		},
		"invalid leverage": {
			msg: &MsgOpenPosition{
				Sender:               sample.AccAddress(),
				TokenPair:            "NIBI:USDN",
				Side:                 Side_BUY,
				QuoteAssetAmount:     sdk.NewInt(100),
				Leverage:             sdk.ZeroDec(),
				BaseAssetAmountLimit: sdk.NewInt(100),
			},
			wantErr: true,
		},
		"invalid quote asset amount": {
			msg: &MsgOpenPosition{
				Sender:               sample.AccAddress(),
				TokenPair:            "NIBI:USDN",
				Side:                 Side_BUY,
				QuoteAssetAmount:     sdk.NewInt(0),
				Leverage:             sdk.NewDec(10),
				BaseAssetAmountLimit: sdk.NewInt(100),
			},
			wantErr: true,
		},
		"invalid token pair": {
			msg: &MsgOpenPosition{
				Sender:               sample.AccAddress(),
				TokenPair:            "NIBI-USDN",
				Side:                 Side_BUY,
				QuoteAssetAmount:     sdk.NewInt(0),
				Leverage:             sdk.NewDec(10),
				BaseAssetAmountLimit: sdk.NewInt(100),
			},
			wantErr: true,
		},
		"invalid base asset amount limit": {
			msg: &MsgOpenPosition{
				Sender:               sample.AccAddress(),
				TokenPair:            "NIBI:USDN",
				Side:                 Side_BUY,
				QuoteAssetAmount:     sdk.NewInt(0),
				Leverage:             sdk.NewDec(10),
				BaseAssetAmountLimit: sdk.ZeroInt(),
			},
			wantErr: true,
		},
	}

	for name, tc := range cases {
		tc := tc
		name := name
		t.Run(name, func(t *testing.T) {
			err := tc.msg.ValidateBasic()
			if err != nil && tc.wantErr == false {
				t.Fatalf("unexpected error: %s", err)
			}
			if err == nil && tc.wantErr == true {
				t.Fatalf("expected error: %s", err)
			}
		})
	}
}
