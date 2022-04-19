package keeper

import (
	"fmt"
	"github.com/NibiruChain/nibiru/x/lockup/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"time"
)

var (
	lockNamespace       = []byte{0x0}
	lockIDNamespace     = append(lockNamespace, 0x0) // maps next lock ID
	lockObjectNamespace = append(lockNamespace, 0x1) // maps lock ID => lock bytes
	lockAddrTimeIndex   = append(lockNamespace, 0x2) // maps address and unlock time => lock ID
	lockAddrIndex       = append(lockNamespace, 0x3) // maps address => lock ID
	lockTimeIndex       = append(lockNamespace, 0x4) // maps unlock time => lock ID

	lockIDKey = []byte{0x0} // lock ID key
)

const (
	// LockStartID is the ID of the first lock.
	LockStartID uint64 = 0
)

func (k LockupKeeper) LocksState(ctx sdk.Context) LockState {
	return newLockState(ctx, k.storeKey, k.cdc)
}

func newLockState(ctx sdk.Context, key sdk.StoreKey, cdc codec.BinaryCodec) LockState {
	store := ctx.KVStore(key) // get keeper KV
	return LockState{
		ctx:           ctx,
		cdc:           cdc,
		id:            prefix.NewStore(store, lockIDNamespace),
		locks:         prefix.NewStore(store, lockObjectNamespace),
		addrTimeIndex: prefix.NewStore(store, lockAddrTimeIndex),
		addrIndex:     prefix.NewStore(store, lockAddrIndex),
		timeIndex:     prefix.NewStore(store, lockTimeIndex),
	}
}

type LockState struct {
	ctx           sdk.Context
	cdc           codec.BinaryCodec
	id            sdk.KVStore
	locks         sdk.KVStore
	addrTimeIndex sdk.KVStore
	addrIndex     sdk.KVStore
	timeIndex     sdk.KVStore
}

// Create creates a new types.Lock, and sets the lock ID.
func (s LockState) Create(l *types.Lock) {
	if l.LockId != 0 {
		panic("lock ID should not be set")
	}

	id := s.nextPrimaryKey()
	pk := sdk.Uint64ToBigEndian(id) // TODO(mercilex): processed twice, maybe next primary key can return the bytes version

	addrTimeIndex := s.keyAddrTime(l.Owner, l.EndTime, pk)
	addrIndex := s.keyAddr(l.Owner, pk)
	timeIndex := s.keyTime(l.EndTime, pk)

	s.locks.Set(pk, s.cdc.MustMarshal(l))        // save lock object
	s.addrTimeIndex.Set(addrTimeIndex, []byte{}) // maps addr + unlock time to lock ID
	s.addrIndex.Set(addrIndex, []byte{})         // maps addr to lock ID
	s.timeIndex.Set(timeIndex, []byte{})         // maps unlock time to lock ID

	l.LockId = id
}

func (s LockState) Delete(l *types.Lock) error {
	lockPrimaryKey := sdk.Uint64ToBigEndian(l.LockId)
	if !s.locks.Has(lockPrimaryKey) {
		return types.ErrLockupNotFound.Wrapf("%d", l.LockId)
	}

	s.locks.Delete(lockPrimaryKey)                                            // clear object
	s.addrTimeIndex.Delete(s.keyAddrTime(l.Owner, l.EndTime, lockPrimaryKey)) // clear address and unlock time index
	s.addrIndex.Delete(s.keyAddr(l.Owner, lockPrimaryKey))                    // clear address index
	s.timeIndex.Delete(s.keyTime(l.EndTime, lockPrimaryKey))                  // clear unlock time index
	return nil
}

func (s LockState) Get(id uint64) (*types.Lock, error) {
	switch lockBytes := s.locks.Get(sdk.Uint64ToBigEndian(id)); lockBytes {
	case nil:
		return nil, types.ErrLockupNotFound.Wrapf("%d", id)
	default:
		lock := new(types.Lock)
		s.cdc.MustUnmarshal(lockBytes, lock)
		return lock, nil
	}
}

func (s LockState) IterateLockedCoins(addr sdk.AccAddress) sdk.Coins {
	key := s.keyAddrTime(addr.String(), s.ctx.BlockTime(), nil)

	iter := s.addrTimeIndex.Iterator(key, nil)
	defer iter.Close()

	coins := sdk.NewCoins()
	for ; iter.Valid(); iter.Next() {

		lock := new(types.Lock)

		primaryKey := iter.Key()[len(key):] // strip index key and just keep primary key
		if !s.locks.Has(primaryKey) {
			panic(fmt.Errorf("state corruption: %v", primaryKey))
		}
		s.cdc.MustUnmarshal(s.locks.Get(primaryKey), lock)
		coins = coins.Add(lock.Coins...)
	}

	return coins
}

func (s LockState) IterateUnlockedCoins(addr sdk.AccAddress) sdk.Coins {
	key := s.keyAddrTime(addr.String(), s.ctx.BlockTime(), nil)

	iter := s.addrTimeIndex.Iterator(nil, key)
	defer iter.Close()

	coins := sdk.NewCoins()
	for ; iter.Valid(); iter.Next() {

		lock := new(types.Lock)

		primaryKey := iter.Key()[len(key):] // strip index key and just keep primary key
		if !s.locks.Has(primaryKey) {
			panic(fmt.Errorf("state corruption: %v", primaryKey))
		}
		s.cdc.MustUnmarshal(s.locks.Get(primaryKey), lock)
		coins = coins.Add(lock.Coins...)
	}

	return coins

}

// IterateTotalLockedCoins returns the total amount of locked coins
func (s LockState) IterateTotalLockedCoins() sdk.Coins {
	key := s.keyTime(s.ctx.BlockTime(), nil)

	iter := s.timeIndex.Iterator(key, nil)
	defer iter.Close()

	coins := sdk.NewCoins()
	for ; iter.Valid(); iter.Next() {
		lock := new(types.Lock)

		primaryKey := iter.Key()[len(key):] // strip index key and just keep primary key
		if !s.locks.Has(primaryKey) {
			panic(fmt.Errorf("state corruption: %v", primaryKey))
		}
		s.cdc.MustUnmarshal(s.locks.Get(primaryKey), lock)
		coins = coins.Add(lock.Coins...)
	}

	return coins
}

func (s LockState) nextPrimaryKey() uint64 {
	idBytes := s.id.Get(lockIDKey)
	var id uint64

	switch idBytes {
	case nil:
		id = LockStartID
	default:
		id = sdk.BigEndianToUint64(idBytes)
	}

	s.id.Set(lockIDKey, sdk.Uint64ToBigEndian(id+1))

	return id
}

// keyAddrTime generates a key which associates address + unlock time to a types.Lock
func (s LockState) keyAddrTime(addr string, endTime time.Time, pk []byte) []byte {
	// TODO(mercilex): key size can be pre-computed
	// TODO(mercilex): are bech32 string addr const size? this means we can avoid 0xff suffixing

	key := append([]byte(addr), 0xFF) // addr + null termination, assumes no 0xff in string

	// proper sorted big endian int64 x.x
	timeBytes := make([]byte, 8)
	timeUnixNano := endTime.UnixNano()

	timeBytes[0] = byte(timeUnixNano >> 56)
	timeBytes[1] = byte(timeUnixNano >> 48)
	timeBytes[2] = byte(timeUnixNano >> 40)
	timeBytes[3] = byte(timeUnixNano >> 32)
	timeBytes[4] = byte(timeUnixNano >> 24)
	timeBytes[5] = byte(timeUnixNano >> 16)
	timeBytes[6] = byte(timeUnixNano >> 8)
	timeBytes[7] = byte(timeUnixNano)
	timeBytes[0] ^= 0x80

	key = append(key, timeBytes...)
	// index key is composed

	// now add the primary key
	key = append(key, pk...)
	return key
}

// keyAddr creates a key which associates all types.Lock to an address
func (s LockState) keyAddr(addr string, pk []byte) []byte {
	// TODO(mercilex): key size can be pre-computed
	// TODO(mercilex): are bech32 string addr const size? this means we can avoid 0xff suffixing

	key := append([]byte(addr), 0xFF) // addr + null termination, assumes no 0xff in string
	return append(key, pk...)         // append primary key for iteration
}

func (s LockState) keyTime(endTime time.Time, pk []byte) []byte {
	key := make([]byte, 8, 8+len(pk)) // size of primary key + size of time in bytes, init 8 elements

	timeUnixNano := endTime.UnixNano()

	key[0] = byte(timeUnixNano >> 56)
	key[1] = byte(timeUnixNano >> 48)
	key[2] = byte(timeUnixNano >> 40)
	key[3] = byte(timeUnixNano >> 32)
	key[4] = byte(timeUnixNano >> 24)
	key[5] = byte(timeUnixNano >> 16)
	key[6] = byte(timeUnixNano >> 8)
	key[7] = byte(timeUnixNano)
	key[0] ^= 0x80

	return append(key, pk...)
}
