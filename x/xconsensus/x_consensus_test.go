package xconsensus_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	cmtproto "github.com/NibiruChain/nibiru/v2/bft/proto/tendermint/types"
	cmttypes "github.com/NibiruChain/nibiru/v2/bft/types"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil"
	"github.com/NibiruChain/nibiru/v2/x/sudo"
	wasmkeeper "github.com/NibiruChain/nibiru/v2/x/wasm/keeper"
	wasm "github.com/NibiruChain/nibiru/v2/x/wasm/types"
	"github.com/NibiruChain/nibiru/v2/x/xconsensus"
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
func defaultParams(t *testing.T) xconsensus.ConsensusParams {
	t.Helper()
	params, err := xconsensus.ConsensusParamsFromProto(
		cmttypes.DefaultConsensusParams().ToProto())
	require.NoError(t, err)
	return params
}

// updateParams returns the params in the update_params fixture.
func updateParams(t *testing.T) xconsensus.ConsensusParams {
	t.Helper()
	params := defaultParams(t)
	params.Validator.PubKeyTypes = []string{"ed25519", "secp256k1"}
	ms := uint64(200)
	params.MinBlockCommitDurationMs = &ms
	return params
}

func TestEmbeddedWasm(t *testing.T) {
	require.NotEmpty(t, xconsensus.XConsensusWasm)
	require.True(t,
		bytes.HasPrefix(xconsensus.XConsensusWasm, []byte("\x00asm")),
		"embedded bytes are not a Wasm module")
	sum := sha256.Sum256(xconsensus.XConsensusWasm)
	require.Equal(t, xconsensus.XConsensusWasmSHA256, hex.EncodeToString(sum[:]))
}

func TestMessagesMatchContractJSON(t *testing.T) {
	t.Run("instantiate", func(t *testing.T) {
		msg := xconsensus.InstantiateMsg{Params: defaultParams(t)}
		got, err := json.Marshal(msg)
		require.NoError(t, err)
		fixture := readFixture(t, "instantiate_msg.json")
		require.JSONEq(t, string(fixture), string(got))

		var decoded xconsensus.InstantiateMsg
		require.NoError(t, json.Unmarshal(fixture, &decoded))
		require.Equal(t, msg, decoded)
	})

	t.Run("update_params", func(t *testing.T) {
		msg := xconsensus.ExecuteMsg{
			UpdateParams: &xconsensus.UpdateParamsMsg{Params: updateParams(t)},
		}
		got, err := json.Marshal(msg)
		require.NoError(t, err)
		fixture := readFixture(t, "update_params_msg.json")
		require.JSONEq(t, string(fixture), string(got))

		var decoded xconsensus.ExecuteMsg
		require.NoError(t, json.Unmarshal(fixture, &decoded))
		require.Equal(t, msg, decoded)
	})

	t.Run("query and migrate", func(t *testing.T) {
		got, err := json.Marshal(xconsensus.QueryParams())
		require.NoError(t, err)
		require.JSONEq(t, `{"params":{}}`, string(got))
		got, err = json.Marshal(xconsensus.MigrateMsg{})
		require.NoError(t, err)
		require.JSONEq(t, `{}`, string(got))
	})

	t.Run("int64 extremes stay exact", func(t *testing.T) {
		params := defaultParams(t)
		params.Block.MaxGas = 1<<63 - 1
		params.Evidence.MaxAgeDurationNs = -1 << 63
		bz, err := json.Marshal(params)
		require.NoError(t, err)
		require.Contains(t, string(bz), `"max_gas":"9223372036854775807"`)
		require.Contains(t, string(bz),
			`"max_age_duration_ns":"-9223372036854775808"`)
		var decoded xconsensus.ConsensusParams
		require.NoError(t, json.Unmarshal(bz, &decoded))
		require.Equal(t, params, decoded)
	})
}

func TestProtoConversion(t *testing.T) {
	pb := cmttypes.DefaultConsensusParams().ToProto()
	params, err := xconsensus.ConsensusParamsFromProto(pb)
	require.NoError(t, err)
	require.Nil(t, params.MinBlockCommitDurationMs)
	require.Equal(t, int64(48*time.Hour), params.Evidence.MaxAgeDurationNs)

	// The version comes from the caller, never from the contract.
	version := &cmtproto.VersionParams{App: 7}
	want := pb
	want.Version = version
	require.Equal(t, want, params.ToProto(version))

	_, ok := params.MinBlockCommitDuration()
	require.False(t, ok)
	ms := uint64(250)
	params.MinBlockCommitDurationMs = &ms
	d, ok := params.MinBlockCommitDuration()
	require.True(t, ok)
	require.Equal(t, 250*time.Millisecond, d)

	_, err = xconsensus.ConsensusParamsFromProto(cmtproto.ConsensusParams{})
	require.Error(t, err)
}

// TestContractOnChain stores the embedded Wasm in a real app, instantiates
// it from the JSON fixture, and checks that only the x/sudo root, read
// through the Stargate QuerySudoers path, can update the params.
func TestContractOnChain(t *testing.T) {
	deps := evmtest.NewTestDeps()
	ctx := deps.Ctx()
	keeper := wasmkeeper.NewDefaultPermissionKeeper(deps.App.WasmKeeper)
	creator := deps.Sender.NibiruAddr

	codeID, checksum, err := keeper.Create(
		ctx, creator, xconsensus.XConsensusWasm,
		&wasm.AccessConfig{Permission: wasm.AccessTypeEverybody},
	)
	require.NoError(t, err)
	require.Equal(t, xconsensus.XConsensusWasmSHA256, hex.EncodeToString(checksum))

	// No admin: the code is only migratable by governance.
	contract, _, err := keeper.Instantiate(
		ctx, codeID, creator, nil,
		readFixture(t, "instantiate_msg.json"), "x-consensus", sdk.Coins{},
	)
	require.NoError(t, err)

	queryParams := func() xconsensus.ConsensusParams {
		bz, err := json.Marshal(xconsensus.QueryParams())
		require.NoError(t, err)
		resp, err := deps.App.WasmKeeper.QuerySmart(ctx, contract, bz)
		require.NoError(t, err)
		var params xconsensus.ConsensusParams
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
	tooLong := xconsensus.MaxMinBlockCommitDurationMs + 1
	bad.MinBlockCommitDurationMs = &tooLong
	badMsg, err := json.Marshal(xconsensus.ExecuteMsg{
		UpdateParams: &xconsensus.UpdateParamsMsg{Params: bad},
	})
	require.NoError(t, err)
	_, err = keeper.Execute(ctx, contract, creator, badMsg, sdk.Coins{})
	require.ErrorContains(t, err, "min_block_commit_duration_ms must be at most")
	require.Equal(t, updateParams(t), queryParams())
}
