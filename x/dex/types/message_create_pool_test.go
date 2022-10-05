package types

import (
	"testing"

	"github.com/NibiruChain/nibiru/x/testutil"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
)

func TestValidateBasic(t *testing.T) {
	tests := []struct {
		name        string
		msg         MsgCreatePool
		expectedErr error
	}{
		{
			name: "invalid address",
			msg: MsgCreatePool{
				Creator: "invalid_address",
				PoolParams: &PoolParams{
					SwapFee: sdk.MustNewDecFromStr("0.003"),
					ExitFee: sdk.MustNewDecFromStr("0.003"),
				},
				PoolAssets: []PoolAsset{
					{
						Token:  sdk.NewInt64Coin("aaa", 1),
						Weight: sdk.NewInt(1),
					},
					{
						Token:  sdk.NewInt64Coin("bbb", 1),
						Weight: sdk.NewInt(1),
					},
				},
			},
			expectedErr: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "invalid swap fee, too small",
			msg: MsgCreatePool{
				Creator: testutil.AccAddress().String(),
				PoolParams: &PoolParams{
					SwapFee: sdk.MustNewDecFromStr("-0.003"),
					ExitFee: sdk.MustNewDecFromStr("0.003"),
				},
				PoolAssets: []PoolAsset{
					{
						Token:  sdk.NewInt64Coin("aaa", 1),
						Weight: sdk.NewInt(1),
					},
					{
						Token:  sdk.NewInt64Coin("bbb", 1),
						Weight: sdk.NewInt(1),
					},
				},
			},
			expectedErr: ErrInvalidSwapFee,
		},
		{
			name: "invalid swap fee, too large",
			msg: MsgCreatePool{
				Creator: testutil.AccAddress().String(),
				PoolParams: &PoolParams{
					SwapFee: sdk.MustNewDecFromStr("1.1"),
					ExitFee: sdk.MustNewDecFromStr("0.003"),
				},
				PoolAssets: []PoolAsset{
					{
						Token:  sdk.NewInt64Coin("aaa", 1),
						Weight: sdk.NewInt(1),
					},
					{
						Token:  sdk.NewInt64Coin("bbb", 1),
						Weight: sdk.NewInt(1),
					},
				},
			},
			expectedErr: ErrInvalidSwapFee,
		},
		{
			name: "invalid exit fee, too small",
			msg: MsgCreatePool{
				Creator: testutil.AccAddress().String(),
				PoolParams: &PoolParams{
					SwapFee: sdk.MustNewDecFromStr("0.003"),
					ExitFee: sdk.MustNewDecFromStr("-0.003"),
				},
				PoolAssets: []PoolAsset{
					{
						Token:  sdk.NewInt64Coin("aaa", 1),
						Weight: sdk.NewInt(1),
					},
					{
						Token:  sdk.NewInt64Coin("bbb", 1),
						Weight: sdk.NewInt(1),
					},
				},
			},
			expectedErr: ErrInvalidExitFee,
		},
		{
			name: "invalid exit fee, too large",
			msg: MsgCreatePool{
				Creator: testutil.AccAddress().String(),
				PoolParams: &PoolParams{
					SwapFee: sdk.MustNewDecFromStr("0.003"),
					ExitFee: sdk.MustNewDecFromStr("1.1"),
				},
				PoolAssets: []PoolAsset{
					{
						Token:  sdk.NewInt64Coin("aaa", 1),
						Weight: sdk.NewInt(1),
					},
					{
						Token:  sdk.NewInt64Coin("bbb", 1),
						Weight: sdk.NewInt(1),
					},
				},
			},
			expectedErr: ErrInvalidExitFee,
		},
		{
			name: "too few assets",
			msg: MsgCreatePool{
				Creator: testutil.AccAddress().String(),
				PoolParams: &PoolParams{
					SwapFee: sdk.MustNewDecFromStr("0.003"),
					ExitFee: sdk.MustNewDecFromStr("0.003"),
				},
				PoolAssets: []PoolAsset{
					{
						Token:  sdk.NewInt64Coin("aaa", 1),
						Weight: sdk.NewInt(1),
					},
				},
			},
			expectedErr: ErrTooFewPoolAssets,
		},
		{
			name: "too many assets",
			msg: MsgCreatePool{
				Creator: testutil.AccAddress().String(),
				PoolParams: &PoolParams{
					SwapFee: sdk.MustNewDecFromStr("0.003"),
					ExitFee: sdk.MustNewDecFromStr("0.003"),
				},
				PoolAssets: []PoolAsset{
					{
						Token:  sdk.NewInt64Coin("aaa", 1),
						Weight: sdk.NewInt(1),
					},
					{
						Token:  sdk.NewInt64Coin("bbb", 1),
						Weight: sdk.NewInt(1),
					},
					{
						Token:  sdk.NewInt64Coin("ccc", 1),
						Weight: sdk.NewInt(1),
					},
				},
			},
			expectedErr: ErrTooManyPoolAssets,
		},
		{
			name: "invalid token weight",
			msg: MsgCreatePool{
				Creator: testutil.AccAddress().String(),
				PoolParams: &PoolParams{
					SwapFee: sdk.MustNewDecFromStr("0.003"),
					ExitFee: sdk.MustNewDecFromStr("0.003"),
				},
				PoolAssets: []PoolAsset{
					{
						Token:  sdk.NewInt64Coin("aaa", 1),
						Weight: sdk.NewInt(0),
					},
					{
						Token:  sdk.NewInt64Coin("bbb", 1),
						Weight: sdk.NewInt(0),
					},
				},
			},
			expectedErr: ErrInvalidTokenWeight,
		},
		{
			name: "valid create pool message",
			msg: MsgCreatePool{
				Creator: testutil.AccAddress().String(),
				PoolParams: &PoolParams{
					SwapFee: sdk.MustNewDecFromStr("0.003"),
					ExitFee: sdk.MustNewDecFromStr("0.003"),
				},
				PoolAssets: []PoolAsset{
					{
						Token:  sdk.NewInt64Coin("aaa", 1),
						Weight: sdk.NewInt(1),
					},
					{
						Token:  sdk.NewInt64Coin("bbb", 1),
						Weight: sdk.NewInt(1),
					},
				},
			},
			expectedErr: nil,
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := tc.msg.ValidateBasic()
			if tc.expectedErr != nil {
				require.ErrorIs(t, err, tc.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
