package simapp

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"testing"

	sdkSimapp "github.com/cosmos/cosmos-sdk/simapp"
	"github.com/cosmos/cosmos-sdk/simapp/helpers"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	simulationtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"
	"github.com/stretchr/testify/require"
	dbm "github.com/tendermint/tm-db"

	"github.com/NibiruChain/nibiru/x/testutil/testapp"
)

func init() {
	sdkSimapp.GetSimulatorFlags()
}

func TestFullAppSimulation(tb *testing.T) {
	config, db, dir, _, _, err := sdkSimapp.SetupSimulation("goleveldb-app-sim", "Simulation")
	if err != nil {
		tb.Fatalf("simulation setup failed: %s", err.Error())
	}

	defer func() {
		db.Close()
		err = os.RemoveAll(dir)
		if err != nil {
			tb.Fatal(err)
		}
	}()

	nibiru := testapp.NewNibiruApp( /*shouldUseDefaultGenesis*/ true)

	// Run randomized simulation:
	_, simParams, simErr := simulation.SimulateFromSeed(
		/* tb */ tb,
		/* w */ os.Stdout,
		/* app */ nibiru.BaseApp,
		/* appStateFn */ AppStateFn(nibiru.AppCodec(), nibiru.SimulationManager()),
		/* randAccFn */ simulationtypes.RandomAccounts, // Replace with own random account function if using keys other than secp256k1
		/* ops */ sdkSimapp.SimulationOperations(nibiru, nibiru.AppCodec(), config), // Run all registered operations
		/* blockedAddrs */ nibiru.ModuleAccountAddrs(),
		/* config */ config,
		/* cdc */ nibiru.AppCodec(),
	)

	// export state and simParams before the simulation error is checked
	if err = sdkSimapp.CheckExportSimulation(nibiru, config, simParams); err != nil {
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
			app := testapp.NewNibiruApp( /*shouldUseDefaultGenesis*/ true)

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
