package app_test

import (
	"github.com/NibiruChain/nibiru/v2/app"
	abci "github.com/NibiruChain/nibiru/v2/bft/abci/types"
	"github.com/NibiruChain/nibiru/v2/bft/libs/log"
	cmtproto "github.com/NibiruChain/nibiru/v2/bft/proto/tendermint/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/client/flags"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/server"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/testutil/sims"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/testutil/testdata"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/module"
	authtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/gov"
	govtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/gov/types"
	v1 "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/gov/types/v1"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/gov/types/v1beta1"
	paramproposal "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/params/types/proposal"
	upgradetypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/upgrade/types"
	ibcclienttypes "github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/core/02-client/types"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testapp"
	tmdb "github.com/cometbft/cometbft-db"
	"github.com/stretchr/testify/require"
	"path/filepath"
	"testing"
	"time"
)

// TestExplicitGovernanceWiring protects the behavior previously supplied by
// depinject: module registration, governance authority, native and legacy
// message routing, and proposal persistence. Routing an authorized parameter
// update through the app also proves the service uses the app's keeper.
func TestExplicitGovernanceWiring(t *testing.T) {
	nibiru, ctx := testapp.NewNibiruTestAppAndContext()
	keeper := nibiru.GovKeeper
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	require.Equal(t, authority, keeper.GetAuthority())
	require.Equal(t, authority, keeper.GetGovernanceAccount(ctx).GetAddress().String())
	require.IsType(t, gov.AppModule{}, nibiru.ModuleManager.Modules[govtypes.ModuleName])
	require.Equal(t, uint64(gov.ConsensusVersion), nibiru.UpgradeKeeper.GetModuleVersionMap(ctx)[govtypes.ModuleName])

	// Native and legacy message services must be registered against the same
	// explicit keeper, and the app's four legacy proposal routes must survive.
	require.NotNil(t, nibiru.MsgServiceRouter().Handler(&v1.MsgSubmitProposal{}))
	require.NotNil(t, nibiru.MsgServiceRouter().Handler(&v1beta1.MsgSubmitProposal{}))
	for _, route := range []string{
		govtypes.RouterKey, paramproposal.RouterKey, upgradetypes.RouterKey, ibcclienttypes.RouterKey,
	} {
		require.True(t, keeper.LegacyRouter().HasRoute(route), route)
	}

	params := keeper.GetParams(ctx)
	updated := params
	updated.MinDeposit = sdk.NewCoins(sdk.NewInt64Coin(params.MinDeposit[0].Denom, 123456789))
	message := &v1.MsgUpdateParams{Authority: authority, Params: updated}
	handler := nibiru.MsgServiceRouter().Handler(message)
	require.NotNil(t, handler)
	_, err := handler(ctx, message)
	require.NoError(t, err)
	require.Equal(t, updated, keeper.GetParams(ctx))

	_, _, proposer := testdata.KeyTestPubAddr()
	message.Authority = proposer.String()
	message.Params = params
	_, err = handler(ctx, message)
	require.Error(t, err)
	require.Equal(t, updated, keeper.GetParams(ctx), "unauthorized message must not change params")

	legacy, err := v1.NewLegacyContent(v1beta1.NewTextProposal("Legacy route", "Still supported"), authority)
	require.NoError(t, err)
	proposal, err := keeper.SubmitProposal(ctx, []sdk.Msg{legacy}, "", "Legacy route", "Still supported", proposer)
	require.NoError(t, err)
	require.Equal(t, v1.StatusDepositPeriod, proposal.Status)
	stored, found := keeper.GetProposal(ctx, proposal.Id)
	require.True(t, found)
	require.Equal(t, proposal, stored)
	response, err := keeper.Proposal(sdk.WrapSDKContext(ctx), &v1.QueryProposalRequest{ProposalId: proposal.Id})
	require.NoError(t, err)
	require.Equal(t, stored, *response.Proposal)
}

// TestExplicitGovernanceWiringRestart reconstructs the app on a committed
// database. Explicit initialization must reconnect governance to its existing
// proposals and parameters and restore message services and legacy routes.
func TestExplicitGovernanceWiringRestart(t *testing.T) {
	db := tmdb.NewMemDB()
	nibiru, _ := testapp.NewNibiruTestAppWithDB(db, app.GenesisState{})
	ctx := testapp.NewContext(nibiru)
	params := nibiru.GovKeeper.GetParams(ctx)
	params.MinDeposit = sdk.NewCoins(sdk.NewInt64Coin(params.MinDeposit[0].Denom, 123456789))
	require.NoError(t, nibiru.GovKeeper.SetParams(ctx, params))
	_, _, proposer := testdata.KeyTestPubAddr()
	legacy, err := v1.NewLegacyContent(
		v1beta1.NewTextProposal("Persisted proposal", "Survives reconstruction"),
		nibiru.GovKeeper.GetAuthority(),
	)
	require.NoError(t, err)
	_, err = nibiru.GovKeeper.SubmitProposal(ctx, []sdk.Msg{legacy}, "", "Persisted proposal", "Survives reconstruction", proposer)
	require.NoError(t, err)
	want := gov.ExportGenesis(ctx, nibiru.GovKeeper)

	nibiru.EndBlock(abci.RequestEndBlock{Height: ctx.BlockHeight()})
	nibiru.Commit()
	height := nibiru.LastBlockHeight()
	restarted := app.NewNibiruApp(log.NewNopLogger(), db, nil, true, sims.EmptyAppOptions{})
	require.Equal(t, height, restarted.LastBlockHeight())
	header := cmtproto.Header{Height: height + 1, Time: ctx.BlockTime().Add(time.Second)}
	restarted.BeginBlock(abci.RequestBeginBlock{Header: header})
	restartedCtx := restarted.NewContext(false, header)
	require.Equal(t, want, gov.ExportGenesis(restartedCtx, restarted.GovKeeper))
	require.NotNil(t, restarted.MsgServiceRouter().Handler(&v1.MsgUpdateParams{}))
	require.True(t, restarted.GovKeeper.LegacyRouter().HasRoute(govtypes.RouterKey))
}

// TestLegacyUpgradeWiringOptions checks operator settings formerly passed
// through depinject. Losing skip heights, the node home, or the BaseApp version
// setter would change upgrade execution or the file watched by Cosmovisor.
func TestLegacyUpgradeWiringOptions(t *testing.T) {
	home := t.TempDir()
	nibiru := app.NewNibiruApp(log.NewNopLogger(), tmdb.NewMemDB(), nil, true, sims.AppOptionsMap{
		flags.FlagHome:                home,
		server.FlagUnsafeSkipUpgrades: []int{12, 34},
	})

	require.True(t, nibiru.UpgradeKeeper.IsSkipHeight(12))
	require.True(t, nibiru.UpgradeKeeper.IsSkipHeight(34))
	require.False(t, nibiru.UpgradeKeeper.IsSkipHeight(13))
	infoPath, err := nibiru.UpgradeKeeper.GetUpgradeInfoPath()
	require.NoError(t, err)
	require.Equal(t, filepath.Join(home, "data", upgradetypes.UpgradeInfoFilename), infoPath)
	require.Same(t, nibiru.BaseApp, nibiru.UpgradeKeeper.GetVersionSetter())
}

// TestLegacyUpgradeWiringExecutesPlan exercises the registered governance
// message service and first BeginBlock module after explicit keeper setup.
// It checks authorization, handler execution, protocol-version advancement,
// pending-plan removal, and completion history as one application flow.
func TestLegacyUpgradeWiringExecutesPlan(t *testing.T) {
	nibiru, ctx := testapp.NewNibiruTestAppAndContext()
	versions := nibiru.ModuleManager.GetVersionMap()
	require.Equal(t, versions, nibiru.UpgradeKeeper.GetModuleVersionMap(ctx))
	require.Equal(t, uint64(3), versions[upgradetypes.ModuleName])
	require.Equal(t, upgradetypes.ModuleName, nibiru.ModuleManager.OrderBeginBlockers[0])

	// Exercise the registered Msg service to preserve its governance authority
	// and its storage of a pending height-based plan.
	plan := upgradetypes.Plan{Name: "legacy-wiring-test", Height: ctx.BlockHeight()}
	handler := nibiru.MsgServiceRouter().Handler(&upgradetypes.MsgSoftwareUpgrade{})
	require.NotNil(t, handler)
	_, err := handler(ctx, &upgradetypes.MsgSoftwareUpgrade{
		Authority: authtypes.NewModuleAddress("not-governance").String(),
		Plan:      plan,
	})
	require.Error(t, err)
	_, found := nibiru.UpgradeKeeper.GetUpgradePlan(ctx)
	require.False(t, found)

	_, err = handler(ctx, &upgradetypes.MsgSoftwareUpgrade{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Plan:      plan,
	})
	require.NoError(t, err)

	before := nibiru.AppVersion()
	called := false
	nibiru.UpgradeKeeper.SetUpgradeHandler(plan.Name, func(_ sdk.Context, got upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		called = true
		require.Equal(t, plan, got)
		require.Equal(t, versions, fromVM)
		return fromVM, nil
	})
	nibiru.BeginBlocker(ctx, abci.RequestBeginBlock{Header: ctx.BlockHeader()})
	require.True(t, called)
	require.Equal(t, before+1, nibiru.AppVersion())
	require.Equal(t, versions, nibiru.UpgradeKeeper.GetModuleVersionMap(ctx))
	_, found = nibiru.UpgradeKeeper.GetUpgradePlan(ctx)
	require.False(t, found)
	require.Equal(t, ctx.BlockHeight(), nibiru.UpgradeKeeper.GetDoneHeight(ctx, plan.Name))
}
