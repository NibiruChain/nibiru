package keeper_test

import (
	"os"
	"testing"
	"time"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wasmvm "github.com/NibiruChain/nibiru/v2/lib/wasmvm-ffi"
	wasmvmtypes "github.com/NibiruChain/nibiru/v2/lib/wasmvm-ffi/wvm"

	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/crypto/keys/ed25519"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/wasm/keeper"
	"github.com/NibiruChain/nibiru/v2/x/wasm/types"
)

func TestSnapshotter(t *testing.T) {
	t.Skip("TODO(x/wasm): enable after the Nibiru unit test app configures a snapshot store")

	specs := map[string]struct {
		wasmFiles []string
	}{
		"single contract": {
			wasmFiles: []string{"../testdata/reflect.wasm"},
		},
		"multiple contract": {
			wasmFiles: []string{"../testdata/reflect.wasm", "../testdata/burner.wasm", "../testdata/reflect.wasm"},
		},
		"duplicate contracts": {
			wasmFiles: []string{"../testdata/reflect.wasm", "../testdata/reflect.wasm"},
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			// setup source app
			srcWasmApp, genesisAddr := newWasmExampleApp(t)

			// store wasm codes on chain
			ctx := srcWasmApp.NewUncachedContext(false, tmproto.Header{
				ChainID: "foo",
				Height:  srcWasmApp.LastBlockHeight() + 1,
				Time:    time.Now(),
			})
			wasmKeeper := srcWasmApp.WasmKeeper
			require.NoError(t, wasmKeeper.SetParams(ctx, types.DefaultParams()))
			contractKeeper := keeper.NewDefaultPermissionKeeper(&wasmKeeper)

			srcCodeIDToChecksum := make(map[uint64][]byte, len(spec.wasmFiles))
			for i, v := range spec.wasmFiles {
				wasmCode, err := os.ReadFile(v)
				require.NoError(t, err)
				codeID, checksum, err := contractKeeper.Create(ctx, genesisAddr, wasmCode, nil)
				require.NoError(t, err)
				require.Equal(t, uint64(i+1), codeID)
				srcCodeIDToChecksum[codeID] = checksum
			}
			// create snapshot
			srcWasmApp.Commit()
			snapshotHeight := uint64(srcWasmApp.LastBlockHeight())
			snapshot, err := srcWasmApp.SnapshotManager().Create(snapshotHeight)
			require.NoError(t, err)
			assert.NotNil(t, snapshot)

			originalMaxWasmSize := types.MaxWasmSize
			types.MaxWasmSize = 1
			t.Cleanup(func() {
				types.MaxWasmSize = originalMaxWasmSize
			})

			// when snapshot imported into dest app instance
			destWasmApp, _ := testapp.NewNibiruTestApp(app.GenesisState{})
			require.NoError(t, destWasmApp.SnapshotManager().Restore(*snapshot))
			for i := uint32(0); i < snapshot.Chunks; i++ {
				chunkBz, err := srcWasmApp.SnapshotManager().LoadChunk(snapshot.Height, snapshot.Format, i)
				require.NoError(t, err)
				end, err := destWasmApp.SnapshotManager().RestoreChunk(chunkBz)
				require.NoError(t, err)
				if end {
					break
				}
			}

			// then all wasm contracts are imported
			wasmKeeper = destWasmApp.WasmKeeper
			ctx = destWasmApp.NewUncachedContext(false, tmproto.Header{
				ChainID: "foo",
				Height:  destWasmApp.LastBlockHeight() + 1,
				Time:    time.Now(),
			})

			destCodeIDToChecksum := make(map[uint64][]byte, len(spec.wasmFiles))
			wasmKeeper.IterateCodeInfos(ctx, func(id uint64, info types.CodeInfo) bool {
				bz, err := wasmKeeper.GetByteCode(ctx, id)
				require.NoError(t, err)

				hash, err := wasmvm.CreateChecksum(bz)
				require.NoError(t, err)
				destCodeIDToChecksum[id] = hash[:]
				assert.Equal(t, hash[:], wasmvmtypes.Checksum(info.CodeHash))
				return false
			})
			assert.Equal(t, srcCodeIDToChecksum, destCodeIDToChecksum)
		})
	}
}

func newWasmExampleApp(t *testing.T) (*app.NibiruApp, sdk.AccAddress) {
	t.Helper()
	senderPrivKey := ed25519.GenPrivKey()

	senderAddr := senderPrivKey.PubKey().Address().Bytes()
	amount, ok := sdk.NewIntFromString("10000000000000000000")
	require.True(t, ok)

	wasmApp, ctx := testapp.NewNibiruTestAppAndContext()
	require.NoError(t, wasmApp.WasmKeeper.SetParams(ctx, types.DefaultParams()))
	require.NoError(t, testapp.FundAccount(
		wasmApp.BankKeeper,
		ctx,
		senderAddr,
		sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, amount)),
	))

	return wasmApp, senderAddr
}
