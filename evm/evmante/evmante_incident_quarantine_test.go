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

// TestAnteStepIncidentQuarantine covers the mainnet quarantine added after the
// August 2026 FunToken pool drain. The incident signer is the EOA that deployed
// the drain contract and signed the exploit transaction. Mainnet must reject
// that signer in CheckTx and DeliverTx without blocking unrelated accounts or
// applying the incident rule to testnet.
func TestAnteStepIncidentQuarantine(t *testing.T) {
	incidentSigner := "0x947Be8Ce20F2b6deFC3ed8593aaebF5E9974841B"
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
			from:      incidentSigner,
			isCheckTx: true,
			wantErr:   true,
		},
		{
			name:    "mainnet DeliverTx rejects attacker EVM alias",
			chainID: appconst.SDK_CHAIN_ID_MAINNET,
			from:    incidentSigner,
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
			from:    incidentSigner,
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
