package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/NibiruChain/nibiru/v2/app/ante"
	"github.com/NibiruChain/nibiru/v2/evm"
	"github.com/NibiruChain/nibiru/v2/evm/evmstate"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	authsigning "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth/signing"
	"github.com/NibiruChain/nibiru/v2/x/collections"
	xoracle "github.com/NibiruChain/nibiru/v2/x/oracle"
	"github.com/NibiruChain/nibiru/v2/x/sudo"
	wasmkeeper "github.com/NibiruChain/nibiru/v2/x/wasm/keeper"
	wasm "github.com/NibiruChain/nibiru/v2/x/wasm/types"
)

// proposalLane is the block section a transaction is proposed in. A lower
// value is proposed earlier: every selected transaction of a lane precedes
// every selected transaction of a later lane. See [classifyProposalTx] and
// [splitNonEVMProposalTxs].
//
// Values are explicit and start at 1, so the zero value is no lane, and
// promotion compares lanes by value. Keep them in proposal order.
type proposalLane uint8

const (
	// proposalLaneOracle holds zero-gas actor transactions that call the Sai
	// oracle contract ([isOracleLaneTx]): every message executes the oracle,
	// and the first message's sender is the only tx-level signer. It also
	// holds earlier transactions sharing a signer with them, which
	// [splitNonEVMProposalTxs] promotes ahead of them.
	proposalLaneOracle proposalLane = 1
	// proposalLaneZeroGas holds the remaining zero-gas actor transactions
	// ([ante.IsZeroGasActorTx]), such as keeper and admin calls.
	proposalLaneZeroGas proposalLane = 2
	// proposalLaneEVM holds Ethereum transactions. PrepareProposal fills it
	// from [evm.Mempool], not from RequestPrepareProposal.Txs.
	proposalLaneEVM proposalLane = 3
	// proposalLaneDefault holds every other non-EVM transaction.
	proposalLaneDefault proposalLane = 4
)

// classifyProposalTx assigns tx to its [proposalLane]. zeroGasActors is the
// x/sudo allowlist and oracle the [oracleLaneRule], both from the proposal's
// state.
func classifyProposalTx(
	tx sdk.Tx,
	zeroGasActors sudo.ZeroGasActors,
	oracle oracleLaneRule,
) proposalLane {
	switch {
	case evm.IsEthTx(tx):
		return proposalLaneEVM
	case !ante.IsZeroGasActorTx(tx, zeroGasActors):
		return proposalLaneDefault
	case isOracleLaneTx(tx, oracle):
		return proposalLaneOracle
	default:
		return proposalLaneZeroGas
	}
}

// oracleLaneRule identifies calls to the Sai oracle contract for
// [proposalLaneOracle]. PrepareProposal builds it once per proposal with
// [resolveOracleLaneRule].
type oracleLaneRule struct {
	// saiOracle is the bech32 address of the Sai oracle contract, as the
	// x-oracle adapter reports it. Empty selects the fallback: a message
	// counts as an oracle call when it is a price write by its JSON key
	// ([isOraclePriceWriteMsg]), whatever contract it executes.
	saiOracle string
}

// matches reports whether msg counts as a call to the Sai oracle: it executes
// saiOracle, or, under the fallback, it carries a price write.
func (r oracleLaneRule) matches(msg *wasm.MsgExecuteContract) bool {
	if r.saiOracle != "" {
		return msg.Contract == r.saiOracle
	}
	return isOraclePriceWriteMsg(msg.Msg)
}

// isOracleLaneTx reports whether tx belongs in [proposalLaneOracle]: it has at
// least one message, every message is a Wasm execute that oracle matches
// ([oracleLaneRule.matches]), and its only tx-level signer is the first
// message's sender. Any execute of the oracle contract qualifies, including
// price writes, expiration updates, and permission or admin calls. A call
// with a co-signer, a message from another sender, or a distinct fee payer
// does not; a fee payer equal to the sender adds no signer. Such a call is
// classified by the remaining rules, normally into [proposalLaneZeroGas].
//
// The sole-signer condition keeps [splitNonEVMProposalTxs] from following an
// oracle call to accounts other than its sender. Without it, any account that
// co-signs or pays for an oracle call would have its earlier transactions,
// and transitively those of every account sharing a transaction with it,
// proposed ahead of the block's oracle calls.
func isOracleLaneTx(tx sdk.Tx, oracle oracleLaneRule) bool {
	msgs := tx.GetMsgs()
	if len(msgs) == 0 {
		return false
	}
	for _, msg := range msgs {
		msgExec, ok := msg.(*wasm.MsgExecuteContract)
		if !ok || !oracle.matches(msgExec) {
			return false
		}
	}
	senders := msgs[0].GetSigners()
	signers := proposalTxSigners(tx)
	return len(senders) == 1 && len(signers) == 1 &&
		signers[0].Equals(senders[0])
}

// proposalTxSigners returns the tx-level signers of a non-EVM transaction:
// the message signers plus the fee payer, whose sequences ante checks. It
// panics, through GetSigners, on an invalid bech32 signer or fee payer.
func proposalTxSigners(tx sdk.Tx) (signers []sdk.AccAddress) {
	// Ante checks and increments the sequence of every tx-level signer,
	// which includes a fee payer that signs no message.
	if sigTx, isSigTx := tx.(authsigning.SigVerifiableTx); isSigTx {
		return sigTx.GetSigners()
	}
	for _, msg := range tx.GetMsgs() {
		signers = append(signers, msg.GetSigners()...)
	}
	return signers
}

// oraclePriceWriteMsgKeys are the externally tagged (snake_case serde)
// execute variants with which the Sai oracle contract records prices.
var oraclePriceWriteMsgKeys = map[string]struct{}{
	"set_prices": {},
	"set_price":  {},
}

// isOraclePriceWriteMsg reports whether a Wasm execute message is a JSON
// object with exactly one top-level key naming an oracle price write. It is
// the fallback of [oracleLaneRule] when the Sai oracle address cannot be read
// from state, as on a localnet or in a test without the x-oracle adapter: it
// matches the message rather than a contract address, so it needs no
// configuration, but it admits only price writes, to any contract. Malformed
// JSON, trailing data, or extra keys return false. The variant payload is not
// validated; the contract does that on execution.
func isOraclePriceWriteMsg(raw []byte) bool {
	dec := json.NewDecoder(bytes.NewReader(raw))
	if tok, err := dec.Token(); err != nil || tok != json.Delim('{') {
		return false
	}
	keyTok, err := dec.Token()
	if err != nil {
		return false
	}
	key, ok := keyTok.(string)
	if !ok {
		return false
	}
	if _, ok := oraclePriceWriteMsgKeys[key]; !ok {
		return false
	}
	var value json.RawMessage
	if err := dec.Decode(&value); err != nil {
		return false
	}
	if tok, err := dec.Token(); err != nil || tok != json.Delim('}') {
		return false
	}
	_, err = dec.Token()
	return err == io.EOF
}

// nonEVMProposalTx is a decoded non-EVM candidate from
// RequestPrepareProposal.Txs with its original outer bytes.
type nonEVMProposalTx struct {
	txBytes   []byte
	gasWanted uint64
}

// splitNonEVMProposalTxs decodes the CometBFT candidates once and groups the
// non-EVM transactions by [classifyProposalTx], preserving relative order
// within each lane. Undecodable and EVM candidates are dropped, so the
// map has no [proposalLaneEVM] entry. A candidate whose classification
// panics is also dropped ([classifyProposalCandidate]).
//
// Lanes never reorder one signer's transactions, fee payers included, since
// ante checks every tx-level signer's account sequence. Instead of pushing a
// later transaction back, the split promotes earlier ones: walking the
// candidates backward, each moves up to the earliest lane of any later
// candidate that shares a signer with it. A fast-lane transaction therefore
// keeps its lane, and same-signer transactions that precede it ride ahead of
// it. A feeder that sends any tx at sequence N before its price write at N+1
// keeps the price write in [proposalLaneOracle], with the earlier tx first.
//
// Promotion follows every shared signer, so a co-signer or fee payer of a
// fast-lane transaction also has its earlier transactions promoted. That
// requires an allowlisted sender to sign a transaction naming that account,
// and it is preferred to delaying the allowlisted sender's transaction behind
// EVM. An oracle-lane transaction has its sender as its only signer
// ([isOracleLaneTx]), so promotion into [proposalLaneOracle] reaches only the
// earlier transactions of oracle callers and of accounts that share an
// earlier transaction with them.
//
// Promotion runs over all candidates, before verification and the byte and
// gas budgets skip any of them. A transaction promoted for an oracle call that
// is then skipped keeps its oracle-lane position. The block stays valid, since
// the transaction still precedes its signer's later transactions and
// ProcessProposal does not check lane order; the transaction merely leads the
// block without the oracle call that justified it.
//
// After the backward pass, lanes are non-decreasing along every signer's
// transactions in candidate order, so no transaction at sequence N+1 is
// verified ahead of the same account's transaction at N.
func splitNonEVMProposalTxs(
	decode sdk.TxDecoder,
	candidates [][]byte,
	zeroGasActors sudo.ZeroGasActors,
	oracle oracleLaneRule,
) map[proposalLane][]nonEVMProposalTx {
	type classified struct {
		candidate nonEVMProposalTx
		lane      proposalLane
		signers   []string
	}
	txs := make([]classified, 0, len(candidates))
	for _, txBytes := range candidates {
		tx, err := decode(txBytes)
		if err != nil {
			continue
		}
		lane, signers, ok := classifyProposalCandidate(tx, zeroGasActors, oracle)
		if !ok || lane == proposalLaneEVM {
			continue
		}
		item := classified{
			candidate: nonEVMProposalTx{txBytes: txBytes},
			lane:      lane,
			signers:   make([]string, len(signers)),
		}
		for i, signer := range signers {
			item.signers[i] = signer.String()
		}
		if gasTx, ok := tx.(interface{ GetGas() uint64 }); ok {
			item.candidate.gasWanted = gasTx.GetGas()
		}
		txs = append(txs, item)
	}

	// earliestLater maps a signer to the lane of its nearest later
	// candidate, which after promotion is the earliest lane of all of them.
	earliestLater := make(map[string]proposalLane)
	for i := len(txs) - 1; i >= 0; i-- {
		for _, signer := range txs[i].signers {
			if next, seen := earliestLater[signer]; seen && next < txs[i].lane {
				txs[i].lane = next
			}
		}
		for _, signer := range txs[i].signers {
			earliestLater[signer] = txs[i].lane
		}
	}
	lanes := make(map[proposalLane][]nonEVMProposalTx)
	for _, item := range txs {
		lanes[item.lane] = append(lanes[item.lane], item.candidate)
	}
	return lanes
}

// classifyProposalCandidate returns the [classifyProposalTx] lane of a decoded
// but unverified candidate and, for non-EVM transactions, its tx-level
// signers: the message signers plus the fee payer, whose sequences ante
// checks. ok is false when classification panics, for example when
// GetSigners meets an invalid bech32 sender. Such a candidate cannot pass
// signature verification, so it is dropped rather than placed in a lane; the
// recover keeps one malformed candidate from reaching the handler-level
// recover, which would empty the whole proposal.
func classifyProposalCandidate(
	tx sdk.Tx,
	zeroGasActors sudo.ZeroGasActors,
	oracle oracleLaneRule,
) (lane proposalLane, signers []sdk.AccAddress, ok bool) {
	defer func() {
		if recover() != nil {
			lane, signers, ok = proposalLaneDefault, nil, false
		}
	}()
	lane = classifyProposalTx(tx, zeroGasActors, oracle)
	if lane == proposalLaneEVM {
		return lane, nil, true
	}
	return lane, proposalTxSigners(tx), true
}

// oracleLaneQueryGasLimit bounds the gas of the x-oracle adapter config query
// in [resolveOracleLaneRule], in SDK gas. It matches the x-consensus params
// query limit (consensus QueryGasLimit), which is sized for one storage read
// plus the fixed contract instance cost with ample headroom.
const oracleLaneQueryGasLimit uint64 = 1_000_000

// ErrOracleAdapterNotConfigured reports that state names no x-oracle adapter
// or that the adapter runs in fixture mode, so there is no Sai oracle address
// to match. It selects the [oracleLaneRule] fallback without an error log.
var ErrOracleAdapterNotConfigured = errors.New("x-oracle adapter has no Sai oracle")

// ResolveSaiOracleAddr returns the Sai oracle address that PrepareProposal
// uses for [proposalLaneOracle], read from the committed state in ctx. It is
// empty when the lane uses the fallback; the error says why, as described in
// [resolveOracleLaneRule].
func (app *NibiruApp) ResolveSaiOracleAddr(ctx sdk.Context) (string, error) {
	rule, err := resolveOracleLaneRule(ctx, app.EvmKeeper, app.WasmKeeper)
	return rule.saiOracle, err
}

// resolveOracleLaneRule returns the [oracleLaneRule] for one proposal from
// the committed state in ctx. It reads the Sai oracle address in two steps:
//
//  1. The EVM params Wasm plugin [evm.WasmPluginNameXOracle] names the
//     x-oracle adapter contract.
//  2. A smart query `{"config":{}}` to the adapter returns its mode; in proxy
//     mode, mode.proxy.sai_oracle_addr is the Sai oracle contract.
//
// The query runs through [wasmkeeper.Keeper.QuerySmartBounded]: on a branch of
// ctx that is discarded, under its own gas meter limited to
// [oracleLaneQueryGasLimit], with panics turned into errors. It therefore
// consumes none of the block's gas, writes no state, and cannot panic out of
// PrepareProposal; a recover here also covers the plugin read.
//
// On any failure the rule falls back to matching price writes by message key
// ([isOraclePriceWriteMsg]), so a chain without the adapter, such as a
// localnet or a test app, keeps an oracle lane. The returned error says why:
// [ErrOracleAdapterNotConfigured] when the plugin is absent or the adapter is
// in fixture mode, and another error when the query or its response fails.
func resolveOracleLaneRule(
	ctx sdk.Context,
	evmKeeper *evmstate.Keeper,
	wasmKeeper wasmkeeper.Keeper,
) (rule oracleLaneRule, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			rule, err = oracleLaneRule{}, fmt.Errorf(
				"reading the Sai oracle address panicked: %v", recovered)
		}
	}()
	saiOracle, err := saiOracleAddr(ctx, evmKeeper, wasmKeeper)
	if err != nil {
		return oracleLaneRule{}, err
	}
	return oracleLaneRule{saiOracle: saiOracle.String()}, nil
}

// saiOracleAddr performs the lookup of [resolveOracleLaneRule].
func saiOracleAddr(
	ctx sdk.Context,
	evmKeeper *evmstate.Keeper,
	wasmKeeper wasmkeeper.Keeper,
) (sdk.AccAddress, error) {
	readCtx, _ := ctx.CacheContext()
	readCtx = readCtx.WithGasMeter(sdk.NewGasMeter(oracleLaneQueryGasLimit))
	adapter, err := evmKeeper.EVMState().WasmPlugins.Get(
		readCtx, evm.WasmPluginNameXOracle)
	if errors.Is(err, collections.ErrNotFound) {
		return nil, ErrOracleAdapterNotConfigured
	}
	if err != nil {
		return nil, fmt.Errorf("reading the x-oracle Wasm plugin: %w", err)
	}
	if adapter.Empty() {
		return nil, ErrOracleAdapterNotConfigured
	}

	req, err := json.Marshal(xoracle.XOracleAdapterQueryMsg{Config: &struct{}{}})
	if err != nil {
		return nil, err
	}
	respBz, _, err := wasmKeeper.QuerySmartBounded(
		ctx, adapter, req, oracleLaneQueryGasLimit)
	if err != nil {
		return nil, fmt.Errorf("querying the x-oracle adapter config: %w", err)
	}
	var resp struct {
		Mode xoracle.XOracleAdapterMode `json:"mode"`
	}
	if err := json.Unmarshal(respBz, &resp); err != nil {
		return nil, fmt.Errorf("decoding the x-oracle adapter config: %w", err)
	}
	switch {
	case resp.Mode.Proxy != nil:
		saiOracle, err := sdk.AccAddressFromBech32(resp.Mode.Proxy.SaiOracleAddr)
		if err != nil {
			return nil, fmt.Errorf("x-oracle adapter Sai oracle address: %w", err)
		}
		return saiOracle, nil
	case resp.Mode.Fixture != nil:
		return nil, ErrOracleAdapterNotConfigured
	default:
		return nil, fmt.Errorf("x-oracle adapter config has no known mode: %s", respBz)
	}
}
