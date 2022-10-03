package types

import (
	"testing"
	"time"

	"github.com/NibiruChain/nibiru/x/testutil"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestMsgCreateIncentivizationProgram_ValidateBasic(t *testing.T) {
	type test struct {
		msg     *MsgCreateIncentivizationProgram
		wantErr string
	}
	validAddr := testutil.AccAddress().String()
	validDenom := "denom"
	validDuration := 48 * time.Hour
	validTime := time.Now()
	validEpochs := int64(100)
	validInitialFunds := sdk.NewCoins(sdk.NewInt64Coin("test", 1000))

	cases := map[string]test{
		"success": {
			msg: &MsgCreateIncentivizationProgram{
				Sender:            validAddr,
				LpDenom:           validDenom,
				MinLockupDuration: &validDuration,
				StartTime:         &validTime,
				Epochs:            validEpochs,
				InitialFunds:      validInitialFunds,
			},
			wantErr: "",
		},
		"time is invalid": {
			msg: &MsgCreateIncentivizationProgram{
				Sender:            validAddr,
				LpDenom:           validDenom,
				MinLockupDuration: &validDuration,
				StartTime:         &time.Time{},
				Epochs:            validEpochs,
				InitialFunds:      validInitialFunds,
			},
			wantErr: "invalid time",
		},
		"lp denom is invalid": {
			msg: &MsgCreateIncentivizationProgram{
				Sender:            validAddr,
				LpDenom:           "",
				MinLockupDuration: &validDuration,
				StartTime:         &validTime,
				Epochs:            validEpochs,
				InitialFunds:      validInitialFunds,
			},
			wantErr: "invalid denom",
		},
		"invalid addr": {
			msg: &MsgCreateIncentivizationProgram{
				Sender:            "",
				LpDenom:           validDenom,
				MinLockupDuration: &validDuration,
				StartTime:         &validTime,
				Epochs:            validEpochs,
				InitialFunds:      validInitialFunds,
			},
			wantErr: "invalid address",
		},
		"invalid epochs": {
			msg: &MsgCreateIncentivizationProgram{
				Sender:            validAddr,
				LpDenom:           validDenom,
				MinLockupDuration: &validDuration,
				StartTime:         &validTime,
				Epochs:            0,
				InitialFunds:      validInitialFunds,
			},
			wantErr: "invalid epochs",
		},
		"invalid min lock duration - nil": {
			msg: &MsgCreateIncentivizationProgram{
				Sender:            validAddr,
				LpDenom:           validDenom,
				MinLockupDuration: nil,
				StartTime:         &validTime,
				Epochs:            validEpochs,
				InitialFunds:      validInitialFunds,
			},
			wantErr: "invalid duration",
		},
		"invalid min lock duration - zero": {
			msg: &MsgCreateIncentivizationProgram{
				Sender:            validAddr,
				LpDenom:           validDenom,
				MinLockupDuration: func() *time.Duration { d := time.Duration(0); return &d }(),
				StartTime:         &validTime,
				Epochs:            validEpochs,
				InitialFunds:      validInitialFunds,
			},
			wantErr: "invalid duration",
		},
		"invalid initial funds": {
			msg: &MsgCreateIncentivizationProgram{
				Sender:            validAddr,
				LpDenom:           validDenom,
				MinLockupDuration: &validDuration,
				StartTime:         &validTime,
				Epochs:            validEpochs,
				InitialFunds: sdk.Coins{sdk.Coin{
					Denom:  "dKSAODKOASKDOASKD_CDSADC_SA",
					Amount: sdk.Int{},
				}},
			},
			wantErr: "invalid initial funds",
		},
	}

	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			err := tc.msg.ValidateBasic()
			if tc.wantErr == "" && err != nil {
				t.Fatalf("unexpected error: %s", err.Error())
			}
			if tc.wantErr != "" && err == nil {
				t.Fatalf("expected error: %s", tc.wantErr)
			}
			if tc.wantErr != "" && err != nil {
				require.Contains(t, err.Error(), tc.wantErr)
			}
		})
	}
}

func TestMsgFundIncentivizationProgram_ValidateBasic(t *testing.T) {
	type test struct {
		msg     *MsgFundIncentivizationProgram
		wantErr string
	}

	cases := map[string]test{
		"success": {
			msg: &MsgFundIncentivizationProgram{
				Sender: testutil.AccAddress().String(),
				Id:     0,
				Funds:  sdk.NewCoins(sdk.NewInt64Coin("test", 1000)),
			},
		},
		"invalid funds": {
			msg: &MsgFundIncentivizationProgram{
				Sender: testutil.AccAddress().String(),
				Id:     0,
				Funds: sdk.Coins{sdk.Coin{
					Denom:  "dKSAODKOASKDOASKD_CDSADC_SA",
					Amount: sdk.Int{},
				}},
			},
			wantErr: "invalid funds",
		},
		"zero funds": {
			msg: &MsgFundIncentivizationProgram{
				Sender: testutil.AccAddress().String(),
				Id:     0,
			},
			wantErr: "no funding provided",
		},
	}

	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			err := tc.msg.ValidateBasic()
			if tc.wantErr == "" && err != nil {
				t.Fatalf("unexpected error: %s", err.Error())
			}
			if tc.wantErr != "" && err == nil {
				t.Fatalf("expected error: %s", tc.wantErr)
			}
			if tc.wantErr != "" && err != nil {
				require.Contains(t, err.Error(), tc.wantErr)
			}
		})
	}
}
