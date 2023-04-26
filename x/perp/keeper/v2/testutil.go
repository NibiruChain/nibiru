package keeper

import (
	"github.com/NibiruChain/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/perp/types"
	v2types "github.com/NibiruChain/nibiru/x/perp/types/v2"
)

func SetPosition(k Keeper, ctx sdk.Context, pos v2types.Position) {
	k.Positions.Insert(ctx, collections.Join(pos.Pair, sdk.MustAccAddressFromBech32(pos.TraderAddress)), pos)
}

func SetPairMetadata(k Keeper, ctx sdk.Context, pm types.PairMetadata) {
	k.PairsMetadata.Insert(ctx, pm.Pair, pm)
}
