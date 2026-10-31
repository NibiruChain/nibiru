package app_test

import (
	"testing"
	"time"

	tmdb "github.com/cometbft/cometbft-db"
	"github.com/stretchr/testify/require"

	abci "github.com/NibiruChain/nibiru/v2/bft/abci/types"
	"github.com/NibiruChain/nibiru/v2/bft/libs/log"
	cmtproto "github.com/NibiruChain/nibiru/v2/bft/proto/tendermint/types"

	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/testutil/sims"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/testutil/testdata"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	authtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/gov"
	govtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/gov/types"
	v1 "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/gov/types/v1"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/gov/types/v1beta1"
	paramproposal "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/params/types/proposal"
	upgradetypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/upgrade/types"
	ibcclienttypes "github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/core/02-client/types"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testapp"
)

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
