package types

import (
	"fmt"
	"testing"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestMsgAddMargin_ValidateBasic(t *testing.T) {
	type test struct {
		msg         *MsgAddMargin
		expectedErr error
	}

	cases := map[string]test{
		"ok": {
			msg: &MsgAddMargin{
				Sender: testutil.AccAddress().String(),
				Pair:   "NIBI:NUSD",
				Margin: sdk.NewInt64Coin("NUSD", 100),
			},
			expectedErr: nil,
		},
		"empty address": {
			msg: &MsgAddMargin{
				Sender: "",
				Pair:   "NIBI:NUSD",
				Margin: sdk.NewInt64Coin("NUSD", 100),
			},
			expectedErr: fmt.Errorf("empty address string is not allowed"),
		},
		"invalid address": {
			msg: &MsgAddMargin{
				Sender: "foobar",
				Pair:   "NIBI:NUSD",
				Margin: sdk.NewInt64Coin("NUSD", 100),
			},
			expectedErr: fmt.Errorf("decoding bech32 failed"),
		},
		"invalid token pair": {
			msg: &MsgAddMargin{
				Sender: testutil.AccAddress().String(),
				Pair:   "NIBI-NUSD",
				Margin: sdk.NewInt64Coin("NUSD", 100),
			},
			expectedErr: asset.ErrInvalidTokenPair,
		},
		"invalid margin amount": {
			msg: &MsgAddMargin{
				Sender: testutil.AccAddress().String(),
				Pair:   "NIBI:NUSD",
				Margin: sdk.NewInt64Coin("NUSD", 0),
			},
			expectedErr: fmt.Errorf("margin must be positive"),
		},
		"invalid margin denom": {
			msg: &MsgAddMargin{
				Sender: testutil.AccAddress().String(),
				Pair:   "NIBI:NUSD",
				Margin: sdk.NewInt64Coin("USDC", 100),
			},
			expectedErr: fmt.Errorf("invalid margin denom"),
		},
	}

	for name, tc := range cases {
		tc := tc
		name := name
		t.Run(name, func(t *testing.T) {
			err := tc.msg.ValidateBasic()
			if tc.expectedErr != nil {
				require.ErrorContains(t, err, tc.expectedErr.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMsgRemoveMargin_ValidateBasic(t *testing.T) {
	type test struct {
		msg         *MsgRemoveMargin
		expectedErr error
	}

	cases := map[string]test{
		"ok": {
			msg: &MsgRemoveMargin{
				Sender: testutil.AccAddress().String(),
				Pair:   "NIBI:NUSD",
				Margin: sdk.NewInt64Coin("NUSD", 100),
			},
			expectedErr: nil,
		},
		"empty address": {
			msg: &MsgRemoveMargin{
				Sender: "",
				Pair:   "NIBI:NUSD",
				Margin: sdk.NewInt64Coin("NUSD", 100),
			},
			expectedErr: fmt.Errorf("empty address string is not allowed"),
		},
		"invalid address": {
			msg: &MsgRemoveMargin{
				Sender: "foobar",
				Pair:   "NIBI:NUSD",
				Margin: sdk.NewInt64Coin("NUSD", 100),
			},
			expectedErr: fmt.Errorf("decoding bech32 failed"),
		},
		"invalid token pair": {
			msg: &MsgRemoveMargin{
				Sender: testutil.AccAddress().String(),
				Pair:   "NIBI-NUSD",
				Margin: sdk.NewInt64Coin("NUSD", 100),
			},
			expectedErr: asset.ErrInvalidTokenPair,
		},
		"invalid margin amount": {
			msg: &MsgRemoveMargin{
				Sender: testutil.AccAddress().String(),
				Pair:   "NIBI:NUSD",
				Margin: sdk.NewInt64Coin("NUSD", 0),
			},
			expectedErr: fmt.Errorf("margin must be positive"),
		},
		"invalid margin denom": {
			msg: &MsgRemoveMargin{
				Sender: testutil.AccAddress().String(),
				Pair:   "NIBI:NUSD",
				Margin: sdk.NewInt64Coin("USDC", 100),
			},
			expectedErr: fmt.Errorf("invalid margin denom"),
		},
	}

	for name, tc := range cases {
		tc := tc
		name := name
		t.Run(name, func(t *testing.T) {
			err := tc.msg.ValidateBasic()
			if tc.expectedErr != nil {
				require.ErrorContains(t, err, tc.expectedErr.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMsgOpenPosition_ValidateBasic(t *testing.T) {
	type test struct {
		msg     *MsgOpenPosition
		wantErr bool
	}

	cases := map[string]test{
		"ok": {
			msg: &MsgOpenPosition{
				Sender:               testutil.AccAddress().String(),
				Pair:                 "NIBI:NUSD",
				Side:                 Side_BUY,
				QuoteAssetAmount:     sdk.NewInt(100),
				Leverage:             sdk.NewDec(10),
				BaseAssetAmountLimit: sdk.NewInt(100),
			},
			wantErr: false,
		},

		"invalid side": {
			msg: &MsgOpenPosition{
				Sender:               testutil.AccAddress().String(),
				Pair:                 "NIBI:NUSD",
				Side:                 3,
				QuoteAssetAmount:     sdk.NewInt(100),
				Leverage:             sdk.NewDec(10),
				BaseAssetAmountLimit: sdk.NewInt(100),
			},
			wantErr: true,
		},
		"invalid side 2": {
			msg: &MsgOpenPosition{
				Sender:               testutil.AccAddress().String(),
				Pair:                 "NIBI:NUSD",
				Side:                 Side_SIDE_UNSPECIFIED,
				QuoteAssetAmount:     sdk.NewInt(100),
				Leverage:             sdk.NewDec(10),
				BaseAssetAmountLimit: sdk.NewInt(100),
			},
			wantErr: true,
		},
		"invalid address": {
			msg: &MsgOpenPosition{
				Sender:               "",
				Pair:                 "NIBI:NUSD",
				Side:                 Side_SELL,
				QuoteAssetAmount:     sdk.NewInt(100),
				Leverage:             sdk.NewDec(10),
				BaseAssetAmountLimit: sdk.NewInt(100),
			},
			wantErr: true,
		},
		"invalid leverage": {
			msg: &MsgOpenPosition{
				Sender:               testutil.AccAddress().String(),
				Pair:                 "NIBI:NUSD",
				Side:                 Side_BUY,
				QuoteAssetAmount:     sdk.NewInt(100),
				Leverage:             sdk.ZeroDec(),
				BaseAssetAmountLimit: sdk.NewInt(100),
			},
			wantErr: true,
		},
		"invalid quote asset amount": {
			msg: &MsgOpenPosition{
				Sender:               testutil.AccAddress().String(),
				Pair:                 "NIBI:NUSD",
				Side:                 Side_BUY,
				QuoteAssetAmount:     sdk.NewInt(0),
				Leverage:             sdk.NewDec(10),
				BaseAssetAmountLimit: sdk.NewInt(100),
			},
			wantErr: true,
		},
		"invalid token pair": {
			msg: &MsgOpenPosition{
				Sender:               testutil.AccAddress().String(),
				Pair:                 "NIBI-NUSD",
				Side:                 Side_BUY,
				QuoteAssetAmount:     sdk.NewInt(0),
				Leverage:             sdk.NewDec(10),
				BaseAssetAmountLimit: sdk.NewInt(100),
			},
			wantErr: true,
		},
		"invalid base asset amount limit": {
			msg: &MsgOpenPosition{
				Sender:               testutil.AccAddress().String(),
				Pair:                 "NIBI:NUSD",
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

func TestMsgMultiLiquidate_ValidateBasic(t *testing.T) {
	type test struct {
		msg     *MsgMultiLiquidate
		wantErr bool
	}

	cases := map[string]test{
		"success": {
			msg: &MsgMultiLiquidate{
				Sender: testutil.AccAddress().String(),
				Liquidations: []*MsgMultiLiquidate_SingleLiquidation{
					{
						Pair:   asset.Registry.Pair(denoms.BTC, denoms.NUSD),
						Trader: testutil.AccAddress().String(),
					},
				}},
			wantErr: false,
		},
		"invalid token pair": {
			msg: &MsgMultiLiquidate{
				Sender: testutil.AccAddress().String(),
				Liquidations: []*MsgMultiLiquidate_SingleLiquidation{
					{
						Pair:   asset.Registry.Pair(denoms.BTC, denoms.NUSD),
						Trader: testutil.AccAddress().String(),
					},
					{
						Pair:   "invalid",
						Trader: testutil.AccAddress().String(),
					},
				}},
			wantErr: true,
		},
		"invalid liquidated address": {
			msg: &MsgMultiLiquidate{
				Sender: testutil.AccAddress().String(),
				Liquidations: []*MsgMultiLiquidate_SingleLiquidation{
					{
						Pair:   asset.Registry.Pair(denoms.BTC, denoms.NUSD),
						Trader: testutil.AccAddress().String(),
					},
					{
						Pair:   asset.Registry.Pair(denoms.BTC, denoms.NUSD),
						Trader: "invalid",
					},
				}},
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
