package app

import sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"

// ResolveSaiOracleAddr exposes [resolveOracleLaneRule] to external tests. It
// returns the resolved Sai oracle address, empty under the fallback.
func ResolveSaiOracleAddr(app *NibiruApp, ctx sdk.Context) (string, error) {
	rule, err := resolveOracleLaneRule(ctx, app.EvmKeeper, app.WasmKeeper)
	return rule.saiOracle, err
}

// ErrOracleAdapterNotConfigured exposes [errOracleAdapterNotConfigured].
var ErrOracleAdapterNotConfigured = errOracleAdapterNotConfigured
