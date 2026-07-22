package evmante

import (
	"fmt"

	cmttypes "github.com/cometbft/cometbft/types"

	"github.com/NibiruChain/nibiru/v2/evm"
	"github.com/NibiruChain/nibiru/v2/evm/evmstate"
)

var _ AnteStep = AnteStepMempoolAdmission

// AnteStepMempoolAdmission rejects an authenticated CheckTxType_New EVM
// transaction when its sender and nonce slot is occupied or its sender has
// reached the live-slot limit. The check is read-only; BaseApp reserves the
// slot through evm.Mempool.Insert after every ante step succeeds.
func AnteStepMempoolAdmission(
	sdb *evmstate.SDB,
	k *evmstate.Keeper,
	msgEthTx *evm.MsgEthereumTx,
	simulate bool,
	opts AnteOptionsEVM,
) error {
	ctx := sdb.Ctx()
	if !ctx.IsCheckTx() || ctx.IsReCheckTx() || simulate {
		return nil
	}
	pool := opts.GetEVMMempool()
	if pool == nil {
		return nil
	}
	if len(ctx.TxBytes()) == 0 {
		return fmt.Errorf("EVM mempool admission requires original transaction bytes")
	}
	txData, err := evm.UnpackTxData(msgEthTx.Data)
	if err != nil {
		return err
	}
	return pool.CheckNewTx(
		cmttypes.Tx(ctx.TxBytes()).Key(),
		msgEthTx.FromAddr(),
		txData.GetNonce(),
	)
}
