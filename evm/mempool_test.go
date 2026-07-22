package evm_test

import (
	"errors"
	"testing"

	cmttypes "github.com/cometbft/cometbft/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/evm"
	"github.com/NibiruChain/nibiru/v2/evm/evmtest"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	sdkmempool "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/mempool"
)

type testMempoolTx struct {
	ctx sdk.Context
	tx  sdk.Tx
	key cmttypes.TxKey
}

func newTestMempoolTx(
	t *testing.T,
	deps *evmtest.TestDeps,
	nonce uint64,
	priority int64,
) testMempoolTx {
	t.Helper()
	msg := evmtest.HappyTransferTx(deps, nonce)
	sender := msg.From
	gasLimit := msg.GetGas()
	tx := evmtest.BuildTx(deps, true, gasLimit, nil, msg)

	// Ethereum RPC transactions encode with an empty From field. Signature
	// verification populates From on the decoded transaction before insertion.
	msg.From = ""
	txBytes, err := deps.App.GetTxConfig().TxEncoder()(tx)
	require.NoError(t, err)
	msg.From = sender

	ctx := deps.Ctx().
		WithTxBytes(txBytes).
		WithPriority(priority).
		WithGasMeter(sdk.NewGasMeter(gasLimit))
	return testMempoolTx{
		ctx: ctx,
		tx:  tx,
		key: cmttypes.Tx(txBytes).Key(),
	}
}

func TestMempoolInsertAndSnapshot(t *testing.T) {
	deps := evmtest.NewTestDeps()
	pool := evm.NewMempool(2)
	tx0 := newTestMempoolTx(t, &deps, 0, 7)
	tx1 := newTestMempoolTx(t, &deps, 1, 9)

	require.NoError(t, pool.Insert(tx0.ctx, tx0.tx))
	require.NoError(t, pool.Insert(tx1.ctx, tx1.tx))
	require.Equal(t, 2, pool.CountTx())

	snapshot := pool.Snapshot()
	require.Len(t, snapshot, 1)
	require.Len(t, snapshot[0].Txs, 2)
	require.Equal(t, uint64(0), snapshot[0].Txs[0].Nonce)
	require.Equal(t, uint64(1), snapshot[0].Txs[1].Nonce)
	require.Equal(t, int64(7), snapshot[0].Txs[0].Priority)
	require.Equal(t, tx0.key, snapshot[0].Txs[0].TxKey)
	require.Equal(t, tx0.ctx.TxBytes(), snapshot[0].Txs[0].TxBytes)

	snapshot[0].Txs[0].TxBytes[0] ^= 0xff
	require.Equal(t, tx0.ctx.TxBytes(), pool.Snapshot()[0].Txs[0].TxBytes)
}

func TestMempoolExactRetransmissionIsIdempotent(t *testing.T) {
	deps := evmtest.NewTestDeps()
	pool := evm.NewMempool(1)
	tx := newTestMempoolTx(t, &deps, 0, 1)

	require.NoError(t, pool.Insert(tx.ctx, tx.tx))
	require.NoError(t, pool.CheckNewTx(tx.key, deps.Sender.EthAddr, 0))
	require.NoError(t, pool.Insert(tx.ctx, tx.tx))
	require.Equal(t, 1, pool.CountTx())
}

func TestMempoolRejectsCollisionAndSenderLimit(t *testing.T) {
	deps := evmtest.NewTestDeps()
	pool := evm.NewMempool(2)
	tx0 := newTestMempoolTx(t, &deps, 0, 1)
	collision := newTestMempoolTx(t, &deps, 0, 2)
	tx1 := newTestMempoolTx(t, &deps, 1, 1)
	tx2 := newTestMempoolTx(t, &deps, 2, 1)

	require.NoError(t, pool.Insert(tx0.ctx, tx0.tx))
	err := pool.CheckNewTx(collision.key, deps.Sender.EthAddr, 0)
	require.ErrorIs(t, err, evm.ErrMempoolNonceCollision)
	require.ErrorIs(t, pool.Insert(collision.ctx, collision.tx), evm.ErrMempoolNonceCollision)

	require.NoError(t, pool.Insert(tx1.ctx, tx1.tx))
	err = pool.CheckNewTx(tx2.key, deps.Sender.EthAddr, 2)
	require.ErrorIs(t, err, evm.ErrMempoolSenderLimit)
	require.ErrorIs(t, pool.Insert(tx2.ctx, tx2.tx), evm.ErrMempoolSenderLimit)
	require.Equal(t, 2, pool.CountTx())
}

func TestMempoolCheckRecheck(t *testing.T) {
	deps := evmtest.NewTestDeps()
	pool := evm.NewMempool(4)
	tx10 := newTestMempoolTx(t, &deps, 10, 1)
	tx11 := newTestMempoolTx(t, &deps, 11, 1)
	tx12 := newTestMempoolTx(t, &deps, 12, 1)

	for _, tx := range []testMempoolTx{tx10, tx11, tx12} {
		require.NoError(t, pool.Insert(tx.ctx, tx.tx))
	}
	require.NoError(t, pool.CheckRecheck(tx12.key, deps.Sender.EthAddr, 10, 12))
	require.ErrorIs(
		t,
		pool.CheckRecheck(tx11.key, deps.Sender.EthAddr, 9, 11),
		evm.ErrMempoolNonceGap,
	)
	require.ErrorIs(
		t,
		pool.CheckRecheck(tx10.key, deps.Sender.EthAddr, 10, 11),
		evm.ErrMempoolTxMismatch,
	)
}

func TestMempoolRemoveByTxKey(t *testing.T) {
	deps := evmtest.NewTestDeps()
	pool := evm.NewMempool(2)
	tx := newTestMempoolTx(t, &deps, 0, 1)
	require.NoError(t, pool.Insert(tx.ctx, tx.tx))

	require.NoError(t, pool.RemoveByTxKey(tx.key))
	require.Zero(t, pool.CountTx())
	require.Empty(t, pool.Snapshot())
	require.ErrorIs(t, pool.RemoveByTxKey(tx.key), sdkmempool.ErrTxNotFound)
}

func TestMempoolTracksMinimumLiveNonce(t *testing.T) {
	deps := evmtest.NewTestDeps()
	pool := evm.NewMempool(4)
	tx10 := newTestMempoolTx(t, &deps, 10, 1)
	tx11 := newTestMempoolTx(t, &deps, 11, 1)
	tx12 := newTestMempoolTx(t, &deps, 12, 1)

	for _, tx := range []testMempoolTx{tx12, tx10, tx11} {
		require.NoError(t, pool.Insert(tx.ctx, tx.tx))
	}
	require.Equal(t, uint64(10), pool.Snapshot()[0].MinNonce)

	require.NoError(t, pool.RemoveByTxKey(tx11.key))
	require.Equal(t, uint64(10), pool.Snapshot()[0].MinNonce)

	require.NoError(t, pool.RemoveByTxKey(tx10.key))
	require.Equal(t, uint64(12), pool.Snapshot()[0].MinNonce)

	require.NoError(t, pool.RemoveByTxKey(tx12.key))
	require.Empty(t, pool.Snapshot())
}

func TestMempoolIgnoresNonEVMTransactions(t *testing.T) {
	deps := evmtest.NewTestDeps()
	pool := evm.NewMempool(1)
	tx := deps.App.GetTxConfig().NewTxBuilder().GetTx()

	require.NoError(t, pool.Insert(deps.Ctx(), tx))
	require.NoError(t, pool.Remove(tx))
	require.Zero(t, pool.CountTx())
}

func TestNewMempoolRejectsZeroLimit(t *testing.T) {
	require.Panics(t, func() { evm.NewMempool(0) })
}

func TestMempoolErrorsAreStable(t *testing.T) {
	require.True(t, errors.Is(evm.ErrMempoolNonceCollision, evm.ErrMempoolNonceCollision))
}
