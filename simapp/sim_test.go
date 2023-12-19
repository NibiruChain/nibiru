package simapp_test

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
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/app"
	appsim "github.com/NibiruChain/nibiru/app/sim"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
)

// SimAppChainID hardcoded chainID for simulation
const SimAppChainID = "simulation-app"

type SimulationTestSuite struct {
	suite.Suite
}

func TestSimulationTestSuite(t *testing.T) {
	suite.Run(t, new(SimulationTestSuite))
}

var _ suite.SetupTestSuite = (*SimulationTestSuite)(nil)

func init() {
	// We call GetSimulatorFlags here in order to set the value for
	// 'simapp.FlagEnabledValue', which enables simulations
	appsim.GetSimulatorFlags()
}

// SetupTest: Runs before every test in the suite.
func (s *SimulationTestSuite) SetupTest() {
	testutil.BeforeIntegrationSuite(s.T())
	if !simapp.FlagEnabledValue {
		s.T().Skip("skipping application simulation")
	}
}

func (s *SimulationTestSuite) TestFullAppSimulation() {
	t := s.T()
	config := simcli.NewConfigFromFlags()
	config.ChainID = SimAppChainID

	db, dir, _, skip, err := helpers.SetupSimulation(
		config,
		"goleveldb-app-sim",
		"Simulation",
		simcli.FlagVerboseValue, simcli.FlagEnabledValue,
	)
	if skip {
		t.Skip("skipping application simulation")
	}
	require.NoError(t, err, "simulation setup failed")

	defer func() {
		db.Close()
		err = os.RemoveAll(dir)
		if err != nil {
			t.Fatal(err)
		}
	}()

	encoding := app.MakeEncodingConfig()
	app := testapp.NewNibiruTestApp(app.NewDefaultGenesisState(encoding.Marshaler))

	// Run randomized simulation:
	_, simParams, simErr := simulation.SimulateFromSeed(
		/* tb */ t,
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
		t.Fatal(err)
	}

	if simErr != nil {
		t.Fatal(simErr)
	}

	if config.Commit {
		simapp.PrintStats(db)
	}
}

func (s *SimulationTestSuite) TestAppStateDeterminism() {
	t := s.T()

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
