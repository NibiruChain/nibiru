package upgrades_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	cmtproto "github.com/NibiruChain/nibiru/v2/bft/proto/tendermint/types"

	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/module"
	upgradetypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/upgrade/types"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/app/upgrades"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/xconsensus"
)

// preUpgradeParams stands in for the x/consensus params of a live chain
// before v2.22, with a Version that the upgrade must keep.
func preUpgradeParams() cmtproto.ConsensusParams {
	return cmtproto.ConsensusParams{
		Block: &cmtproto.BlockParams{MaxBytes: 22_020_096, MaxGas: 300_000_000},
		Evidence: &cmtproto.EvidenceParams{
			MaxAgeNumBlocks: 100_000,
			MaxAgeDuration:  48 * time.Hour,
			MaxBytes:        1_048_576,
		},
		Validator: &cmtproto.ValidatorParams{PubKeyTypes: []string{"ed25519"}},
		Version:   &cmtproto.VersionParams{App: 0},
	}
}

// runUpgrade2_22 runs the v2.22 handler like x/upgrade does: with the stored
// version map, recording the result only on success. It returns the events
// the handler emitted.
func runUpgrade2_22(
	t *testing.T, nibiru *app.NibiruApp, ctx sdk.Context,
) sdk.Events {
	t.Helper()
	handler := upgrades.Upgrade2_22_0.Handler.Handler(
		nibiru.ModuleManager, nibiru.Configurator(), &nibiru.PublicKeepers)
	ctx = ctx.WithEventManager(sdk.NewEventManager())
	plan := upgradetypes.Plan{
		Name: upgrades.Upgrade2_22_0.UpgradeName, Height: ctx.BlockHeight(),
	}
	versions, err := handler(ctx, plan,
		nibiru.UpgradeKeeper.GetModuleVersionMap(ctx))
	require.NoError(t, err)
	nibiru.UpgradeKeeper.SetModuleVersionMap(ctx, versions)
	return ctx.EventManager().Events()
}

// preUpgradeChain returns a chain without x-consensus whose x/consensus store
// holds params, and whose version map predates the xconsensus module, so the
// handler sees exactly what a live chain at the upgrade height has.
func preUpgradeChain(
	t *testing.T, chainID string, params *cmtproto.ConsensusParams,
) (*app.NibiruApp, sdk.Context) {
	t.Helper()
	nibiru, _ := testapp.NewNibiruTestAppPreXConsensus()
	ctx := testapp.NewContext(nibiru).WithChainID(chainID)
	_, found := nibiru.XConsensusKeeper.ContractAddr(ctx)
	require.False(t, found)
	nibiru.ConsensusParamsKeeper.Set(ctx, params)
	versions := nibiru.UpgradeKeeper.GetModuleVersionMap(ctx)
	delete(versions, xconsensus.ModuleName)
	nibiru.UpgradeKeeper.SetModuleVersionMap(ctx, module.VersionMap(versions))
	return nibiru, ctx
}

// TestUpgrade2_22_0CreatesXConsensus checks the upgrade on mainnet and
// testnet chain IDs: x-consensus is created at the predicted address, as its
// own Wasm admin, with exactly the stored params and the commit wait unset,
// and the stored params, Version included, are unchanged by the next refresh. A second run is a
// no-op. The mainnet chain ID exercises the Wasm deployer guard, which the
// governance permission keeper passes.
func TestUpgrade2_22_0CreatesXConsensus(t *testing.T) {
	for _, chainID := range []string{
		appconst.SDK_CHAIN_ID_MAINNET, "nibiru-testnet-2", "localnet",
	} {
		t.Run(chainID, func(t *testing.T) {
			pre := preUpgradeParams()
			nibiru, ctx := preUpgradeChain(t, chainID, &pre)

			events := runUpgrade2_22(t, nibiru, ctx)
			for _, ev := range events {
				require.NotEqual(t, "upgrade_failure", ev.Type, ev.Attributes)
			}
			versions := nibiru.UpgradeKeeper.GetModuleVersionMap(ctx)
			require.Equal(t, uint64(1), versions[xconsensus.ModuleName])

			addr, found := nibiru.XConsensusKeeper.ContractAddr(ctx)
			require.True(t, found)
			info := nibiru.WasmKeeper.GetContractInfo(ctx, addr)
			require.Equal(t, xconsensus.CreatorAddr().String(), info.Creator)
			require.Equal(t, addr.String(), info.Admin)
			checksum := nibiru.WasmKeeper.GetCodeInfo(ctx, info.CodeID).CodeHash
			require.Equal(t, xconsensus.PredictContractAddr(checksum), addr)

			req, err := json.Marshal(xconsensus.QueryParams())
			require.NoError(t, err)
			resp, err := nibiru.WasmKeeper.QuerySmart(ctx, addr, req)
			require.NoError(t, err)
			var got xconsensus.ConsensusParams
			require.NoError(t, json.Unmarshal(resp, &got))
			want, err := xconsensus.ConsensusParamsFromProto(pre)
			require.NoError(t, err)
			require.Equal(t, want, got)
			require.Nil(t, got.MinBlockCommitDurationMs)
			_, ok := nibiru.XConsensusKeeper.CommitWait()
			require.False(t, ok)

			// The first refresh after the upgrade writes nothing new.
			require.NoError(t, nibiru.XConsensusKeeper.Refresh(ctx))
			require.Equal(t, pre, *nibiru.GetConsensusParams(ctx))

			// Running the handler again creates nothing.
			runUpgrade2_22(t, nibiru, ctx)
			var count int
			nibiru.WasmKeeper.IterateContractsByCreator(ctx, xconsensus.CreatorAddr(),
				func(sdk.AccAddress) bool { count++; return false })
			require.Equal(t, 1, count)
		})
	}
}

// TestUpgrade2_22_0FailureContinues checks the failure path: with stored
// params that cannot seed the contract (no validator params), the handler
// reports an upgrade_failure event, keeps the migrations, creates no
// contract, and does not halt.
func TestUpgrade2_22_0FailureContinues(t *testing.T) {
	broken := preUpgradeParams()
	broken.Validator = nil
	nibiru, ctx := preUpgradeChain(t, "localnet", &broken)
	events := runUpgrade2_22(t, nibiru, ctx)

	var failed bool
	for _, ev := range events {
		if ev.Type == "upgrade_failure" {
			failed = true
		}
	}
	require.True(t, failed, "expected an upgrade_failure event")
	_, found := nibiru.XConsensusKeeper.ContractAddr(ctx)
	require.False(t, found)
	versions := nibiru.UpgradeKeeper.GetModuleVersionMap(ctx)
	require.Equal(t, uint64(1), versions[xconsensus.ModuleName])
}
