package app

import (
	"context"
	"time"

	cs "github.com/NibiruChain/nibiru/v2/bft/consensus"
	cmtproto "github.com/NibiruChain/nibiru/v2/bft/proto/tendermint/types"

	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	consensustypes "github.com/NibiruChain/nibiru/v2/x/consensus/types"

	"github.com/NibiruChain/nibiru/v2/x/xconsensus"
)

// CommitTimeoutFunc returns the hook Nibiru BFT uses for its commit wait
// (node.CommitTimeoutFunc). The x-consensus value min_block_commit_duration_ms
// replaces config.toml timeout_commit on every node once it is set.
//
// The engine calls the hook under its consensus mutex after committing H,
// with height H+1, so the closure only loads an atomic. The x/xconsensus
// EndBlocker sets it during EndBlock H, before that commit, which makes a
// change executed in H govern the wait before H+1, together with the app
// params and header of H+1. The height argument is not needed for that.
func (app *NibiruApp) CommitTimeoutFunc() cs.CommitTimeoutFunc {
	keeper := app.XConsensusKeeper
	return func(int64) (time.Duration, bool) {
		return keeper.CommitWait()
	}
}

// warmXConsensusCommitWait loads the commit wait from the latest committed
// state, so the first height after a restart already uses the chain value
// instead of config.toml. It runs after app.Load and before the node starts.
// Params need no warm-up: they are in the x/consensus store. A failure only
// costs one height of config.toml pacing, so it is logged, not fatal.
func (app *NibiruApp) warmXConsensusCommitWait() {
	height := app.LastBlockHeight()
	if height == 0 {
		return // Fresh chain: InitChainer creates the contract unset.
	}
	// Wasm needs a positive block time. The committed header time is not
	// kept by baseapp, and this read is node-local and never touches state,
	// so the wall clock is fine.
	ctx := app.NewContext(true, cmtproto.Header{
		Height: height, Time: time.Now().UTC(),
	})
	if err := app.XConsensusKeeper.WarmCommitWait(ctx); err != nil {
		app.Logger().Error(
			"could not load the x-consensus commit wait; using config.toml "+
				"timeout_commit until the next block",
			"module", xconsensus.ModuleName, "height", height, "err", err)
	}
}

// consensusMsgUpdateParamsURL is the type URL of x/consensus MsgUpdateParams.
var consensusMsgUpdateParamsURL = sdk.MsgTypeURL(&consensustypes.MsgUpdateParams{})

// xconsensusCircuitBreaker rejects x/consensus MsgUpdateParams in the message
// router, so the x-consensus contract is the only writer of consensus params.
//
// It is defense in depth. Gov proposals execute through this router in the
// gov EndBlocker, where an ante filter would not see them, and the
// x/xconsensus EndBlocker, ordered last, would overwrite such a write in the
// same block anyway. Blocking in the router also stops the message through
// authz, Wasm and EVM dispatch. The x/consensus Params query stays available.
//
// The router holds a single circuit breaker, and no other code sets one today.
// Adding x/circuit or another breaker later would replace this one through
// SetCircuit, so the new breaker must compose with this check.
type xconsensusCircuitBreaker struct{}

// IsAllowed reports false only for x/consensus MsgUpdateParams.
func (xconsensusCircuitBreaker) IsAllowed(
	_ context.Context, typeURL string,
) (bool, error) {
	return typeURL != consensusMsgUpdateParamsURL, nil
}
