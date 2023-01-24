package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
)

type OracleKeeper interface {
	GetExchangeRate(ctx sdk.Context, pair common.AssetPair) (sdk.Dec, error)
	SetPrice(ctx sdk.Context, pair common.AssetPair, price sdk.Dec)
}
