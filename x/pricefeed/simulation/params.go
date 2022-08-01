package simulation

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/NibiruChain/nibiru/x/common"

	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"

	"github.com/NibiruChain/nibiru/x/pricefeed/types"
)

const (
	AssetPairsKey            = "Pairs"
	TwapLookbackWindowKey    = "TwapLookbackWindow"
	maxAssetPairs            = 100
	maxPostedPrices          = 100
	maxLookbackWindowMinutes = 7 * 24 * 60
)

func RandomizedParams(r *rand.Rand) []simtypes.ParamChange {
	return []simtypes.ParamChange{
		simulation.NewSimParamChange(types.ModuleName, AssetPairsKey, func(r *rand.Rand) string {
			return string(types.Amino.MustMarshalJSON(genAssetPairs(r)))
		}),
		simulation.NewSimParamChange(types.ModuleName, TwapLookbackWindowKey, func(r *rand.Rand) string {
			return fmt.Sprintf("\"%d\"", genTwapLookbackWindow(r))
		}),
	}
}

func genAssetPairs(r *rand.Rand) common.AssetPairs {
	assetPairs := make(common.AssetPairs, r.Intn(maxAssetPairs))
	for i := range assetPairs {
		a1, a2 := simtypes.RandStringOfLength(r, 5), simtypes.RandStringOfLength(r, 5)
		pair := common.MustNewAssetPair(fmt.Sprintf("%s:%s", a1, a2))
		assetPairs[i] = pair
	}
	return assetPairs
}

func genTwapLookbackWindow(r *rand.Rand) time.Duration {
	return time.Duration(r.Intn(maxLookbackWindowMinutes)) * time.Minute
}
