package ante_test

import (
	"testing"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	sdkclienttx "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/client/tx"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/authz"
	banktypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/bank/types"
	wasmtypes "github.com/NibiruChain/nibiru/v2/x/wasm/types"

	"github.com/NibiruChain/nibiru/v2/app/ante"
	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/evm/evmtest"
)

func TestAnteDecIncidentQuarantine(t *testing.T) {
	attacker := eth.EthAddrToNibiruAddr(
		gethcommon.HexToAddress("0x947Be8Ce20F2b6deFC3ed8593aaebF5E9974841B"),
	)
	require.Equal(t, "nibi1j3a73n3q72mdalp7mpvn4t4lt6vhfpqm00fup8", attacker.String())
	other := sdk.AccAddress("unrelated-user-addr")
	eris := sdk.MustAccAddressFromBech32(
		"nibi1udqqx30cw8nwjxtl4l28ym9hhrp933zlq8dqxfjzcdhvl8y24zcqpzmh8m",
	)

	tests := []struct {
		name    string
		chainID string
		msg     sdk.Msg
		wantErr bool
	}{
		{
			name:    "mainnet direct bank message",
			chainID: appconst.SDK_CHAIN_ID_MAINNET,
			msg:     banktypes.NewMsgSend(attacker, other, sdk.NewCoins(sdk.NewInt64Coin("unibi", 1))),
			wantErr: true,
		},
		{
			name:    "mainnet direct Eris withdrawal",
			chainID: appconst.SDK_CHAIN_ID_MAINNET,
			msg: &wasmtypes.MsgExecuteContract{
				Sender:   attacker.String(),
				Contract: eris.String(),
				Msg:      []byte(`{"withdraw_unbonded":{}}`),
			},
			wantErr: true,
		},
		{
			name:    "mainnet Eris withdrawal with alternate receiver",
			chainID: appconst.SDK_CHAIN_ID_MAINNET,
			msg: &wasmtypes.MsgExecuteContract{
				Sender:   attacker.String(),
				Contract: eris.String(),
				Msg:      []byte(`{"withdraw_unbonded":{"receiver":"` + other.String() + `"}}`),
			},
			wantErr: true,
		},
		{
			name:    "mainnet authz grantee cannot exercise attacker authority",
			chainID: appconst.SDK_CHAIN_ID_MAINNET,
			msg: func() sdk.Msg {
				inner := banktypes.NewMsgSend(attacker, other, sdk.NewCoins(sdk.NewInt64Coin("unibi", 1)))
				exec := authz.NewMsgExec(other, []sdk.Msg{inner})
				return &exec
			}(),
			wantErr: true,
		},
		{
			name:    "mainnet nested authz cannot exercise attacker authority",
			chainID: appconst.SDK_CHAIN_ID_MAINNET,
			msg: func() sdk.Msg {
				innerMsg := banktypes.NewMsgSend(attacker, other, sdk.NewCoins(sdk.NewInt64Coin("unibi", 1)))
				innerExec := authz.NewMsgExec(other, []sdk.Msg{innerMsg})
				outerExec := authz.NewMsgExec(other, []sdk.Msg{&innerExec})
				return &outerExec
			}(),
			wantErr: true,
		},
		{
			name:    "mainnet unrelated account",
			chainID: appconst.SDK_CHAIN_ID_MAINNET,
			msg:     banktypes.NewMsgSend(other, attacker, sdk.NewCoins(sdk.NewInt64Coin("unibi", 1))),
		},
		{
			name:    "non-mainnet attacker address",
			chainID: "nibiru-testnet-2",
			msg:     banktypes.NewMsgSend(attacker, other, sdk.NewCoins(sdk.NewInt64Coin("unibi", 1))),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			deps := evmtest.NewTestDeps()
			ctx := deps.Ctx().WithChainID(tc.chainID)
			txBuilder, err := sdkclienttx.Factory{}.
				WithChainID(tc.chainID).
				WithTxConfig(deps.App.GetTxConfig()).
				BuildUnsignedTx(tc.msg)
			require.NoError(t, err)

			calledNext := false
			next := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
				calledNext = true
				return ctx, nil
			}
			_, err = (ante.AnteDecIncidentQuarantine{}).AnteHandle(
				ctx, txBuilder.GetTx(), false, next,
			)
			if tc.wantErr {
				require.ErrorContains(t, err, "is quarantined")
				require.False(t, calledNext)
				return
			}
			require.NoError(t, err)
			require.True(t, calledNext)
		})
	}
}

func TestAnteDecIncidentQuarantine_CheckAndDeliverMatch(t *testing.T) {
	deps := evmtest.NewTestDeps()
	attacker := eth.EthAddrToNibiruAddr(
		gethcommon.HexToAddress("0x947Be8Ce20F2b6deFC3ed8593aaebF5E9974841B"),
	)
	msg := banktypes.NewMsgSend(
		attacker,
		sdk.AccAddress("unrelated-user-addr"),
		sdk.NewCoins(sdk.NewInt64Coin("unibi", 1)),
	)
	txBuilder, err := sdkclienttx.Factory{}.
		WithChainID(appconst.SDK_CHAIN_ID_MAINNET).
		WithTxConfig(deps.App.GetTxConfig()).
		BuildUnsignedTx(msg)
	require.NoError(t, err)

	for _, isCheckTx := range []bool{true, false} {
		ctx := deps.Ctx().
			WithChainID(appconst.SDK_CHAIN_ID_MAINNET).
			WithIsCheckTx(isCheckTx)
		_, err := (ante.AnteDecIncidentQuarantine{}).AnteHandle(
			ctx, txBuilder.GetTx(), false, evmtest.NextNoOpAnteHandler,
		)
		require.ErrorContains(t, err, "is quarantined")
	}
}

func TestAnteDecIncidentQuarantine_RejectsFeeGrantUse(t *testing.T) {
	deps := evmtest.NewTestDeps()
	attacker := eth.EthAddrToNibiruAddr(
		gethcommon.HexToAddress("0x947Be8Ce20F2b6deFC3ed8593aaebF5E9974841B"),
	)
	other := sdk.AccAddress("unrelated-user-addr")
	msg := banktypes.NewMsgSend(
		other,
		sdk.AccAddress("another-user-address"),
		sdk.NewCoins(sdk.NewInt64Coin("unibi", 1)),
	)
	txBuilder, err := sdkclienttx.Factory{}.
		WithChainID(appconst.SDK_CHAIN_ID_MAINNET).
		WithTxConfig(deps.App.GetTxConfig()).
		BuildUnsignedTx(msg)
	require.NoError(t, err)
	txBuilder.SetFeeGranter(attacker)

	ctx := deps.Ctx().WithChainID(appconst.SDK_CHAIN_ID_MAINNET)
	_, err = (ante.AnteDecIncidentQuarantine{}).AnteHandle(
		ctx, txBuilder.GetTx(), false, evmtest.NextNoOpAnteHandler,
	)
	require.ErrorContains(t, err, "is quarantined and cannot grant fees")
}
