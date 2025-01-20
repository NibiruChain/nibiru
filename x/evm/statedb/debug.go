package statedb

// Copyright (c) 2023-2024 Nibi, Inc.

import (
	"github.com/ethereum/go-ethereum/common"
)

// DebugDirtiesCount is a test helper to inspect how many entries in the journal
// are still dirty (uncommitted). After calling [StateDB.Commit], this function
// should return zero.
func (s *StateDB) DebugDirtiesCount() int {
	dirtiesCount := 0
	for _, dirtyCount := range s.Journal.dirties {
		dirtiesCount += dirtyCount
	}
	return dirtiesCount
}

// DebugDirties is a test helper that returns the journal's dirty account changes map.
func (s *StateDB) DebugDirties() map[common.Address]int {
	return s.Journal.dirties
}

// DebugEntries is a test helper that returns the sequence of [JournalChange]
// objects added during execution.
func (s *StateDB) DebugEntries() []JournalChange {
	return s.Journal.entries
}

// DebugStateObjects is a test helper that returns returns a copy of the
// [StateDB.stateObjects] map.
func (s *StateDB) DebugStateObjects() map[common.Address]*stateObject {
	copyOfMap := make(map[common.Address]*stateObject)
	for key, val := range s.stateObjects {
		copyOfMap[key] = val
	}
	return copyOfMap
}
