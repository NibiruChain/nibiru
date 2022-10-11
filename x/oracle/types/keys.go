package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
)

const (
	// ModuleName is the name of the oracle module
	ModuleName = "oracle"

	// StoreKey is the string store representation
	StoreKey = ModuleName

	// RouterKey is the msg router key for the oracle module
	RouterKey = ModuleName

	// QuerierRoute is the query router key for the oracle module
	QuerierRoute = ModuleName
)

// Keys for oracle store
// Items are stored with the following key: values
//
// - 0x01<pair_Bytes>: sdk.Dec
//
// - 0x02<valAddress_Bytes>: accAddress
//
// - 0x03<valAddress_Bytes>: int64
//
// - 0x04<valAddress_Bytes>: AggregateExchangeRatePrevote
//
// - 0x05<valAddress_Bytes>: AggregateExchangeRateVote
//
// - 0x06<pair_Bytes>: sdk.Dec
var (
	// Keys for store prefixes
	MissCounterKey        = []byte{0x03} // prefix for each key to a miss counter
	PairsKey              = []byte{0x06} // prefix for each key to a pair
	PairRewardsKey        = []byte{0x07} // prefix for each key to a pair's rewards
	PairRewardsCounterKey = []byte{0x08} // prefix for the singleton of pair rewards ID.
)

// GetMissCounterKey - stored by *Validator* address
func GetMissCounterKey(v sdk.ValAddress) []byte {
	return append(MissCounterKey, address.MustLengthPrefix(v)...)
}

// GetPairKey - stored by *pair* bytes
func GetPairKey(d string) []byte {
	return append(PairsKey, append([]byte(d), 0x00)...)
}

// ExtractPairFromPairKey - split pair from the pair key
func ExtractPairFromPairKey(key []byte) (pair string) {
	pair = string(key[1 : len(key)-1])
	return
}

// GetPairRewardsKey returns the primary key for the PairRewards object.
func GetPairRewardsKey(pair string, id uint64) []byte {
	// TODO(mercilex): precompute key size
	key := append(PairRewardsKey, []byte(pair)...)
	key = append(key, 0x00)
	key = append(key, sdk.Uint64ToBigEndian(id)...)
	return key
}

// GetPairRewardsPrefixKey returns the primary key prefix
// to iterate over the PairRewards instances of the pair.
func GetPairRewardsPrefixKey(pair string) []byte {
	key := append(PairRewardsKey, []byte(pair)...)
	key = append(key, 0x00)
	return key
}
