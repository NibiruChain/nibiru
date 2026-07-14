package v7

import (
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
)

// ConnectionKeeper expected IBC connection keeper
type ConnectionKeeper interface {
	CreateSentinelLocalhostConnection(ctx sdk.Context)
}
