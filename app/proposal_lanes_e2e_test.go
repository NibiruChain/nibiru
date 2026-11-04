package app_test

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"testing"
	"time"

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
	xoracle "github.com/NibiruChain/nibiru/v2/x/oracle"
	"github.com/NibiruChain/nibiru/v2/x/sudo"
	wasm "github.com/NibiruChain/nibiru/v2/x/wasm/types"
)

// laneE2EAccount is a funded account with a real key, for txs that pass the
// app's own ante handler.
type laneE2EAccount struct {
	priv   cryptotypes.PrivKey
	addr   sdk.AccAddress
	accNum uint64
}

// newLaneE2EAccount creates and funds an account in the uncommitted state of
// deps.
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
// the given sequences, with the fee paid by feePayer when set. corrupt flips
// a byte of the first signature, so the app's ante handler rejects the tx.
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

// laneE2EFixture is a committed chain with two feeders and a keeper on the
// zero-gas allowlist, payers and traders off it, and the app's real
// PrepareProposal (BaseApp verifier, app ante handler) and stock
// ProcessProposal. With resolved set, a proxy-mode x-oracle adapter names
// the oracle contract; otherwise the oracle lane uses the message-key
// fallback.
type laneE2EFixture struct {
	deps                   evmtest.TestDeps
	feederA, feederB, keep laneE2EAccount
	payers, traders        []laneE2EAccount
	oracle, perp, other    string
}

func newLaneE2EFixture(t *testing.T, resolved bool) *laneE2EFixture {
	t.Helper()
	deps := evmtest.NewTestDeps()
	f := &laneE2EFixture{
		feederA: newLaneE2EAccount(t, &deps),
		feederB: newLaneE2EAccount(t, &deps),
		keep:    newLaneE2EAccount(t, &deps),
		payers: []laneE2EAccount{
			newLaneE2EAccount(t, &deps), newLaneE2EAccount(t, &deps),
		},
		traders: []laneE2EAccount{
			newLaneE2EAccount(t, &deps), newLaneE2EAccount(t, &deps),
		},
		oracle: evmtest.NewEthPrivAcc().NibiruAddr.String(),
		perp:   evmtest.NewEthPrivAcc().NibiruAddr.String(),
		other:  evmtest.NewEthPrivAcc().NibiruAddr.String(),
	}
	deps.App.SudoKeeper.ZeroGasActors.Set(deps.Ctx(), sudo.ZeroGasActors{
		Senders: []string{
			f.feederA.addr.String(), f.feederB.addr.String(), f.keep.addr.String(),
		},
		Contracts: []string{f.oracle, f.perp},
	})
	if resolved {
		adapter := instantiateXOracleAdapter(t, &deps,
			xoracle.XOracleAdapterProxyModeOf(f.oracle))
		setXOraclePlugin(t, &deps, adapter.String())
	}
	deps.App.Commit()
	f.deps = deps
	return f
}

// tx signs a single-signer Wasm execute from account at seq. A non-zero fee
// is paid in unibi.
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

// laneE2EBlockTime is the proposal time of every e2e request. CometBFT
// always sets one; the x-oracle adapter query fails without it.
var laneE2EBlockTime = time.Unix(1_800_000_000, 0).UTC()

func (f *laneE2EFixture) height() int64 {
	return f.deps.App.LastBlockHeight() + 1
}

// process runs the app's stock ProcessProposal on txs.
func (f *laneE2EFixture) process(txs ...[]byte) bool {
	return f.deps.App.ProcessProposal(abci.RequestProcessProposal{
		Height: f.height(),
		Time:   laneE2EBlockTime,
		Txs:    txs,
	}).IsAccepted()
}

// prepareAndProcess runs the app's PrepareProposal and then its
// ProcessProposal on the result.
func (f *laneE2EFixture) prepareAndProcess(
	maxTxBytes int64,
	txs ...[]byte,
) ([][]byte, bool) {
	prepared := f.deps.App.PrepareProposal(abci.RequestPrepareProposal{
		Height:     f.height(),
		Time:       laneE2EBlockTime,
		MaxTxBytes: maxTxBytes,
		Txs:        txs,
	})
	return prepared.Txs, f.process(prepared.Txs...)
}

// proposalTxsSize returns the protobuf-encoded size CometBFT counts for txs.
func proposalTxsSize(txs ...[]byte) int64 {
	return cmttypes.ComputeProtoSizeForTxs(cmttypes.ToTxs(txs))
}

// TestPrepareProposalLanesPassRealProcess proves the block the app's
// PrepareProposal builds, with the real ante handler as verifier, is accepted
// by the app's stock ProcessProposal, which replays ante in block order.
// Feeder A sends a paid call X at sequence 0 and its oracle call PA at
// sequence 1; feeder B sends PB; the keeper sends a fill. X is promoted ahead
// of PA into the oracle lane.
//
//   - All valid: X, PA, PB, then the fill.
//   - PA fails ante (bad signature): X was verified ahead of it and stays at
//     the front without PA. The block is valid; only placement changed.
//   - PA does not fit the priority byte cap: likewise, X stays at the front.
//   - Feeder B's expiration update leads the keeper's fill only when the Sai
//     oracle address is resolved from the x-oracle adapter.
//
// The stock handler rejects a bad signature and a sequence out of order, so
// acceptance is not vacuous.
func TestPrepareProposalLanesPassRealProcess(t *testing.T) {
	for _, resolved := range []bool{true, false} {
		t.Run(fmt.Sprintf("resolved=%t", resolved), func(t *testing.T) {
			f := newLaneE2EFixture(t, resolved)
			x := f.tx(t, f.feederA, 0, 1_000, false, f.other, `{"claim":{}}`)
			pa := f.tx(t, f.feederA, 1, 0, false, f.oracle, setPricesMsg)
			paBadSig := f.tx(t, f.feederA, 1, 0, true, f.oracle, setPricesMsg)
			pb := f.tx(t, f.feederB, 0, 0, false, f.oracle, setPriceMsg)
			fill := f.tx(t, f.keep, 0, 0, false, f.perp, triggerFillMsg)

			require.True(t, f.process(x, pa, pb, fill))
			require.False(t, f.process(x, paBadSig, pb, fill), "bad signature")
			require.False(t, f.process(pa, x, pb, fill), "sequence order")

			txs, accepted := f.prepareAndProcess(1_000_000, fill, x, pa, pb)
			require.Equal(t, [][]byte{x, pa, pb, fill}, txs, "all valid")
			require.True(t, accepted)

			txs, accepted = f.prepareAndProcess(1_000_000, fill, x, paBadSig, pb)
			require.Equal(t, [][]byte{x, pb, fill}, txs, "rejected oracle call")
			require.True(t, accepted)

			bigPA := f.tx(t, f.feederA, 1, 0, false, f.oracle,
				`{"set_prices":{"prices":[`+strings.Repeat(`[0,"1"],`, 200)+`[0,"1"]]}}`)
			txs, accepted = f.prepareAndProcess(
				4*proposalTxsSize(x, pb, fill), fill, x, bigPA, pb)
			require.Equal(t, [][]byte{x, pb, fill}, txs, "oracle call over the cap")
			require.True(t, accepted)

			// An expiration update is an oracle call only when the Sai
			// oracle address is resolved; the fallback admits price writes
			// alone.
			expiry := f.tx(t, f.feederB, 0, 0, false, f.oracle, updateExpirationMsg)
			txs, accepted = f.prepareAndProcess(1_000_000, fill, expiry)
			if resolved {
				require.Equal(t, [][]byte{expiry, fill}, txs, "resolved")
			} else {
				require.Equal(t, [][]byte{fill, expiry}, txs, "fallback")
			}
			require.True(t, accepted)
		})
	}
}

// laneRealCandidates generates one random candidate list of really signed
// txs from the accounts of a [laneE2EFixture].
type laneRealCandidates struct {
	t       *testing.T
	f       *laneE2EFixture
	rng     *rand.Rand
	nextSeq map[string]uint64
	txs     [][]byte
}

// add signs a tx carrying msgs, msgs[i] from senders[i] to contracts[i],
// with the fee paid by feePayer when set. Tx-level signers are the distinct
// senders in order, then feePayer when distinct. Some txs carry an
// out-of-sequence number or a corrupted signature, which the real ante
// handler rejects.
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

// generate appends n random candidates: oracle calls (price writes and
// expiration updates), some sponsored or batched across feeders; feeder
// price writes to the perp contract; feeder and keeper calls outside the
// allowlist; keeper fills, some sponsored or carrying another account's
// message; calls signed by three accounts; and trader txs.
func (c *laneRealCandidates) generate(n int) {
	f := c.f
	feeders := []laneE2EAccount{f.feederA, f.feederB}
	everyone := append(append(append([]laneE2EAccount{}, feeders...),
		f.payers...), append(append([]laneE2EAccount{}, f.traders...), f.keep)...)
	maybePayer := func(oneIn int) *laneE2EAccount {
		if c.rng.IntN(oneIn) == 0 {
			payer := pick(c.rng, f.payers)
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
		case roll < 26:
			s, ct, m := one(pick(c.rng, feeders), f.oracle, pick(c.rng,
				[]string{setPricesMsg, setPriceMsg, updateExpirationMsg}))
			c.add(maybePayer(6), s, ct, m)
		case roll < 29:
			c.add(nil, []laneE2EAccount{f.feederA, f.feederB},
				[]string{f.oracle, f.oracle}, []string{setPricesMsg, setPriceMsg})
		case roll < 33:
			s, ct, m := one(pick(c.rng, feeders), f.perp, setPricesMsg)
			c.add(nil, s, ct, m)
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
			s, ct, m := one(pick(c.rng, f.traders), f.perp, triggerFillMsg)
			c.add(maybePayer(4), s, ct, m)
		}
	}
}

// TestPrepareProposalLanesRealPropertyPassesProcess is a seeded property
// test over the app's real handlers and the key liveness property of the
// proposer-side lanes: for random candidate lists of really signed txs,
// budgets, and ante outcomes, under both the resolved Sai oracle and the
// message-key fallback, the block that PrepareProposal builds with the
// BaseApp verifier and the app ante handler is always accepted by the app's
// stock ProcessProposal. Set NIBIRU_LANE_REAL_PROPERTY_ITERS to raise the
// iteration count (per oracle mode) and NIBIRU_LANE_PROPERTY_SEED to change
// the seed.
func TestPrepareProposalLanesRealPropertyPassesProcess(t *testing.T) {
	iterations := lanePropertyEnvInt(t, "NIBIRU_LANE_REAL_PROPERTY_ITERS", 100)
	if testing.Short() {
		iterations = min(iterations, 20)
	}
	seed := lanePropertySeed(t)
	for _, resolved := range []bool{true, false} {
		t.Run(fmt.Sprintf("resolved=%t", resolved), func(t *testing.T) {
			f := newLaneE2EFixture(t, resolved)
			rng := rand.New(rand.NewPCG(seed, seed^0x5851f42d4c957f2d))
			var proposed, withOracle int
			for iteration := range iterations {
				candidates := &laneRealCandidates{
					t: t, f: f, rng: rng, nextSeq: map[string]uint64{},
				}
				candidates.generate(4 + rng.IntN(25))
				maxTxBytes := max(1,
					proposalTxsSize(candidates.txs...)*int64(10+rng.IntN(120))/100)
				if rng.IntN(15) == 0 {
					maxTxBytes = -1
				}
				txs, accepted := f.prepareAndProcess(maxTxBytes, candidates.txs...)
				require.Truef(t, accepted,
					"seed %d iteration %d: ProcessProposal rejected PrepareProposal output",
					seed, iteration)
				proposed += len(txs)
				if containsOracleCall(txs, f.oracle) {
					withOracle++
				}
			}
			t.Logf("seed %d: %d iterations, %d txs proposed, %d blocks with an oracle call",
				seed, iterations, proposed, withOracle)
			require.Positive(t, withOracle)
		})
	}
}

// containsOracleCall reports whether any tx in txs names the oracle
// contract. It scans raw bytes and is only a coverage statistic.
func containsOracleCall(txs [][]byte, oracle string) bool {
	for _, txBytes := range txs {
		if strings.Contains(string(txBytes), oracle) {
			return true
		}
	}
	return false
}
