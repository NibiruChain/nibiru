package store

import (
	dbm "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/libs/log"

	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/store/cache"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/store/rootmulti"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/store/types"
)

func NewCommitMultiStore(db dbm.DB) types.CommitMultiStore {
	return rootmulti.NewStore(db, log.NewNopLogger())
}

func NewCommitKVStoreCacheManager() types.MultiStorePersistentCache {
	return cache.NewCommitKVStoreCacheManager(cache.DefaultCommitKVStoreCacheSize)
}
