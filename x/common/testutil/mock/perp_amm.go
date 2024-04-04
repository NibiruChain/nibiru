package mock

import (
	sdkmath "cosmossdk.io/math"
	asset "github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/perp/v2/types"
)

// default market with sensible values for tests
func TestAMMDefault() *types.AMM {
	return &types.AMM{
		Pair:            asset.NewPair(denoms.BTC, denoms.NUSD),
		Version:         1,
		BaseReserve:     sdkmath.LegacyNewDec(1e12),
		QuoteReserve:    sdkmath.LegacyNewDec(1e12),
		SqrtDepth:       sdkmath.LegacyNewDec(1e12),
		PriceMultiplier: sdkmath.LegacyOneDec(),
		TotalLong:       sdkmath.LegacyZeroDec(),
		TotalShort:      sdkmath.LegacyZeroDec(),
		SettlementPrice: sdkmath.LegacyZeroDec(),
	}
}

// default market with sensible values for tests
func TestAMM(sqrtK sdkmath.LegacyDec, priceMultiplier sdkmath.LegacyDec) *types.AMM {
	return &types.AMM{
		Pair:            asset.NewPair(denoms.BTC, denoms.NUSD),
		BaseReserve:     sqrtK,
		QuoteReserve:    sqrtK,
		SqrtDepth:       sqrtK,
		PriceMultiplier: priceMultiplier,
		TotalLong:       sdkmath.LegacyZeroDec(),
		TotalShort:      sdkmath.LegacyZeroDec(),
	}
}
