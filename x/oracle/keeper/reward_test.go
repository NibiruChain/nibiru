package keeper

import (
	"github.com/NibiruChain/nibiru/x/oracle/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestKeeper_PairRewards(t *testing.T) {
	i := CreateTestInput(t)
	k, ctx := i.OracleKeeper, i.Ctx

	t.Run("get, set, delete", func(t *testing.T) {
		// test delete not found
		err := k.DeletePairReward(ctx, "idk", 0)
		require.Error(t, err)
		// test set and get (found)
		reward := &types.PairReward{
			Pair:        "BTC-USD",
			Id:          10,
			VotePeriods: 100,
			Coins:       sdk.NewCoins(sdk.NewInt64Coin("atom", 1000)),
		}
		k.SetPairReward(ctx, reward)
		gotReward, err := k.GetPairReward(ctx, reward.Pair, reward.Id)
		require.NoError(t, err)
		require.Equal(t, reward, gotReward)
		// test delete and get (not found)
		err = k.DeletePairReward(ctx, reward.Pair, reward.Id)
		require.NoError(t, err)
		_, err = k.GetPairReward(ctx, "idk", 0)
		require.Error(t, err)
	})

	t.Run("next key", func(t *testing.T) {
		key := k.NextPairRewardKey(ctx)
		require.Equal(t, key, uint64(0))
		key = k.NextPairRewardKey(ctx)
		require.Equal(t, key, uint64(1))
	})

	t.Run("iterations", func(t *testing.T) {
		reward1 := &types.PairReward{
			Pair:        "BTC:USD",
			VotePeriods: 100,
			Coins:       sdk.NewCoins(sdk.NewCoin("atom", sdk.NewInt(1000))),
		}
		k.CreatePairReward(ctx, reward1)

		reward2 := &types.PairReward{
			Pair:        "ETH:USD",
			VotePeriods: 50,
			Coins:       sdk.NewCoins(sdk.NewInt64Coin("atom", 100)),
		}
		k.CreatePairReward(ctx, reward2)

		reward3 := &types.PairReward{
			Pair:        "BTC:USD",
			VotePeriods: 30,
			Coins:       sdk.NewCoins(sdk.NewInt64Coin("atom", 200), sdk.NewInt64Coin("nibi", 1000)),
		}
		k.CreatePairReward(ctx, reward3)

		// this tests prefix overlaps
		reward4 := &types.PairReward{
			Pair:        "BTC:USDT",
			VotePeriods: 3,
			Coins:       sdk.NewCoins(sdk.NewInt64Coin("atom", 200)),
		}
		k.CreatePairReward(ctx, reward4)

		// iterate by pair
		expected := []*types.PairReward{reward1, reward3}
		got := []*types.PairReward{}
		k.IteratePairRewards(ctx, reward1.Pair, func(rewards *types.PairReward) (stop bool) {
			got = append(got, rewards)
			return false
		})
		require.Equal(t, expected, got)

		// iterate all
		expected = []*types.PairReward{reward1, reward3, reward4, reward2}
		got = nil
		k.IterateAllPairRewards(ctx, func(rewards *types.PairReward) (stop bool) {
			got = append(got, rewards)
			return false
		})
		require.Equal(t, expected, got)

	})

}
