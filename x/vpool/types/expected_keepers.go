package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
)

type OracleKeeper interface {
	GetExchangeRate(ctx sdk.Context, pair asset.Pair) (sdk.Dec, error)
	SetPrice(ctx sdk.Context, pair asset.Pair, price sdk.Dec)
}
