package ante_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/evm/evmtest"
	wasmtest "github.com/NibiruChain/nibiru/v2/evm/precompile/test"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/baseapp"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/client/tx"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/crypto/hd"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/crypto/keyring"
	txtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/tx"
	authtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth/types"
	gov "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/gov/types/v1"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/sudo"
	wasm "github.com/NibiruChain/nibiru/v2/x/wasm/types"
)

// Each Simulate call forks committed state. Delivery has the same voter and
// proposal, but BeginBlock can drain the fee collector before delivery.
type appSimulationConn struct {
	chain *app.NibiruApp
	calls int
	gas   []uint64
}

func (c *appSimulationConn) Invoke(_ context.Context, _ string, request, response interface{}, _ ...grpc.CallOption) error {
	c.calls++
	info, result, err := c.chain.Simulate(request.(*txtypes.SimulateRequest).TxBytes)
	if err != nil {
		return err
	}
	c.gas = append(c.gas, info.GasUsed)
	*response.(*txtypes.SimulateResponse) = txtypes.SimulateResponse{GasInfo: &info, Result: result}
	return nil
}
func (*appSimulationConn) NewStream(context.Context, *grpc.StreamDesc, string, ...grpc.CallOption) (grpc.ClientStream, error) {
	panic("unused")
}

func TestAutomaticGasCoversGovernanceExecution(t *testing.T) {
	for _, collectorFunded := range []bool{false, true} {
		for _, adjustment := range []float64{1, 1.05} {
			t.Run(fmt.Sprintf("adjustment=%g/collector-funded=%t", adjustment, collectorFunded), func(t *testing.T) {
				chain, ctx := testapp.NewNibiruTestAppAndContext()
				baseapp.SetChainID("gas-test")(chain.BaseApp)
				config := chain.GetTxConfig()
				kb := keyring.NewInMemory(chain.AppCodec())
				record, _, err := kb.NewMnemonic("voter", keyring.English, hd.CreateHDPath(118, 0, 0).String(), "", hd.Secp256k1)
				require.NoError(t, err)
				addr, err := record.GetAddress()
				require.NoError(t, err)
				require.NoError(t, testapp.FundAccount(chain.BankKeeper, ctx, addr, unibi(1_000_000)))
				// Match the archived voter, whose public key was already in state.
				pubkey, err := record.GetPubKey()
				require.NoError(t, err)
				voter := chain.AccountKeeper.GetAccount(ctx, addr)
				require.NoError(t, voter.SetPubKey(pubkey))
				chain.AccountKeeper.SetAccount(ctx, voter)
				if collectorFunded {
					require.NoError(t, testapp.FundAccount(chain.BankKeeper, ctx, addr, unibi(1_000_000)))
					require.NoError(t, chain.BankKeeper.SendCoinsFromAccountToModule(ctx, addr, authtypes.FeeCollectorName, unibi(1_000_000)))
				}
				proposal, err := gov.NewProposal(nil, 38, ctx.BlockTime(), ctx.BlockTime().Add(time.Hour), "", "Gas regression", "Gas regression", addr)
				require.NoError(t, err)
				proposal.Status = gov.StatusVotingPeriod
				chain.GovKeeper.SetProposal(ctx, proposal)
				chain.Commit()
				chain.BeginBlock(abci.RequestBeginBlock{Header: tmproto.Header{Height: 2, Time: ctx.BlockTime(), ChainID: "gas-test"}})
				ctx = chain.GetContextForDeliverTx(nil)
				require.True(t, chain.BankKeeper.GetBalance(ctx, chain.AccountKeeper.GetModuleAddress(authtypes.FeeCollectorName), "unibi").IsZero(), "BeginBlock drains prior fees")
				acc := chain.AccountKeeper.GetAccount(ctx, addr)
				f := tx.Factory{}.WithTxConfig(config).WithKeybase(kb).WithFromName("voter").WithChainID(ctx.ChainID()).
					WithAccountNumber(acc.GetAccountNumber()).WithSequence(acc.GetSequence()).WithSimulateAndExecute(true).
					WithGasPrices("0.025unibi").WithGasAdjustment(adjustment)
				msg := gov.NewMsgVote(addr, 38, gov.OptionYes, "")
				conn := &appSimulationConn{chain: chain}
				res, limit, err := tx.CalculateGas(conn, f, msg)
				require.NoError(t, err)
				require.LessOrEqual(t, conn.calls, 4)
				require.Greater(t, conn.gas[1], conn.gas[0], "fee-bearing simulation must include Bank payment")
				require.Positive(t, res.GasInfo.GasUsed)
				// Simulations must leave the voter sequence and balance untouched.
				require.Equal(t, acc.GetSequence(), chain.AccountKeeper.GetAccount(ctx, addr).GetSequence())
				require.EqualValues(t, 1_000_000, chain.BankKeeper.GetBalance(ctx, addr, "unibi").Amount.Int64())
				built, err := f.WithGas(limit).BuildUnsignedTx(msg)
				require.NoError(t, err)
				require.NoError(t, tx.Sign(f.WithGas(limit), "voter", built, true))
				bz, err := config.TxEncoder()(built.GetTx())
				require.NoError(t, err)
				delivered := chain.DeliverTx(abci.RequestDeliverTx{Tx: bz})
				require.True(t, delivered.IsOK(), "%+v", delivered)
				require.LessOrEqual(t, uint64(delivered.GasUsed), limit)
				require.Greater(t, uint64(delivered.GasUsed), res.GasInfo.GasUsed, "simulation omits counter costs")
				_, found := chain.GovKeeper.GetVote(ctx, 38, addr)
				require.True(t, found)
				t.Logf("measured=%d adjustment=%g limit=%d execution=%d calls=%d", res.GasInfo.GasUsed, adjustment, limit, delivered.GasUsed, conn.calls)
			})
		}
	}
}

// A zero measurement does not cover the whitelist lookup before the fixed
// zero meter is installed. Exercise this with a real contract and ante chain.
func TestZeroGasEstimateRequiresExplicitLimit(t *testing.T) {
	deps := evmtest.NewTestDeps()
	baseapp.SetChainID(deps.Ctx().ChainID())(deps.App.BaseApp)
	header := deps.Ctx().BlockHeader()
	header.ChainID = deps.Ctx().ChainID()
	deps.App.BeginBlock(abci.RequestBeginBlock{Header: header})
	require.NoError(t, testapp.FundAccount(deps.App.BankKeeper, deps.Ctx(), deps.Sender.NibiruAddr, unibi(1_000_000)))
	assertions := new(suite.Suite)
	assertions.SetT(t)
	contracts := wasmtest.SetupWasmContracts(&deps, assertions)
	deps.Commit()
	chain := deps.App
	ctx := deps.Ctx()
	config := chain.GetTxConfig()
	kb := keyring.NewInMemory(chain.AppCodec())
	record, _, err := kb.NewMnemonic("gasless", keyring.English, hd.CreateHDPath(118, 0, 0).String(), "", hd.Secp256k1)
	require.NoError(t, err)
	addr, err := record.GetAddress()
	require.NoError(t, err)
	require.NoError(t, testapp.FundAccount(chain.BankKeeper, ctx, addr, unibi(1_000_000)))
	chain.SudoKeeper.ZeroGasActors.Set(ctx, sudo.ZeroGasActors{Senders: []string{addr.String()}, Contracts: []string{contracts[1].String()}})
	params := chain.GetConsensusParams(ctx)
	params.Block.MaxGas = -1
	chain.StoreConsensusParams(ctx, params)
	baseapp.SetChainID(ctx.ChainID())(chain.BaseApp)
	chain.Commit()
	chain.BeginBlock(abci.RequestBeginBlock{Header: tmproto.Header{Height: 2, Time: ctx.BlockTime(), ChainID: ctx.ChainID()}})
	ctx = chain.GetContextForDeliverTx(nil)
	acc := chain.AccountKeeper.GetAccount(ctx, addr)
	f := tx.Factory{}.WithTxConfig(config).WithKeybase(kb).WithFromName("gasless").WithChainID(ctx.ChainID()).
		WithAccountNumber(acc.GetAccountNumber()).WithSequence(acc.GetSequence()).WithSimulateAndExecute(true).
		WithGasPrices("0.025unibi").WithGasAdjustment(1.05)
	msg := &wasm.MsgExecuteContract{Sender: addr.String(), Contract: contracts[1].String(), Msg: wasm.RawContractMessage(`{"increment":{}}`)}
	conn := &appSimulationConn{chain: chain}
	res, limit, err := tx.CalculateGas(conn, f, msg)
	require.NoError(t, err)
	require.Zero(t, res.GasInfo.GasUsed)
	// The old estimate of zero must fail before reaching the zero-gas meter.
	old, err := f.WithGas(0).BuildUnsignedTx(msg)
	require.NoError(t, err)
	require.NoError(t, tx.Sign(f, "gasless", old, true))
	bz, err := config.TxEncoder()(old.GetTx())
	require.NoError(t, err)
	failure := chain.CheckTx(abci.RequestCheckTx{Tx: bz})
	require.EqualValues(t, 11, failure.Code, "%+v", failure)
	require.Contains(t, failure.Log, "ReadFlat")
	// The fee/counter repair deliberately preserves zero measurements. This
	// records its remaining limitation, and verifies the explicit-limit remedy.
	require.Zero(t, limit)
	for index, explicitLimit := range []uint64{300000, 500000} {
		manual := f.WithGas(explicitLimit).WithSequence(acc.GetSequence() + uint64(index))
		built, err := manual.BuildUnsignedTx(msg)
		require.NoError(t, err)
		require.NoError(t, tx.Sign(manual, "gasless", built, true))
		bz, err = config.TxEncoder()(built.GetTx())
		require.NoError(t, err)
		check := chain.CheckTx(abci.RequestCheckTx{Tx: bz})
		require.True(t, check.IsOK(), "%+v", check)
		balanceBefore := chain.BankKeeper.GetBalance(ctx, addr, "unibi")
		delivered := chain.DeliverTx(abci.RequestDeliverTx{Tx: bz})
		require.True(t, delivered.IsOK(), "%+v", delivered)
		require.Zero(t, delivered.GasUsed)
		require.Equal(t, balanceBefore, chain.BankKeeper.GetBalance(ctx, addr, "unibi"), "the declared fee must remain uncharged for a zero-gas actor")
		t.Logf("zero-limit CheckTx failed at %d gas; measured=%d auto limit=%d explicit limit=%d execution=%d", failure.GasUsed, res.GasInfo.GasUsed, limit, explicitLimit, delivered.GasUsed)
	}
}
