package types

import (
	"testing"
	"time"

	"github.com/NibiruChain/nibiru/x/testutil"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestMsgLockTokens_ValidateBasic(t *testing.T) {
	type test struct {
		msg     *MsgLockTokens
		wantErr string
	}

	validAddr := testutil.AccAddress().String()
	validDuration := 1 * time.Hour
	validCoins := sdk.NewCoins(sdk.NewInt64Coin("test", 100))

	cases := map[string]test{
		"success": {
			msg: &MsgLockTokens{
				Owner:    validAddr,
				Duration: validDuration,
				Coins:    validCoins,
			},
		},
		"invalid address": {
			msg: &MsgLockTokens{
				Owner:    "",
				Duration: validDuration,
				Coins:    validCoins,
			},
			wantErr: "invalid address",
		},
		"invalid coins": {
			msg: &MsgLockTokens{
				Owner:    validAddr,
				Duration: validDuration,
				Coins:    sdk.Coins{sdk.Coin{}},
			},
			wantErr: "invalid coins",
		},
		"zero coins": {
			msg: &MsgLockTokens{
				Owner:    validAddr,
				Duration: validDuration,
				Coins:    sdk.NewCoins(),
			},
			wantErr: "zero coins",
		},
		"invalid duration": {
			msg: &MsgLockTokens{
				Owner:    validAddr,
				Duration: 0,
				Coins:    validCoins,
			},
			wantErr: "duration should be positive",
		},
	}

	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			err := tc.msg.ValidateBasic()
			if tc.wantErr == "" && err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if tc.wantErr != "" && err == nil {
				t.Fatalf("expected error: %s", err)
			}
			if tc.wantErr != "" {
				require.Contains(t, err.Error(), tc.wantErr)
			}
		})
	}
}

func TestMsgInitiateUnlock_ValidateBasic(t *testing.T) {
	type test struct {
		msg     *MsgInitiateUnlock
		wantErr string
	}

	cases := map[string]test{
		"success": {
			msg: &MsgInitiateUnlock{
				Owner:  testutil.AccAddress().String(),
				LockId: 0,
			},
		},
		"invalid address": {
			msg:     &MsgInitiateUnlock{Owner: "invalid address", LockId: 0},
			wantErr: "invalid address",
		},
	}

	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			err := tc.msg.ValidateBasic()
			if tc.wantErr == "" && err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if tc.wantErr != "" && err == nil {
				t.Fatalf("expected error: %s", err)
			}
			if tc.wantErr != "" {
				require.Contains(t, err.Error(), tc.wantErr)
			}
		})
	}
}
