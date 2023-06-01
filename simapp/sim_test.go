package simapp

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"testing"

	sdkSimapp "cosmossdk.io/simapp"
	dbm "github.com/cometbft/cometbft-db"
	helpers "github.com/cosmos/cosmos-sdk/testutil/sims"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	simulationtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
)

func init() {
	sdkSimapp.GetSimulatorFlags()
}

func TestFullAppSimulation(tb *testing.T) {
	config, db, dir, _, skip, err := sdkSimapp.SetupSimulation("goleveldb-app-sim", "Simulation")
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

	encoding := app.MakeTestEncodingConfig()
	app := testapp.NewNibiruTestApp(app.NewDefaultGenesisState(encoding.Codec))

	// Run randomized simulation:
	_, simParams, simErr := simulation.SimulateFromSeed(
		/* tb */ tb,
		/* w */ os.Stdout,
		/* app */ app.BaseApp,
		/* appStateFn */ AppStateFn(app.AppCodec(), app.SimulationManager()),
		/* randAccFn */ simulationtypes.RandomAccounts, // Replace with own random account function if using keys other than secp256k1
		/* ops */ sdkSimapp.SimulationOperations(app, app.AppCodec(), config), // Run all registered operations
		/* blockedAddrs */ app.ModuleAccountAddrs(),
		/* config */ config,
		/* cdc */ app.AppCodec(),
	)

	// export state and simParams before the simulation error is checked
	if err = sdkSimapp.CheckExportSimulation(app, config, simParams); err != nil {
		tb.Fatal(err)
	}

	if simErr != nil {
		tb.Fatal(simErr)
	}

	if config.Commit {
		sdkSimapp.PrintStats(db)
	}
}

func TestAppStateDeterminism(t *testing.T) {
	if !sdkSimapp.FlagEnabledValue {
		t.Skip("skipping application simulation")
	}

	encoding := app.MakeTestEncodingConfig()

	config := sdkSimapp.NewConfigFromFlags()
	config.InitialBlockHeight = 1
	config.ExportParamsPath = ""
	config.OnOperation = false
	config.AllInvariants = false
	config.ChainID = helpers.SimAppChainID

	numSeeds := 3
	numTimesToRunPerSeed := 5
	appHashList := make([]json.RawMessage, numTimesToRunPerSeed)

	for i := 0; i < numSeeds; i++ {
		config.Seed = rand.Int63()

		for j := 0; j < numTimesToRunPerSeed; j++ {
			db := dbm.NewMemDB()
			app := testapp.NewNibiruTestApp(app.NewDefaultGenesisState(encoding.Codec))

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
				sdkSimapp.SimulationOperations(app, app.AppCodec(), config),
				app.ModuleAccountAddrs(),
				config,
				app.AppCodec(),
			)
			require.NoError(t, err)

			if config.Commit {
				sdkSimapp.PrintStats(db)
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
