package v7

import (
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
)

// MigrateLocalhostClient initialises the 09-localhost client state and sets it in state.
func MigrateLocalhostClient(ctx sdk.Context, clientKeeper ClientKeeper) error {
	return clientKeeper.CreateLocalhostClient(ctx)
}
