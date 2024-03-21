package ante_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/app/ante"
	"github.com/NibiruChain/nibiru/x/common/testutil"
)

func (s *AnteTestSuite) TestAnteDecoratorStakingCommission() {
	// Define a no-op next handler to make this a unit test.
	var nextAnteHandler sdk.AnteHandler = func(
		ctx sdk.Context, tx sdk.Tx, simulate bool,
	) (newCtx sdk.Context, err error) {
		return ctx, nil
	}

	// Mock description for testing.
	mockDescription := stakingtypes.Description{
		Moniker:         "mock-moniker",
		Identity:        "mock-identity",
		Website:         "mock-website",
		SecurityContact: "mock-security-contact",
		Details:         "mock-details",
	}

	// Define validator address.
	valAddr := sdk.ValAddress(testutil.AccAddress()).String()

	// Define commission rate.
	commissionRate := sdk.NewDecWithPrec(10, 2)

	// Define happy messages.
	happyMsgs := []sdk.Msg{
		&stakingtypes.MsgCreateValidator{
			Description:       mockDescription,
			Commission:        stakingtypes.CommissionRates{Rate: sdk.NewDecWithPrec(6, 2)},
			MinSelfDelegation: sdk.NewInt(1),
			DelegatorAddress:  testutil.AccAddress().String(),
			ValidatorAddress:  valAddr,
			Pubkey:            nil,
			Value:             sdk.NewInt64Coin("unibi", 1),
		},
		&stakingtypes.MsgEditValidator{
			Description:      mockDescription,
			ValidatorAddress: valAddr,
			CommissionRate:   &commissionRate,
		},
	}

	// Define function to create sad messages.
	createSadMsgs := func() []sdk.Msg {
		sadMsgCreateVal := *(happyMsgs[0]).(*stakingtypes.MsgCreateValidator)
		sadMsgCreateVal.Commission.Rate = sdk.NewDecWithPrec(26, 2)

		sadMsgEditVal := *(happyMsgs[1]).(*stakingtypes.MsgEditValidator)
		newCommissionRate := sdk.NewDecWithPrec(26, 2)
		sadMsgEditVal.CommissionRate = &newCommissionRate

		return []sdk.Msg{
			&sadMsgCreateVal,
			&sadMsgEditVal,
		}
	}
	sadMsgs := createSadMsgs()

	// Test cases.
	for _, tc := range []struct {
		name    string
		txMsgs  []sdk.Msg
		wantErr string
	}{
		{
			name:    "happy blank",
			txMsgs:  []sdk.Msg{},
			wantErr: "",
		},
		{
			name: "happy msgs",
			txMsgs: []sdk.Msg{
				happyMsgs[0],
				happyMsgs[1],
			},
			wantErr: "",
		},
		{
			name: "sad: max commission on create validator",
			txMsgs: []sdk.Msg{
				sadMsgs[0],
				happyMsgs[1],
			},
			wantErr: ante.ErrMaxValidatorCommission.Error(),
		},
		{
			name: "sad: max commission on edit validator",
			txMsgs: []sdk.Msg{
				happyMsgs[0],
				sadMsgs[1],
			},
			wantErr: ante.ErrMaxValidatorCommission.Error(),
		},
	} {
		s.T().Run(tc.name, func(t *testing.T) {
			txGasCoins := sdk.NewCoins(
				sdk.NewCoin("unibi", sdk.NewInt(1_000)),
				sdk.NewCoin("utoken", sdk.NewInt(500)),
			)

			encCfg := app.MakeEncodingConfig()
			txBuilder, err := tx.Factory{}.
				WithFees(txGasCoins.String()).
				WithChainID(s.ctx.ChainID()).
				WithTxConfig(encCfg.TxConfig).
				BuildUnsignedTx(tc.txMsgs...)
			s.NoError(err)

			anteDecorator := ante.AnteDecoratorStakingCommission{}
			simulate := true
			s.ctx, err = anteDecorator.AnteHandle(
				s.ctx, txBuilder.GetTx(), simulate, nextAnteHandler,
			)

			if tc.wantErr != "" {
				s.ErrorContains(err, tc.wantErr)
				return
			}
			s.NoError(err)
		})
	}
}
