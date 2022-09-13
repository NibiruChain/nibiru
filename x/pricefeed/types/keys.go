package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// ModuleName defines the module name
	ModuleName = "pricefeed"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey is the message route for slashing
	RouterKey = ModuleName

	// QuerierRoute defines the module's query routing key
	QuerierRoute = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_pricefeed"
)

var (
	// Snapshot prefix for the median oracle price at a specific point in time
	PriceSnapshotPrefix = []byte{0x03}
)

func PriceSnapshotKey(pairId string, blockHeight int64) []byte {
	return append(
		PriceSnapshotPrefix,
		append(
			[]byte(pairId),
			sdk.Uint64ToBigEndian(uint64(blockHeight))...,
		)...,
	)
}
