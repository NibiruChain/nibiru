package incentivization_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	"github.com/NibiruChain/nibiru/x/incentivization"
	"github.com/NibiruChain/nibiru/x/incentivization/keeper"
	"github.com/NibiruChain/nibiru/x/incentivization/types"
	"github.com/NibiruChain/nibiru/x/testutil"
)

func TestAppModule_InitGenesis_ExportGenesis(t *testing.T) {
	app := testutil.NewTestApp(false)

	am := incentivization.NewAppModule(app.AppCodec(), app.IncentivizationKeeper)
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
	am.InitGenesis(ctx, app.AppCodec(), genesisRaw)
	// export again
	genesisRaw = am.ExportGenesis(ctx, app.AppCodec())
	genesis = new(types.GenesisState)
	app.AppCodec().MustUnmarshalJSON(genesisRaw, genesis)
	require.Equal(t, programs, genesis.IncentivizationPrograms) // must be equal to export
}
