package wasmbinding

import (
	perpkeeper "github.com/NibiruChain/nibiru/x/perp/keeper"
)

type QueryPlugin struct {
	perpKeeper *perpkeeper.Keeper
}

// NewQueryPlugin returns a reference to a new QueryPlugin.
func NewQueryPlugin(pk *perpkeeper.Keeper) *QueryPlugin {
	return &QueryPlugin{
		perpKeeper: pk,
	}
}
