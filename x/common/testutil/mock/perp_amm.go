package mock

import (
	asset "github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	v2types "github.com/NibiruChain/nibiru/x/perp/types/v2"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// default market with sensible values for tests
func TestAMMDefault() *v2types.AMM {
	return &v2types.AMM{
		Pair:            asset.NewPair(denoms.BTC, denoms.NUSD),
		BaseReserve:     sdk.NewDec(1e12),
		QuoteReserve:    sdk.NewDec(1e12),
		SqrtDepth:       sdk.NewDec(1e12),
		PriceMultiplier: sdk.OneDec(),
		TotalLong:       sdk.ZeroDec(),
		TotalShort:      sdk.ZeroDec(),
	}
}

// default market with sensible values for tests
func TestAMM(sqrtK sdk.Dec, priceMultiplier sdk.Dec) *v2types.AMM {
	return &v2types.AMM{
		Pair:            asset.NewPair(denoms.BTC, denoms.NUSD),
		BaseReserve:     sqrtK,
		QuoteReserve:    sqrtK,
		SqrtDepth:       sqrtK,
		PriceMultiplier: priceMultiplier,
		TotalLong:       sdk.ZeroDec(),
		TotalShort:      sdk.ZeroDec(),
	}
}
