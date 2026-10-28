package keeper

// SetQueryGasLimit overrides the params query gas limit, so tests can force
// an out-of-gas query on the real contract. Production code always uses
// QueryGasLimit.
func (k *Keeper) SetQueryGasLimit(limit uint64) { k.queryGasLimit = limit }
