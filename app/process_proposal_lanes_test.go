package app_test

import (
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	abci "github.com/NibiruChain/nibiru/v2/bft/abci/types"
	tmproto "github.com/NibiruChain/nibiru/v2/bft/proto/tendermint/types"
	cmttypes "github.com/NibiruChain/nibiru/v2/bft/types"

	"github.com/NibiruChain/nibiru/v2/app"
	srvconfig "github.com/NibiruChain/nibiru/v2/app/server/config"
	"github.com/NibiruChain/nibiru/v2/evm/evmstate"
	"github.com/NibiruChain/nibiru/v2/evm/evmtest"
	cryptotypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/crypto/types"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/module"
	signingtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/tx/signing"
	upgradetypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/upgrade/types"
	"github.com/NibiruChain/nibiru/v2/x/sudo"
	wasm "github.com/NibiruChain/nibiru/v2/x/wasm/types"
)

// laneRulesDoneHeight is the height at which tests mark
// [app.LaneRulesUpgradeName] done. The lane rules apply from the next height.
const laneRulesDoneHeight int64 = 10

// activateLaneRules marks [app.LaneRulesUpgradeName] done at
// [laneRulesDoneHeight] through the upgrade keeper, as the upgrade would.
func activateLaneRules(t *testing.T, deps *evmtest.TestDeps) {
	t.Helper()
	keeper := deps.App.UpgradeKeeper
	keeper.SetUpgradeHandler(app.LaneRulesUpgradeName, func(
		_ sdk.Context, _ upgradetypes.Plan, vm module.VersionMap,
	) (module.VersionMap, error) {
		return vm, nil
	})
	ctx := deps.Ctx().WithBlockHeight(laneRulesDoneHeight)
	keeper.ApplyUpgrade(ctx, upgradetypes.Plan{
		Name:   app.LaneRulesUpgradeName,
		Height: laneRulesDoneHeight,
	})
	require.Equal(t, laneRulesDoneHeight,
		keeper.GetDoneHeight(deps.Ctx(), app.LaneRulesUpgradeName))
}

// acceptAllProcessProposal stands in for the stock ProcessProposal handler,
// which would reject the unsigned test transactions, so tests observe only
// the lane rules.
func acceptAllProcessProposal(
	sdk.Context, abci.RequestProcessProposal,
) abci.ResponseProcessProposal {
	return abci.ResponseProcessProposal{
		Status: abci.ResponseProcessProposal_ACCEPT,
	}
}

// laneRulesCtx returns the test context with consensus Block.MaxBytes and
// MaxGas set, as BaseApp sets them for proposal handlers.
func laneRulesCtx(deps *evmtest.TestDeps, maxBytes, maxGas int64) sdk.Context {
	return deps.Ctx().WithConsensusParams(&tmproto.ConsensusParams{
		Block: &tmproto.BlockParams{MaxBytes: maxBytes, MaxGas: maxGas},
	})
}

// processLaneRules runs [app.NewLaneOrderProcessProposalHandler] over txs at
// height and returns whether the proposal was accepted.
func processLaneRules(
	ctx sdk.Context,
	deps *evmtest.TestDeps,
	height int64,
	txs [][]byte,
) bool {
	handler := app.NewLaneOrderProcessProposalHandler(
		deps.App, acceptAllProcessProposal)
	resp := handler(ctx, abci.RequestProcessProposal{Height: height, Txs: txs})
	return resp.IsAccepted()
}

// laneRulesFixture is a lane-rule test setup with the lane rules active, one
// EVM transaction in the app mempool, and Sai zero-gas actors.
type laneRulesFixture struct {
	deps   evmtest.TestDeps
	actors saiProposalActors
	evmTx  []byte
	ctx    sdk.Context
}

func newLaneRulesFixture(t *testing.T) *laneRulesFixture {
	t.Helper()
	deps := evmtest.NewTestDeps()
	account := evmstate.NewEmptyAccount()
	require.NoError(t, deps.EvmKeeper.SetAccount(
		deps.Ctx(), deps.Sender.EthAddr, *account))
	evmTx := insertProposalEVMTransaction(t, &deps, 0, 100)
	actors := setSaiProposalActors(&deps)
	activateLaneRules(t, &deps)
	return &laneRulesFixture{
		deps:   deps,
		actors: actors,
		evmTx:  evmTx,
		ctx:    laneRulesCtx(&deps, 1_000_000, -1),
	}
}

// process runs the lane rules at the first enforced height.
func (f *laneRulesFixture) process(txs ...[]byte) bool {
	return processLaneRules(f.ctx, &f.deps, laneRulesDoneHeight+1, txs)
}

// TestLaneRulesRejectViolations proves ProcessProposal rejects a block, once
// the lane rules are active, when an EVM tx or an unrelated default-lane tx
// precedes a price write, when the transactions cannot be decoded or
// classified, or when the oracle and zero-gas lanes exceed the byte cap.
func TestLaneRulesRejectViolations(t *testing.T) {
	f := newLaneRulesFixture(t)
	a := f.actors
	setPrices := buildProposalWasmTx(t, &f.deps,
		proposalWasmExec(a.feeder, a.oracle, setPricesMsg))
	traderTx := buildProposalWasmTx(t, &f.deps,
		proposalWasmExec(a.trader, a.perp, triggerFillMsg))
	keeperFill := buildProposalWasmTx(t, &f.deps,
		proposalWasmExec(a.keeper, a.perp, triggerFillMsg))
	invalidSender := buildProposalWasmTx(t, &f.deps,
		proposalWasmExec("not-a-bech32-address", a.oracle, setPricesMsg))

	require.True(t, f.process(setPrices, keeperFill, f.evmTx, traderTx))

	require.False(t, f.process(f.evmTx, setPrices), "EVM before price write")
	require.False(t, f.process(traderTx, setPrices), "default before price write")
	require.False(t, f.process(keeperFill, setPrices), "zero-gas before price write")
	require.False(t, f.process(setPrices, []byte("not a tx")), "undecodable")
	require.False(t, f.process(invalidSender, traderTx), "unclassifiable")

	// The cap is Block.MaxBytes / 4. Fit the price write but not the fill.
	priceSize := cmttypes.ComputeProtoSizeForTxs([]cmttypes.Tx{setPrices})
	tight := laneRulesCtx(&f.deps, 4*priceSize, -1)
	require.True(t, processLaneRules(tight, &f.deps, laneRulesDoneHeight+1,
		[][]byte{setPrices, f.evmTx, traderTx}))
	require.False(t, processLaneRules(tight, &f.deps, laneRulesDoneHeight+1,
		[][]byte{setPrices, keeperFill}), "priority bytes over cap")
}

// TestLaneRulesAcceptValidBlocks proves the lane rules accept a block with
// no price write in any lane order and a price write preceded by an earlier
// tx from the same feeder. A sponsored price write is classified zero-gas, so
// its fee payer's earlier tx may not precede the block's price writes.
func TestLaneRulesAcceptValidBlocks(t *testing.T) {
	f := newLaneRulesFixture(t)
	a := f.actors
	otherContract := evmtest.NewEthPrivAcc().NibiruAddr.String()
	payer := evmtest.NewEthPrivAcc().NibiruAddr

	traderTx := buildProposalWasmTx(t, &f.deps,
		proposalWasmExec(a.trader, a.perp, triggerFillMsg))
	keeperFill := buildProposalWasmTx(t, &f.deps,
		proposalWasmExec(a.keeper, a.perp, triggerFillMsg))
	require.True(t, f.process(traderTx, f.evmTx, keeperFill), "no price write")
	require.True(t, f.process(), "empty block")

	feederCall := buildSequencedProposalWasmTx(t, &f.deps, nil, []uint64{0},
		proposalWasmExec(a.feeder, otherContract, `{"claim":{}}`))
	setPrices := buildSequencedProposalWasmTx(t, &f.deps, nil, []uint64{1},
		proposalWasmExec(a.feeder, a.oracle, setPricesMsg))
	require.True(t, f.process(feederCall, setPrices, keeperFill, f.evmTx),
		"promoted same-signer predecessor")

	// A price write with a distinct fee payer is not a price write, so it
	// promotes its payer's earlier tx only into the zero-gas lane, which may
	// not precede a price write.
	payerTx := buildSequencedProposalWasmTx(t, &f.deps, nil, []uint64{0},
		proposalWasmExec(payer.String(), otherContract, `{"claim":{}}`))
	sponsoredPrices := buildSequencedProposalWasmTx(t, &f.deps, payer,
		[]uint64{2, 1}, proposalWasmExec(a.feeder, a.oracle, setPriceMsg))
	require.False(t, f.process(payerTx, setPrices, sponsoredPrices, traderTx),
		"sponsored price write admits no fee payer predecessor")
	require.True(t, f.process(setPrices, payerTx, sponsoredPrices, traderTx),
		"sponsored price write in the zero-gas lane")
}

// TestLaneRulesInactiveBeforeUpgrade proves a block that breaks the lane
// rules is accepted, as by the stock handler, while
// [app.LaneRulesUpgradeName] has not run and at its own height, and rejected
// from the next height.
func TestLaneRulesInactiveBeforeUpgrade(t *testing.T) {
	deps := evmtest.NewTestDeps()
	actors := setSaiProposalActors(&deps)
	traderTx := buildProposalWasmTx(t, &deps,
		proposalWasmExec(actors.trader, actors.perp, triggerFillMsg))
	setPrices := buildProposalWasmTx(t, &deps,
		proposalWasmExec(actors.feeder, actors.oracle, setPricesMsg))
	violating := [][]byte{traderTx, setPrices, []byte("not a tx")}
	ctx := laneRulesCtx(&deps, 1_000_000, -1)

	require.True(t, processLaneRules(ctx, &deps, 1_000, violating))
	activateLaneRules(t, &deps)
	require.True(t, processLaneRules(ctx, &deps, laneRulesDoneHeight, violating))
	require.False(t,
		processLaneRules(ctx, &deps, laneRulesDoneHeight+1, violating))
}

// TestLaneRulesPrepareRegroupsOrphanedPromotion pins the subsequence pitfall.
// Feeder A sends a call at sequence 0, promoted into the oracle lane for A's
// price write at sequence 1, which the verifier rejects. Feeder B's price
// write follows. Kept in promotion order, A's call would precede B's price
// write with no later A price write to justify it, and every validator would
// reject the block. PrepareProposal regroups the selection so A's call moves
// behind B's price write, and ProcessProposal accepts the result.
func TestLaneRulesPrepareRegroupsOrphanedPromotion(t *testing.T) {
	f := newLaneRulesFixture(t)
	a := f.actors
	otherContract := evmtest.NewEthPrivAcc().NibiruAddr.String()
	feederB := evmtest.NewEthPrivAcc().NibiruAddr.String()
	f.deps.App.SudoKeeper.ZeroGasActors.Set(f.deps.Ctx(), sudo.ZeroGasActors{
		Senders:   []string{a.feeder, feederB, a.keeper},
		Contracts: []string{a.oracle, a.perp},
	})

	feederACall := buildSequencedProposalWasmTx(t, &f.deps, nil, []uint64{0},
		proposalWasmExec(a.feeder, otherContract, `{"claim":{}}`))
	feederAPrices := buildSequencedProposalWasmTx(t, &f.deps, nil, []uint64{1},
		proposalWasmExec(a.feeder, a.oracle, setPricesMsg))
	feederBPrices := buildSequencedProposalWasmTx(t, &f.deps, nil, []uint64{0},
		proposalWasmExec(feederB, a.oracle, setPricesMsg))

	require.False(t, f.process(feederACall, feederBPrices),
		"unrepaired promotion order breaks the oracle-first rule")

	verifier := &laneRulesPrepareVerifier{
		sequences: sequencePrepareVerifier{
			decode: f.deps.App.GetTxConfig().TxDecoder(),
			next:   map[string]uint64{},
		},
		reject: map[string]bool{string(feederAPrices): true},
	}
	handler := app.NewEVMPrepareProposalHandler(f.deps.App, verifier)
	response := handler(f.ctx, abci.RequestPrepareProposal{
		Height:     laneRulesDoneHeight + 1,
		MaxTxBytes: 1_000_000,
		Txs:        [][]byte{feederACall, feederAPrices, feederBPrices},
	})

	require.Equal(t,
		[][]byte{feederBPrices, feederACall, f.evmTx}, response.Txs)
	require.True(t, f.process(response.Txs...))
}

// laneRulesPrepareVerifier is [sequencePrepareVerifier] that also rejects
// listed transactions, as ante does for a tx with a bad signature or
// insufficient funds. A rejected tx advances no sequence.
type laneRulesPrepareVerifier struct {
	sequences sequencePrepareVerifier
	reject    map[string]bool
}

func (v *laneRulesPrepareVerifier) PrepareProposalVerifyTxBytes(
	txBytes []byte,
) error {
	if v.reject[string(txBytes)] {
		return fmt.Errorf("rejected by test verifier")
	}
	return v.sequences.PrepareProposalVerifyTxBytes(txBytes)
}

// laneRulesPropertyIterations returns the iteration count for
// [TestLaneRulesPrepareOutputPassesProcess], raised with
// NIBIRU_LANE_PROPERTY_ITERS.
func laneRulesPropertyIterations(t *testing.T) int {
	t.Helper()
	iterations := 2_000
	if raw := os.Getenv("NIBIRU_LANE_PROPERTY_ITERS"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		require.NoError(t, err)
		iterations = parsed
	}
	if testing.Short() {
		iterations = min(iterations, 200)
	}
	return iterations
}

// laneRulesPropertyWorld is the account setup of the randomized lane test:
// several feeders and a keeper on the zero-gas allowlist, payers and traders
// off it, and pubkeys reused across signatures to keep tx building cheap.
type laneRulesPropertyWorld struct {
	feeders, payers, traders []string
	keeper                   string
	oracle, perp, other      string
	pubKeys                  []cryptotypes.PubKey
}

func newLaneRulesPropertyWorld(deps *evmtest.TestDeps) laneRulesPropertyWorld {
	newAddr := func() string { return evmtest.NewEthPrivAcc().NibiruAddr.String() }
	world := laneRulesPropertyWorld{
		feeders: []string{newAddr(), newAddr(), newAddr()},
		payers:  []string{newAddr(), newAddr()},
		traders: []string{newAddr(), newAddr()},
		keeper:  newAddr(),
		oracle:  newAddr(),
		perp:    newAddr(),
		other:   newAddr(),
	}
	for range 4 {
		world.pubKeys = append(world.pubKeys, evmtest.NewEthPrivAcc().PrivKey.PubKey())
	}
	deps.App.SudoKeeper.ZeroGasActors.Set(deps.Ctx(), sudo.ZeroGasActors{
		Senders:   append(append([]string{}, world.feeders...), world.keeper),
		Contracts: []string{world.oracle, world.perp},
	})
	return world
}

// laneRulesCandidates generates one random CometBFT candidate list.
type laneRulesCandidates struct {
	t       *testing.T
	deps    *evmtest.TestDeps
	world   laneRulesPropertyWorld
	rng     *rand.Rand
	nextSeq map[string]uint64
	txs     [][]byte
	reject  map[string]bool
}

func pick[T any](rng *rand.Rand, values []T) T {
	return values[rng.IntN(len(values))]
}

// add encodes a tx from sender carrying msgs, optionally paid by feePayer,
// with the next sequence of each tx-level signer. Some txs get an
// out-of-sequence number, a large memo, or a verifier rejection.
func (c *laneRulesCandidates) add(
	feePayer string,
	msgs ...*wasm.MsgExecuteContract,
) {
	signers := []string{msgs[0].Sender}
	if feePayer != "" && feePayer != msgs[0].Sender {
		signers = append(signers, feePayer)
	}
	outOfSequence := c.rng.IntN(20) == 0
	seqs := make([]uint64, len(signers))
	for i, signer := range signers {
		seqs[i] = c.nextSeq[signer]
		if outOfSequence {
			seqs[i] += 3
		} else {
			c.nextSeq[signer]++
		}
	}

	sdkMsgs := make([]sdk.Msg, 0, len(msgs))
	for _, msg := range msgs {
		sdkMsgs = append(sdkMsgs, msg)
	}
	builder := c.deps.App.GetTxConfig().NewTxBuilder()
	require.NoError(c.t, builder.SetMsgs(sdkMsgs...))
	builder.SetGasLimit(uint64(1+c.rng.IntN(4)) * 50_000_000)
	if feePayer != "" {
		builder.SetFeePayer(sdk.MustAccAddressFromBech32(feePayer))
	}
	if c.rng.IntN(10) == 0 {
		builder.SetMemo(strings.Repeat("m", 300+c.rng.IntN(3_000)))
	} else {
		builder.SetMemo(strconv.Itoa(len(c.txs)))
	}
	sigs := make([]signingtypes.SignatureV2, 0, len(seqs))
	for i, seq := range seqs {
		sigs = append(sigs, signingtypes.SignatureV2{
			PubKey: c.world.pubKeys[i],
			Data: &signingtypes.SingleSignatureData{
				SignMode: signingtypes.SignMode_SIGN_MODE_DIRECT,
			},
			Sequence: seq,
		})
	}
	require.NoError(c.t, builder.SetSignatures(sigs...))
	txBytes, err := c.deps.App.GetTxConfig().TxEncoder()(builder.GetTx())
	require.NoError(c.t, err)
	if c.rng.IntN(15) == 0 {
		c.reject[string(txBytes)] = true
	}
	c.txs = append(c.txs, txBytes)
}

// generate appends n random candidates: price writes from several feeders,
// some co-signed by a fee payer; feeder and keeper calls outside the
// allowlist; keeper fills, some sponsored; trader and payer txs; and EVM txs,
// which PrepareProposal ignores in RequestPrepareProposal.Txs.
func (c *laneRulesCandidates) generate(n int, evmTxs [][]byte) {
	w := c.world
	maybePayer := func(oneIn int) string {
		if c.rng.IntN(oneIn) == 0 {
			return pick(c.rng, w.payers)
		}
		return ""
	}
	for range n {
		switch roll := c.rng.IntN(100); {
		case roll < 25:
			msg := pick(c.rng, []string{setPricesMsg, setPriceMsg})
			c.add(maybePayer(5),
				proposalWasmExec(pick(c.rng, w.feeders), w.oracle, msg))
		case roll < 38:
			c.add("", proposalWasmExec(
				pick(c.rng, w.feeders), w.other, `{"claim":{}}`))
		case roll < 53:
			c.add(maybePayer(4),
				proposalWasmExec(w.keeper, w.perp, triggerFillMsg))
		case roll < 60:
			c.add("", proposalWasmExec(w.keeper, w.other, `{"claim":{}}`))
		case roll < 65:
			feeder := pick(c.rng, w.feeders)
			c.add("",
				proposalWasmExec(feeder, w.oracle, setPricesMsg),
				proposalWasmExec(feeder, w.other, `{"claim":{}}`))
		case roll < 75:
			c.add("", proposalWasmExec(
				pick(c.rng, w.payers), w.perp, triggerFillMsg))
		case roll < 90:
			c.add(maybePayer(4), proposalWasmExec(
				pick(c.rng, w.traders), w.perp, triggerFillMsg))
		default:
			c.txs = append(c.txs, pick(c.rng, evmTxs))
		}
	}
}

// TestLaneRulesPrepareOutputPassesProcess is a seeded property test: for
// random candidate lists, budgets, and verifier outcomes, the block that
// [app.NewEVMPrepareProposalHandler] builds always passes the lane rules of
// [app.NewLaneOrderProcessProposalHandler]. Set NIBIRU_LANE_PROPERTY_ITERS to
// raise the iteration count and NIBIRU_LANE_PROPERTY_SEED to change the seed.
func TestLaneRulesPrepareOutputPassesProcess(t *testing.T) {
	deps := evmtest.NewTestDeps()
	account := evmstate.NewEmptyAccount()
	require.NoError(t, deps.EvmKeeper.SetAccount(
		deps.Ctx(), deps.Sender.EthAddr, *account))
	evmTxs := insertProposalEVMTransactions(t, &deps, 4)
	world := newLaneRulesPropertyWorld(&deps)
	activateLaneRules(t, &deps)

	seed := uint64(20261104)
	if raw := os.Getenv("NIBIRU_LANE_PROPERTY_SEED"); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 64)
		require.NoError(t, err)
		seed = parsed
	}
	rng := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
	decode := deps.App.GetTxConfig().TxDecoder()

	minEVMSize := int64(math.MaxInt64)
	for _, evmTx := range evmTxs {
		minEVMSize = min(minEVMSize,
			cmttypes.ComputeProtoSizeForTxs([]cmttypes.Tx{evmTx}))
	}

	iterations := laneRulesPropertyIterations(t)
	var withPriceWrite, withRejections int
	for iteration := range iterations {
		candidates := &laneRulesCandidates{
			t: t, deps: &deps, world: world, rng: rng,
			nextSeq: map[string]uint64{}, reject: map[string]bool{},
		}
		candidates.generate(4+rng.IntN(30), evmTxs)

		total := cmttypes.ComputeProtoSizeForTxs(
			cmttypes.ToTxs(append(append([][]byte{}, candidates.txs...), evmTxs...)))
		maxTxBytes := max(1, total*int64(10+rng.IntN(120))/100)
		blockMaxBytes := maxTxBytes + int64(rng.IntN(20_000))
		if rng.IntN(10) == 0 {
			blockMaxBytes = max(1, maxTxBytes*int64(50+rng.IntN(50))/100)
		}
		maxGas := int64(-1)
		if rng.IntN(3) == 0 {
			maxGas = int64(1+rng.IntN(15)) * 50_000_000
		}
		ctx := laneRulesCtx(&deps, blockMaxBytes, maxGas)

		verifier := &laneRulesPrepareVerifier{
			sequences: sequencePrepareVerifier{decode: decode, next: map[string]uint64{}},
			reject:    candidates.reject,
		}
		handler := app.NewEVMPrepareProposalHandler(deps.App, verifier)
		height := laneRulesDoneHeight + 1 + int64(iteration)
		response := handler(ctx, abci.RequestPrepareProposal{
			Height:     height,
			MaxTxBytes: maxTxBytes,
			Txs:        candidates.txs,
		})

		require.Truef(t,
			processLaneRules(ctx, &deps, height, response.Txs),
			"seed %d iteration %d: ProcessProposal rejected PrepareProposal output",
			seed, iteration)
		require.LessOrEqual(t,
			cmttypes.ComputeProtoSizeForTxs(cmttypes.ToTxs(response.Txs)), maxTxBytes)
		// The verifier accepts every EVM tx, so an empty proposal while one
		// fits means PrepareProposal's own lane check fell back to empty,
		// which would hide a construction bug from the assertion above.
		if minEVMSize <= maxTxBytes {
			require.NotEmptyf(t, response.Txs,
				"seed %d iteration %d: PrepareProposal fell back to empty", seed, iteration)
		}

		if containsPriceWrite(response.Txs, world.oracle) {
			withPriceWrite++
		}
		if len(candidates.reject) > 0 {
			withRejections++
		}
	}
	t.Logf("seed %d: %d iterations, %d blocks with a price write, %d with verifier rejections",
		seed, iterations, withPriceWrite, withRejections)
	require.Positive(t, withPriceWrite)
}

// containsPriceWrite reports whether any tx in txs carries a price write
// message to oracle. It scans raw bytes and is only a coverage statistic.
func containsPriceWrite(txs [][]byte, oracle string) bool {
	for _, txBytes := range txs {
		if strings.Contains(string(txBytes), oracle) &&
			strings.Contains(string(txBytes), "set_price") {
			return true
		}
	}
	return false
}

// TestLaneRulesUpgradeRegistered proves the lane rules are keyed on an
// upgrade the binary registers, so activation happens in the binary that runs
// the upgrade rather than in any binary that sees a plan of that name.
func TestLaneRulesUpgradeRegistered(t *testing.T) {
	var names []string
	for _, upgrade := range app.Upgrades {
		names = append(names, upgrade.UpgradeName)
	}
	require.Contains(t, names, app.LaneRulesUpgradeName)
}

// TestLaneRulesNodePolicy proves the app.toml "nibiru-bft.lane-rules"
// policy: "on" enforces without the upgrade, as on a localnet or a chain
// restarted from exported genesis, "off" stops enforcing after it, and
// "auto" follows the upgrade.
func TestLaneRulesNodePolicy(t *testing.T) {
	deps := evmtest.NewTestDeps()
	actors := setSaiProposalActors(&deps)
	traderTx := buildProposalWasmTx(t, &deps,
		proposalWasmExec(actors.trader, actors.perp, triggerFillMsg))
	setPrices := buildProposalWasmTx(t, &deps,
		proposalWasmExec(actors.feeder, actors.oracle, setPricesMsg))
	violating := [][]byte{traderTx, setPrices}
	ctx := laneRulesCtx(&deps, 1_000_000, -1)
	height := laneRulesDoneHeight + 1
	t.Cleanup(func() { deps.App.SetLaneRulesMode(srvconfig.LaneRulesAuto) })

	deps.App.SetLaneRulesMode(srvconfig.LaneRulesAuto)
	require.True(t, processLaneRules(ctx, &deps, height, violating))
	deps.App.SetLaneRulesMode(srvconfig.LaneRulesOn)
	require.False(t, processLaneRules(ctx, &deps, height, violating))

	activateLaneRules(t, &deps)
	deps.App.SetLaneRulesMode(srvconfig.LaneRulesAuto)
	require.False(t, processLaneRules(ctx, &deps, height, violating))
	deps.App.SetLaneRulesMode(srvconfig.LaneRulesOff)
	require.True(t, processLaneRules(ctx, &deps, height, violating))
}
