package keeper

import (
	"fmt"

	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
)

// QuerySmartBounded runs a smart query against contractAddr under a fresh gas
// meter limited to gasLimit, discards any state the query touches, and turns
// every panic into an error. It returns the raw response and the gas the
// query used, including on failure.
//
// It is the first piece of a pattern for state that a Wasm contract owns but
// the chain reads every block: the contract is the only writer, and a module
// store holds a deterministic copy that the app refreshes from the contract
// under a bounded gas meter. Readers use the module store, so a slow or broken
// contract can never stall a block or halt the chain; the refresh just fails
// and the copy keeps its last good value. x-consensus (x/xconsensus) is the
// first user.
//
// Why each step matters:
//   - Bounded gas. QuerySmart on an infinite gas meter, which is what
//     BeginBlock and EndBlock contexts carry, hands the VM unlimited Wasm gas
//     (runtimeGasForContract), and node-local query_gas_limit only applies to
//     gRPC. A contract loop would hang every node. The fresh meter is
//     deterministic: Wasm gas is metered the same way on every node, so every
//     node agrees on whether the query ran out of gas.
//   - Cache context. Smart queries are read-only by contract, but the gas and
//     store layers still record reads and the query stack depth. Running in
//     ctx.CacheContext() and never calling its write function keeps the
//     caller's store and events exactly as they were.
//   - Recover. Running out of gas panics with sdk.ErrorOutOfGas
//     (consumeRuntimeGas), and a VM or store bug could panic too. Callers in
//     BeginBlock or EndBlock must not panic, so the panic becomes an error.
//
// The caller's gas meter is not charged. That is the intent for block hooks,
// where the cost is bounded by gasLimit instead. Do not use this to serve
// user-paid queries.
func (k Keeper) QuerySmartBounded(
	ctx sdk.Context, contractAddr sdk.AccAddress, req []byte, gasLimit uint64,
) (resp []byte, gasUsed uint64, err error) {
	cacheCtx, _ := ctx.CacheContext()
	gasMeter := sdk.NewGasMeter(gasLimit)
	cacheCtx = cacheCtx.WithGasMeter(gasMeter)

	defer func() {
		gasUsed = gasMeter.GasConsumedToLimit()
		if r := recover(); r != nil {
			resp = nil
			if oog, ok := r.(sdk.ErrorOutOfGas); ok {
				err = fmt.Errorf(
					"wasm smart query ran out of gas (limit %d): %s",
					gasLimit, oog.Descriptor)
				return
			}
			err = fmt.Errorf("wasm smart query panicked: %v", r)
		}
	}()

	resp, err = k.QuerySmart(cacheCtx, contractAddr, req)
	return resp, gasUsed, err
}
