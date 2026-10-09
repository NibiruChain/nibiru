package upgrades

import (
	"errors"
	"testing"

	dbm "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/libs/log"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/store"
	storetypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/store/types"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
)

func TestUpgrade221CachedStepContainsFailures(t *testing.T) {
	for _, outcome := range []string{"success", "error", "panic", "out of gas"} {
		t.Run(outcome, func(t *testing.T) {
			db := dbm.NewMemDB()
			ms := store.NewCommitMultiStore(db)
			key := storetypes.NewKVStoreKey("upgrade-test")
			ms.MountStoreWithDB(key, storetypes.StoreTypeIAVL, db)
			require.NoError(t, ms.LoadLatestVersion())
			ctx := sdk.NewContext(ms, tmproto.Header{}, false, log.NewNopLogger())
			ctx = ctx.WithGasMeter(sdk.NewGasMeter(100_000))
			beforeGas := ctx.GasMeter().GasConsumed()
			err := (Handler_v2_21{}).runCachedUpgradeStep(ctx, func(cached sdk.Context) error {
				cached.KVStore(key).Set([]byte("step"), []byte("written"))
				cached.EventManager().EmitEvent(sdk.NewEvent("custom_write"))
				switch outcome {
				case "error":
					return errors.New("injected error after write")
				case "panic":
					panic("injected panic after write")
				case "out of gas":
					cached.GasMeter().ConsumeGas(upgradeStepGasLimitV221+1, "injected exhaustion")
				}
				return nil
			})
			require.Equal(t, beforeGas, ctx.GasMeter().GasConsumed())
			if outcome == "success" {
				require.NoError(t, err)
				require.Equal(t, []byte("written"), ctx.KVStore(key).Get([]byte("step")))
				require.Len(t, ctx.EventManager().Events(), 1)
			} else {
				require.ErrorContains(t, err, outcome)
				require.Nil(t, ctx.KVStore(key).Get([]byte("step")))
				require.Empty(t, ctx.EventManager().Events())
				// A later independent step can still commit after discarded writes.
				require.NoError(t, (Handler_v2_21{}).runCachedUpgradeStep(ctx, func(cached sdk.Context) error {
					cached.KVStore(key).Set([]byte("later"), []byte("committed"))
					return nil
				}))
				require.Equal(t, []byte("committed"), ctx.KVStore(key).Get([]byte("later")))
			}
		})
	}
}
