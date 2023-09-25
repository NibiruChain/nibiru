package mock

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	asset "github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/perp/v2/types"
)

// default market with sensible values for tests
func TestAMMDefault() *types.AMM {
	return &types.AMM{
		Pair:            asset.NewPair(denoms.BTC, denoms.NUSD),
		Version:         1,
		BaseReserve:     sdk.NewDec(1e12),
		QuoteReserve:    sdk.NewDec(1e12),
		SqrtDepth:       sdk.NewDec(1e12),
		PriceMultiplier: sdk.OneDec(),
		TotalLong:       sdk.ZeroDec(),
		TotalShort:      sdk.ZeroDec(),
	}
}

// default market with sensible values for tests
func TestAMM(sqrtK sdk.Dec, priceMultiplier sdk.Dec) *types.AMM {
	return &types.AMM{
		Pair:            asset.NewPair(denoms.BTC, denoms.NUSD),
		BaseReserve:     sqrtK,
		QuoteReserve:    sqrtK,
		SqrtDepth:       sqrtK,
		PriceMultiplier: priceMultiplier,
		TotalLong:       sdk.ZeroDec(),
		TotalShort:      sdk.ZeroDec(),
	}
}
