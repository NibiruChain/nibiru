package action

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/testutil"
)

func SetPairPrice(pair asset.Pair, price sdk.Dec) testutil.Action {
	return &setPairPrice{
		Pair:  pair,
		Price: price,
	}
}

type setPairPrice struct {
	Pair  asset.Pair
	Price sdk.Dec
}

func (s setPairPrice) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error) {
	app.OracleKeeper.SetPrice(ctx, s.Pair, s.Price)

	return ctx, nil
}
