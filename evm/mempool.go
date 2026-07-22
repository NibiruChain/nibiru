package evm

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"

	cmttypes "github.com/cometbft/cometbft/types"
	gethcommon "github.com/ethereum/go-ethereum/common"

	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	sdkmempool "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/mempool"
)

var (
	_ sdkmempool.Mempool  = (*Mempool)(nil)
	_ sdkmempool.Iterator = (*mempoolIterator)(nil)

	// ErrMempoolNonceCollision means another transaction owns the requested
	// authenticated sender and EVM nonce slot.
	ErrMempoolNonceCollision = errors.New("EVM mempool nonce slot already occupied")
	// ErrMempoolSenderLimit means a sender owns the configured maximum number
	// of live EVM nonce slots.
	ErrMempoolSenderLimit = errors.New("EVM mempool sender slot limit reached")
	// ErrMempoolNonceGap means a transaction is not part of a complete state
	// nonce chain during CheckTxType_Recheck.
	ErrMempoolNonceGap = errors.New("EVM mempool state nonce chain is incomplete")
	// ErrMempoolTxMismatch means a transaction does not own the nonce slot it
	// identifies during recheck or decoded-transaction removal.
	ErrMempoolTxMismatch = errors.New("EVM mempool transaction does not own nonce slot")
)

// MempoolTx is an immutable snapshot of one EVM transaction held by Mempool.
//
// TxBytes contains the original outer SDK transaction bytes received by
// BaseApp.CheckTx. Proposal construction must return these bytes instead of
// re-encoding the decoded transaction because EVM ante validation populates
// MsgEthereumTx.From.
type MempoolTx struct {
	TxBytes   []byte
	TxKey     cmttypes.TxKey
	EVMHash   gethcommon.Hash
	Sender    gethcommon.Address
	Nonce     uint64
	Priority  int64
	GasWanted uint64
	ArrivalID uint64
}

// MempoolSender is a point-in-time copy of one sender's live EVM nonce slots.
// MinNonce is the lowest live nonce in Txs; it is not necessarily the sender's
// committed state nonce. Txs is ordered by EVM nonce in ascending order and may
// contain nonce gaps.
type MempoolSender struct {
	Sender   gethcommon.Address
	MinNonce uint64
	Txs      []MempoolTx
}

type mempoolSlotKey struct {
	sender gethcommon.Address
	nonce  uint64
}

type mempoolTx struct {
	tx sdk.Tx
	MempoolTx
}

type mempoolSender struct {
	slots    map[uint64]*mempoolTx
	minNonce uint64
}

// Mempool is Nibiru's node-local EVM transaction index.
//
// Mempool implements the Cosmos SDK mempool interface but indexes only
// standard EVM transactions. Non-EVM insertion and removal are no-ops because
// the custom PrepareProposal function obtains non-EVM candidates from
// RequestPrepareProposal.Txs.
//
// Each authenticated sender may own at most maxSlotsPerSender live nonce
// slots. The first accepted transaction owns a sender and nonce slot; a
// different transaction for that slot is rejected rather than replacing it.
// Mempool is process-local and must not participate in consensus validation.
type Mempool struct {
	mu sync.RWMutex

	bySender map[gethcommon.Address]*mempoolSender
	byTxKey  map[cmttypes.TxKey]mempoolSlotKey

	txCount           int
	nextArrivalID     uint64
	maxSlotsPerSender uint64
}

// NewMempool returns an empty EVM mempool with the given live-slot limit.
// maxSlotsPerSender must be greater than zero.
func NewMempool(maxSlotsPerSender uint64) *Mempool {
	if maxSlotsPerSender == 0 {
		panic("EVM mempool max slots per sender must be greater than zero")
	}
	return &Mempool{
		bySender:          make(map[gethcommon.Address]*mempoolSender),
		byTxKey:           make(map[cmttypes.TxKey]mempoolSlotKey),
		maxSlotsPerSender: maxSlotsPerSender,
	}
}

// CheckNewTx performs the read-only CheckTxType_New admission guard. A
// successful result does not reserve a nonce slot; Insert repeats the checks
// atomically after the complete ante handler succeeds.
func (m *Mempool) CheckNewTx(
	txKey cmttypes.TxKey,
	sender gethcommon.Address,
	nonce uint64,
) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.checkNewTx(txKey, sender, nonce)
}

func (m *Mempool) checkNewTx(
	txKey cmttypes.TxKey,
	sender gethcommon.Address,
	nonce uint64,
) error {
	senderEntry, found := m.bySender[sender]
	if !found {
		return nil
	}
	if existing, found := senderEntry.slots[nonce]; found {
		if existing.TxKey == txKey {
			return nil
		}
		return fmt.Errorf(
			"%w: sender %s, nonce %d", ErrMempoolNonceCollision, sender, nonce,
		)
	}
	if uint64(len(senderEntry.slots)) >= m.maxSlotsPerSender {
		return fmt.Errorf(
			"%w: sender %s, limit %d",
			ErrMempoolSenderLimit, sender, m.maxSlotsPerSender,
		)
	}
	return nil
}

// CheckRecheck determines whether an indexed EVM transaction remains part of
// a complete state nonce chain during CheckTxType_Recheck. It validates only;
// BaseApp removes the transaction by TxKey when the surrounding recheck fails.
func (m *Mempool) CheckRecheck(
	txKey cmttypes.TxKey,
	sender gethcommon.Address,
	stateNonce uint64,
	txNonce uint64,
) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	senderEntry, found := m.bySender[sender]
	if !found {
		return fmt.Errorf("%w: sender %s has no live slots", ErrMempoolNonceGap, sender)
	}
	target, found := senderEntry.slots[txNonce]
	if !found || target.TxKey != txKey {
		return fmt.Errorf(
			"%w: sender %s, nonce %d", ErrMempoolTxMismatch, sender, txNonce,
		)
	}
	if txNonce < stateNonce {
		return fmt.Errorf(
			"%w: sender %s, state nonce %d, transaction nonce %d",
			ErrMempoolNonceGap, sender, stateNonce, txNonce,
		)
	}
	for nonce := stateNonce; ; nonce++ {
		if _, found := senderEntry.slots[nonce]; !found {
			return fmt.Errorf(
				"%w: sender %s, missing nonce %d before transaction nonce %d",
				ErrMempoolNonceGap, sender, nonce, txNonce,
			)
		}
		if nonce == txNonce {
			break
		}
	}
	return nil
}

// Insert atomically admits an EVM transaction after its complete ante handler
// succeeds. The original transaction bytes in sdk.Context are the authoritative
// bytes used for TxKey calculation and later proposal construction.
func (m *Mempool) Insert(goCtx context.Context, tx sdk.Tx) error {
	if !IsEthTx(tx) {
		return nil
	}

	sdkCtx := sdk.UnwrapSDKContext(goCtx)
	txBytes := sdkCtx.TxBytes()
	if len(txBytes) == 0 {
		return fmt.Errorf("EVM mempool insertion requires original transaction bytes")
	}

	msg, err := RequireStandardEVMTxMsg(tx)
	if err != nil {
		return err
	}
	if msg.From == "" {
		return fmt.Errorf("EVM mempool insertion requires an authenticated sender")
	}
	txData, err := UnpackTxData(msg.Data)
	if err != nil {
		return fmt.Errorf("unpack EVM transaction data for mempool insertion: %w", err)
	}
	ethTx, err := msg.AsTransactionSafe()
	if err != nil {
		return err
	}

	sender := msg.FromAddr()
	nonce := txData.GetNonce()
	txKey := cmttypes.Tx(txBytes).Key()

	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.checkNewTx(txKey, sender, nonce); err != nil {
		return err
	}
	if slot, found := m.byTxKey[txKey]; found {
		if slot.sender == sender && slot.nonce == nonce {
			return nil
		}
		return fmt.Errorf(
			"%w: transaction key already indexes sender %s, nonce %d",
			ErrMempoolTxMismatch, slot.sender, slot.nonce,
		)
	}

	senderEntry, found := m.bySender[sender]
	if !found {
		senderEntry = &mempoolSender{
			slots:    make(map[uint64]*mempoolTx),
			minNonce: nonce,
		}
		m.bySender[sender] = senderEntry
	} else if nonce < senderEntry.minNonce {
		senderEntry.minNonce = nonce
	}
	entry := &mempoolTx{
		tx: tx,
		MempoolTx: MempoolTx{
			TxBytes:   bytes.Clone(txBytes),
			TxKey:     txKey,
			EVMHash:   ethTx.Hash(),
			Sender:    sender,
			Nonce:     nonce,
			Priority:  sdkCtx.Priority(),
			GasWanted: sdkCtx.GasMeter().Limit(),
			ArrivalID: m.nextArrivalID,
		},
	}
	m.nextArrivalID++
	senderEntry.slots[nonce] = entry
	m.byTxKey[txKey] = mempoolSlotKey{sender: sender, nonce: nonce}
	m.txCount++
	return nil
}

// Select returns a read-only iterator over a point-in-time copy of the decoded
// EVM transactions. The custom PrepareProposal function uses Snapshot because
// it also requires original bytes and EVM-specific metadata.
func (m *Mempool) Select(context.Context, [][]byte) sdkmempool.Iterator {
	m.mu.RLock()
	entries := make([]*mempoolTx, 0, m.txCount)
	for _, senderEntry := range m.bySender {
		for _, entry := range senderEntry.slots {
			entries = append(entries, entry)
		}
	}
	m.mu.RUnlock()
	if len(entries) == 0 {
		return nil
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].ArrivalID < entries[j].ArrivalID
	})
	txs := make([]sdk.Tx, len(entries))
	for idx, entry := range entries {
		txs[idx] = entry.tx
	}
	return &mempoolIterator{txs: txs}
}

// CountTx returns the number of live EVM transactions in the mempool.
func (m *Mempool) CountTx() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.txCount
}

// Remove removes tx only when its signed Ethereum hash matches the transaction
// occupying the derived sender and nonce slot. BaseApp lifecycle integration
// should prefer RemoveByTxKey because Remove cannot recover the original outer
// SDK transaction bytes.
func (m *Mempool) Remove(tx sdk.Tx) error {
	if !IsEthTx(tx) {
		return nil
	}
	msg, err := RequireStandardEVMTxMsg(tx)
	if err != nil {
		return err
	}
	if msg.From == "" {
		return sdkmempool.ErrTxNotFound
	}
	txData, err := UnpackTxData(msg.Data)
	if err != nil {
		return err
	}
	ethTx, err := msg.AsTransactionSafe()
	if err != nil {
		return err
	}

	sender := msg.FromAddr()
	nonce := txData.GetNonce()
	m.mu.Lock()
	defer m.mu.Unlock()
	senderEntry, found := m.bySender[sender]
	if !found {
		return sdkmempool.ErrTxNotFound
	}
	entry, found := senderEntry.slots[nonce]
	if !found {
		return sdkmempool.ErrTxNotFound
	}
	if entry.EVMHash != ethTx.Hash() {
		return fmt.Errorf(
			"%w: sender %s, nonce %d", ErrMempoolTxMismatch, sender, nonce,
		)
	}
	m.removeEntry(entry.TxKey, mempoolSlotKey{sender: sender, nonce: nonce})
	return nil
}

// RemoveByTxKey removes exactly the transaction identified by CometBFT's
// outer-transaction key.
func (m *Mempool) RemoveByTxKey(txKey cmttypes.TxKey) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	slot, found := m.byTxKey[txKey]
	if !found {
		return sdkmempool.ErrTxNotFound
	}
	m.removeEntry(txKey, slot)
	return nil
}

func (m *Mempool) removeEntry(txKey cmttypes.TxKey, slot mempoolSlotKey) {
	delete(m.byTxKey, txKey)
	senderEntry := m.bySender[slot.sender]
	delete(senderEntry.slots, slot.nonce)
	if len(senderEntry.slots) == 0 {
		delete(m.bySender, slot.sender)
	} else if slot.nonce == senderEntry.minNonce {
		first := true
		for nonce := range senderEntry.slots {
			if first || nonce < senderEntry.minNonce {
				senderEntry.minNonce = nonce
				first = false
			}
		}
	}
	m.txCount--
}

// Snapshot returns a deep copy of the mempool grouped by sender. Sender groups
// have deterministic address order, and transactions within a group have
// ascending nonce order.
func (m *Mempool) Snapshot() []MempoolSender {
	m.mu.RLock()
	defer m.mu.RUnlock()

	snapshot := make([]MempoolSender, 0, len(m.bySender))
	for sender, senderEntry := range m.bySender {
		txs := make([]MempoolTx, 0, len(senderEntry.slots))
		for _, entry := range senderEntry.slots {
			tx := entry.MempoolTx
			tx.TxBytes = bytes.Clone(entry.TxBytes)
			txs = append(txs, tx)
		}
		sort.Slice(txs, func(i, j int) bool { return txs[i].Nonce < txs[j].Nonce })
		snapshot = append(snapshot, MempoolSender{
			Sender:   sender,
			MinNonce: senderEntry.minNonce,
			Txs:      txs,
		})
	}
	sort.Slice(snapshot, func(i, j int) bool {
		return bytes.Compare(snapshot[i].Sender[:], snapshot[j].Sender[:]) < 0
	})
	return snapshot
}

type mempoolIterator struct {
	txs   []sdk.Tx
	index int
}

func (it *mempoolIterator) Tx() sdk.Tx {
	return it.txs[it.index]
}

func (it *mempoolIterator) Next() sdkmempool.Iterator {
	next := it.index + 1
	if next >= len(it.txs) {
		return nil
	}
	return &mempoolIterator{txs: it.txs, index: next}
}
