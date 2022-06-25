package simapp

import (
	"os"
	"testing"

	sdkSimapp "github.com/cosmos/cosmos-sdk/simapp"
	simulationtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"

	"github.com/NibiruChain/nibiru/x/testutil/testapp"
)

// Profile with:
// /usr/local/go/bin/go test -benchmem -run=^$ github.com/NibiruChain/nibiru/simapp -bench ^BenchmarkFullAppSimulation$ -Commit=true -cpuprofile cpu.out
func BenchmarkFullAppSimulation(b *testing.B) {
	// -Enabled=true -NumBlocks=1000 -BlockSize=200 \
	// -Period=1 -Commit=true -Seed=57 -v -timeout 24h
	sdkSimapp.FlagEnabledValue = true
	sdkSimapp.FlagNumBlocksValue = 1000
	sdkSimapp.FlagBlockSizeValue = 200
	sdkSimapp.FlagCommitValue = true
	sdkSimapp.FlagVerboseValue = true
	// sdkSimapp.FlagPeriodValue = 1000
	fullAppSimulation(b, false)
}

func TestFullAppSimulation(t *testing.T) {
	// -Enabled=true -NumBlocks=1000 -BlockSize=200 \
	// -Period=1 -Commit=true -Seed=57 -v -timeout 24h
	sdkSimapp.FlagEnabledValue = true
	sdkSimapp.FlagNumBlocksValue = 50
	sdkSimapp.FlagBlockSizeValue = 50
	sdkSimapp.FlagCommitValue = true
	sdkSimapp.FlagVerboseValue = true
	sdkSimapp.FlagPeriodValue = 10
	sdkSimapp.FlagSeedValue = 10
	fullAppSimulation(t, true)
}

func fullAppSimulation(tb testing.TB, is_testing bool) {
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
