package consensus_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	cmttypes "github.com/NibiruChain/nibiru/v2/bft/types"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/consensus"
	"github.com/NibiruChain/nibiru/v2/x/consensus/types"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil"
	"github.com/NibiruChain/nibiru/v2/x/sudo"
	wasmkeeper "github.com/NibiruChain/nibiru/v2/x/wasm/keeper"
	wasm "github.com/NibiruChain/nibiru/v2/x/wasm/types"
)

// fixtureDir holds the JSON fixtures shared with the Rust tests in
// wasm-contracts/x-consensus/src/tests.rs.
const fixtureDir = "../../wasm-contracts/x-consensus/testdata"

// readFixture returns a fixture file's bytes.
func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	bz, err := os.ReadFile(filepath.Join(fixtureDir, name))
	require.NoError(t, err)
	return bz
}

// defaultParams returns the engine defaults (bft/types DefaultConsensusParams)
// as contract params, which is what the instantiate fixture holds.
func defaultParams(t *testing.T) types.ConsensusParams {
	t.Helper()
	params, err := types.ConsensusParamsFromProto(
		cmttypes.DefaultConsensusParams().ToProto())
	require.NoError(t, err)
	return params
}

// updateParams returns the params in the update_params fixture.
func updateParams(t *testing.T) types.ConsensusParams {
	t.Helper()
	params := defaultParams(t)
	params.Validator.PubKeyTypes = []string{"ed25519", "secp256k1"}
	ms := uint64(200)
	params.MinBlockCommitDurationMs = &ms
	return params
}

// TestEmbeddedWasm checks that the embedded x_consensus.wasm.gz decompresses,
// within the Wasm size limit, to a Wasm module whose SHA-256 matches
// ContractWasmSHA256. ContractWasm decompresses lazily, so this test is what
// catches a corrupt archive before a release, and a rebuilt artifact cannot
// ship with a stale checksum, which also fixes the published contract
// address. It also checks that callers get independent copies.
func TestEmbeddedWasm(t *testing.T) {
	code := consensus.ContractWasm()
	require.NotEmpty(t, code)
	require.True(t, bytes.HasPrefix(code, []byte("\x00asm")),
		"embedded bytes are not a Wasm module")
	sum := sha256.Sum256(code)
	require.Equal(t, consensus.ContractWasmSHA256, hex.EncodeToString(sum[:]))

	code[0] = 0xff
	require.True(t,
		bytes.HasPrefix(consensus.ContractWasm(), []byte("\x00asm")),
		"a caller's write must not reach the next caller")
}

// TestContractOnChain stores the embedded Wasm in a real app, instantiates
// it from the JSON fixture, and checks that only the x/sudo root, read
// through the Stargate QuerySudoers path, can update the params. It also
// checks that the contract rejects one more than
// types.MaxMinBlockCommitDurationMs on instantiate and on update and accepts
// the bound itself, which keeps the Go constant equal to the Rust one.
func TestContractOnChain(t *testing.T) {
	deps := evmtest.NewTestDeps()
	ctx := deps.Ctx()
	keeper := wasmkeeper.NewDefaultPermissionKeeper(deps.App.WasmKeeper)
	creator := deps.Sender.NibiruAddr

	codeID, checksum, err := keeper.Create(
		ctx, creator, consensus.ContractWasm(),
		&wasm.AccessConfig{Permission: wasm.AccessTypeEverybody},
	)
	require.NoError(t, err)
	require.Equal(t, consensus.ContractWasmSHA256, hex.EncodeToString(checksum))

	// The contract applies the commit wait bound at instantiate too.
	tooLong := types.MaxMinBlockCommitDurationMs + 1
	badInit := defaultParams(t)
	badInit.MinBlockCommitDurationMs = &tooLong
	badInitMsg, err := json.Marshal(types.InstantiateMsg{Params: badInit})
	require.NoError(t, err)
	_, _, err = keeper.Instantiate(
		ctx, codeID, creator, nil, badInitMsg, "x-consensus", sdk.Coins{})
	require.ErrorContains(t, err, "min_block_commit_duration_ms must be at most")

	// No admin: in this test only governance could migrate. Production
	// creation (EnsureXConsensus) makes the contract its own admin.
	contract, _, err := keeper.Instantiate(
		ctx, codeID, creator, nil,
		readFixture(t, "instantiate_msg.json"), "x-consensus", sdk.Coins{},
	)
	require.NoError(t, err)

	queryParams := func() types.ConsensusParams {
		bz, err := json.Marshal(types.QueryParams())
		require.NoError(t, err)
		resp, err := deps.App.WasmKeeper.QuerySmart(ctx, contract, bz)
		require.NoError(t, err)
		var params types.ConsensusParams
		require.NoError(t, json.Unmarshal(resp, &params))
		return params
	}
	require.Equal(t, defaultParams(t), queryParams())

	updateMsg := readFixture(t, "update_params_msg.json")
	setRoot := func(root string) {
		deps.App.SudoKeeper.Sudoers.Set(ctx, sudo.Sudoers{Root: root})
	}

	setRoot(testutil.ADDR_SUDO_ROOT)
	_, err = keeper.Execute(ctx, contract, creator, updateMsg, sdk.Coins{})
	require.ErrorContains(t, err, "is not the x/sudo root")

	setRoot("")
	_, err = keeper.Execute(ctx, contract, creator, updateMsg, sdk.Coins{})
	require.ErrorContains(t, err, "x/sudo has no root set")
	require.Equal(t, defaultParams(t), queryParams())

	setRoot(creator.String())
	_, err = keeper.Execute(ctx, contract, creator, updateMsg, sdk.Coins{})
	require.NoError(t, err)
	require.Equal(t, updateParams(t), queryParams())

	// Validation runs on chain too.
	bad := updateParams(t)
	bad.MinBlockCommitDurationMs = &tooLong
	badMsg, err := json.Marshal(types.ExecuteMsg{
		UpdateParams: &types.UpdateParamsMsg{Params: bad},
	})
	require.NoError(t, err)
	_, err = keeper.Execute(ctx, contract, creator, badMsg, sdk.Coins{})
	require.ErrorContains(t, err, "min_block_commit_duration_ms must be at most")
	require.Equal(t, updateParams(t), queryParams())

	// The bound itself is allowed.
	atMax := updateParams(t)
	maxMs := types.MaxMinBlockCommitDurationMs
	atMax.MinBlockCommitDurationMs = &maxMs
	atMaxMsg, err := json.Marshal(types.ExecuteMsg{
		UpdateParams: &types.UpdateParamsMsg{Params: atMax},
	})
	require.NoError(t, err)
	_, err = keeper.Execute(ctx, contract, creator, atMaxMsg, sdk.Coins{})
	require.NoError(t, err)
	require.Equal(t, atMax, queryParams())
}
