package lockup_test

import (
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	"github.com/NibiruChain/nibiru/x/lockup"
	"github.com/NibiruChain/nibiru/x/lockup/types"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
	"github.com/NibiruChain/nibiru/x/testutil/testapp"
)

func TestAppModule_ExportGenesis_ImportGenesis(t *testing.T) {
	app := testapp.NewNibiruApp(false)
	am := lockup.NewAppModule(app.AppCodec(), app.LockupKeeper, app.AccountKeeper, app.BankKeeper)

	ctxUncached := app.NewContext(false, tmproto.Header{Time: time.Now()})
	// create locks
	ctx, _ := ctxUncached.CacheContext()
	var locks []*types.Lock
	for i := 0; i < 100; i++ {
		addr := sample.AccAddress()
		coins := sdk.NewCoins(
			sdk.NewInt64Coin("test", 1+int64(i)*100),
		)
		err := simapp.FundAccount(app.BankKeeper, ctx, addr, coins)
		require.NoError(t, err)

		lock, err := app.LockupKeeper.LockTokens(ctx, addr, coins, time.Duration(i)*time.Hour*24)
		require.NoError(t, err)

		locks = append(locks, lock)
	}

	// export genesis
	genesisRaw := am.ExportGenesis(ctx, app.AppCodec())
	genesis := new(types.GenesisState)
	app.AppCodec().MustUnmarshalJSON(genesisRaw, genesis)

	require.Equal(t, locks, genesis.Locks)

	// test import
	ctx, _ = ctxUncached.CacheContext()
	// we need to refund accounts
	for _, lock := range genesis.Locks {
		owner, err := sdk.AccAddressFromBech32(lock.Owner)
		require.NoError(t, err)
		err = simapp.FundAccount(app.BankKeeper, ctx, owner, lock.Coins)
		require.NoError(t, err)
	}

	am.InitGenesis(ctx, app.AppCodec(), genesisRaw)
	exportedAfterImport := am.ExportGenesis(ctx, app.AppCodec())
	require.Equal(t, genesisRaw, exportedAfterImport)
}
