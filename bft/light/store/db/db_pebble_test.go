//go:build pebbledb

package db

import (
	"testing"

	"github.com/stretchr/testify/require"

	dbm "github.com/cometbft/cometbft-db"
)

// Pebble pools iterators on Close and forbids a second Close, so this fails
// under -race if Prune closes its iterator twice.
func TestPrune_ConcurrentReaders_PebbleDB(t *testing.T) {
	db, err := dbm.NewDB("prune_readers", dbm.PebbleDBBackend, t.TempDir())
	require.NoError(t, err)
	defer db.Close()
	pruneWithReaders(t, db)
}
