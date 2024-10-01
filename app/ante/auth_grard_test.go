package ante_test

import (
	"time"

	sdkclienttx "github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/app/ante"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
)

func (s *AnteTestSuite) TestAnteDecoratorAuthzGuard() {
	testCases := []struct {
		name    string
		txMsg   func() sdk.Msg
		wantErr string
	}{
		{
			name: "sad: authz generic grant with evm message",
			txMsg: func() sdk.Msg {
				someTime := time.Now()
				expiryTime := someTime.Add(time.Hour)
				genericGrant, err := authz.NewGrant(
					someTime,
					authz.NewGenericAuthorization(sdk.MsgTypeURL(&evm.MsgEthereumTx{})), &expiryTime,
				)
				s.Require().NoError(err)
				return &authz.MsgGrant{Grant: genericGrant}
			},
			wantErr: "not allowed",
		},
		{
			name: "happy: authz generic grant with non evm message",
			txMsg: func() sdk.Msg {
				someTime := time.Now()
				expiryTime := someTime.Add(time.Hour)
				genericGrant, err := authz.NewGrant(
					someTime,
					authz.NewGenericAuthorization(sdk.MsgTypeURL(&stakingtypes.MsgCreateValidator{})), &expiryTime,
				)
				s.Require().NoError(err)
				return &authz.MsgGrant{Grant: genericGrant}
			},
			wantErr: "",
		},
		{
			name: "happy: authz non generic grant",
			txMsg: func() sdk.Msg {
				someTime := time.Now()
				expiryTime := someTime.Add(time.Hour)
				genericGrant, err := authz.NewGrant(
					someTime,
					&banktypes.SendAuthorization{},
					&expiryTime,
				)
				s.Require().NoError(err)
				return &authz.MsgGrant{Grant: genericGrant}
			},
			wantErr: "",
		},
		{
			name: "happy: non authz message",
			txMsg: func() sdk.Msg {
				return &evm.MsgEthereumTx{}
			},
			wantErr: "",
		},
		{
			name: "sad: authz exec with a single evm message",
			txMsg: func() sdk.Msg {
				msgExec := authz.NewMsgExec(
					sdk.AccAddress("nibiuser"),
					[]sdk.Msg{
						&evm.MsgEthereumTx{},
					},
				)
				return &msgExec
			},
			wantErr: "ExtensionOptionsEthereumTx",
		},
		{
			name: "sad: authz exec with evm message and non evm message",
			txMsg: func() sdk.Msg {
				msgExec := authz.NewMsgExec(
					sdk.AccAddress("nibiuser"),
					[]sdk.Msg{
						&banktypes.MsgSend{},
						&evm.MsgEthereumTx{},
					},
				)
				return &msgExec
			},
			wantErr: "ExtensionOptionsEthereumTx",
		},
		{
			name: "happy: authz exec without evm messages",
			txMsg: func() sdk.Msg {
				msgExec := authz.NewMsgExec(
					sdk.AccAddress("nibiuser"),
					[]sdk.Msg{
						&banktypes.MsgSend{},
					},
				)
				return &msgExec
			},
			wantErr: "",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			deps := evmtest.NewTestDeps()
			anteDec := ante.AnteDecoratorAuthzGuard{}

			encCfg := app.MakeEncodingConfig()
			txBuilder, err := sdkclienttx.Factory{}.
				WithChainID(s.ctx.ChainID()).
				WithTxConfig(encCfg.TxConfig).
				BuildUnsignedTx(tc.txMsg())
			s.Require().NoError(err)

			_, err = anteDec.AnteHandle(
				deps.Ctx, txBuilder.GetTx(), false, evmtest.NextNoOpAnteHandler,
			)
			if tc.wantErr != "" {
				s.Require().ErrorContains(err, tc.wantErr)
				return
			}
			s.Require().NoError(err)
		})
	}
}
