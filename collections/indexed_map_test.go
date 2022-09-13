package collections

import (
	"testing"

	"github.com/NibiruChain/nibiru/collections/keys"
)

type testIndexes struct {
	// Address indexes by address
	Address *MultiIndex[keys.StringKey, keys.Uint8Key, lock]
	// AddressStartTime indexes by address and start time
	AddressStartTime *MultiIndex[keys.Pair[keys.StringKey, keys.Uint8Key], keys.Uint8Key, lock]
}

func (t testIndexes) IndexList() []Index[keys.Uint8Key, lock] {
	return []Index[keys.Uint8Key, lock]{t.Address}
}

func TestIndexedMap(t *testing.T) {
	/*
		sk, _, cdc := deps()
		_ = NewIndexedMap[keys.Uint8Key, lock, *lock, testIndexes](cdc, sk, 0, testIndexes{
			Address: NewMultiIndex[keys.StringKey, keys.Uint8Key, lock](sk, func(v lock) keys.StringKey {
				return v.Address
			}),
			AddressStartTime: NewMultiIndex[keys.Pair[keys.StringKey, keys.Uint8Key], keys.Uint8Key, lock](sk, func(v lock) keys.Pair[keys.StringKey, keys.Uint8Key] {
				return keys.Join(v.Address, v.Start)
			}),
		})
	*/
}
