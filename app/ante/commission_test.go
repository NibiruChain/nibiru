package ante_test

import (
	"testing"

	"cosmossdk.io/math"
	sdkclienttx "github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/app/ante"
	"github.com/NibiruChain/nibiru/x/common/testutil"
)

func (s *AnteTestSuite) TestAnteDecoratorStakingCommission() {
	// nextAnteHandler: A no-op next handler to make this a unit test.
	var nextAnteHandler sdk.AnteHandler = func(
		ctx sdk.Context, tx sdk.Tx, simulate bool,
	) (newCtx sdk.Context, err error) {
		return ctx, nil
	}

	mockDescription := stakingtypes.Description{
		Moniker:         "mock-moniker",
		Identity:        "mock-identity",
		Website:         "mock-website",
		SecurityContact: "mock-security-contact",
		Details:         "mock-details",
	}

	valAddr := sdk.ValAddress(testutil.AccAddress()).String()
	commissionRatePointer := new(math.LegacyDec)
	*commissionRatePointer = math.LegacyNewDecWithPrec(10, 2)
	happyMsgs := []sdk.Msg{
		&stakingtypes.MsgCreateValidator{
			Description: mockDescription,
			Commission: stakingtypes.CommissionRates{
				Rate:          math.LegacyNewDecWithPrec(6, 2), // 6%
				MaxRate:       math.LegacyNewDec(420),
				MaxChangeRate: math.LegacyNewDec(420),
			},
			MinSelfDelegation: math.NewInt(1),
			DelegatorAddress:  testutil.AccAddress().String(),
			ValidatorAddress:  valAddr,
			Pubkey:            &codectypes.Any{},
			Value:             sdk.NewInt64Coin("unibi", 1),
		},
		&stakingtypes.MsgEditValidator{
			Description:       mockDescription,
			ValidatorAddress:  valAddr,
			CommissionRate:    commissionRatePointer, // 10%
			MinSelfDelegation: nil,
		},
	}

	createSadMsgs := func() []sdk.Msg {
		sadMsgCreateVal := new(stakingtypes.MsgCreateValidator)
		*sadMsgCreateVal = *(happyMsgs[0]).(*stakingtypes.MsgCreateValidator)
		sadMsgCreateVal.Commission.Rate = math.LegacyNewDecWithPrec(26, 2)

		sadMsgEditVal := new(stakingtypes.MsgEditValidator)
		*sadMsgEditVal = *(happyMsgs[1]).(*stakingtypes.MsgEditValidator)
		newCommissionRate := new(math.LegacyDec)
		*newCommissionRate = math.LegacyNewDecWithPrec(26, 2)
		sadMsgEditVal.CommissionRate = newCommissionRate

		return []sdk.Msg{
			sadMsgCreateVal,
			sadMsgEditVal,
		}
	}
	sadMsgs := createSadMsgs()

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
				sdk.NewCoin("unibi", math.NewInt(1_000)),
				sdk.NewCoin("utoken", math.NewInt(500)),
			)

			encCfg := app.MakeEncodingConfig()
			txBuilder, err := sdkclienttx.Factory{}.
				WithFees(txGasCoins.String()).
				WithChainID(s.ctx.ChainID()).
				WithTxConfig(encCfg.TxConfig).
				WithChainID("nibi-test-chain").
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
