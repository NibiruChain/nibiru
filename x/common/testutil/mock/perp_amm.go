package mock

import (
	asset "github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	v2types "github.com/NibiruChain/nibiru/x/perp/types/v2"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// default market with sensible values for tests
func TestAMM() *v2types.AMM {
	return &v2types.AMM{
		Pair:            asset.NewPair(denoms.BTC, denoms.NUSD),
		BaseReserve:     sdk.NewDec(10e12),
		QuoteReserve:    sdk.NewDec(10e12),
		SqrtDepth:       sdk.NewDec(10e12),
		PriceMultiplier: sdk.OneDec(),
		Bias:            sdk.ZeroDec(),
	}
}
