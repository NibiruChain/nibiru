package app_test

import (
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/evm/evmstate"
	"github.com/NibiruChain/nibiru/v2/evm/evmtest"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
)

type acceptingPrepareVerifier struct {
	seen [][]byte
}

func (v *acceptingPrepareVerifier) PrepareProposalVerifyTxBytes(txBytes []byte) error {
	v.seen = append(v.seen, txBytes)
	return nil
}

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
