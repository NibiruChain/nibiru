package keeper

import (
	"fmt"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/lockup/types"
)

var (
	lockNamespace            = []byte{0x0}
	lockIDNamespace          = append(lockNamespace, 0x0) // maps next lock ID
	lockObjectNamespace      = append(lockNamespace, 0x1) // maps lock ID => lock bytes
	lockAddrTimeIndex        = append(lockNamespace, 0x2) // maps address and unlock time => lock ID
	lockAddrIndex            = append(lockNamespace, 0x3) // maps address => lock ID
	lockTimeIndex            = append(lockNamespace, 0x4) // maps unlock time => lock ID
	lockDenomIndex           = append(lockNamespace, 0x5) // maps denom to lock ID
	lockDenomUnlockTimeIndex = append(lockNamespace, 0x6) // maps denom + unlock time to lock ID

	lockIDKey = []byte{0x0} // lock ID key
)

const (
	// LockStartID is the ID of the first lock.
	LockStartID uint64 = 0
)

func (k Keeper) LocksState(ctx sdk.Context) LockState {
	return newLockState(ctx, k.storeKey, k.cdc)
}

func newLockState(ctx sdk.Context, key sdk.StoreKey, cdc codec.BinaryCodec) LockState {
	store := ctx.KVStore(key) // get keeper KV
	return LockState{
		ctx:            ctx,
		cdc:            cdc,
		id:             prefix.NewStore(store, lockIDNamespace),
		locks:          prefix.NewStore(store, lockObjectNamespace),
		addrTimeIndex:  prefix.NewStore(store, lockAddrTimeIndex),
		addrIndex:      prefix.NewStore(store, lockAddrIndex),
		timeIndex:      prefix.NewStore(store, lockTimeIndex),
		denomIndex:     prefix.NewStore(store, lockDenomIndex),
		denomTimeIndex: prefix.NewStore(store, lockDenomUnlockTimeIndex),
	}
}

type LockState struct {
	ctx            sdk.Context
	cdc            codec.BinaryCodec
	id             sdk.KVStore
	locks          sdk.KVStore
	addrTimeIndex  sdk.KVStore
	addrIndex      sdk.KVStore
	timeIndex      sdk.KVStore
	denomIndex     sdk.KVStore
	denomTimeIndex sdk.KVStore
}

// Create creates a new types.Lock, and sets the lock ID.
func (s LockState) Create(l *types.Lock) {
	if l.LockId != 0 {
		panic("lock ID should not be set")
	}

	id := s.nextPrimaryKey()
	pk := sdk.Uint64ToBigEndian(id) // TODO(mercilex): processed twice, maybe next primary key can return the bytes version
	l.LockId = id                   // sets lock ID so that is mapped in state

	s.locks.Set(pk, s.cdc.MustMarshal(l)) // save lock object
	s.index(pk, l)                        // generate indexes
}

func (s LockState) index(pk []byte, l *types.Lock) {
	addrTimeIndex := s.keyAddrTime(l.Owner, l.EndTime, pk)
	addrIndex := s.keyAddr(l.Owner, pk)
	timeIndex := s.keyTime(l.EndTime, pk)

	s.addrTimeIndex.Set(addrTimeIndex, []byte{}) // maps addr + unlock time to lock ID
	s.addrIndex.Set(addrIndex, []byte{})         // maps addr to lock ID
	s.timeIndex.Set(timeIndex, []byte{})         // maps unlock time to lock ID

	for _, coin := range l.Coins {
		s.denomIndex.Set(s.keyDenom(coin.Denom, pk), []byte{})                    // maps denom to lock ID
		s.denomTimeIndex.Set(s.keyDenomTime(coin.Denom, l.EndTime, pk), []byte{}) // maps denom unlock time to lock ID
	}
}

func (s LockState) unindex(pk []byte, l *types.Lock) {
	s.addrTimeIndex.Delete(s.keyAddrTime(l.Owner, l.EndTime, pk)) // clear address and unlock time index
	s.addrIndex.Delete(s.keyAddr(l.Owner, pk))                    // clear address index
	s.timeIndex.Delete(s.keyTime(l.EndTime, pk))                  // clear unlock time index

	// clears associations between lock ID
	// and coins locked and their unlock time.
	for _, coin := range l.Coins {
		s.denomIndex.Delete(s.keyDenom(coin.Denom, pk))
		s.denomTimeIndex.Delete(s.keyDenomTime(coin.Denom, l.EndTime, pk))
	}
}

func (s LockState) Update(update *types.Lock) error {
	pk := sdk.Uint64ToBigEndian(update.LockId)
	// get old lock
	oldLockBytes := s.locks.Get(pk)
	if oldLockBytes == nil {
		return types.ErrLockupNotFound.Wrapf("%d", update.LockId)
	}

	oldLock := new(types.Lock)
	s.cdc.MustUnmarshal(oldLockBytes, oldLock)
	// update indexes
	s.unindex(pk, oldLock)
	s.index(pk, update)
	// update lock
	s.locks.Set(pk, s.cdc.MustMarshal(update))
	return nil
}

func (s LockState) Delete(l *types.Lock) error {
	lockPrimaryKey := sdk.Uint64ToBigEndian(l.LockId)
	if !s.locks.Has(lockPrimaryKey) {
		return types.ErrLockupNotFound.Wrapf("%d", l.LockId)
	}

	s.locks.Delete(lockPrimaryKey) // clear object
	s.unindex(lockPrimaryKey, l)   // unindex
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

// UnlockedIDsByAddress returns the list of types.Lock IDs which can be
// unlocked given the lock owner sdk.AccAddress.
func (s LockState) UnlockedIDsByAddress(addr sdk.AccAddress) []uint64 {
	iter := prefix.NewStore(s.addrTimeIndex, s.keyAddr(addr.String(), nil)). // this creates a store which prefixes over addr's lock namespace
											Iterator(nil, s.keyTime(s.ctx.BlockTime(), nil)) // this iterates over locks with end time <= current time
	defer iter.Close()

	var ids []uint64

	for ; iter.Valid(); iter.Next() {
		primaryKey := iter.Key()[8:]
		ids = append(ids, sdk.BigEndianToUint64(primaryKey))
	}

	return ids
}

func (s LockState) IterateLockedCoins(addr sdk.AccAddress) sdk.Coins {
	iter := prefix.NewStore(s.addrTimeIndex, s.keyAddr(addr.String(), nil)). // this creates a store which prefixes over addr's lock namespace
											Iterator(s.keyTime(s.ctx.BlockTime(), nil), nil) // this iterates over locks with end time >= current time
	defer iter.Close()

	coins := sdk.NewCoins()
	for ; iter.Valid(); iter.Next() {
		lock := new(types.Lock)

		primaryKey := iter.Key()[8:] // we're stripping the size of the time in form of bytes of the key to get only the primary key
		if !s.locks.Has(primaryKey) {
			panic(fmt.Errorf("state corruption: %v", primaryKey))
		}
		s.cdc.MustUnmarshal(s.locks.Get(primaryKey), lock)
		coins = coins.Add(lock.Coins...)
	}

	return coins
}

func (s LockState) IterateUnlockedCoins(addr sdk.AccAddress) sdk.Coins {
	iter := prefix.NewStore(s.addrTimeIndex, s.keyAddr(addr.String(), nil)). // this creates a store which prefixes over addr's lock namespace
											Iterator(nil, s.keyTime(s.ctx.BlockTime(), nil)) // this iterates over locks with end time <= current time
	defer iter.Close()

	coins := sdk.NewCoins()
	for ; iter.Valid(); iter.Next() {
		lock := new(types.Lock)

		primaryKey := iter.Key()[8:] // strip index key and just keep primary key
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

func (s LockState) IterateCoinsByDenomUnlockingAfter(denom string, unlockingAfter time.Time, f func(id uint64) (stop bool)) {
	// NOTE(mercilex): here we're adding 1 sec because our indexing precision is in seconds
	// maybe the most proper way would be to increase by one byte the value of the keyTime key.
	iter := prefix.NewStore(s.denomTimeIndex, s.keyDenom(denom, nil)). // iterate only over a certain denom
										Iterator(s.keyTime(unlockingAfter.Add(1*time.Second), nil), nil) // after the provided time

	for ; iter.Valid(); iter.Next() {
		primaryKey := sdk.BigEndianToUint64(iter.Key()[8:])
		if f(primaryKey) {
			break
		}
	}
}

func (s LockState) IterateCoinsByDenomUnlockingBefore(denom string, unlockingBefore time.Time, f func(id uint64) (stop bool)) {
	iter := prefix.NewStore(s.denomTimeIndex, s.keyDenom(denom, nil)). // iterate only over a certain denom
										Iterator(nil, s.keyTime(unlockingBefore.Add(-1*time.Second), nil)) // before the provided time

	for ; iter.Valid(); iter.Next() {
		primaryKey := sdk.BigEndianToUint64(iter.Key()[8:])
		if f(primaryKey) {
			break
		}
	}
}

func (s LockState) IterateLocksByAddress(addr sdk.AccAddress, do func(id uint64) (stop bool)) {
	key := s.keyAddr(addr.String(), nil)
	iter := prefix.NewStore(s.addrIndex, key).Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		if !do(sdk.BigEndianToUint64(iter.Key())) {
			break
		}
	}
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
	timeUnix := endTime.Unix()

	timeBytes[0] = byte(timeUnix >> 56)
	timeBytes[1] = byte(timeUnix >> 48)
	timeBytes[2] = byte(timeUnix >> 40)
	timeBytes[3] = byte(timeUnix >> 32)
	timeBytes[4] = byte(timeUnix >> 24)
	timeBytes[5] = byte(timeUnix >> 16)
	timeBytes[6] = byte(timeUnix >> 8)
	timeBytes[7] = byte(timeUnix)
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

	timeUnix := endTime.Unix()

	key[0] = byte(timeUnix >> 56)
	key[1] = byte(timeUnix >> 48)
	key[2] = byte(timeUnix >> 40)
	key[3] = byte(timeUnix >> 32)
	key[4] = byte(timeUnix >> 24)
	key[5] = byte(timeUnix >> 16)
	key[6] = byte(timeUnix >> 8)
	key[7] = byte(timeUnix)
	key[0] ^= 0x80

	return append(key, pk...)
}

func (s LockState) keyDenom(denom string, pk []byte) []byte {
	// TODO(mercilex): maybe more efficient
	denomKey := append([]byte(denom), 0xFF)
	return append(denomKey, pk...)
}

func (s LockState) keyDenomTime(denom string, t time.Time, pk []byte) []byte {
	// TODO(mercilex): maybe more efficient
	return append(append([]byte(denom), 0xFF), s.keyTime(t, pk)...)
}

func (s LockState) IterateLocks(do func(lock *types.Lock) (stop bool)) {
	iter := s.locks.Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		lock := new(types.Lock)
		s.cdc.MustUnmarshal(iter.Value(), lock)
		if do(lock) {
			break
		}
	}
}
