package keeper

import (
	"fmt"
	"github.com/NibiruChain/nibiru/collections/keys"
	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/types"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"testing"
)

func TestKeeper_AllPairPositions(t *testing.T) {
	k, _, ctx := getKeeper(t)
	for i := 0; i < 100; i++ {
		addr := sample.AccAddress().String()
		pair := common.PairBTCStable
		d := sdk.MustNewDecFromStr(fmt.Sprintf("%d", i))
		k.Positions.Insert(ctx, keys.Join(pair, keys.String(addr)), types.Position{
			TraderAddress:                       addr,
			Pair:                                pair,
			Size_:                               d.MulInt64(10),
			Margin:                              d.MulInt64(1),
			OpenNotional:                        d.MulInt64(10),
			LastUpdateCumulativePremiumFraction: d.QuoInt64(100),
			BlockNumber:                         int64(i),
		})
	}

	pos := k.AllPairPositions(ctx, common.PairBTCStable)
	for _, p := range pos {
		t.Logf("%s %s", p.Key, &p.Value)
	}
}
