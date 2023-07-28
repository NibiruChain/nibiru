package types

import (
	"testing"

	"github.com/NibiruChain/nibiru/x/common/testutil"

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
						Weight: sdk.OneInt(),
					},
					{
						Token:  sdk.NewInt64Coin("bbb", 1),
						Weight: sdk.OneInt(),
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
						Weight: sdk.OneInt(),
					},
					{
						Token:  sdk.NewInt64Coin("bbb", 1),
						Weight: sdk.OneInt(),
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
						Weight: sdk.OneInt(),
					},
					{
						Token:  sdk.NewInt64Coin("bbb", 1),
						Weight: sdk.OneInt(),
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
						Weight: sdk.OneInt(),
					},
					{
						Token:  sdk.NewInt64Coin("bbb", 1),
						Weight: sdk.OneInt(),
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
						Weight: sdk.OneInt(),
					},
					{
						Token:  sdk.NewInt64Coin("bbb", 1),
						Weight: sdk.OneInt(),
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
						Weight: sdk.OneInt(),
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
						Weight: sdk.OneInt(),
					},
					{
						Token:  sdk.NewInt64Coin("bbb", 1),
						Weight: sdk.OneInt(),
					},
					{
						Token:  sdk.NewInt64Coin("ccc", 1),
						Weight: sdk.OneInt(),
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
						Weight: sdk.ZeroInt(),
					},
					{
						Token:  sdk.NewInt64Coin("bbb", 1),
						Weight: sdk.ZeroInt(),
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
						Weight: sdk.OneInt(),
					},
					{
						Token:  sdk.NewInt64Coin("bbb", 1),
						Weight: sdk.OneInt(),
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

func TestMsgExitPool_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgExitPool
		err  error
	}{
		{
			name: "invalid address",
			msg: MsgExitPool{
				Sender: "invalid_address",
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			name: "valid address",
			msg: MsgExitPool{
				Sender: testutil.AccAddress().String(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestMsgJoinPool_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgJoinPool
		err  error
	}{
		{
			name: "invalid address",
			msg: MsgJoinPool{
				Sender: "invalid_address",
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			name: "valid address",
			msg: MsgJoinPool{
				Sender: testutil.AccAddress().String(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestMsgSwapAssets_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgSwapAssets
		err  error
	}{
		{
			name: "invalid address",
			msg: MsgSwapAssets{
				Sender:        "invalid_address",
				PoolId:        1,
				TokenIn:       sdk.NewInt64Coin("foo", 1),
				TokenOutDenom: "bar",
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "invalid pool id",
			msg: MsgSwapAssets{
				Sender:        testutil.AccAddress().String(),
				PoolId:        0,
				TokenIn:       sdk.NewInt64Coin("foo", 1),
				TokenOutDenom: "bar",
			},
			err: ErrInvalidPoolId,
		},
		{
			name: "invalid tokens in",
			msg: MsgSwapAssets{
				Sender:        testutil.AccAddress().String(),
				PoolId:        1,
				TokenIn:       sdk.NewInt64Coin("foo", 0),
				TokenOutDenom: "bar",
			},
			err: ErrInvalidTokenIn,
		},
		{
			name: "invalid token out denom",
			msg: MsgSwapAssets{
				Sender:        testutil.AccAddress().String(),
				PoolId:        1,
				TokenIn:       sdk.NewInt64Coin("foo", 1),
				TokenOutDenom: "",
			},
			err: ErrInvalidTokenOutDenom,
		},
		{
			name: "valid message",
			msg: MsgSwapAssets{
				Sender:        testutil.AccAddress().String(),
				PoolId:        1,
				TokenIn:       sdk.NewInt64Coin("foo", 1),
				TokenOutDenom: "bar",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
		})
	}
}
