package evm_test

import (
	"errors"
	"testing"

	cmttypes "github.com/cometbft/cometbft/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/evm"
	"github.com/NibiruChain/nibiru/v2/evm/evmtest"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
)

// testMempoolTx is a fixture for [evm.Mempool] tests that retains the insertion
// context, decoded transaction, and outer [cmttypes.TxKey].
type testMempoolTx struct {
	ctx sdk.Context
	tx  sdk.Tx
	key cmttypes.TxKey
}

// newTestMempoolTx builds an EVM transfer with RPC-shaped encoding: empty
// [evm.MsgEthereumTx.From] in the outer bytes, then restores the authenticated
// sender on the decoded message for [evm.Mempool.Insert].
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

// TestMempoolInsertAndSnapshot proves [evm.Mempool.Snapshot] is a deep copy:
// mutating returned [evm.MempoolTx.TxBytes] does not change pool state, and
// original outer bytes and priorities are preserved.
func TestMempoolInsertAndSnapshot(t *testing.T) {
	deps := evmtest.NewTestDeps()
	mp := evm.NewMempool(2)
	tx0 := newTestMempoolTx(t, &deps, 0, 7)
	tx1 := newTestMempoolTx(t, &deps, 1, 9)

	require.NoError(t, mp.Insert(tx0.ctx, tx0.tx))
	require.NoError(t, mp.Insert(tx1.ctx, tx1.tx))
	require.Equal(t, 2, mp.CountTx())

	snapshot := mp.Snapshot()
	require.Len(t, snapshot, 1)
	require.Len(t, snapshot[0].Txs, 2)
	require.Equal(t, uint64(0), snapshot[0].Txs[0].Nonce)
	require.Equal(t, uint64(1), snapshot[0].Txs[1].Nonce)
	require.Equal(t, int64(7), snapshot[0].Txs[0].Priority)
	require.Equal(t, tx0.key, snapshot[0].Txs[0].TxKey)
	require.Equal(t, tx0.ctx.TxBytes(), snapshot[0].Txs[0].TxBytes)

	snapshot[0].Txs[0].TxBytes[0] ^= 0xff
	require.Equal(t, tx0.ctx.TxBytes(), mp.Snapshot()[0].Txs[0].TxBytes)
}

// TestMempoolExactRetransmissionIsIdempotent proves that inserting the same
// [cmttypes.TxKey] twice is a no-op. CometBFT may call CheckTxType_New again
// after transaction-cache eviction while the application entry is still live.
func TestMempoolExactRetransmissionIsIdempotent(t *testing.T) {
	deps := evmtest.NewTestDeps()
	mp := evm.NewMempool(1)
	tx := newTestMempoolTx(t, &deps, 0, 1)

	require.NoError(t, mp.Insert(tx.ctx, tx.tx))
	require.NoError(t, mp.CheckNewTx(tx.key, deps.Sender.EthAddr, 0))
	require.NoError(t, mp.Insert(tx.ctx, tx.tx))
	require.Equal(t, 1, mp.CountTx())
}

// TestMempoolRejectsCollisionAndSenderLimit proves no fee replacement: a
// different [cmttypes.TxKey] for an occupied (sender, nonce) returns
// [evm.ErrMempoolNonceCollision], and a third distinct slot fails with
// [evm.ErrMempoolSenderLimit] when the limit is 2.
func TestMempoolRejectsCollisionAndSenderLimit(t *testing.T) {
	deps := evmtest.NewTestDeps()
	mp := evm.NewMempool(2)
	tx0 := newTestMempoolTx(t, &deps, 0, 1)
	collision := newTestMempoolTx(t, &deps, 0, 2)
	tx1 := newTestMempoolTx(t, &deps, 1, 1)
	tx2 := newTestMempoolTx(t, &deps, 2, 1)

	require.NoError(t, mp.Insert(tx0.ctx, tx0.tx))
	err := mp.CheckNewTx(collision.key, deps.Sender.EthAddr, 0)
	require.ErrorIs(t, err, evm.ErrMempoolNonceCollision)
	require.ErrorIs(t, mp.Insert(collision.ctx, collision.tx), evm.ErrMempoolNonceCollision)

	require.NoError(t, mp.Insert(tx1.ctx, tx1.tx))
	err = mp.CheckNewTx(tx2.key, deps.Sender.EthAddr, 2)
	require.ErrorIs(t, err, evm.ErrMempoolSenderLimit)
	require.ErrorIs(t, mp.Insert(tx2.ctx, tx2.tx), evm.ErrMempoolSenderLimit)
	require.Equal(t, 2, mp.CountTx())
}

// TestMempoolCheckRecheck proves [evm.Mempool.CheckRecheck] retains a complete
// state nonce chain from the committed state nonce, rejects tails after a gap
// ([evm.ErrMempoolNonceGap]), and rejects a [cmttypes.TxKey] that does not own
// the claimed slot ([evm.ErrMempoolTxMismatch]).
func TestMempoolCheckRecheck(t *testing.T) {
	deps := evmtest.NewTestDeps()
	mp := evm.NewMempool(4)
	tx10 := newTestMempoolTx(t, &deps, 10, 1)
	tx11 := newTestMempoolTx(t, &deps, 11, 1)
	tx12 := newTestMempoolTx(t, &deps, 12, 1)

	for _, tx := range []testMempoolTx{tx10, tx11, tx12} {
		require.NoError(t, mp.Insert(tx.ctx, tx.tx))
	}
	require.NoError(t, mp.CheckRecheck(tx12.key, deps.Sender.EthAddr, 10, 12))
	require.ErrorIs(
		t,
		mp.CheckRecheck(tx11.key, deps.Sender.EthAddr, 9, 11),
		evm.ErrMempoolNonceGap,
	)
	require.ErrorIs(
		t,
		mp.CheckRecheck(tx10.key, deps.Sender.EthAddr, 10, 11),
		evm.ErrMempoolTxMismatch,
	)
}

// TestMempoolRemoveByTxKey proves [evm.Mempool.RemoveByTxKey] removes by outer
// [cmttypes.TxKey] and is idempotent for failed-recheck and DeliverTx cleanup.
func TestMempoolRemoveByTxKey(t *testing.T) {
	deps := evmtest.NewTestDeps()
	mp := evm.NewMempool(2)
	tx := newTestMempoolTx(t, &deps, 0, 1)
	require.NoError(t, mp.Insert(tx.ctx, tx.tx))

	require.NoError(t, mp.RemoveByTxKey(tx.key))
	require.Zero(t, mp.CountTx())
	require.Empty(t, mp.Snapshot())
	require.NoError(t, mp.RemoveByTxKey(tx.key))
}

// TestMempoolTracksMinimumLiveNonce proves [evm.MempoolSender.MinNonce] tracks
// the lowest live slot after out-of-order insertion and removals, not the
// committed state nonce, and that the sender entry is deleted when empty.
func TestMempoolTracksMinimumLiveNonce(t *testing.T) {
	deps := evmtest.NewTestDeps()
	mp := evm.NewMempool(4)
	tx10 := newTestMempoolTx(t, &deps, 10, 1)
	tx11 := newTestMempoolTx(t, &deps, 11, 1)
	tx12 := newTestMempoolTx(t, &deps, 12, 1)

	for _, tx := range []testMempoolTx{tx12, tx10, tx11} {
		require.NoError(t, mp.Insert(tx.ctx, tx.tx))
	}
	require.Equal(t, uint64(10), mp.Snapshot()[0].MinNonce)

	require.NoError(t, mp.RemoveByTxKey(tx11.key))
	require.Equal(t, uint64(10), mp.Snapshot()[0].MinNonce)

	require.NoError(t, mp.RemoveByTxKey(tx10.key))
	require.Equal(t, uint64(12), mp.Snapshot()[0].MinNonce)

	require.NoError(t, mp.RemoveByTxKey(tx12.key))
	require.Empty(t, mp.Snapshot())
}

// TestMempoolIgnoresNonEVMTransactions proves [evm.Mempool.Insert] and
// [evm.Mempool.Remove] are no-ops for non-EVM transactions; PrepareProposal
// obtains non-EVM candidates from RequestPrepareProposal.Txs.
func TestMempoolIgnoresNonEVMTransactions(t *testing.T) {
	deps := evmtest.NewTestDeps()
	mp := evm.NewMempool(1)
	tx := deps.App.GetTxConfig().NewTxBuilder().GetTx()

	require.NoError(t, mp.Insert(deps.Ctx(), tx))
	require.NoError(t, mp.Remove(tx))
	require.Zero(t, mp.CountTx())
}

// TestNewMempoolRejectsZeroLimit proves [evm.NewMempool] panics when the live
// slot limit is zero so production cannot construct an unbound mempool.
func TestNewMempoolRejectsZeroLimit(t *testing.T) {
	require.Panics(t, func() { evm.NewMempool(0) })
}

// TestMempoolErrorsAreStable proves mempool sentinel errors remain comparable
// with [errors.Is] for ante and BaseApp callers.
func TestMempoolErrorsAreStable(t *testing.T) {
	require.True(t, errors.Is(evm.ErrMempoolNonceCollision, evm.ErrMempoolNonceCollision))
}
