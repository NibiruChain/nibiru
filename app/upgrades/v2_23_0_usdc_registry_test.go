package upgrades_test

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/app/upgrades"
	"github.com/NibiruChain/nibiru/v2/evm/evmtest"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	"github.com/NibiruChain/nibiru/v2/x/sudo"
	wasmkeeper "github.com/NibiruChain/nibiru/v2/x/wasm/keeper"
	wasmtypes "github.com/NibiruChain/nibiru/v2/x/wasm/types"
)

const holderSetFixtureSHA256 = "0a18618d6d1dcf9ef0cf4a92252abf879b96a22066b1e7e57e2efd1c958204c8"

func holderSetBytecode(t *testing.T) ([]byte, []byte) {
	t.Helper()
	compressed, err := os.ReadFile("testdata/v2.23/usdc_holder_set.wasm.gz")
	require.NoError(t, err)
	reader, err := gzip.NewReader(bytes.NewReader(compressed))
	require.NoError(t, err)
	bytecode, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.NoError(t, reader.Close())
	checksum := sha256.Sum256(bytecode)
	require.Equal(t, holderSetFixtureSHA256, hex.EncodeToString(checksum[:]))
	return bytecode, checksum[:]
}

func holderSetAddress(n int) string { return fmt.Sprintf("0x%040x", n) }

func deployHolderSet(t *testing.T, deps evmtest.TestDeps, creator sdk.AccAddress, initial []string, salt []byte) (sdk.AccAddress, []byte) {
	t.Helper()
	bytecode, checksum := holderSetBytecode(t)
	keeper := wasmkeeper.NewDefaultPermissionKeeper(deps.App.WasmKeeper)
	codeID, gotChecksum, err := keeper.Create(deps.Ctx(), creator, bytecode, &wasmtypes.AllowEverybody)
	require.NoError(t, err)
	require.Equal(t, checksum, gotChecksum)
	msg, err := json.Marshal(map[string]any{"admin": creator.String(), "addresses": initial})
	require.NoError(t, err)
	contract, _, err := keeper.Instantiate2(deps.Ctx(), codeID, creator, nil, msg,
		"temporary v2.23 holder set", sdk.Coins{}, salt, false)
	require.NoError(t, err)
	require.Equal(t, wasmkeeper.BuildContractAddressPredictable(checksum, creator, salt, nil), contract)
	return contract, checksum
}

// TestUSDCHolderSetKeeper executes the gzipped fixture through the real Wasm
// keeper. It models predeployment, batched SQL imports, and a last-minute
// operator update before the migration reads every page. No balances are stored.
func TestUSDCHolderSetKeeper(t *testing.T) {
	deps := evmtest.NewTestDeps()
	deps.SetCtx(deps.Ctx().WithGasMeter(sdk.NewInfiniteGasMeter()))
	creator := sdk.AccAddress(bytes.Repeat([]byte{17}, 20))
	contract, checksum := deployHolderSet(t, deps, creator, []string{}, []byte("usdc-holders-v1"))
	empty, err := upgrades.ReadUSDCHolderSet(deps.Ctx(), deps.App.WasmKeeper, contract, checksum)
	require.NoError(t, err)
	require.Empty(t, empty)
	keeper := wasmkeeper.NewDefaultPermissionKeeper(deps.App.WasmKeeper)
	update := func(sender sdk.AccAddress, add, remove []string) error {
		if add == nil {
			add = []string{}
		}
		if remove == nil {
			remove = []string{}
		}
		msg, err := json.Marshal(map[string]any{"update_addresses": map[string]any{"add": add, "remove": remove}})
		require.NoError(t, err)
		// Ordinary message execution commits only successful contract writes.
		ctx, commit := deps.Ctx().CacheContext()
		_, err = keeper.Execute(ctx, contract, sender, msg, sdk.Coins{})
		if err == nil {
			commit()
		}
		return err
	}
	for start := 1; start <= 1007; start += 500 {
		var add []string
		for n := start; n < start+500 && n <= 1007; n++ {
			add = append(add, strings.ToUpper(holderSetAddress(n)))
		}
		require.NoError(t, update(creator, add, nil))
	}
	require.ErrorContains(t, update(sdk.AccAddress(bytes.Repeat([]byte{18}, 20)), []string{holderSetAddress(2000)}, nil), "unauthorized")
	require.Error(t, update(creator, []string{holderSetAddress(2000), "invalid"}, []string{holderSetAddress(1)}))
	// The final operator push changes the candidate set after initial loading.
	require.NoError(t, update(creator,
		[]string{holderSetAddress(1008), holderSetAddress(1008)}, []string{holderSetAddress(1)}))
	got, err := upgrades.ReadUSDCHolderSet(deps.Ctx(), deps.App.WasmKeeper, contract, checksum)
	require.NoError(t, err)
	var want []gethcommon.Address
	for n := 2; n <= 1008; n++ {
		want = append(want, gethcommon.HexToAddress(holderSetAddress(n)))
	}
	require.Equal(t, want, got)
	// Neither a wrong artifact nor a missing predeployment can yield candidates.
	wrongChecksum := bytes.Repeat([]byte{42}, 32)
	got, err = upgrades.ReadUSDCHolderSet(deps.Ctx(), deps.App.WasmKeeper, contract, wrongChecksum)
	require.ErrorContains(t, err, "unexpected bytecode")
	require.Nil(t, got)
	missing := sdk.AccAddress(bytes.Repeat([]byte{99}, 32))
	before := deps.App.WasmKeeper.PeekAutoIncrementID(deps.Ctx(), wasmtypes.KeySequenceCodeID)
	got, err = upgrades.ReadUSDCHolderSet(deps.Ctx(), deps.App.WasmKeeper, missing, checksum)
	require.ErrorContains(t, err, "not deployed")
	require.Nil(t, got)
	require.Equal(t, before, deps.App.WasmKeeper.PeekAutoIncrementID(deps.Ctx(), wasmtypes.KeySequenceCodeID))
}

// TestUSDCHolderSetInstantiate2Parity proves address parity with the same real
// artifact/creator/salt and different chains, initial state, and code IDs.
func TestUSDCHolderSetInstantiate2Parity(t *testing.T) {
	var shared sdk.AccAddress
	creator := sdk.AccAddress(bytes.Repeat([]byte{17}, 20))
	for index, chainID := range []string{"localnet", "nibiru-testnet-3", "cataclysm-1"} {
		t.Run(chainID, func(t *testing.T) {
			deps := evmtest.NewTestDeps()
			deps.SetCtx(deps.Ctx().WithChainID(chainID).WithGasMeter(sdk.NewInfiniteGasMeter()))
			// Provision deployment authority explicitly, including on mainnet.
			deps.App.SudoKeeper.Sudoers.Set(deps.Ctx(), sudo.Sudoers{Root: creator.String()})
			bytecode, _ := holderSetBytecode(t)
			keeper := wasmkeeper.NewDefaultPermissionKeeper(deps.App.WasmKeeper)
			for n := 0; n < index; n++ {
				_, _, err := keeper.Create(deps.Ctx(), creator, bytecode, &wasmtypes.AllowEverybody)
				require.NoError(t, err)
			}
			initial := []string{holderSetAddress(index + 1)}
			contract, checksum := deployHolderSet(t, deps, creator, initial, []byte("usdc-holders-v1"))
			if shared == nil {
				shared = contract
			}
			require.Equal(t, shared, contract)
			got, err := upgrades.ReadUSDCHolderSet(deps.Ctx(), deps.App.WasmKeeper, contract, checksum)
			require.NoError(t, err)
			require.Equal(t, []gethcommon.Address{gethcommon.HexToAddress(initial[0])}, got)
		})
	}
}
