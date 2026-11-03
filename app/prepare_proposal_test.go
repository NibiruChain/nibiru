package app_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	abci "github.com/NibiruChain/nibiru/v2/bft/abci/types"
	tmproto "github.com/NibiruChain/nibiru/v2/bft/proto/tendermint/types"
	cmttypes "github.com/NibiruChain/nibiru/v2/bft/types"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/evm/evmstate"
	"github.com/NibiruChain/nibiru/v2/evm/evmtest"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	signingtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/tx/signing"
	authsigning "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth/signing"
	"github.com/NibiruChain/nibiru/v2/x/sudo"
	wasm "github.com/NibiruChain/nibiru/v2/x/wasm/types"
)

// acceptingPrepareVerifier records the outer bytes passed to prepare-proposal
// verification so tests can assert byte identity against insertion.
type acceptingPrepareVerifier struct {
	seen [][]byte
}

func (v *acceptingPrepareVerifier) PrepareProposalVerifyTxBytes(txBytes []byte) error {
	v.seen = append(v.seen, txBytes)
	return nil
}

// insertProposalEVMTransaction inserts one EVM transaction into [evm.Mempool]
// using RPC-shaped encoding: [evm.MsgEthereumTx.From] is empty in the outer
// bytes, then restored on the decoded message for authenticated insertion.
// Returns the original outer bytes that proposal selection must preserve.
func insertProposalEVMTransaction(
	t *testing.T,
	deps *evmtest.TestDeps,
	nonce uint64,
	priority int64,
) []byte {
	t.Helper()
	msg := evmtest.HappyTransferTx(deps, nonce)
	sender := msg.From
	tx := evmtest.BuildTx(deps, true, msg.GetGas(), nil, msg)
	msg.From = ""
	txBytes, err := deps.App.GetTxConfig().TxEncoder()(tx)
	require.NoError(t, err)
	msg.From = sender
	ctx := deps.Ctx().
		WithTxBytes(txBytes).
		WithPriority(priority).
		WithGasMeter(sdk.NewGasMeter(msg.GetGas()))
	require.NoError(t, deps.App.EvmMempool.Insert(ctx, tx))
	return txBytes
}

// TestEVMPrepareProposalPreservesStateNonceChainBytes proves
// [app.NewEVMPrepareProposalHandler] selects only the contiguous state nonce
// chain from committed state nonce 10 (slots 10 and 11), omits gapped nonce 13,
// and returns the original outer bytes captured at insertion.
func TestEVMPrepareProposalPreservesStateNonceChainBytes(t *testing.T) {
	deps := evmtest.NewTestDeps()
	account := evmstate.NewEmptyAccount()
	account.Nonce = 10
	require.NoError(t, deps.EvmKeeper.SetAccount(deps.Ctx(), deps.Sender.EthAddr, *account))

	tx10 := insertProposalEVMTransaction(t, &deps, 10, 1)
	tx11 := insertProposalEVMTransaction(t, &deps, 11, 100)
	insertProposalEVMTransaction(t, &deps, 13, 1)

	verifier := new(acceptingPrepareVerifier)
	handler := app.NewEVMPrepareProposalHandler(deps.App, verifier)
	ctx := deps.Ctx().WithConsensusParams(&tmproto.ConsensusParams{
		Block: &tmproto.BlockParams{MaxBytes: 1_000_000, MaxGas: -1},
	})
	response := handler(ctx, abci.RequestPrepareProposal{
		MaxTxBytes: 1_000_000,
	})

	require.Equal(t, [][]byte{tx10, tx11}, response.Txs)
	require.Equal(t, response.Txs, verifier.seen)
}

// TestEVMPrepareProposalTreatsMissingAccountAsNonceZero proves that a sender
// with no committed account is treated as state nonce 0, so a first-time or
// zero-gas sender's nonce-0 transaction remains proposable. Ante creates the
// account only on deliver; proposal selection must not skip nil accounts.
func TestEVMPrepareProposalTreatsMissingAccountAsNonceZero(t *testing.T) {
	deps := evmtest.NewTestDeps()
	require.Nil(t, deps.EvmKeeper.GetAccount(deps.Ctx(), deps.Sender.EthAddr))
	tx0 := insertProposalEVMTransaction(t, &deps, 0, 1)

	verifier := new(acceptingPrepareVerifier)
	handler := app.NewEVMPrepareProposalHandler(deps.App, verifier)
	ctx := deps.Ctx().WithConsensusParams(&tmproto.ConsensusParams{
		Block: &tmproto.BlockParams{MaxBytes: 1_000_000, MaxGas: -1},
	})
	response := handler(ctx, abci.RequestPrepareProposal{
		MaxTxBytes: 1_000_000,
	})

	require.Equal(t, [][]byte{tx0}, response.Txs)
	require.Equal(t, response.Txs, verifier.seen)
}

// TestEVMPrepareProposalPassesThroughNonEVMOrder proves the non-EVM proposal
// lane preserves CometBFT's relative order from RequestPrepareProposal.Txs.
func TestEVMPrepareProposalPassesThroughNonEVMOrder(t *testing.T) {
	deps := evmtest.NewTestDeps()
	buildNonEVM := func(memo string) []byte {
		builder := deps.App.GetTxConfig().NewTxBuilder()
		builder.SetMemo(memo)
		tx := builder.GetTx()
		bytes, err := deps.App.GetTxConfig().TxEncoder()(tx)
		require.NoError(t, err)
		return bytes
	}
	txA := buildNonEVM("a")
	txB := buildNonEVM("b")

	verifier := new(acceptingPrepareVerifier)
	handler := app.NewEVMPrepareProposalHandler(deps.App, verifier)
	ctx := deps.Ctx().WithConsensusParams(&tmproto.ConsensusParams{
		Block: &tmproto.BlockParams{MaxBytes: 1_000_000, MaxGas: -1},
	})
	response := handler(ctx, abci.RequestPrepareProposal{
		MaxTxBytes: 1_000_000,
		Txs:        [][]byte{txA, txB},
	})

	require.Equal(t, [][]byte{txA, txB}, response.Txs)
}

// TestEVMPrepareProposalCountsProtoEncodedTxSize proves the handler budgets
// RequestPrepareProposal.MaxTxBytes by CometBFT's protobuf-encoded tx size, not
// raw length. Two txs whose raw lengths exactly fill MaxTxBytes exceed it once
// encoded, so only the first is selected.
func TestEVMPrepareProposalCountsProtoEncodedTxSize(t *testing.T) {
	deps := evmtest.NewTestDeps()
	buildNonEVM := func(memo string) []byte {
		builder := deps.App.GetTxConfig().NewTxBuilder()
		builder.SetMemo(memo)
		bytes, err := deps.App.GetTxConfig().TxEncoder()(builder.GetTx())
		require.NoError(t, err)
		return bytes
	}
	txA := buildNonEVM("a")
	txB := buildNonEVM("b")
	rawLen := int64(len(txA) + len(txB))
	require.Greater(t, cmttypes.ComputeProtoSizeForTxs([]cmttypes.Tx{txA, txB}), rawLen)

	verifier := new(acceptingPrepareVerifier)
	handler := app.NewEVMPrepareProposalHandler(deps.App, verifier)
	ctx := deps.Ctx().WithConsensusParams(&tmproto.ConsensusParams{
		Block: &tmproto.BlockParams{MaxBytes: 1_000_000, MaxGas: -1},
	})
	response := handler(ctx, abci.RequestPrepareProposal{
		MaxTxBytes: rawLen,
		Txs:        [][]byte{txA, txB},
	})

	require.Equal(t, [][]byte{txA}, response.Txs)
}

// TestEVMPrepareProposalPrioritizesExecutableSenderHeads proves that across
// senders, higher-priority executable heads are selected first, without
// breaking within-sender nonce order.
func TestEVMPrepareProposalPrioritizesExecutableSenderHeads(t *testing.T) {
	deps := evmtest.NewTestDeps()
	setAccount := func() {
		account := evmstate.NewEmptyAccount()
		require.NoError(t, deps.EvmKeeper.SetAccount(deps.Ctx(), deps.Sender.EthAddr, *account))
	}

	setAccount()
	lowPriority := insertProposalEVMTransaction(t, &deps, 0, 1)
	deps.Sender = evmtest.NewEthPrivAcc()
	setAccount()
	highPriority := insertProposalEVMTransaction(t, &deps, 0, 100)

	verifier := new(acceptingPrepareVerifier)
	handler := app.NewEVMPrepareProposalHandler(deps.App, verifier)
	ctx := deps.Ctx().WithConsensusParams(&tmproto.ConsensusParams{
		Block: &tmproto.BlockParams{MaxBytes: 1_000_000, MaxGas: -1},
	})
	response := handler(ctx, abci.RequestPrepareProposal{MaxTxBytes: 1_000_000})

	require.Equal(t, [][]byte{highPriority, lowPriority}, response.Txs)
}

// buildProposalWasmExecuteTx encodes an unsigned non-EVM transaction whose
// messages each send an oracle `set_prices` write to contract from sender, as
// CometBFT hands candidates to PrepareProposal.
func buildProposalWasmExecuteTx(
	t *testing.T,
	deps *evmtest.TestDeps,
	sender string,
	contracts ...string,
) []byte {
	t.Helper()
	msgs := make([]*wasm.MsgExecuteContract, 0, len(contracts))
	for _, contract := range contracts {
		msgs = append(msgs, proposalWasmExec(sender, contract, setPricesMsg))
	}
	return buildProposalWasmTx(t, deps, msgs...)
}

// Sai oracle and perp execute messages used as proposal candidates.
const (
	setPricesMsg        = `{"set_prices":{"prices":[[0,"97000.5"],[1,"3500"]]}}`
	setPriceMsg         = `{"set_price":{"token_id":0,"price_usd":"97000.5"}}`
	updateExpirationMsg = `{"update_expiration_time":{"expiration_time":60}}`
	triggerFillMsg      = `{"execute_trigger_order":{"trade_id":7}}`
)

// proposalWasmExec returns a Wasm execute message from sender to contract.
func proposalWasmExec(sender, contract, msg string) *wasm.MsgExecuteContract {
	return &wasm.MsgExecuteContract{
		Sender:   sender,
		Contract: contract,
		Msg:      wasm.RawContractMessage(msg),
	}
}

// buildProposalWasmTx encodes an unsigned non-EVM transaction carrying msgs,
// as CometBFT hands candidates to PrepareProposal.
func buildProposalWasmTx(
	t *testing.T,
	deps *evmtest.TestDeps,
	msgs ...*wasm.MsgExecuteContract,
) []byte {
	t.Helper()
	sdkMsgs := make([]sdk.Msg, 0, len(msgs))
	for _, msg := range msgs {
		sdkMsgs = append(sdkMsgs, msg)
	}
	builder := deps.App.GetTxConfig().NewTxBuilder()
	require.NoError(t, builder.SetMsgs(sdkMsgs...))
	builder.SetGasLimit(100_000_000)
	txBytes, err := deps.App.GetTxConfig().TxEncoder()(builder.GetTx())
	require.NoError(t, err)
	return txBytes
}

// TestEVMPrepareProposalSelectsZeroGasActorTxsFirst proves that zero-gas
// actor transactions are proposed ahead of every EVM transaction, so a trade
// in the same block executes against the price written in that block. Every
// message built by [buildProposalWasmExecuteTx] is a `set_prices` write, so
// both the feeder's write and the keeper's batch to the perp and oracle
// contracts land in the oracle lane. Every other non-EVM transaction,
// including a call to a zero-gas contract from an unlisted sender, a listed
// sender calling an unlisted contract, or a batch that includes an unlisted
// contract, keeps its place after the EVM transactions. Relative CometBFT
// order is preserved within each lane.
func TestEVMPrepareProposalSelectsZeroGasActorTxsFirst(t *testing.T) {
	deps := evmtest.NewTestDeps()
	account := evmstate.NewEmptyAccount()
	require.NoError(t, deps.EvmKeeper.SetAccount(deps.Ctx(), deps.Sender.EthAddr, *account))
	evmTx := insertProposalEVMTransaction(t, &deps, 0, 100)

	oracleFeeder := evmtest.NewEthPrivAcc().NibiruAddr.String()
	keeperBot := evmtest.NewEthPrivAcc().NibiruAddr.String()
	trader := evmtest.NewEthPrivAcc().NibiruAddr.String()
	oracleContract := evmtest.NewEthPrivAcc().NibiruAddr.String()
	perpContract := evmtest.NewEthPrivAcc().NibiruAddr.String()
	otherContract := evmtest.NewEthPrivAcc().NibiruAddr.String()
	deps.App.SudoKeeper.ZeroGasActors.Set(deps.Ctx(), sudo.ZeroGasActors{
		Senders:   []string{oracleFeeder, keeperBot},
		Contracts: []string{oracleContract, perpContract},
	})

	traderToOracle := buildProposalWasmExecuteTx(t, &deps, trader, oracleContract)
	oracleWrite := buildProposalWasmExecuteTx(t, &deps, oracleFeeder, oracleContract)
	feederToOther := buildProposalWasmExecuteTx(t, &deps, oracleFeeder, otherContract)
	keeperBatch := buildProposalWasmExecuteTx(t, &deps, keeperBot, perpContract, oracleContract)
	mixedBatch := buildProposalWasmExecuteTx(t, &deps, keeperBot, perpContract, otherContract)

	verifier := new(acceptingPrepareVerifier)
	handler := app.NewEVMPrepareProposalHandler(deps.App, verifier)
	ctx := deps.Ctx().WithConsensusParams(&tmproto.ConsensusParams{
		Block: &tmproto.BlockParams{MaxBytes: 1_000_000, MaxGas: -1},
	})
	response := handler(ctx, abci.RequestPrepareProposal{
		MaxTxBytes: 1_000_000,
		Txs:        [][]byte{traderToOracle, oracleWrite, feederToOther, keeperBatch, mixedBatch},
	})

	require.Equal(t, [][]byte{
		oracleWrite, keeperBatch, // oracle lane: every message is set_prices
		evmTx,                                     // EVM lane
		traderToOracle, feederToOther, mixedBatch, // default lane
	}, response.Txs)
	require.Equal(t, response.Txs, verifier.seen)
}

// insertProposalEVMTransactions inserts n EVM transactions from the test
// sender at nonces 0 through n-1 and returns their outer bytes in nonce order.
func insertProposalEVMTransactions(t *testing.T, deps *evmtest.TestDeps, n int) [][]byte {
	t.Helper()
	txs := make([][]byte, 0, n)
	for nonce := range n {
		txs = append(txs, insertProposalEVMTransaction(t, deps, uint64(nonce), 100))
	}
	return txs
}

// requireEVMPrefixAfter asserts that txs is lead followed by a non-empty,
// strict prefix of evmTxs: EVM filled the rest of the block but not all of it
// fit, so space was scarce.
func requireEVMPrefixAfter(t *testing.T, lead []byte, evmTxs, txs [][]byte) {
	t.Helper()
	require.NotEmpty(t, txs)
	require.Equal(t, lead, txs[0])
	rest := txs[1:]
	require.NotEmpty(t, rest)
	require.Less(t, len(rest), len(evmTxs))
	require.Equal(t, evmTxs[:len(rest)], rest)
}

// TestEVMPrepareProposalZeroGasLaneWinsScarceSpace proves the zero-gas lanes
// are budgeted before EVM transactions: when the EVM mempool alone could fill
// RequestPrepareProposal.MaxTxBytes, the oracle write, which fits the
// priority-lane cap, is proposed first and EVM transactions fill the rest.
func TestEVMPrepareProposalZeroGasLaneWinsScarceSpace(t *testing.T) {
	deps := evmtest.NewTestDeps()
	account := evmstate.NewEmptyAccount()
	require.NoError(t, deps.EvmKeeper.SetAccount(deps.Ctx(), deps.Sender.EthAddr, *account))
	evmTxs := insertProposalEVMTransactions(t, &deps, 8)

	oracleFeeder := evmtest.NewEthPrivAcc().NibiruAddr.String()
	oracleContract := evmtest.NewEthPrivAcc().NibiruAddr.String()
	deps.App.SudoKeeper.ZeroGasActors.Set(deps.Ctx(), sudo.ZeroGasActors{
		Senders:   []string{oracleFeeder},
		Contracts: []string{oracleContract},
	})
	oracleWrite := buildProposalWasmExecuteTx(t, &deps, oracleFeeder, oracleContract)
	maxTxBytes := 4 * cmttypes.ComputeProtoSizeForTxs([]cmttypes.Tx{oracleWrite})

	verifier := new(acceptingPrepareVerifier)
	handler := app.NewEVMPrepareProposalHandler(deps.App, verifier)
	ctx := deps.Ctx().WithConsensusParams(&tmproto.ConsensusParams{
		Block: &tmproto.BlockParams{MaxBytes: 1_000_000, MaxGas: -1},
	})
	response := handler(ctx, abci.RequestPrepareProposal{
		MaxTxBytes: maxTxBytes,
		Txs:        [][]byte{oracleWrite},
	})

	requireEVMPrefixAfter(t, oracleWrite, evmTxs, response.Txs)
}

// saiProposalActors is a mainnet-shaped x/sudo ZeroGasActors setup: the
// allowlist holds the oracle and perp contracts, and its senders include the
// price feeder, the keeper, and an admin.
type saiProposalActors struct {
	feeder, keeper, admin, trader string
	oracle, perp                  string
}

func setSaiProposalActors(deps *evmtest.TestDeps) saiProposalActors {
	actors := saiProposalActors{
		feeder: evmtest.NewEthPrivAcc().NibiruAddr.String(),
		keeper: evmtest.NewEthPrivAcc().NibiruAddr.String(),
		admin:  evmtest.NewEthPrivAcc().NibiruAddr.String(),
		trader: evmtest.NewEthPrivAcc().NibiruAddr.String(),
		oracle: evmtest.NewEthPrivAcc().NibiruAddr.String(),
		perp:   evmtest.NewEthPrivAcc().NibiruAddr.String(),
	}
	deps.App.SudoKeeper.ZeroGasActors.Set(deps.Ctx(), sudo.ZeroGasActors{
		Senders:   []string{actors.feeder, actors.keeper, actors.admin},
		Contracts: []string{actors.oracle, actors.perp},
	})
	return actors
}

// TestEVMPrepareProposalOracleLaneBeforeZeroGasLane proves the four proposal
// lanes: oracle price writes (`set_prices` and `set_price`), then other
// zero-gas actor transactions, then EVM transactions, then everything else.
// The keeper fill arrives first in RequestPrepareProposal.Txs but is proposed
// after the price writes, so it executes against the price written in the
// same block. Relative CometBFT order holds within each lane.
func TestEVMPrepareProposalOracleLaneBeforeZeroGasLane(t *testing.T) {
	deps := evmtest.NewTestDeps()
	account := evmstate.NewEmptyAccount()
	require.NoError(t, deps.EvmKeeper.SetAccount(deps.Ctx(), deps.Sender.EthAddr, *account))
	evmTx := insertProposalEVMTransaction(t, &deps, 0, 100)
	actors := setSaiProposalActors(&deps)

	keeperFill := buildProposalWasmTx(t, &deps,
		proposalWasmExec(actors.keeper, actors.perp, triggerFillMsg))
	traderTx := buildProposalWasmTx(t, &deps,
		proposalWasmExec(actors.trader, actors.perp, triggerFillMsg))
	setPrices := buildProposalWasmTx(t, &deps,
		proposalWasmExec(actors.feeder, actors.oracle, setPricesMsg))
	keeperFill2 := buildProposalWasmTx(t, &deps,
		proposalWasmExec(actors.keeper, actors.perp, `{"liquidate":{"trade_id":8}}`))
	setPrice := buildProposalWasmTx(t, &deps,
		proposalWasmExec(actors.feeder, actors.oracle, setPriceMsg))

	verifier := new(acceptingPrepareVerifier)
	handler := app.NewEVMPrepareProposalHandler(deps.App, verifier)
	ctx := deps.Ctx().WithConsensusParams(&tmproto.ConsensusParams{
		Block: &tmproto.BlockParams{MaxBytes: 1_000_000, MaxGas: -1},
	})
	response := handler(ctx, abci.RequestPrepareProposal{
		MaxTxBytes: 1_000_000,
		Txs:        [][]byte{keeperFill, traderTx, setPrices, keeperFill2, setPrice},
	})

	require.Equal(t, [][]byte{
		setPrices, setPrice, // oracle lane
		keeperFill, keeperFill2, // zero-gas lane
		evmTx,    // EVM lane
		traderTx, // default lane
	}, response.Txs)
	require.Equal(t, response.Txs, verifier.seen)
}

// TestEVMPrepareProposalOracleLaneRequiresPriceWrites proves the oracle lane
// matches the message, not the contract: a zero-gas actor calling the oracle
// contract with a non-price message, a batch mixing a price write with a
// non-price message, and a malformed or multi-key JSON message all fall to
// the zero-gas lane. A `set_prices` call from an unlisted sender gets no
// priority at all.
func TestEVMPrepareProposalOracleLaneRequiresPriceWrites(t *testing.T) {
	deps := evmtest.NewTestDeps()
	account := evmstate.NewEmptyAccount()
	require.NoError(t, deps.EvmKeeper.SetAccount(deps.Ctx(), deps.Sender.EthAddr, *account))
	evmTx := insertProposalEVMTransaction(t, &deps, 0, 100)
	actors := setSaiProposalActors(&deps)

	unlistedSetPrices := buildProposalWasmTx(t, &deps,
		proposalWasmExec(actors.trader, actors.oracle, setPricesMsg))
	expiryUpdate := buildProposalWasmTx(t, &deps,
		proposalWasmExec(actors.keeper, actors.oracle, updateExpirationMsg))
	mixedBatch := buildProposalWasmTx(t, &deps,
		proposalWasmExec(actors.admin, actors.oracle, setPricesMsg),
		proposalWasmExec(actors.admin, actors.oracle, updateExpirationMsg))
	malformed := buildProposalWasmTx(t, &deps,
		proposalWasmExec(actors.admin, actors.oracle, `{"set_prices":{"prices":[]}`))
	twoKeys := buildProposalWasmTx(t, &deps, proposalWasmExec(
		actors.admin, actors.oracle,
		`{"set_prices":{"prices":[]},"update_expiration_time":{}}`))
	setPrices := buildProposalWasmTx(t, &deps,
		proposalWasmExec(actors.feeder, actors.oracle, setPricesMsg))

	verifier := new(acceptingPrepareVerifier)
	handler := app.NewEVMPrepareProposalHandler(deps.App, verifier)
	ctx := deps.Ctx().WithConsensusParams(&tmproto.ConsensusParams{
		Block: &tmproto.BlockParams{MaxBytes: 1_000_000, MaxGas: -1},
	})
	response := handler(ctx, abci.RequestPrepareProposal{
		MaxTxBytes: 1_000_000,
		Txs: [][]byte{
			unlistedSetPrices, expiryUpdate, mixedBatch, malformed, twoKeys,
			setPrices,
		},
	})

	require.Equal(t, [][]byte{
		setPrices,                                    // oracle lane
		expiryUpdate, mixedBatch, malformed, twoKeys, // zero-gas lane
		evmTx,             // EVM lane
		unlistedSetPrices, // default lane
	}, response.Txs)
}

// TestEVMPrepareProposalOracleLaneWinsScarceSpace proves the oracle lane is
// budgeted first: the priority-lane cap (1/4 of MaxTxBytes) fits the price
// write or the keeper fill but not both, so only the price write is proposed,
// even though the keeper fill arrived first. EVM transactions, which alone
// would overflow the block, fill the remaining space.
func TestEVMPrepareProposalOracleLaneWinsScarceSpace(t *testing.T) {
	deps := evmtest.NewTestDeps()
	account := evmstate.NewEmptyAccount()
	require.NoError(t, deps.EvmKeeper.SetAccount(deps.Ctx(), deps.Sender.EthAddr, *account))
	evmTxs := insertProposalEVMTransactions(t, &deps, 8)
	actors := setSaiProposalActors(&deps)

	keeperFill := buildProposalWasmTx(t, &deps,
		proposalWasmExec(actors.keeper, actors.perp, triggerFillMsg))
	setPrices := buildProposalWasmTx(t, &deps,
		proposalWasmExec(actors.feeder, actors.oracle, setPricesMsg))
	maxTxBytes := 4 * cmttypes.ComputeProtoSizeForTxs([]cmttypes.Tx{setPrices})
	require.GreaterOrEqual(t, maxTxBytes/4,
		cmttypes.ComputeProtoSizeForTxs([]cmttypes.Tx{keeperFill}))

	verifier := new(acceptingPrepareVerifier)
	handler := app.NewEVMPrepareProposalHandler(deps.App, verifier)
	ctx := deps.Ctx().WithConsensusParams(&tmproto.ConsensusParams{
		Block: &tmproto.BlockParams{MaxBytes: 1_000_000, MaxGas: -1},
	})
	response := handler(ctx, abci.RequestPrepareProposal{
		MaxTxBytes: maxTxBytes,
		Txs:        [][]byte{keeperFill, setPrices},
	})

	requireEVMPrefixAfter(t, setPrices, evmTxs, response.Txs)
}

// TestEVMPrepareProposalCapsPriorityLaneBytes proves a flood of zero-fee
// zero-gas actor transactions cannot take the whole block: lanes 1 and 2
// together stop at 1/4 of MaxTxBytes, here exactly the price write plus one
// keeper tx. The price write still wins its place despite arriving last, the
// rest of the flood is skipped, and the EVM and default lanes keep their
// room.
func TestEVMPrepareProposalCapsPriorityLaneBytes(t *testing.T) {
	deps := evmtest.NewTestDeps()
	account := evmstate.NewEmptyAccount()
	require.NoError(t, deps.EvmKeeper.SetAccount(deps.Ctx(), deps.Sender.EthAddr, *account))
	evmTx := insertProposalEVMTransaction(t, &deps, 0, 100)
	actors := setSaiProposalActors(&deps)

	flood := make([][]byte, 0, 20)
	for tradeID := range 20 {
		flood = append(flood, buildProposalWasmTx(t, &deps, proposalWasmExec(
			actors.keeper, actors.perp,
			fmt.Sprintf(`{"liquidate":{"trade_id":%d}}`, 10+tradeID))))
	}
	setPrices := buildProposalWasmTx(t, &deps,
		proposalWasmExec(actors.feeder, actors.oracle, setPricesMsg))
	traderTx := buildProposalWasmTx(t, &deps,
		proposalWasmExec(actors.trader, actors.perp, triggerFillMsg))
	maxTxBytes := 4 * cmttypes.ComputeProtoSizeForTxs([]cmttypes.Tx{setPrices, flood[0]})

	verifier := new(acceptingPrepareVerifier)
	handler := app.NewEVMPrepareProposalHandler(deps.App, verifier)
	ctx := deps.Ctx().WithConsensusParams(&tmproto.ConsensusParams{
		Block: &tmproto.BlockParams{MaxBytes: 1_000_000, MaxGas: -1},
	})
	candidates := append(append([][]byte{}, flood...), traderTx, setPrices)
	response := handler(ctx, abci.RequestPrepareProposal{
		MaxTxBytes: maxTxBytes,
		Txs:        candidates,
	})

	require.Equal(t, [][]byte{setPrices, flood[0], evmTx, traderTx}, response.Txs)
}

// rejectingPrepareVerifier records every verified candidate and rejects the
// ones listed in reject, as the BaseApp verifier does for a tx that fails ante.
type rejectingPrepareVerifier struct {
	reject [][]byte
	seen   [][]byte
}

func (v *rejectingPrepareVerifier) PrepareProposalVerifyTxBytes(txBytes []byte) error {
	v.seen = append(v.seen, txBytes)
	for _, rejected := range v.reject {
		if string(rejected) == string(txBytes) {
			return fmt.Errorf("rejected by test verifier")
		}
	}
	return nil
}

// sequencePrepareVerifier models the account sequence check that the BaseApp
// verifier runs against cumulative prepare-proposal state: a non-EVM tx is
// accepted only when every tx-level signer, fee payer included, presents its
// next expected sequence, and acceptance advances each of them. Sequences
// start at 0. EVM txs are accepted as is.
type sequencePrepareVerifier struct {
	decode sdk.TxDecoder
	next   map[string]uint64
}

func (v *sequencePrepareVerifier) PrepareProposalVerifyTxBytes(txBytes []byte) error {
	tx, err := v.decode(txBytes)
	if err != nil {
		return err
	}
	sigTx, ok := tx.(authsigning.SigVerifiableTx)
	if !ok {
		return nil
	}
	sigs, err := sigTx.GetSignaturesV2()
	if err != nil || len(sigs) == 0 {
		return nil
	}
	signers := sigTx.GetSigners()
	if len(signers) != len(sigs) {
		return fmt.Errorf("expected %d signatures, got %d", len(signers), len(sigs))
	}
	for i, signer := range signers {
		if sigs[i].Sequence != v.next[signer.String()] {
			return fmt.Errorf("account sequence mismatch for %s: expected %d, got %d",
				signer, v.next[signer.String()], sigs[i].Sequence)
		}
	}
	for _, signer := range signers {
		v.next[signer.String()]++
	}
	return nil
}

// buildSequencedProposalWasmTx encodes a non-EVM tx carrying msgs with one
// signer info per tx-level signer, at the given account sequences in signer
// order (message signers, then feePayer when set and distinct). The signature
// bytes are empty; only the sequences matter to [sequencePrepareVerifier].
func buildSequencedProposalWasmTx(
	t *testing.T,
	deps *evmtest.TestDeps,
	feePayer sdk.AccAddress,
	seqs []uint64,
	msgs ...*wasm.MsgExecuteContract,
) []byte {
	t.Helper()
	sdkMsgs := make([]sdk.Msg, 0, len(msgs))
	for _, msg := range msgs {
		sdkMsgs = append(sdkMsgs, msg)
	}
	builder := deps.App.GetTxConfig().NewTxBuilder()
	require.NoError(t, builder.SetMsgs(sdkMsgs...))
	builder.SetGasLimit(100_000_000)
	if feePayer != nil {
		builder.SetFeePayer(feePayer)
	}
	sigs := make([]signingtypes.SignatureV2, 0, len(seqs))
	for _, seq := range seqs {
		sigs = append(sigs, signingtypes.SignatureV2{
			PubKey: evmtest.NewEthPrivAcc().PrivKey.PubKey(),
			Data: &signingtypes.SingleSignatureData{
				SignMode: signingtypes.SignMode_SIGN_MODE_DIRECT,
			},
			Sequence: seq,
		})
	}
	require.NoError(t, builder.SetSignatures(sigs...))
	txBytes, err := deps.App.GetTxConfig().TxEncoder()(builder.GetTx())
	require.NoError(t, err)
	return txBytes
}

// TestEVMPrepareProposalKeepsSignerOrderAcrossLanes proves lanes never move a
// signer's tx ahead of its own earlier tx. The keeper sends A at sequence 0 to
// an unlisted contract (default lane), then trigger fill B at sequence 1 to
// the perp contract (zero-gas lane). B is demoted to the default lane behind
// A, so both pass the sequence check and land in one block. The feeder's price
// write is unaffected and still leads the block.
func TestEVMPrepareProposalKeepsSignerOrderAcrossLanes(t *testing.T) {
	deps := evmtest.NewTestDeps()
	account := evmstate.NewEmptyAccount()
	require.NoError(t, deps.EvmKeeper.SetAccount(deps.Ctx(), deps.Sender.EthAddr, *account))
	evmTx := insertProposalEVMTransaction(t, &deps, 0, 100)
	actors := setSaiProposalActors(&deps)
	otherContract := evmtest.NewEthPrivAcc().NibiruAddr.String()

	keeperA := buildSequencedProposalWasmTx(t, &deps, nil, []uint64{0},
		proposalWasmExec(actors.keeper, otherContract, `{"claim":{}}`))
	keeperB := buildSequencedProposalWasmTx(t, &deps, nil, []uint64{1},
		proposalWasmExec(actors.keeper, actors.perp, triggerFillMsg))
	setPrices := buildSequencedProposalWasmTx(t, &deps, nil, []uint64{0},
		proposalWasmExec(actors.feeder, actors.oracle, setPricesMsg))

	verifier := &sequencePrepareVerifier{
		decode: deps.App.GetTxConfig().TxDecoder(),
		next:   map[string]uint64{},
	}
	handler := app.NewEVMPrepareProposalHandler(deps.App, verifier)
	ctx := deps.Ctx().WithConsensusParams(&tmproto.ConsensusParams{
		Block: &tmproto.BlockParams{MaxBytes: 1_000_000, MaxGas: -1},
	})
	response := handler(ctx, abci.RequestPrepareProposal{
		MaxTxBytes: 1_000_000,
		Txs:        [][]byte{keeperA, keeperB, setPrices},
	})

	require.Equal(t, [][]byte{setPrices, evmTx, keeperA, keeperB}, response.Txs)
}

// TestEVMPrepareProposalKeepsFeePayerOrderAcrossLanes proves demotion counts
// the fee payer as a signer, as ante does. Account P sends a default-lane tx
// at sequence 0, then pays fees for a keeper trigger fill that carries P's
// sequence 1. The fill shares signer P with the earlier tx, so it is demoted
// behind it and both land in one block.
func TestEVMPrepareProposalKeepsFeePayerOrderAcrossLanes(t *testing.T) {
	deps := evmtest.NewTestDeps()
	account := evmstate.NewEmptyAccount()
	require.NoError(t, deps.EvmKeeper.SetAccount(deps.Ctx(), deps.Sender.EthAddr, *account))
	evmTx := insertProposalEVMTransaction(t, &deps, 0, 100)
	actors := setSaiProposalActors(&deps)
	payer := evmtest.NewEthPrivAcc().NibiruAddr

	payerTx := buildSequencedProposalWasmTx(t, &deps, nil, []uint64{0},
		proposalWasmExec(payer.String(), actors.perp, triggerFillMsg))
	sponsoredFill := buildSequencedProposalWasmTx(t, &deps, payer, []uint64{0, 1},
		proposalWasmExec(actors.keeper, actors.perp, triggerFillMsg))

	verifier := &sequencePrepareVerifier{
		decode: deps.App.GetTxConfig().TxDecoder(),
		next:   map[string]uint64{},
	}
	handler := app.NewEVMPrepareProposalHandler(deps.App, verifier)
	ctx := deps.Ctx().WithConsensusParams(&tmproto.ConsensusParams{
		Block: &tmproto.BlockParams{MaxBytes: 1_000_000, MaxGas: -1},
	})
	response := handler(ctx, abci.RequestPrepareProposal{
		MaxTxBytes: 1_000_000,
		Txs:        [][]byte{payerTx, sponsoredFill},
	})

	require.Equal(t, [][]byte{evmTx, payerTx, sponsoredFill}, response.Txs)
}

// TestEVMPrepareProposalDropsUnclassifiableCandidate proves a decodable
// candidate whose sender is not valid bech32, which panics in GetSigners, is
// dropped on its own instead of emptying the proposal through the
// handler-level recover.
func TestEVMPrepareProposalDropsUnclassifiableCandidate(t *testing.T) {
	deps := evmtest.NewTestDeps()
	account := evmstate.NewEmptyAccount()
	require.NoError(t, deps.EvmKeeper.SetAccount(deps.Ctx(), deps.Sender.EthAddr, *account))
	evmTx := insertProposalEVMTransaction(t, &deps, 0, 100)
	actors := setSaiProposalActors(&deps)

	invalidSender := buildProposalWasmTx(t, &deps,
		proposalWasmExec("not-a-bech32-address", actors.oracle, setPricesMsg))
	setPrices := buildProposalWasmTx(t, &deps,
		proposalWasmExec(actors.feeder, actors.oracle, setPricesMsg))
	traderTx := buildProposalWasmTx(t, &deps,
		proposalWasmExec(actors.trader, actors.perp, triggerFillMsg))

	verifier := new(acceptingPrepareVerifier)
	handler := app.NewEVMPrepareProposalHandler(deps.App, verifier)
	ctx := deps.Ctx().WithConsensusParams(&tmproto.ConsensusParams{
		Block: &tmproto.BlockParams{MaxBytes: 1_000_000, MaxGas: -1},
	})
	response := handler(ctx, abci.RequestPrepareProposal{
		MaxTxBytes: 1_000_000,
		Txs:        [][]byte{invalidSender, setPrices, traderTx},
	})

	require.Equal(t, [][]byte{setPrices, evmTx, traderTx}, response.Txs)
	require.Equal(t, response.Txs, verifier.seen)
}

// TestEVMPrepareProposalSkipsRejectedLaneCandidates proves a candidate in the
// oracle or zero-gas lane that fails verification is skipped without blocking
// later candidates in the same or later lanes.
func TestEVMPrepareProposalSkipsRejectedLaneCandidates(t *testing.T) {
	deps := evmtest.NewTestDeps()
	account := evmstate.NewEmptyAccount()
	require.NoError(t, deps.EvmKeeper.SetAccount(deps.Ctx(), deps.Sender.EthAddr, *account))
	evmTx := insertProposalEVMTransaction(t, &deps, 0, 100)
	actors := setSaiProposalActors(&deps)

	badPrices := buildProposalWasmTx(t, &deps,
		proposalWasmExec(actors.feeder, actors.oracle, setPricesMsg))
	goodPrice := buildProposalWasmTx(t, &deps,
		proposalWasmExec(actors.feeder, actors.oracle, setPriceMsg))
	badFill := buildProposalWasmTx(t, &deps,
		proposalWasmExec(actors.keeper, actors.perp, triggerFillMsg))
	goodFill := buildProposalWasmTx(t, &deps,
		proposalWasmExec(actors.keeper, actors.perp, `{"liquidate":{"trade_id":8}}`))

	verifier := &rejectingPrepareVerifier{reject: [][]byte{badPrices, badFill}}
	handler := app.NewEVMPrepareProposalHandler(deps.App, verifier)
	ctx := deps.Ctx().WithConsensusParams(&tmproto.ConsensusParams{
		Block: &tmproto.BlockParams{MaxBytes: 1_000_000, MaxGas: -1},
	})
	response := handler(ctx, abci.RequestPrepareProposal{
		MaxTxBytes: 1_000_000,
		Txs:        [][]byte{badFill, goodFill, badPrices, goodPrice},
	})

	require.Equal(t, [][]byte{goodPrice, goodFill, evmTx}, response.Txs)
	require.Equal(t, [][]byte{badPrices, goodPrice, badFill, goodFill, evmTx}, verifier.seen)
}

// TestEVMPrepareProposalZeroGasLanesUseDeclaredGas pins a known hazard: lanes
// 1 and 2 are charged their declared gas limit (100M per tx here, as gasless
// Sai txs declare) against a finite max_gas, even though they execute on a
// zero gas meter. With max_gas equal to two declared limits, the price write
// and keeper fill use the whole budget, and neither the EVM tx nor the
// default-lane tx fits. ProcessProposal sums declared gas the same way, so
// charging less here would produce blocks that fail ProcessProposal.
func TestEVMPrepareProposalZeroGasLanesUseDeclaredGas(t *testing.T) {
	deps := evmtest.NewTestDeps()
	account := evmstate.NewEmptyAccount()
	require.NoError(t, deps.EvmKeeper.SetAccount(deps.Ctx(), deps.Sender.EthAddr, *account))
	_ = insertProposalEVMTransaction(t, &deps, 0, 100)
	actors := setSaiProposalActors(&deps)

	keeperFill := buildProposalWasmTx(t, &deps,
		proposalWasmExec(actors.keeper, actors.perp, triggerFillMsg))
	setPrices := buildProposalWasmTx(t, &deps,
		proposalWasmExec(actors.feeder, actors.oracle, setPricesMsg))
	traderTx := buildProposalWasmTx(t, &deps,
		proposalWasmExec(actors.trader, actors.perp, triggerFillMsg))

	verifier := new(acceptingPrepareVerifier)
	handler := app.NewEVMPrepareProposalHandler(deps.App, verifier)
	ctx := deps.Ctx().WithConsensusParams(&tmproto.ConsensusParams{
		Block: &tmproto.BlockParams{MaxBytes: 1_000_000, MaxGas: 200_000_000},
	})
	response := handler(ctx, abci.RequestPrepareProposal{
		MaxTxBytes: 1_000_000,
		Txs:        [][]byte{traderTx, keeperFill, setPrices},
	})

	require.Equal(t, [][]byte{setPrices, keeperFill}, response.Txs)
}
