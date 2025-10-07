package evmstate

// Copyright (c) 2023-2024 Nibi, Inc.

import (
	"github.com/ethereum/go-ethereum/common"
)

// DebugDirtiesCount is a test helper to inspect how many entries in the journal
// are still dirty (uncommitted). After calling [SDB.Commit], this function
// should return zero.
func (s *SDB) DebugDirtiesCount() int {
	dirtiesCount := 0
	for _, dirtyCount := range s.Journal.dirties {
		dirtiesCount += dirtyCount
	}
	return dirtiesCount
}

// DebugDirties is a test helper that returns the journal's dirty account changes map.
func (s *SDB) DebugDirties() map[common.Address]int {
	return s.Journal.dirties
}

// DebugStateObjects is a test helper that returns returns a copy of the
// [SDB.stateObjects] map.
func (s *SDB) DebugStateObjects() map[common.Address]*stateObject {
	copyOfMap := make(map[common.Address]*stateObject)
	for key, val := range s.stateObjects {
		copyOfMap[key] = val
	}
	return copyOfMap
}
