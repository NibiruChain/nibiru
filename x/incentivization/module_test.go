package incentivization_test

import (
	"testing"
	"time"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/stretchr/testify/require"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	"github.com/NibiruChain/nibiru/x/incentivization"
	"github.com/NibiruChain/nibiru/x/incentivization/keeper"
	"github.com/NibiruChain/nibiru/x/incentivization/types"
	"github.com/NibiruChain/nibiru/x/testutil"
)

func TestAppModule_InitGenesis_ExportGenesis(t *testing.T) {
	app := testutil.NewTestApp(false)

	am := incentivization.NewAppModule(app.AppCodec(), app.IncentivizationKeeper, app.AccountKeeper)
	ctxUncached := app.NewContext(false, tmproto.Header{Time: time.Now()})
	ctx, _ := ctxUncached.CacheContext()
	// create some programs
	var programs []*types.IncentivizationProgram
	for i := 0; i < 100; i++ {
		program, err := app.IncentivizationKeeper.CreateIncentivizationProgram(ctx, "denom",
			24*time.Hour+time.Second*time.Duration(i), time.Now().Add(time.Duration(i)*time.Second),
			int64(i)*100+keeper.MinEpochs)
		require.NoError(t, err)
		program.StartTime = program.StartTime.UTC()
		programs = append(programs, program)
	}

	// export genesis
	genesisRaw := am.ExportGenesis(ctx, app.AppCodec())
	genesis := new(types.GenesisState)
	app.AppCodec().MustUnmarshalJSON(genesisRaw, genesis)
	require.Equal(t, programs, genesis.IncentivizationPrograms) // must be equal to creation
	// init genesis
	ctx, _ = ctxUncached.CacheContext()
	require.Panics(t, func() {
		// tests the lack of existence of escrow accounts in auth
		am.InitGenesis(ctx, app.AppCodec(), genesisRaw)
	})
	ctx, _ = ctxUncached.CacheContext()
	ctx = ctx.WithBlockTime(ctxUncached.BlockTime().Add(10 * time.Second)) // set in the future
	// init escrow accounts
	for _, p := range programs {
		escrowAccount := app.AccountKeeper.NewAccount(ctx, authtypes.NewEmptyModuleAccount(keeper.NewEscrowAccountName(p.Id))) // module account that holds the escrowed funds.
		app.AccountKeeper.SetAccount(ctx, escrowAccount)
	}
	// init working genesis
	am.InitGenesis(ctx, app.AppCodec(), genesisRaw)
	// export again
	genesisRaw = am.ExportGenesis(ctx, app.AppCodec())
	genesis = new(types.GenesisState)
	app.AppCodec().MustUnmarshalJSON(genesisRaw, genesis)
	require.Equal(t, programs, genesis.IncentivizationPrograms) // must be equal to export
}
