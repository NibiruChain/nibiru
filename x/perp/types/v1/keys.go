package v1

import (
	"strings"
)

var (
	// StoreKey defines the primary module store key.
	StoreKey = ModuleName

	// RouterKey is the message route for slashing.
	RouterKey = ModuleName

	// QuerierRoute defines the module's query routing key.
	QuerierRoute = ModuleName

	MemStoreKey = strings.Join([]string{"mem", ModuleName}, "_")
)
