package keeper

// import (
// 	"testing"

// 	"github.com/NibiruChain/nibiru/x/common/testutil"

// 	sdk "github.com/cosmos/cosmos-sdk/types"
// 	"github.com/golang/mock/gomock"
// 	"github.com/stretchr/testify/require"

// 	"github.com/NibiruChain/nibiru/x/common/asset"
// 	"github.com/NibiruChain/nibiru/x/perp/types"
// 	v2types "github.com/NibiruChain/nibiru/x/perp/types/v2"
// )

// func TestSettlePosition(t *testing.T) {
// 	t.Run("success - settlement price zero", func(t *testing.T) {
// 		k, dep, ctx := getKeeper(t)
// 		traderAddr := testutil.AccAddress()
// 		pair := asset.MustNewPair("LUNA:UST")

// 		dep.mockPerpAmmKeeper.
// 			EXPECT().
// 			GetSettlementPrice(gomock.Eq(ctx), gomock.Eq(pair)).
// 			Return(sdk.ZeroDec(), error(nil))

// 		dep.mockBankKeeper.EXPECT().
// 			SendCoinsFromModuleToAccount(
// 				ctx, types.VaultModuleAccount, traderAddr,
// 				sdk.NewCoins(sdk.NewCoin("UST", sdk.NewInt(100))),
// 			).
// 			Return(error(nil))

// 		pos := v2types.Position{
// 			TraderAddress: traderAddr.String(),
// 			Pair:          pair,
// 			Size_:         sdk.NewDec(10),
// 			Margin:        sdk.NewDec(100),
// 			OpenNotional:  sdk.NewDec(1000),
// 		}
// 		SetPosition(k, ctx, pos)

// 		coins, err := k.SettlePosition(ctx, pos)
// 		require.NoError(t, err)

// 		require.Equal(t, sdk.NewCoins(
// 			sdk.NewCoin( /*denom=*/ pair.QuoteDenom(), pos.Margin.TruncateInt()),
// 		), coins) // TODO(mercilex): here we should have different denom, depends on Transfer impl
// 	})

// 	t.Run("success - settlement price not zero", func(t *testing.T) {
// 		k, dep, ctx := getKeeper(t)
// 		traderAddr := testutil.AccAddress()
// 		pair := asset.MustNewPair("LUNA:UST") // memeing

// 		dep.mockPerpAmmKeeper.
// 			EXPECT().
// 			GetSettlementPrice(ctx, pair).
// 			Return(sdk.NewDec(1000), error(nil))

// 		dep.mockBankKeeper.EXPECT().
// 			SendCoinsFromModuleToAccount(
// 				ctx, types.VaultModuleAccount, traderAddr, sdk.NewCoins(sdk.NewCoin("UST", sdk.NewInt(99_100)))).
// 			Return(error(nil))

// 		// this means that the user
// 		// has bought 100 contracts
// 		// for an open notional of 1_000 coins
// 		// which means entry price is 10
// 		// now price is 1_000
// 		// which means current pos value is 100_000
// 		// now we need to return the user the profits
// 		// which are 99000 coins
// 		// we also need to return margin which is 100coin
// 		// so total is 99_100 coin
// 		pos := v2types.Position{
// 			TraderAddress: traderAddr.String(),
// 			Pair:          pair,
// 			Size_:         sdk.NewDec(100),
// 			Margin:        sdk.NewDec(100),
// 			OpenNotional:  sdk.NewDec(1000),
// 		}
// 		SetPosition(k, ctx, pos)

// 		coins, err := k.SettlePosition(ctx, pos)
// 		require.NoError(t, err)
// 		require.Equal(t, coins, sdk.NewCoins(
// 			sdk.NewInt64Coin(pair.QuoteDenom(), 99100))) // todo(mercilex): modify denom once transfer is impl
// 	})

// 	t.Run("position size is zero", func(t *testing.T) {
// 		k, _, ctx := getKeeper(t)
// 		traderAddr := testutil.AccAddress()
// 		pair := asset.MustNewPair("LUNA:UST")

// 		pos := v2types.Position{
// 			TraderAddress: traderAddr.String(),
// 			Pair:          pair,
// 			Size_:         sdk.ZeroDec(),
// 		}
// 		SetPosition(k, ctx, pos)

// 		coins, err := k.SettlePosition(ctx, pos)
// 		require.NoError(t, err)
// 		require.Len(t, coins, 0)
// 	})
// }
