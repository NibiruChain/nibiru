package keeper

import (
	sdkmath "cosmossdk.io/math"

	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
)

// TokensToConsensusPower - convert input tokens to potential consensus-engine power
func (k Keeper) TokensToConsensusPower(ctx sdk.Context, tokens sdkmath.Int) int64 {
	return sdk.TokensToConsensusPower(tokens, k.PowerReduction(ctx))
}

// TokensFromConsensusPower - convert input power to tokens
func (k Keeper) TokensFromConsensusPower(ctx sdk.Context, power int64) sdkmath.Int {
	return sdk.TokensFromConsensusPower(power, k.PowerReduction(ctx))
}
