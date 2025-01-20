package keeper_test

import (
	"encoding/json"
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	staking "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethparams "github.com/ethereum/go-ethereum/params"
	"github.com/rs/zerolog/log"

	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/evm/statedb"
)

// TestGasConsumedInvariantSend: The "NibiruBankKeeper" is defined such that
// send operations are meant to have consistent gas consumption regardless of the
// whether the active "StateDB" instance is defined or undefined (nil). This is
// important to prevent consensus failures resulting from nodes reaching an
// inconsistent state after processing the same state transitions and gettng
// conflicting gas results.
func (s *Suite) TestGasConsumedInvariantSend() {
	to := evmtest.NewEthPrivAcc() // arbitrary constant

	testCases := []struct {
		GasConsumedInvariantScenario
		name string
	}{
		{
			name: "StateDB nil",
			GasConsumedInvariantScenario: GasConsumedInvariantScenario{
				BankDenom:  evm.EVMBankDenom,
				NilStateDB: true,
			},
		},

		{
			name: "StateDB defined",
			GasConsumedInvariantScenario: GasConsumedInvariantScenario{
				BankDenom:  evm.EVMBankDenom,
				NilStateDB: false,
			},
		},

		{
			name: "StateDB nil, uniba",
			GasConsumedInvariantScenario: GasConsumedInvariantScenario{
				BankDenom:  "uniba",
				NilStateDB: true,
			},
		},
	}

	s.T().Log("Check that gas consumption is equal in all scenarios")
	var first uint64
	for idx, tc := range testCases {
		s.Run(tc.name, func() {
			gasConsumed := tc.GasConsumedInvariantScenario.Run(s, to)
			s.T().Logf("gasConsumed: %d", gasConsumed)
			s.Require().NotZerof(gasConsumed, "gasConsumed should not be zero")
			if idx == 0 {
				first = gasConsumed
				return
			}
			// Each elem being equal to "first" implies that each elem is equal
			s.Equalf(
				first,
				gasConsumed,
				"Gas consumed should be equal",
			)
		})
	}
}

type GasConsumedInvariantScenario struct {
	BankDenom  string
	NilStateDB bool
}

func (scenario GasConsumedInvariantScenario) Run(
	s *Suite,
	to evmtest.EthPrivKeyAcc,
) (gasConsumed uint64) {
	bankDenom, nilStateDB := scenario.BankDenom, scenario.NilStateDB
	deps := evmtest.NewTestDeps()
	if nilStateDB {
		s.Require().Nil(deps.EvmKeeper.Bank.StateDB)
	} else {
		deps.NewStateDB()
		s.NotNil(deps.EvmKeeper.Bank.StateDB)
	}

	sendCoins := sdk.NewCoins(sdk.NewInt64Coin(bankDenom, 420))
	s.NoError(
		testapp.FundAccount(deps.App.BankKeeper, deps.Ctx, deps.Sender.NibiruAddr, sendCoins),
	)

	gasConsumedBefore := deps.Ctx.GasMeter().GasConsumed()
	s.NoError(
		deps.App.BankKeeper.SendCoins(
			deps.Ctx, deps.Sender.NibiruAddr, to.NibiruAddr, sendCoins,
		),
	)
	gasConsumedAfter := deps.Ctx.GasMeter().GasConsumed()

	s.Greaterf(gasConsumedAfter, gasConsumedBefore,
		"gas meter consumed should not be negative: gas consumed after = %d, gas consumed before = %d ",
		gasConsumedAfter, gasConsumedBefore,
	)

	return gasConsumedAfter - gasConsumedBefore
}

func (s *Suite) TestGasConsumedInvariantSendNotNibi() {
	to := evmtest.NewEthPrivAcc() // arbitrary constant

	testCases := []struct {
		GasConsumedInvariantScenario
		name string
	}{
		{
			name: "StateDB nil A",
			GasConsumedInvariantScenario: GasConsumedInvariantScenario{
				BankDenom:  "dummy_token_A",
				NilStateDB: true,
			},
		},

		{
			name: "StateDB defined A",
			GasConsumedInvariantScenario: GasConsumedInvariantScenario{
				BankDenom:  "dummy_token_A",
				NilStateDB: false,
			},
		},

		{
			name: "StateDB nil B",
			GasConsumedInvariantScenario: GasConsumedInvariantScenario{
				BankDenom:  "dummy_token_B",
				NilStateDB: true,
			},
		},

		{
			name: "StateDB defined B",
			GasConsumedInvariantScenario: GasConsumedInvariantScenario{
				BankDenom:  "dummy_token_B",
				NilStateDB: false,
			},
		},
	}

	s.T().Log("Check that gas consumption is equal in all scenarios")
	var first uint64
	for idx, tc := range testCases {
		s.Run(tc.name, func() {
			gasConsumed := tc.GasConsumedInvariantScenario.Run(s, to)
			if idx == 0 {
				first = gasConsumed
				return
			}
			// Each elem being equal to "first" implies that each elem is equal
			s.Equalf(
				fmt.Sprintf("%d", first),
				fmt.Sprintf("%d", gasConsumed),
				"Gas consumed should be equal",
			)
		})
	}
}

type FunctionalGasConsumedInvariantScenario struct {
	Setup   func(deps *evmtest.TestDeps)
	Measure func(deps *evmtest.TestDeps)
}

func (f FunctionalGasConsumedInvariantScenario) Run(s *Suite) {
	var (
		gasConsumedA uint64 // nil StateDB
		gasConsumedB uint64 // not nil StateDB
	)

	{
		deps := evmtest.NewTestDeps()
		s.Nil(deps.EvmKeeper.Bank.StateDB)

		f.Setup(&deps)

		gasConsumedBefore := deps.Ctx.GasMeter().GasConsumed()
		f.Measure(&deps)
		gasConsumedAfter := deps.Ctx.GasMeter().GasConsumed()
		gasConsumedA = gasConsumedAfter - gasConsumedBefore
	}

	{
		deps := evmtest.NewTestDeps()
		deps.NewStateDB()
		s.NotNil(deps.EvmKeeper.Bank.StateDB)

		f.Setup(&deps)

		gasConsumedBefore := deps.Ctx.GasMeter().GasConsumed()
		f.Measure(&deps)
		gasConsumedAfter := deps.Ctx.GasMeter().GasConsumed()
		gasConsumedB = gasConsumedAfter - gasConsumedBefore
	}

	s.Equalf(
		fmt.Sprintf("%d", gasConsumedA),
		fmt.Sprintf("%d", gasConsumedB),
		"Gas consumed should be equal",
	)
}

// See [Suite.TestGasConsumedInvariantSend].
func (s *Suite) TestGasConsumedInvariantOther() {
	to := evmtest.NewEthPrivAcc() // arbitrary constant
	bankDenom := evm.EVMBankDenom
	coins := sdk.NewCoins(sdk.NewInt64Coin(bankDenom, 420)) // arbitrary constant
	// Use this value because the gas token is involved in both the BaseOp and
	// AfterOp of the "ForceGasInvariant" function.
	s.Run("MintCoins", func() {
		FunctionalGasConsumedInvariantScenario{
			Setup: func(deps *evmtest.TestDeps) {
				s.NoError(
					testapp.FundAccount(deps.App.BankKeeper, deps.Ctx, deps.Sender.NibiruAddr, coins),
				)
			},
			Measure: func(deps *evmtest.TestDeps) {
				s.NoError(
					deps.App.BankKeeper.MintCoins(
						deps.Ctx, evm.ModuleName, coins,
					),
				)
			},
		}.Run(s)
	})

	s.Run("BurnCoins", func() {
		FunctionalGasConsumedInvariantScenario{
			Setup: func(deps *evmtest.TestDeps) {
				s.NoError(
					testapp.FundModuleAccount(deps.App.BankKeeper, deps.Ctx, evm.ModuleName, coins),
				)
			},
			Measure: func(deps *evmtest.TestDeps) {
				s.NoError(
					deps.App.BankKeeper.BurnCoins(
						deps.Ctx, evm.ModuleName, coins,
					),
				)
			},
		}.Run(s)
	})

	s.Run("SendCoinsFromAccountToModule", func() {
		FunctionalGasConsumedInvariantScenario{
			Setup: func(deps *evmtest.TestDeps) {
				s.NoError(
					testapp.FundAccount(deps.App.BankKeeper, deps.Ctx, deps.Sender.NibiruAddr, coins),
				)
			},
			Measure: func(deps *evmtest.TestDeps) {
				s.NoError(
					deps.App.BankKeeper.SendCoinsFromAccountToModule(
						deps.Ctx, deps.Sender.NibiruAddr, evm.ModuleName, coins,
					),
				)
			},
		}.Run(s)
	})

	s.Run("SendCoinsFromModuleToAccount", func() {
		FunctionalGasConsumedInvariantScenario{
			Setup: func(deps *evmtest.TestDeps) {
				s.NoError(
					testapp.FundModuleAccount(deps.App.BankKeeper, deps.Ctx, evm.ModuleName, coins),
				)
			},
			Measure: func(deps *evmtest.TestDeps) {
				s.NoError(
					deps.App.BankKeeper.SendCoinsFromModuleToAccount(
						deps.Ctx, evm.ModuleName, to.NibiruAddr, coins,
					),
				)
			},
		}.Run(s)
	})

	s.Run("SendCoinsFromModuleToModule", func() {
		FunctionalGasConsumedInvariantScenario{
			Setup: func(deps *evmtest.TestDeps) {
				s.NoError(
					testapp.FundModuleAccount(deps.App.BankKeeper, deps.Ctx, evm.ModuleName, coins),
				)
			},
			Measure: func(deps *evmtest.TestDeps) {
				s.NoError(
					deps.App.BankKeeper.SendCoinsFromModuleToModule(
						deps.Ctx, evm.ModuleName, staking.NotBondedPoolName, coins,
					),
				)
			},
		}.Run(s)
	})
}

// TestStateDBReadonlyInvariant: The EVM Keeper's "ApplyEvmMsg" function is used
// in both queries and transaction messages. Queries such as "eth_call",
// "eth_estimateGas", "debug_traceTransaction", "debugTraceCall", and
// "debug_traceBlock" interact with the EVM StateDB inside of "ApplyEvmMsg".
//
// Queries MUST NOT result in lingering effects on the blockchain multistore or
// the keepers, as doing so would result in potential inconsistencies between
// nodes and consensus failures. This test adds cases to make sure that invariant
// is held.
func (s *Suite) TestStateDBReadonlyInvariant() {
	deps := evmtest.NewTestDeps()
	_, _, erc20Contract := evmtest.DeployAndExecuteERC20Transfer(&deps, s.T())
	to := evmtest.NewEthPrivAcc()

	type StateDBWithExplanation struct {
		StateDB     *statedb.StateDB
		Explanation string
	}

	var stateDBs []StateDBWithExplanation
	stateDBs = append(stateDBs, StateDBWithExplanation{
		StateDB:     deps.App.EvmKeeper.Bank.StateDB,
		Explanation: "initial DB after some EthereumTx",
	})

	s.T().Log("eth_call")
	{
		fungibleTokenContract := embeds.SmartContract_ERC20Minter
		jsonTxArgs, err := json.Marshal(&evm.JsonTxArgs{
			From: &deps.Sender.EthAddr,
			Data: (*hexutil.Bytes)(&fungibleTokenContract.Bytecode),
		})
		s.Require().NoError(err)
		req := &evm.EthCallRequest{Args: jsonTxArgs}
		_, err = deps.EvmKeeper.EthCall(deps.GoCtx(), req)
		s.Require().NoError(err)
		stateDBs = append(stateDBs, StateDBWithExplanation{
			StateDB:     deps.App.EvmKeeper.Bank.StateDB,
			Explanation: "DB after eth_call query",
		})
	}

	s.T().Log(`EthereumTx success, err == nil, vmError="insufficient balance for transfer"`)
	{
		balOfSender := deps.App.BankKeeper.GetBalance(
			deps.Ctx, deps.Sender.NibiruAddr, evm.EVMBankDenom)
		tooManyTokensWei := evm.NativeToWei(balOfSender.Amount.AddRaw(420).BigInt())
		txTransferWei := evmtest.TxTransferWei{
			Deps:      &deps,
			To:        to.EthAddr,
			AmountWei: tooManyTokensWei,
		}
		evmResp, err := txTransferWei.Run()
		s.Require().NoErrorf(err, "%#v", evmResp)
		s.Require().Contains(evmResp.VmError, "insufficient balance for transfer")
		stateDBs = append(stateDBs, StateDBWithExplanation{
			StateDB:     deps.App.EvmKeeper.Bank.StateDB,
			Explanation: "DB after EthereumTx with vmError",
		})
	}

	s.T().Log(`EthereumTx success, err == nil, no vmError"`)
	{
		sendCoins := sdk.NewCoins(sdk.NewInt64Coin(evm.EVMBankDenom, 420))
		s.NoError(
			testapp.FundAccount(deps.App.BankKeeper, deps.Ctx, deps.Sender.NibiruAddr, sendCoins),
		)

		ctx := deps.Ctx
		log.Log().Msgf("ctx.GasMeter().GasConsumed() %d", ctx.GasMeter().GasConsumed())
		log.Log().Msgf("ctx.GasMeter().Limit() %d", ctx.GasMeter().Limit())

		wei := evm.NativeToWei(sendCoins[0].Amount.BigInt())
		evmResp, err := evmtest.TxTransferWei{
			Deps:      &deps,
			To:        to.EthAddr,
			AmountWei: wei,
		}.Run()

		s.Require().NoErrorf(err, "%#v", evmResp)
		s.Require().Falsef(evmResp.Failed(), "%#v", evmResp)
		stateDBs = append(stateDBs, StateDBWithExplanation{
			StateDB:     deps.App.EvmKeeper.Bank.StateDB,
			Explanation: "DB after EthereumTx success",
		})

		for _, err := range []error{
			testapp.FundAccount(deps.App.BankKeeper, deps.Ctx, deps.Sender.NibiruAddr, sendCoins),
			testapp.FundFeeCollector(deps.App.BankKeeper, deps.Ctx,
				math.NewIntFromUint64(gethparams.TxGas),
			),
		} {
			s.NoError(err)
		}
		evmResp, err = evmtest.TxTransferWei{
			Deps:      &deps,
			To:        erc20Contract,
			AmountWei: wei,
			GasLimit:  gethparams.TxGas * 2,
		}.Run()
		s.Require().NoErrorf(err, "%#v", evmResp)
		s.Require().Contains(evmResp.VmError, "execution reverted")
	}

	s.T().Log("Verify that the NibiruBankKeeper.StateDB is unaffected")
	var first *statedb.StateDB
	for idx, db := range stateDBs {
		if idx == 0 {
			first = db.StateDB
			continue
		}
		s.True(first == db.StateDB, db.Explanation)
	}
}
