package types_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	cmtproto "github.com/NibiruChain/nibiru/v2/bft/proto/tendermint/types"
	cmttypes "github.com/NibiruChain/nibiru/v2/bft/types"

	"github.com/NibiruChain/nibiru/v2/x/consensus/types"
)

// fixtureDir holds the JSON fixtures shared with the Rust tests in
// wasm-contracts/x-consensus/src/tests.rs.
const fixtureDir = "../../../wasm-contracts/x-consensus/testdata"

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

// TestMessagesMatchContractJSON checks the Go types against the JSON the
// contract accepts. The contract rejects unknown fields, so any drift would
// make every update or the per-block query fail.
func TestMessagesMatchContractJSON(t *testing.T) {
	t.Run("instantiate", func(t *testing.T) {
		msg := types.InstantiateMsg{Params: defaultParams(t)}
		got, err := json.Marshal(msg)
		require.NoError(t, err)
		fixture := readFixture(t, "instantiate_msg.json")
		require.JSONEq(t, string(fixture), string(got))

		var decoded types.InstantiateMsg
		require.NoError(t, json.Unmarshal(fixture, &decoded))
		require.Equal(t, msg, decoded)
	})

	t.Run("update_params", func(t *testing.T) {
		msg := types.ExecuteMsg{
			UpdateParams: &types.UpdateParamsMsg{Params: updateParams(t)},
		}
		got, err := json.Marshal(msg)
		require.NoError(t, err)
		fixture := readFixture(t, "update_params_msg.json")
		require.JSONEq(t, string(fixture), string(got))

		var decoded types.ExecuteMsg
		require.NoError(t, json.Unmarshal(fixture, &decoded))
		require.Equal(t, msg, decoded)
	})

	t.Run("query and migrate", func(t *testing.T) {
		got, err := json.Marshal(types.QueryParams())
		require.NoError(t, err)
		require.JSONEq(t, `{"params":{}}`, string(got))
		got, err = json.Marshal(types.MigrateMsg{})
		require.NoError(t, err)
		require.JSONEq(t, `{}`, string(got))
	})

	t.Run("migrate and update_admin", func(t *testing.T) {
		got, err := json.Marshal(types.ExecuteMsg{
			Migrate: &types.MigrateExecMsg{CodeID: 42, Msg: []byte(`{}`)},
		})
		require.NoError(t, err)
		require.JSONEq(t, `{"migrate":{"code_id":42,"msg":"e30="}}`, string(got))
		got, err = json.Marshal(types.ExecuteMsg{
			UpdateAdmin: &types.UpdateAdminMsg{Admin: "nibi1admin"},
		})
		require.NoError(t, err)
		require.JSONEq(t, `{"update_admin":{"admin":"nibi1admin"}}`, string(got))
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
		var decoded types.ConsensusParams
		require.NoError(t, json.Unmarshal(bz, &decoded))
		require.Equal(t, params, decoded)
	})
}

// TestProtoConversion checks the conversion to and from Nibiru BFT's proto
// type: lossless for the three groups, version taken only from the caller,
// and an incomplete proto rejected.
func TestProtoConversion(t *testing.T) {
	pb := cmttypes.DefaultConsensusParams().ToProto()
	params, err := types.ConsensusParamsFromProto(pb)
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

	_, err = types.ConsensusParamsFromProto(cmtproto.ConsensusParams{})
	require.Error(t, err)
}
