package app_test

import (
	"math/rand/v2"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	abci "github.com/NibiruChain/nibiru/v2/bft/abci/types"
	cmttypes "github.com/NibiruChain/nibiru/v2/bft/types"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/evm/evmtest"
	clienttx "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/client/tx"
	cryptotypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/crypto/types"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	signingtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/tx/signing"
	authsigning "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth/signing"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/sudo"
	wasm "github.com/NibiruChain/nibiru/v2/x/wasm/types"
)

// laneE2EAccount is a funded account with a real key, for transactions that
// pass the app's own ante handler.
type laneE2EAccount struct {
	priv   cryptotypes.PrivKey
	addr   sdk.AccAddress
	accNum uint64
}

// newLaneE2EAccount creates and funds an account in the uncommitted genesis
// state of deps.
func newLaneE2EAccount(t *testing.T, deps *evmtest.TestDeps) laneE2EAccount {
	t.Helper()
	priv := evmtest.NewEthPrivAcc().PrivKey
	addr := sdk.AccAddress(priv.PubKey().Address())
	keeper := deps.App.AccountKeeper
	account := keeper.NewAccountWithAddress(deps.Ctx(), addr)
	keeper.SetAccount(deps.Ctx(), account)
	require.NoError(t, testapp.FundAccount(deps.App.BankKeeper, deps.Ctx(), addr,
		sdk.NewCoins(sdk.NewInt64Coin(appconst.DENOM_UNIBI, 1_000_000_000))))
	return laneE2EAccount{priv: priv, addr: addr, accNum: account.GetAccountNumber()}
}

// signLaneE2ETx encodes msgs signed by signers, in tx-level signer order, at
// the given sequences, with fee paid by feePayer when set. corrupt flips a
// byte of the first signature, so the app's ante handler rejects the tx.
func signLaneE2ETx(
	t *testing.T,
	deps *evmtest.TestDeps,
	feePayer *laneE2EAccount,
	fee sdk.Coins,
	signers []laneE2EAccount,
	seqs []uint64,
	corrupt bool,
	msgs ...sdk.Msg,
) []byte {
	t.Helper()
	txConfig := deps.App.GetTxConfig()
	signMode := txConfig.SignModeHandler().DefaultMode()
	builder := txConfig.NewTxBuilder()
	require.NoError(t, builder.SetMsgs(msgs...))
	builder.SetGasLimit(500_000)
	builder.SetFeeAmount(fee)
	if feePayer != nil {
		builder.SetFeePayer(feePayer.addr)
	}
	sigs := make([]signingtypes.SignatureV2, len(signers))
	for i, signer := range signers {
		sigs[i] = signingtypes.SignatureV2{
			PubKey:   signer.priv.PubKey(),
			Data:     &signingtypes.SingleSignatureData{SignMode: signMode},
			Sequence: seqs[i],
		}
	}
	require.NoError(t, builder.SetSignatures(sigs...))
	for i, signer := range signers {
		sig, err := clienttx.SignWithPrivKey(signMode, authsigning.SignerData{
			Address:       signer.addr.String(),
			ChainID:       eth.EIP155ChainID_Testnet,
			AccountNumber: signer.accNum,
			Sequence:      seqs[i],
		}, builder, signer.priv, txConfig, seqs[i])
		require.NoError(t, err)
		sigs[i] = sig
	}
	if corrupt {
		data := sigs[0].Data.(*signingtypes.SingleSignatureData)
		data.Signature[0] ^= 0xff
	}
	require.NoError(t, builder.SetSignatures(sigs...))
	txBytes, err := txConfig.TxEncoder()(builder.GetTx())
	require.NoError(t, err)
	return txBytes
}

// laneE2EFixture is a committed chain with the lane rules active, two
// feeders and a keeper on the zero-gas allowlist, and the app's real
// PrepareProposal and ProcessProposal handlers: the BaseApp verifier with the
// app ante handler, and the lane rules wrapped around the stock
// ProcessProposal handler.
type laneE2EFixture struct {
	deps                   evmtest.TestDeps
	feederA, feederB, keep laneE2EAccount
	oracle, perp, other    string
}

func newLaneE2EFixture(t *testing.T) *laneE2EFixture {
	t.Helper()
	deps := evmtest.NewTestDeps()
	f := &laneE2EFixture{
		feederA: newLaneE2EAccount(t, &deps),
		feederB: newLaneE2EAccount(t, &deps),
		keep:    newLaneE2EAccount(t, &deps),
		oracle:  evmtest.NewEthPrivAcc().NibiruAddr.String(),
		perp:    evmtest.NewEthPrivAcc().NibiruAddr.String(),
		other:   evmtest.NewEthPrivAcc().NibiruAddr.String(),
	}
	deps.App.SudoKeeper.ZeroGasActors.Set(deps.Ctx(), sudo.ZeroGasActors{
		Senders: []string{
			f.feederA.addr.String(), f.feederB.addr.String(), f.keep.addr.String(),
		},
		Contracts: []string{f.oracle, f.perp},
	})
	activateLaneRules(t, &deps)
	deps.App.Commit()
	f.deps = deps
	return f
}

// tx signs a single-signer tx from account at seq. A non-zero fee is paid
// in unibi.
func (f *laneE2EFixture) tx(
	t *testing.T,
	account laneE2EAccount,
	seq uint64,
	feeAmount int64,
	corrupt bool,
	contract, msg string,
) []byte {
	t.Helper()
	fee := sdk.NewCoins()
	if feeAmount > 0 {
		fee = sdk.NewCoins(sdk.NewInt64Coin(appconst.DENOM_UNIBI, feeAmount))
	}
	return signLaneE2ETx(t, &f.deps, nil, fee, []laneE2EAccount{account},
		[]uint64{seq}, corrupt, &wasm.MsgExecuteContract{
			Sender:   account.addr.String(),
			Contract: contract,
			Msg:      wasm.RawContractMessage(msg),
		})
}

// prepareAndProcess runs the app's PrepareProposal and then its
// ProcessProposal on the result, both at the first enforced height.
func (f *laneE2EFixture) prepareAndProcess(
	t *testing.T,
	maxTxBytes int64,
	txs ...[]byte,
) ([][]byte, bool) {
	t.Helper()
	height := laneRulesDoneHeight + 1
	prepared := f.deps.App.PrepareProposal(abci.RequestPrepareProposal{
		Height:     height,
		MaxTxBytes: maxTxBytes,
		Txs:        txs,
	})
	processed := f.deps.App.ProcessProposal(abci.RequestProcessProposal{
		Height: height,
		Txs:    prepared.Txs,
	})
	return prepared.Txs, processed.IsAccepted()
}

// TestLaneRulesPrepareOutputPassesRealProcess proves the block that the
// app's PrepareProposal builds with the real ante handler as verifier is
// accepted by the app's ProcessProposal, which runs the lane rules and then
// the stock handler's ante replay in block order. Feeder A sends a paid call
// X at sequence 0 and its price write PA at sequence 1; feeder B sends PB.
//
//   - All valid: B's price-write-only group comes first, then A's X and PA.
//   - PA fails ante (bad signature): X was verified ahead of it and stays,
//     after PB, so no price write follows it.
//   - PA does not fit the priority byte cap: X loses its promotion, moves
//     behind PB, and is verified there.
func TestLaneRulesPrepareOutputPassesRealProcess(t *testing.T) {
	f := newLaneE2EFixture(t)
	x := f.tx(t, f.feederA, 0, 1_000, false, f.other, `{"claim":{}}`)
	pa := f.tx(t, f.feederA, 1, 0, false, f.oracle, setPricesMsg)
	paBadSig := f.tx(t, f.feederA, 1, 0, true, f.oracle, setPricesMsg)
	pb := f.tx(t, f.feederB, 0, 0, false, f.oracle, setPriceMsg)
	fill := f.tx(t, f.keep, 0, 0, false, f.perp, triggerFillMsg)

	// The stock handler replays ante: a bad signature or a sequence out of
	// order rejects the block, as do the lane rules.
	require.True(t, f.process(pb, x, pa, fill))
	require.False(t, f.process(pb, x, paBadSig, fill), "bad signature")
	require.False(t, f.process(pb, pa, x, fill), "sequence order")
	require.False(t, f.process(x, pb, fill), "stranded call before price write")

	t.Run("all valid", func(t *testing.T) {
		txs, accepted := f.prepareAndProcess(t, 1_000_000, x, pa, pb, fill)
		require.Equal(t, [][]byte{pb, x, pa, fill}, txs)
		require.True(t, accepted)
	})

	t.Run("skipped price write after its predecessor", func(t *testing.T) {
		txs, accepted := f.prepareAndProcess(t, 1_000_000, x, paBadSig, pb, fill)
		require.Equal(t, [][]byte{pb, x, fill}, txs)
		require.True(t, accepted)
	})

	t.Run("price write over the priority cap", func(t *testing.T) {
		bigPA := f.tx(t, f.feederA, 1, 0, false, f.oracle,
			`{"set_prices":{"prices":[`+strings.Repeat(`[0,"1"],`, 200)+`[0,"1"]]}}`)
		fits := proposalTxsSize(x, pb, fill)
		txs, accepted := f.prepareAndProcess(t, 4*fits, x, bigPA, pb, fill)
		require.Equal(t, [][]byte{pb, x, fill}, txs)
		require.True(t, accepted)
	})

	t.Run("price write after a stranded predecessor is skipped", func(t *testing.T) {
		// Feeder B also has a promoted call before its price write, so both
		// feeders' groups are guarded. A's PA fails after X was verified,
		// so B's price write must not follow X: it is skipped, and B's call
		// stays without it.
		y := f.tx(t, f.feederB, 0, 1_000, false, f.other, `{"claim":{}}`)
		pbAfterY := f.tx(t, f.feederB, 1, 0, false, f.oracle, setPriceMsg)
		txs, accepted := f.prepareAndProcess(t, 1_000_000,
			x, paBadSig, y, pbAfterY, fill)
		require.Equal(t, [][]byte{x, y, fill}, txs)
		require.True(t, accepted)
		require.False(t, f.process(x, y, pbAfterY, fill),
			"B's price write after stranded X breaks the oracle-first rule")
	})
}

// proposalTxsSize returns the protobuf-encoded size CometBFT counts for txs.
func proposalTxsSize(txs ...[]byte) int64 {
	return cmttypes.ComputeProtoSizeForTxs(cmttypes.ToTxs(txs))
}

// process runs the app's ProcessProposal on txs at the first enforced
// height.
func (f *laneE2EFixture) process(txs ...[]byte) bool {
	return f.deps.App.ProcessProposal(abci.RequestProcessProposal{
		Height: laneRulesDoneHeight + 1,
		Txs:    txs,
	}).IsAccepted()
}

// laneRealPropertyIterations returns the iteration count for
// [TestLaneRulesRealPrepareOutputPassesRealProcess], raised with
// NIBIRU_LANE_REAL_PROPERTY_ITERS. Each iteration signs every candidate and
// runs the real ante handler in both handlers, so the default is lower than
// the stub property test's.
func laneRealPropertyIterations(t *testing.T) int {
	t.Helper()
	iterations := 150
	if raw := os.Getenv("NIBIRU_LANE_REAL_PROPERTY_ITERS"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		require.NoError(t, err)
		iterations = parsed
	}
	if testing.Short() {
		iterations = min(iterations, 30)
	}
	return iterations
}

// laneRealCandidates generates one random candidate list of really signed
// txs from the accounts of a [laneE2EFixture] plus payers and traders.
type laneRealCandidates struct {
	t       *testing.T
	f       *laneE2EFixture
	payers  []laneE2EAccount
	traders []laneE2EAccount
	rng     *rand.Rand
	nextSeq map[string]uint64
	txs     [][]byte
}

// add signs a tx carrying msgs from senders, msgs[i] from senders[i], with
// fee paid by feePayer when set. Tx-level signers are the distinct senders
// in order, then feePayer when distinct. Some txs carry an out-of-sequence
// number or a corrupted signature, which the real ante handler rejects.
func (c *laneRealCandidates) add(
	feePayer *laneE2EAccount,
	senders []laneE2EAccount,
	contracts, msgs []string,
) {
	var signers []laneE2EAccount
	seen := map[string]bool{}
	for _, sender := range senders {
		if !seen[sender.addr.String()] {
			seen[sender.addr.String()] = true
			signers = append(signers, sender)
		}
	}
	if feePayer != nil && !seen[feePayer.addr.String()] {
		signers = append(signers, *feePayer)
	}
	outOfSequence := c.rng.IntN(25) == 0
	seqs := make([]uint64, len(signers))
	for i, signer := range signers {
		seqs[i] = c.nextSeq[signer.addr.String()]
		if outOfSequence {
			seqs[i] += 2
		} else {
			c.nextSeq[signer.addr.String()]++
		}
	}
	sdkMsgs := make([]sdk.Msg, len(msgs))
	for i, msg := range msgs {
		sdkMsgs[i] = &wasm.MsgExecuteContract{
			Sender:   senders[i].addr.String(),
			Contract: contracts[i],
			Msg:      wasm.RawContractMessage(msg),
		}
	}
	fee := sdk.NewCoins()
	if c.rng.IntN(2) == 0 {
		fee = sdk.NewCoins(sdk.NewInt64Coin(appconst.DENOM_UNIBI,
			int64(1+c.rng.IntN(5_000))))
	}
	c.txs = append(c.txs, signLaneE2ETx(c.t, &c.f.deps, feePayer, fee,
		signers, seqs, c.rng.IntN(20) == 0, sdkMsgs...))
}

// generate appends n random candidates: price writes, some sponsored or
// batched across feeders; feeder and keeper calls outside the allowlist;
// keeper fills, some sponsored or carrying a trader's message; calls signed
// by three accounts; and trader txs.
func (c *laneRealCandidates) generate(n int) {
	f := c.f
	feeders := []laneE2EAccount{f.feederA, f.feederB}
	everyone := append(append(append([]laneE2EAccount{}, feeders...),
		c.payers...), append(c.traders, f.keep)...)
	maybePayer := func(oneIn int) *laneE2EAccount {
		if c.rng.IntN(oneIn) == 0 {
			payer := pick(c.rng, c.payers)
			return &payer
		}
		return nil
	}
	one := func(sender laneE2EAccount, contract, msg string) (
		[]laneE2EAccount, []string, []string,
	) {
		return []laneE2EAccount{sender}, []string{contract}, []string{msg}
	}
	for range n {
		switch roll := c.rng.IntN(100); {
		case roll < 28:
			s, ct, m := one(pick(c.rng, feeders), f.oracle,
				pick(c.rng, []string{setPricesMsg, setPriceMsg}))
			c.add(maybePayer(6), s, ct, m)
		case roll < 31:
			c.add(nil, []laneE2EAccount{f.feederA, f.feederB},
				[]string{f.oracle, f.oracle}, []string{setPricesMsg, setPriceMsg})
		case roll < 45:
			s, ct, m := one(pick(c.rng, feeders), f.other, `{"claim":{}}`)
			c.add(nil, s, ct, m)
		case roll < 58:
			s, ct, m := one(f.keep, f.perp, triggerFillMsg)
			c.add(maybePayer(4), s, ct, m)
		case roll < 63:
			c.add(maybePayer(3), []laneE2EAccount{f.keep, pick(c.rng, everyone)},
				[]string{f.perp, f.perp}, []string{triggerFillMsg, triggerFillMsg})
		case roll < 70:
			c.add(maybePayer(2),
				[]laneE2EAccount{pick(c.rng, everyone), pick(c.rng, everyone)},
				[]string{f.other, f.perp}, []string{`{"claim":{}}`, triggerFillMsg})
		default:
			s, ct, m := one(pick(c.rng, c.traders), f.perp, triggerFillMsg)
			c.add(maybePayer(4), s, ct, m)
		}
	}
}

// TestLaneRulesRealPrepareOutputPassesRealProcess is a seeded property test
// over the app's real handlers: for random candidate lists of really signed
// txs, budgets, and ante outcomes, the block that PrepareProposal builds with
// the BaseApp verifier and the app ante handler is accepted by the app's
// ProcessProposal, which runs the lane rules and then the stock handler's
// ante replay in block order. Set NIBIRU_LANE_REAL_PROPERTY_ITERS to raise the
// iteration count and NIBIRU_LANE_PROPERTY_SEED to change the seed.
func TestLaneRulesRealPrepareOutputPassesRealProcess(t *testing.T) {
	deps := evmtest.NewTestDeps()
	f := &laneE2EFixture{
		feederA: newLaneE2EAccount(t, &deps),
		feederB: newLaneE2EAccount(t, &deps),
		keep:    newLaneE2EAccount(t, &deps),
		oracle:  evmtest.NewEthPrivAcc().NibiruAddr.String(),
		perp:    evmtest.NewEthPrivAcc().NibiruAddr.String(),
		other:   evmtest.NewEthPrivAcc().NibiruAddr.String(),
	}
	payers := []laneE2EAccount{newLaneE2EAccount(t, &deps), newLaneE2EAccount(t, &deps)}
	traders := []laneE2EAccount{newLaneE2EAccount(t, &deps), newLaneE2EAccount(t, &deps)}
	deps.App.SudoKeeper.ZeroGasActors.Set(deps.Ctx(), sudo.ZeroGasActors{
		Senders: []string{
			f.feederA.addr.String(), f.feederB.addr.String(), f.keep.addr.String(),
		},
		Contracts: []string{f.oracle, f.perp},
	})
	activateLaneRules(t, &deps)
	deps.App.Commit()
	f.deps = deps

	seed := laneRulesPropertySeed(t)
	rng := rand.New(rand.NewPCG(seed, seed^0x5851f42d4c957f2d))
	iterations := laneRealPropertyIterations(t)
	var withPriceWrite, proposed int
	for iteration := range iterations {
		candidates := &laneRealCandidates{
			t: t, f: f, payers: payers, traders: traders, rng: rng,
			nextSeq: map[string]uint64{},
		}
		candidates.generate(4 + rng.IntN(25))
		total := proposalTxsSize(candidates.txs...)
		maxTxBytes := max(1, total*int64(10+rng.IntN(120))/100)
		if rng.IntN(15) == 0 {
			maxTxBytes = -1
		}
		txs, accepted := f.prepareAndProcess(t, maxTxBytes, candidates.txs...)
		require.Truef(t, accepted,
			"seed %d iteration %d: ProcessProposal rejected PrepareProposal output",
			seed, iteration)
		proposed += len(txs)
		if containsPriceWrite(txs, f.oracle) {
			withPriceWrite++
		}
	}
	t.Logf("seed %d: %d iterations, %d txs proposed, %d blocks with a price write",
		seed, iterations, proposed, withPriceWrite)
	require.Positive(t, withPriceWrite)
}
