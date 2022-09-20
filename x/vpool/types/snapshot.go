package types

import (
	"github.com/NibiruChain/nibiru/x/common"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func NewReserveSnapshot(ctx sdk.Context, pair common.AssetPair, baseAssetReserve, quoteAssetReserve sdk.Dec) ReserveSnapshot {
	return ReserveSnapshot{
		Pair:              pair.String(),
		BaseAssetReserve:  baseAssetReserve,
		QuoteAssetReserve: quoteAssetReserve,
		TimestampMs:       ctx.BlockTime().UnixMilli(),
		BlockNumber:       ctx.BlockHeight(),
	}
}
