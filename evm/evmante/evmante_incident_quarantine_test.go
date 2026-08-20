package evmante_test

import (
	"testing"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/evm"
	"github.com/NibiruChain/nibiru/v2/evm/evmante"
	"github.com/NibiruChain/nibiru/v2/evm/evmtest"
)

func TestAnteStepIncidentQuarantine(t *testing.T) {
	attacker := "0x947Be8Ce20F2b6deFC3ed8593aaebF5E9974841B"
	unrelated := gethcommon.HexToAddress("0x1111111111111111111111111111111111111111").Hex()

	tests := []struct {
		name      string
		chainID   string
		from      string
		isCheckTx bool
		wantErr   bool
	}{
		{
			name:      "mainnet CheckTx rejects attacker EVM alias",
			chainID:   appconst.SDK_CHAIN_ID_MAINNET,
			from:      attacker,
			isCheckTx: true,
			wantErr:   true,
		},
		{
			name:    "mainnet DeliverTx rejects attacker EVM alias",
			chainID: appconst.SDK_CHAIN_ID_MAINNET,
			from:    attacker,
			wantErr: true,
		},
		{
			name:    "mainnet permits unrelated EVM sender",
			chainID: appconst.SDK_CHAIN_ID_MAINNET,
			from:    unrelated,
		},
		{
			name:    "testnet permits incident address",
			chainID: "nibiru-testnet-2",
			from:    attacker,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			deps := evmtest.NewTestDeps()
			deps.SetCtx(
				deps.Ctx().
					WithChainID(tc.chainID).
					WithIsCheckTx(tc.isCheckTx),
			)
			sdb := deps.NewStateDB()
			msg := &evm.MsgEthereumTx{From: tc.from}

			err := evmante.AnteStepIncidentQuarantine(
				sdb, sdb.Keeper(), msg, false, ANTE_OPTIONS_UNUSED,
			)
			if tc.wantErr {
				require.ErrorContains(t, err, "is quarantined")
				return
			}
			require.NoError(t, err)
		})
	}
}
