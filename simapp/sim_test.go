package simapp

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"testing"

	"github.com/cosmos/ibc-go/v7/testing/simapp"

	dbm "github.com/cometbft/cometbft-db"
	helpers "github.com/cosmos/cosmos-sdk/testutil/sims"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	simulationtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"
	simcli "github.com/cosmos/cosmos-sdk/x/simulation/client/cli"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
)

// SimAppChainID hardcoded chainID for simulation
const SimAppChainID = "simulation-app"

func init() {
	app.GetSimulatorFlags()
}

func TestFullAppSimulation(tb *testing.T) {
	config := simcli.NewConfigFromFlags()
	config.ChainID = SimAppChainID

	db, dir, _, skip, err := helpers.SetupSimulation(
		config,
		"goleveldb-app-sim",
		"Simulation",
		simcli.FlagVerboseValue, simcli.FlagEnabledValue,
	)
	if skip {
		tb.Skip("skipping application simulation")
	}
	require.NoError(tb, err, "simulation setup failed")

	defer func() {
		db.Close()
		err = os.RemoveAll(dir)
		if err != nil {
			tb.Fatal(err)
		}
	}()

	encoding := app.MakeEncodingConfig()
	app := testapp.NewNibiruTestApp(app.NewDefaultGenesisState(encoding.Marshaler))

	// Run randomized simulation:
	_, simParams, simErr := simulation.SimulateFromSeed(
		/* tb */ tb,
		/* w */ os.Stdout,
		/* app */ app.BaseApp,
		/* appStateFn */ AppStateFn(app.AppCodec(), app.SimulationManager()),
		/* randAccFn */ simulationtypes.RandomAccounts, // Replace with own random account function if using keys other than secp256k1
		/* ops */ helpers.SimulationOperations(app, app.AppCodec(), config), // Run all registered operations
		/* blockedAddrs */ app.ModuleAccountAddrs(),
		/* config */ config,
		/* cdc */ app.AppCodec(),
	)

	// export state and simParams before the simulation error is checked
	if err = helpers.CheckExportSimulation(app, config, simParams); err != nil {
		tb.Fatal(err)
	}

	if simErr != nil {
		tb.Fatal(simErr)
	}

	if config.Commit {
		simapp.PrintStats(db)
	}
}

func TestAppStateDeterminism(t *testing.T) {
	if !simapp.FlagEnabledValue {
		t.Skip("skipping application simulation")
	}

	encoding := app.MakeEncodingConfig()

	config := simapp.NewConfigFromFlags()
	config.InitialBlockHeight = 1
	config.ExportParamsPath = ""
	config.OnOperation = false
	config.AllInvariants = false
	config.ChainID = SimAppChainID

	numSeeds := 3
	numTimesToRunPerSeed := 5
	appHashList := make([]json.RawMessage, numTimesToRunPerSeed)

	for i := 0; i < numSeeds; i++ {
		config.Seed = rand.Int63()

		for j := 0; j < numTimesToRunPerSeed; j++ {
			db := dbm.NewMemDB()
			app := testapp.NewNibiruTestApp(app.NewDefaultGenesisState(encoding.Marshaler))

			fmt.Printf(
				"running non-determinism simulation; seed %d: %d/%d, attempt: %d/%d\n",
				config.Seed, i+1, numSeeds, j+1, numTimesToRunPerSeed,
			)

			_, _, err := simulation.SimulateFromSeed(
				t,
				os.Stdout,
				app.BaseApp,
				AppStateFn(app.AppCodec(), app.SimulationManager()),
				simtypes.RandomAccounts, // Replace with own random account function if using keys other than secp256k1
				helpers.SimulationOperations(app, app.AppCodec(), config),
				app.ModuleAccountAddrs(),
				config,
				app.AppCodec(),
			)
			require.NoError(t, err)

			if config.Commit {
				simapp.PrintStats(db)
			}

			appHash := app.LastCommitID().Hash
			appHashList[j] = appHash

			if j != 0 {
				require.Equal(
					t, string(appHashList[0]), string(appHashList[j]),
					"non-determinism in seed %d: %d/%d, attempt: %d/%d\n", config.Seed, i+1, numSeeds, j+1, numTimesToRunPerSeed,
				)
			}
		}
	}
}
