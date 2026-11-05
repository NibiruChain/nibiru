package app_test

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	abci "github.com/NibiruChain/nibiru/v2/bft/abci/types"
	tmproto "github.com/NibiruChain/nibiru/v2/bft/proto/tendermint/types"
	cmttypes "github.com/NibiruChain/nibiru/v2/bft/types"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/app/ante"
	"github.com/NibiruChain/nibiru/v2/evm"
	"github.com/NibiruChain/nibiru/v2/evm/evmstate"
	"github.com/NibiruChain/nibiru/v2/evm/evmtest"
	cryptotypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/crypto/types"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	signingtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/tx/signing"
	authsigning "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth/signing"
	xoracle "github.com/NibiruChain/nibiru/v2/x/oracle"
	"github.com/NibiruChain/nibiru/v2/x/sudo"
	wasm "github.com/NibiruChain/nibiru/v2/x/wasm/types"
)

// laneChooser is the source of random choices for the lane test generators:
// a seeded [rand.Rand] in the property tests, fuzz input bytes in the fuzz
// target.
type laneChooser interface {
	IntN(n int) int
}

func pick[T any](choose laneChooser, values []T) T {
	return values[choose.IntN(len(values))]
}

// lanePropertyWorld is the account setup of the randomized lane tests:
// several feeders and a keeper on the zero-gas allowlist, payers and traders
// off it, a proxy-mode x-oracle adapter pointing at the oracle contract, and
// pubkeys reused across signatures to keep tx building cheap.
type lanePropertyWorld struct {
	feeders, payers, traders []string
	keeper                   string
	oracle, perp, other      string
	adapter                  string
	pubKeys                  []cryptotypes.PubKey
	evmTxs                   [][]byte
}

// newLanePropertyWorld sets up the world in deps, including four EVM txs from
// deps.Sender at nonces 0 to 3 in the app mempool.
func newLanePropertyWorld(t testing.TB, deps *evmtest.TestDeps) lanePropertyWorld {
	t.Helper()
	account := evmstate.NewEmptyAccount()
	require.NoError(t, deps.EvmKeeper.SetAccount(
		deps.Ctx(), deps.Sender.EthAddr, *account))
	newAddr := func() string { return evmtest.NewEthPrivAcc().NibiruAddr.String() }
	world := lanePropertyWorld{
		feeders: []string{newAddr(), newAddr(), newAddr()},
		payers:  []string{newAddr(), newAddr()},
		traders: []string{newAddr(), newAddr()},
		keeper:  newAddr(),
		oracle:  newAddr(),
		perp:    newAddr(),
		other:   newAddr(),
	}
	for range 8 {
		world.pubKeys = append(world.pubKeys, evmtest.NewEthPrivAcc().PrivKey.PubKey())
	}
	for nonce := range uint64(4) {
		world.evmTxs = append(world.evmTxs,
			insertProposalEVMTransaction(t, deps, nonce, 100))
	}
	world.adapter = instantiateXOracleAdapter(t, deps,
		xoracle.XOracleAdapterProxyModeOf(world.oracle)).String()
	deps.App.SudoKeeper.ZeroGasActors.Set(deps.Ctx(), world.zeroGasActors())
	return world
}

// zeroGasActors returns the world's x/sudo allowlist: the feeders and the
// keeper may send zero-gas calls to the oracle and perp contracts.
func (w lanePropertyWorld) zeroGasActors() sudo.ZeroGasActors {
	return sudo.ZeroGasActors{
		Senders:   append(append([]string{}, w.feeders...), w.keeper),
		Contracts: []string{w.oracle, w.perp},
	}
}

// laneCandidates generates one random CometBFT candidate list of unsigned
// txs whose signer infos carry account sequences, for
// [sequencePrepareVerifier].
type laneCandidates struct {
	t       testing.TB
	deps    *evmtest.TestDeps
	world   lanePropertyWorld
	choose  laneChooser
	nextSeq map[string]uint64
	txs     [][]byte
	reject  map[string]bool
}

func newLaneCandidates(
	t testing.TB, deps *evmtest.TestDeps, world lanePropertyWorld,
	choose laneChooser,
) *laneCandidates {
	return &laneCandidates{
		t: t, deps: deps, world: world, choose: choose,
		nextSeq: map[string]uint64{}, reject: map[string]bool{},
	}
}

// add encodes a tx carrying msgs, optionally paid by feePayer, with the next
// sequence of each tx-level signer: the distinct message senders in order,
// then feePayer when set and distinct. Some txs get an out-of-sequence
// number, a large memo, or a verifier rejection.
func (c *laneCandidates) add(feePayer string, msgs ...*wasm.MsgExecuteContract) {
	var signers []string
	for _, msg := range msgs {
		if !slices.Contains(signers, msg.Sender) {
			signers = append(signers, msg.Sender)
		}
	}
	if feePayer != "" && !slices.Contains(signers, feePayer) {
		signers = append(signers, feePayer)
	}
	outOfSequence := c.choose.IntN(20) == 0
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
	builder.SetGasLimit(uint64(1+c.choose.IntN(4)) * 50_000_000)
	if feePayer != "" {
		builder.SetFeePayer(sdk.MustAccAddressFromBech32(feePayer))
	}
	if c.choose.IntN(10) == 0 {
		builder.SetMemo(strings.Repeat("m", 300+c.choose.IntN(3_000)))
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
	if c.choose.IntN(15) == 0 {
		c.reject[string(txBytes)] = true
	}
	c.txs = append(c.txs, txBytes)
}

// generate appends n random candidates: oracle calls (price writes and
// expiration updates) from several feeders, some sponsored by a fee payer or
// batched with another feeder's call, which keeps them out of the oracle
// lane; feeder price writes to the perp contract, which only the message-key
// fallback places in the oracle lane; feeder and keeper calls outside the
// allowlist; keeper fills and oracle calls, some sponsored and some carrying
// a later message from another account; calls signed by three or more
// accounts; trader and payer txs; and EVM txs, which PrepareProposal ignores
// in RequestPrepareProposal.Txs.
func (c *laneCandidates) generate(n int) {
	w := c.world
	maybePayer := func(oneIn int) string {
		if c.choose.IntN(oneIn) == 0 {
			return pick(c.choose, w.payers)
		}
		return ""
	}
	anyone := func() string {
		return pick(c.choose, [][]string{w.feeders, w.payers, w.traders,
			{w.keeper}}[c.choose.IntN(4)])
	}
	oracleMsg := func() string {
		return pick(c.choose,
			[]string{setPricesMsg, setPriceMsg, setPricesMsg, updateExpirationMsg})
	}
	for range n {
		switch roll := c.choose.IntN(100); {
		case roll < 20:
			c.add(maybePayer(5),
				proposalWasmExec(pick(c.choose, w.feeders), w.oracle, oracleMsg()))
		case roll < 23:
			c.add("",
				proposalWasmExec(pick(c.choose, w.feeders), w.oracle, setPricesMsg),
				proposalWasmExec(pick(c.choose, w.feeders), w.oracle, oracleMsg()))
		case roll < 27:
			c.add("", proposalWasmExec(
				pick(c.choose, w.feeders), w.perp, setPricesMsg))
		case roll < 36:
			c.add("", proposalWasmExec(
				pick(c.choose, w.feeders), w.other, `{"claim":{}}`))
		case roll < 46:
			c.add(maybePayer(4), proposalWasmExec(w.keeper, w.perp, triggerFillMsg))
		case roll < 50:
			c.add(maybePayer(3),
				proposalWasmExec(w.keeper, w.perp, triggerFillMsg),
				proposalWasmExec(anyone(), w.perp, triggerFillMsg))
		case roll < 54:
			c.add("", proposalWasmExec(w.keeper, w.other, `{"claim":{}}`))
		case roll < 58:
			c.add(maybePayer(4),
				proposalWasmExec(w.keeper, w.oracle, updateExpirationMsg))
		case roll < 62:
			feeder := pick(c.choose, w.feeders)
			c.add("",
				proposalWasmExec(feeder, w.oracle, setPricesMsg),
				proposalWasmExec(feeder, w.other, `{"claim":{}}`))
		case roll < 67:
			c.add(maybePayer(2),
				proposalWasmExec(anyone(), w.other, `{"claim":{}}`),
				proposalWasmExec(anyone(), w.perp, triggerFillMsg),
				proposalWasmExec(anyone(), w.other, `{"claim":{}}`))
		case roll < 74:
			c.add("", proposalWasmExec(
				pick(c.choose, w.payers), w.perp, triggerFillMsg))
		case roll < 89:
			c.add(maybePayer(4), proposalWasmExec(
				pick(c.choose, w.traders), w.perp, triggerFillMsg))
		default:
			c.txs = append(c.txs, pick(c.choose, w.evmTxs))
		}
	}
}

// refLane is the test's own model of the proposal lanes, independent of the
// app's classifier, so a change to the app's rules shows up as a mismatch.
type refLane int

const (
	refLaneOracle  refLane = 1
	refLaneZeroGas refLane = 2
	refLaneEVM     refLane = 3
	refLaneDefault refLane = 4
)

// refTx is a candidate as the reference model sees it.
type refTx struct {
	class   refLane // lane before promotion
	lane    refLane // lane after promotion
	signers []string
}

// refProposalLanes classifies every decodable candidate by the lane rules of
// the proposer: a zero-gas actor tx is an oracle-lane tx when every message
// executes saiOracle (or, with saiOracle empty, carries a single-key
// `set_prices` or `set_price` message) and its first sender is its only
// tx-level signer. It then promotes, walking backward, each tx to the
// earliest lane of any later tx sharing a signer. EVM txs keep their lane.
func refProposalLanes(
	t testing.TB,
	decode sdk.TxDecoder,
	candidates [][]byte,
	zeroGasActors sudo.ZeroGasActors,
	saiOracle string,
) map[string]refTx {
	t.Helper()
	isPriceWrite := func(raw []byte) bool {
		var obj map[string]json.RawMessage
		if json.Unmarshal(raw, &obj) != nil || len(obj) != 1 {
			return false
		}
		_, set := obj["set_prices"]
		_, one := obj["set_price"]
		return set || one
	}
	type entry struct {
		key string
		tx  refTx
	}
	var entries []entry
	for _, txBytes := range candidates {
		tx, err := decode(txBytes)
		require.NoError(t, err)
		if evm.IsEthTx(tx) {
			continue
		}
		var signers []string
		for _, signer := range tx.(authsigning.SigVerifiableTx).GetSigners() {
			signers = append(signers, signer.String())
		}
		class := refLaneDefault
		if ante.IsZeroGasActorTx(tx, zeroGasActors) {
			class = refLaneOracle
			for _, msg := range tx.GetMsgs() {
				exec := msg.(*wasm.MsgExecuteContract)
				matches := exec.Contract == saiOracle
				if saiOracle == "" {
					matches = isPriceWrite(exec.Msg)
				}
				if !matches {
					class = refLaneZeroGas
				}
			}
			sender := tx.GetMsgs()[0].GetSigners()[0].String()
			if len(signers) != 1 || signers[0] != sender {
				class = refLaneZeroGas
			}
		}
		entries = append(entries, entry{string(txBytes),
			refTx{class: class, lane: class, signers: signers}})
	}
	earliest := map[string]refLane{}
	for i := len(entries) - 1; i >= 0; i-- {
		for _, signer := range entries[i].tx.signers {
			if lane, ok := earliest[signer]; ok && lane < entries[i].tx.lane {
				entries[i].tx.lane = lane
			}
		}
		for _, signer := range entries[i].tx.signers {
			earliest[signer] = entries[i].tx.lane
		}
	}
	out := make(map[string]refTx, len(entries))
	for _, e := range entries {
		out[e.key] = e.tx
	}
	return out
}

// laneCheck is one PrepareProposal run to check with [checkPreparedLanes].
type laneCheck struct {
	label         string
	decode        sdk.TxDecoder
	candidates    [][]byte
	evmTxs        [][]byte
	zeroGasActors sudo.ZeroGasActors
	saiOracle     string
	maxTxBytes    int64
	maxGas        int64
	proposed      [][]byte
	verified      [][]byte
}

// laneCheckStats counts how often a property run met the cases the
// invariants are about, so a generator change cannot make them vacuous.
type laneCheckStats struct {
	withOracleTx, withPromotion, withOrphanedPromotion int
}

// checkPreparedLanes asserts the invariants of a PrepareProposal output:
//
//  1. Replaying account sequences in block order accepts every non-EVM tx,
//     as the stock ProcessProposal's ante replay requires.
//  2. The txs PrepareProposal verified are exactly the block, in block order.
//  3. Lanes are in order: the non-EVM txs' promoted reference lanes never
//     decrease along the block, and every EVM tx follows every oracle and
//     zero-gas tx and precedes every default tx. In particular, every tx
//     before the last oracle-lane tx is an oracle-lane tx or a predecessor
//     promoted into the oracle lane by a later tx sharing a signer.
//  4. The block fits MaxTxBytes and max gas, and the oracle and zero-gas
//     lanes together use at most MaxTxBytes / 4.
//
// Invariant 3 does not require that a promoted predecessor's oracle-lane
// tx is in the block. Promotion runs over all candidates, so when the
// oracle-lane tx that promoted a predecessor is then rejected or does not
// fit, the predecessor still leads the block. Without lane enforcement in
// ProcessProposal that is a placement choice, not a validity problem: the
// predecessor still precedes its signer's later txs, so the block passes
// the stock checks. The stats count such orphaned promotions.
func checkPreparedLanes(t testing.TB, c laneCheck, stats *laneCheckStats) {
	t.Helper()
	ref := refProposalLanes(t, c.decode, c.candidates, c.zeroGasActors, c.saiOracle)

	replay := sequencePrepareVerifier{decode: c.decode, next: map[string]uint64{}}
	for idx, txBytes := range c.proposed {
		require.NoErrorf(t, replay.PrepareProposalVerifyTxBytes(txBytes),
			"%s: tx %d fails the sequence replay in block order", c.label, idx)
	}
	require.Equalf(t,
		append([][]byte{}, c.proposed...), append([][]byte{}, c.verified...),
		"%s: verification order differs from block order", c.label)

	isEVM := func(txBytes []byte) bool {
		return slices.ContainsFunc(c.evmTxs, func(evmTx []byte) bool {
			return string(evmTx) == string(txBytes)
		})
	}
	lastOracle, sawEVM := -1, false
	maxLane := refLaneOracle
	var priorityTxs [][]byte
	var gas uint64
	for idx, txBytes := range c.proposed {
		tx, err := c.decode(txBytes)
		require.NoError(t, err)
		gas += tx.(interface{ GetGas() uint64 }).GetGas()
		if isEVM(txBytes) {
			require.LessOrEqualf(t, maxLane, refLaneEVM,
				"%s: EVM tx %d after a default-lane tx", c.label, idx)
			sawEVM = true
			maxLane = max(maxLane, refLaneEVM)
			continue
		}
		r, ok := ref[string(txBytes)]
		require.Truef(t, ok, "%s: tx %d is not a candidate", c.label, idx)
		require.GreaterOrEqualf(t, r.lane, maxLane,
			"%s: tx %d in lane %d after lane %d", c.label, idx, r.lane, maxLane)
		maxLane = r.lane
		if r.lane <= refLaneZeroGas {
			require.Falsef(t, sawEVM, "%s: priority tx %d after an EVM tx",
				c.label, idx)
			priorityTxs = append(priorityTxs, txBytes)
		}
		if r.class == refLaneOracle {
			lastOracle = idx
		}
	}
	for idx := range lastOracle {
		if isEVM(c.proposed[idx]) {
			t.Fatalf("%s: EVM tx %d before the last oracle-lane tx", c.label, idx)
		}
		require.Equalf(t, refLaneOracle, ref[string(c.proposed[idx])].lane,
			"%s: tx %d precedes the last oracle-lane tx without promotion",
			c.label, idx)
	}

	if c.maxTxBytes >= 0 {
		require.LessOrEqual(t,
			cmttypes.ComputeProtoSizeForTxs(cmttypes.ToTxs(c.proposed)), c.maxTxBytes)
		require.LessOrEqualf(t,
			cmttypes.ComputeProtoSizeForTxs(cmttypes.ToTxs(priorityTxs)),
			c.maxTxBytes/4, "%s: priority lanes over MaxTxBytes/4", c.label)
	}
	if c.maxGas >= 0 {
		require.LessOrEqual(t, gas, uint64(c.maxGas))
	}

	if lastOracle >= 0 {
		stats.withOracleTx++
	}
	promoted, orphaned := false, false
	for idx := range lastOracle {
		r := ref[string(c.proposed[idx])]
		if r.class != refLaneOracle {
			promoted = true
		}
	}
	// A promoted predecessor is orphaned when no later oracle-lane tx in the
	// block shares a signer with it.
	for idx, txBytes := range c.proposed {
		r, ok := ref[string(txBytes)]
		if !ok || r.lane != refLaneOracle || r.class == refLaneOracle {
			continue
		}
		anchored := false
		for _, later := range c.proposed[idx+1:] {
			lr, ok := ref[string(later)]
			if ok && lr.class == refLaneOracle &&
				slices.ContainsFunc(lr.signers, func(s string) bool {
					return slices.Contains(r.signers, s)
				}) {
				anchored = true
			}
		}
		orphaned = orphaned || !anchored
	}
	if promoted {
		stats.withPromotion++
	}
	if orphaned {
		stats.withOrphanedPromotion++
	}
}

// laneVerifier is [sequencePrepareVerifier] that also rejects listed txs, as
// ante does for a bad signature or insufficient funds, and records accepted
// txs in order. A rejected tx advances no sequence.
type laneVerifier struct {
	sequences sequencePrepareVerifier
	reject    map[string]bool
	accepted  [][]byte
}

func (v *laneVerifier) PrepareProposalVerifyTxBytes(txBytes []byte) error {
	if v.reject[string(txBytes)] {
		return fmt.Errorf("rejected by test verifier")
	}
	if err := v.sequences.PrepareProposalVerifyTxBytes(txBytes); err != nil {
		return err
	}
	v.accepted = append(v.accepted, txBytes)
	return nil
}

// runLaneCase draws budgets, the allowlist, and the oracle mode from choose,
// generates n candidates, runs PrepareProposal with a [laneVerifier], and
// checks the output with [checkPreparedLanes].
func runLaneCase(
	t testing.TB,
	deps *evmtest.TestDeps,
	world lanePropertyWorld,
	choose laneChooser,
	n int,
	label string,
	stats *laneCheckStats,
) {
	t.Helper()
	candidates := newLaneCandidates(t, deps, world, choose)
	candidates.generate(n)

	// An allowlist without contracts disables zero-gas classification.
	zeroGasActors := world.zeroGasActors()
	if choose.IntN(20) == 0 {
		zeroGasActors.Contracts = nil
	}
	deps.App.SudoKeeper.ZeroGasActors.Set(deps.Ctx(), zeroGasActors)
	saiOracle := ""
	if choose.IntN(2) == 0 {
		setXOraclePlugin(t, deps, world.adapter)
		saiOracle = world.oracle
	} else {
		setXOraclePlugin(t, deps, "")
	}

	total := cmttypes.ComputeProtoSizeForTxs(cmttypes.ToTxs(
		append(append([][]byte{}, candidates.txs...), world.evmTxs...)))
	maxTxBytes := max(1, total*int64(10+choose.IntN(120))/100)
	if choose.IntN(15) == 0 {
		maxTxBytes = -1
	}
	maxGas := int64(-1)
	if choose.IntN(3) == 0 {
		maxGas = int64(1+choose.IntN(15)) * 50_000_000
	}
	ctx := deps.Ctx().WithConsensusParams(&tmproto.ConsensusParams{
		Block: &tmproto.BlockParams{MaxBytes: 22_020_096, MaxGas: maxGas},
	})

	decode := deps.App.GetTxConfig().TxDecoder()
	verifier := &laneVerifier{
		sequences: sequencePrepareVerifier{decode: decode, next: map[string]uint64{}},
		reject:    candidates.reject,
	}
	response := app.NewEVMPrepareProposalHandler(deps.App, verifier)(ctx,
		abci.RequestPrepareProposal{MaxTxBytes: maxTxBytes, Txs: candidates.txs})

	// The verifier accepts every EVM tx, so an empty block while the first
	// EVM tx fits means the handler recovered a panic.
	firstEVM := cmttypes.ComputeProtoSizeForTxs(cmttypes.ToTxs(world.evmTxs[:1]))
	if maxTxBytes < 0 || firstEVM <= maxTxBytes {
		require.NotEmptyf(t, response.Txs, "%s: empty proposal", label)
	}
	checkPreparedLanes(t, laneCheck{
		label:         label,
		decode:        decode,
		candidates:    candidates.txs,
		evmTxs:        world.evmTxs,
		zeroGasActors: zeroGasActors,
		saiOracle:     saiOracle,
		maxTxBytes:    maxTxBytes,
		maxGas:        maxGas,
		proposed:      response.Txs,
		verified:      verifier.accepted,
	}, stats)
}

// lanePropertyEnvInt reads a positive integer test knob from the
// environment, or returns def.
func lanePropertyEnvInt(t *testing.T, name string, def int) int {
	t.Helper()
	raw := os.Getenv(name)
	if raw == "" {
		return def
	}
	parsed, err := strconv.Atoi(raw)
	require.NoError(t, err)
	return parsed
}

// lanePropertySeed returns the seed of the randomized lane tests, set with
// NIBIRU_LANE_PROPERTY_SEED.
func lanePropertySeed(t *testing.T) uint64 {
	t.Helper()
	seed := uint64(20261104)
	if raw := os.Getenv("NIBIRU_LANE_PROPERTY_SEED"); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 64)
		require.NoError(t, err)
		seed = parsed
	}
	return seed
}

// TestPrepareProposalLaneProperties is a seeded property test of
// [app.NewEVMPrepareProposalHandler]: for random candidate lists, budgets,
// allowlists, oracle modes (resolved Sai oracle or message-key fallback),
// and verifier outcomes, the output satisfies [checkPreparedLanes]. Set
// NIBIRU_LANE_PROPERTY_ITERS to raise the iteration count and
// NIBIRU_LANE_PROPERTY_SEED to change the seed.
func TestPrepareProposalLaneProperties(t *testing.T) {
	deps := evmtest.NewTestDeps()
	world := newLanePropertyWorld(t, &deps)
	seed := lanePropertySeed(t)
	rng := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
	iterations := lanePropertyEnvInt(t, "NIBIRU_LANE_PROPERTY_ITERS", 2_000)
	if testing.Short() {
		iterations = min(iterations, 200)
	}
	var stats laneCheckStats
	for iteration := range iterations {
		runLaneCase(t, &deps, world, rng, 4+rng.IntN(30),
			fmt.Sprintf("seed %d iteration %d", seed, iteration), &stats)
	}
	t.Logf("seed %d: %d iterations, %d with an oracle-lane tx, %d with a "+
		"promoted predecessor, %d with an orphaned promotion",
		seed, iterations, stats.withOracleTx, stats.withPromotion,
		stats.withOrphanedPromotion)
	require.Positive(t, stats.withOracleTx)
	require.Positive(t, stats.withPromotion)
}
