package app_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	abci "github.com/NibiruChain/nibiru/v2/bft/abci/types"
	tmproto "github.com/NibiruChain/nibiru/v2/bft/proto/tendermint/types"
	cmttypes "github.com/NibiruChain/nibiru/v2/bft/types"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/evm/evmstate"
	"github.com/NibiruChain/nibiru/v2/evm/evmtest"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
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
// messages each execute contract from sender, as CometBFT hands candidates to
// PrepareProposal.
func buildProposalWasmExecuteTx(
	t *testing.T,
	deps *evmtest.TestDeps,
	sender string,
	contracts ...string,
) []byte {
	t.Helper()
	msgs := make([]sdk.Msg, 0, len(contracts))
	for _, contract := range contracts {
		msgs = append(msgs, &wasm.MsgExecuteContract{
			Sender:   sender,
			Contract: contract,
			Msg:      wasm.RawContractMessage(`{"set_prices":{"prices":[]}}`),
		})
	}
	builder := deps.App.GetTxConfig().NewTxBuilder()
	require.NoError(t, builder.SetMsgs(msgs...))
	builder.SetGasLimit(100_000_000)
	txBytes, err := deps.App.GetTxConfig().TxEncoder()(builder.GetTx())
	require.NoError(t, err)
	return txBytes
}

// TestEVMPrepareProposalSelectsZeroGasActorTxsFirst proves that non-EVM
// zero-gas actor transactions (oracle price writes) are proposed ahead of
// every EVM transaction, so a trade in the same block executes against the
// price written in that block. Every other non-EVM transaction, including a
// call to the zero-gas contract from an unlisted sender, a listed sender
// calling an unlisted contract, or a mixed-contract transaction, keeps its
// place after the EVM transactions. Relative CometBFT order is preserved
// within each lane.
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
		oracleWrite, keeperBatch,
		evmTx,
		traderToOracle, feederToOther, mixedBatch,
	}, response.Txs)
	require.Equal(t, response.Txs, verifier.seen)
}

// TestEVMPrepareProposalZeroGasLaneWinsScarceSpace proves the zero-gas lane is
// budgeted before EVM transactions: when RequestPrepareProposal.MaxTxBytes
// fits either the oracle write or the EVM transaction but not both, the oracle
// write is proposed and the EVM transaction waits for a later block.
func TestEVMPrepareProposalZeroGasLaneWinsScarceSpace(t *testing.T) {
	deps := evmtest.NewTestDeps()
	account := evmstate.NewEmptyAccount()
	require.NoError(t, deps.EvmKeeper.SetAccount(deps.Ctx(), deps.Sender.EthAddr, *account))
	evmTx := insertProposalEVMTransaction(t, &deps, 0, 100)

	oracleFeeder := evmtest.NewEthPrivAcc().NibiruAddr.String()
	oracleContract := evmtest.NewEthPrivAcc().NibiruAddr.String()
	deps.App.SudoKeeper.ZeroGasActors.Set(deps.Ctx(), sudo.ZeroGasActors{
		Senders:   []string{oracleFeeder},
		Contracts: []string{oracleContract},
	})
	oracleWrite := buildProposalWasmExecuteTx(t, &deps, oracleFeeder, oracleContract)
	maxTxBytes := cmttypes.ComputeProtoSizeForTxs([]cmttypes.Tx{oracleWrite, evmTx}) - 1
	require.GreaterOrEqual(t, maxTxBytes, cmttypes.ComputeProtoSizeForTxs([]cmttypes.Tx{evmTx}))

	verifier := new(acceptingPrepareVerifier)
	handler := app.NewEVMPrepareProposalHandler(deps.App, verifier)
	ctx := deps.Ctx().WithConsensusParams(&tmproto.ConsensusParams{
		Block: &tmproto.BlockParams{MaxBytes: 1_000_000, MaxGas: -1},
	})
	response := handler(ctx, abci.RequestPrepareProposal{
		MaxTxBytes: maxTxBytes,
		Txs:        [][]byte{oracleWrite},
	})

	require.Equal(t, [][]byte{oracleWrite}, response.Txs)
}
