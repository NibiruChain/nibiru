package simulation

import (
	"fmt"
	types3 "github.com/cosmos/cosmos-sdk/types"
	types2 "github.com/cosmos/ibc-go/v3/modules/apps/transfer/types"
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
		ibcDenom := generateValidIBCDenom(r)
		normalDenom := generateValidDenom(r)

		pair := common.MustNewAssetPair(fmt.Sprintf("%s:%s", ibcDenom, normalDenom))
		assetPairs[i] = pair
	}

	return assetPairs
}

func generateValidIBCDenom(r *rand.Rand) string {
	for {
		ibcDenom := simtypes.RandStringOfLength(r, 64)
		err := types2.ValidateIBCDenom(ibcDenom)
		if err != nil {
			continue
		}

		return ibcDenom
	}
}

func generateValidDenom(r *rand.Rand) string {
	for {
		normalDenom := simtypes.RandStringOfLength(r, 128)
		err := types3.ValidateDenom(normalDenom)
		if err != nil {
			continue
		}

		return normalDenom
	}
}

func genTwapLookbackWindow(r *rand.Rand) time.Duration {
	return time.Duration(r.Intn(maxLookbackWindowMinutes)) * time.Minute
}
