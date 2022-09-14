package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
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

var (
	// Keys for store prefixes
	PairRewardsKey = []byte{0x07} // prefix for each key to a pair's rewards
)

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
