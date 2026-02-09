package evmstate_test

import (
	"math/big"
	"strconv"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	gethparams "github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"

	"github.com/NibiruChain/nibiru/v2/x/evm"

	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil/testapp"

	abci "github.com/cometbft/cometbft/abci/types"

	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
)

func (s *Suite) TestMsgEthereumTx_CreateContract() {
	testCases := []struct {
		name     string
		scenario func()
	}{
		{
			name: "happy: deploy contract, sufficient gas limit",
			scenario: func() {
				deps := evmtest.NewTestDeps()
				ethAcc := deps.Sender

				// Leftover gas fee is refunded within EthereumTx from the FeeCollector
				// so, the module must have some coins
				err := testapp.FundModuleAccount(
					deps.App.BankKeeper,
					deps.Ctx(),
					authtypes.FeeCollectorName,
					sdk.NewCoins(sdk.NewCoin("unibi", sdkmath.NewInt(1_000_000))),
				)
				s.Require().NoError(err)
				s.T().Log("create eth tx msg, increase gas limit")
				gasLimit := big.NewInt(1_500_000)
				ethTxMsg, err := evmtest.NewMsgEthereumTx(evmtest.ArgsEthTx{
					CreateContract: &evmtest.ArgsCreateContract{
						EthAcc:        ethAcc,
						EthChainIDInt: deps.EvmKeeper.EthChainID(deps.Ctx()),
						GasPrice:      big.NewInt(1),
						Nonce:         deps.NewStateDB().GetNonce(ethAcc.EthAddr),
						GasLimit:      gasLimit,
					},
				})
				s.Require().NoError(err)
				s.Require().NoError(ethTxMsg.ValidateBasic())
				s.Equal(ethTxMsg.GetGas(), gasLimit.Uint64())

				resp, err := deps.App.EvmKeeper.EthereumTx(sdk.WrapSDKContext(deps.Ctx()), ethTxMsg)
				s.Require().NoError(
					err,
					"resp: %s\nblock header: %s",
					resp,
					deps.Ctx().BlockHeader().ProposerAddress,
				)
				s.Require().Empty(resp.VmError)

				// Event "EventContractDeployed" must present
				testutil.RequireContainsTypedEvent(
					s.T(),
					deps.Ctx(),
					&evm.EventContractDeployed{
						Sender:       ethAcc.EthAddr.String(),
						ContractAddr: resp.Logs[0].Address,
					},
				)
			},
		},
		{
			name: "sad: deploy contract, exceed gas limit",
			scenario: func() {
				deps := evmtest.NewTestDeps()
				ethAcc := deps.Sender

				s.T().Log("create eth tx msg, default create contract gas")
				gasLimit := gethparams.TxGasContractCreation
				ethTxMsg, err := evmtest.NewMsgEthereumTx(evmtest.ArgsEthTx{
					CreateContract: &evmtest.ArgsCreateContract{
						EthAcc:        ethAcc,
						EthChainIDInt: deps.EvmKeeper.EthChainID(deps.Ctx()),
						GasPrice:      big.NewInt(1),
						Nonce:         deps.NewStateDB().GetNonce(ethAcc.EthAddr),
					},
				})
				s.Require().NotNilf(ethTxMsg, "err: %v", err)
				s.NoError(err)
				s.Require().NoError(ethTxMsg.ValidateBasic())
				s.Equal(ethTxMsg.GetGas(), gasLimit)

				resp, err := deps.App.EvmKeeper.EthereumTx(sdk.WrapSDKContext(deps.Ctx()), ethTxMsg)
				s.Require().ErrorContains(
					err, core.ErrIntrinsicGas.Error(),
					"resp: %+v", resp,
				)
				s.Require().Nilf(resp, "err: %v", err)
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, tc.scenario)
	}
}

func (s *Suite) TestMsgEthereumTx_ExecuteContract() {
	deps := evmtest.NewTestDeps()
	ethAcc := deps.Sender

	// Leftover gas fee is refunded within EthereumTx from the FeeCollector
	// so, the module must have some coins
	err := testapp.FundModuleAccount(
		deps.App.BankKeeper,
		deps.Ctx(),
		authtypes.FeeCollectorName,
		sdk.NewCoins(sdk.NewCoin("unibi", sdkmath.NewInt(1000_000))),
	)
	s.Require().NoError(err)
	deployResp, err := evmtest.DeployContract(
		&deps, embeds.SmartContract_TestERC20,
	)
	s.Require().NoError(err)
	contractAddr := deployResp.ContractAddr
	testContract := embeds.SmartContract_TestERC20
	to := gethcommon.HexToAddress("0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed")
	input, err := testContract.ABI.Pack("transfer", to, big.NewInt(123))
	s.NoError(err)

	gasLimit := big.NewInt(1000_000)
	argsEthTx := evmtest.ArgsEthTx{
		ExecuteContract: &evmtest.ArgsExecuteContract{
			EthAcc:          ethAcc,
			EthChainIDInt:   deps.EvmKeeper.EthChainID(deps.Ctx()),
			GasPrice:        big.NewInt(1),
			Nonce:           deps.NewStateDB().GetNonce(ethAcc.EthAddr),
			GasLimit:        gasLimit,
			ContractAddress: &contractAddr,
			Data:            input,
		},
	}
	ethTxMsg, err := evmtest.NewMsgEthereumTx(argsEthTx)
	s.NoError(err)
	s.Require().NoError(ethTxMsg.ValidateBasic())
	s.Equal(ethTxMsg.GetGas(), gasLimit.Uint64())
	resp, err := deps.App.EvmKeeper.EthereumTx(sdk.WrapSDKContext(deps.Ctx()), ethTxMsg)
	s.Require().NoError(
		err,
		"resp: %s\nblock header: %s",
		resp,
		deps.Ctx().BlockHeader().ProposerAddress,
	)
	s.Require().Empty(resp.VmError)

	// Event "EventContractExecuted" must present
	testutil.RequireContainsTypedEvent(
		s.T(),
		deps.Ctx(),
		&evm.EventContractExecuted{
			Sender:       ethAcc.EthAddr.String(),
			ContractAddr: resp.Logs[0].Address,
		},
	)
}

func (s *Suite) TestMsgEthereumTx_SimpleTransfer() {
	testCases := []struct {
		name   string
		txType evmtest.GethTxType
	}{
		{
			name:   "happy: AccessListTx",
			txType: gethcore.AccessListTxType,
		},
		{
			name:   "happy: LegacyTx",
			txType: gethcore.LegacyTxType,
		},
	}

	for _, tc := range testCases {
		deps := evmtest.NewTestDeps()
		ethAcc := deps.Sender

		fundedAmount := evm.NativeToWei(big.NewInt(123)).Int64()
		err := testapp.FundAccount(
			deps.App.BankKeeper,
			deps.Ctx(),
			deps.Sender.NibiruAddr,
			sdk.NewCoins(sdk.NewInt64Coin("unibi", fundedAmount)),
		)
		s.Require().NoError(err)

		s.T().Log("create eth tx msg")
		var innerTxData []byte = nil
		var accessList gethcore.AccessList = nil
		to := gethcommon.HexToAddress("0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed")

		ethTxMsg, err := evmtest.NewEthTxMsgFromTxData(
			&deps,
			tc.txType,
			innerTxData,
			deps.NewStateDB().GetNonce(ethAcc.EthAddr),
			&to,
			big.NewInt(fundedAmount),
			gethparams.TxGas,
			accessList,
		)
		s.NoError(err)

		resp, err := deps.App.EvmKeeper.EthereumTx(sdk.WrapSDKContext(deps.Ctx()), ethTxMsg)
		s.Require().NoError(err)
		s.Require().Empty(resp.VmError)

		gasUsed := strconv.FormatUint(resp.GasUsed, 10)
		wantGasUsed := strconv.FormatUint(gethparams.TxGas, 10)
		s.Equal(gasUsed, wantGasUsed)

		// Event "EventTransfer" must present
		testutil.RequireContainsTypedEvent(
			s.T(),
			deps.Ctx(),
			&evm.EventTransfer{
				Sender:    ethAcc.EthAddr.String(),
				Recipient: to.String(),
				Amount:    strconv.FormatInt(fundedAmount, 10),
			},
		)
	}
}

// The following zero-gas tests exercise only the EthereumTx msg_server. They do not
// run the ante handler. The context is given the same zero-gas marker (CtxKeyZeroGasMeta)
// that the ante would set in production. We verify that the msg_server skips RefundGas
// when that marker is present, as it would after ante ran in a real DeliverTx.

// TestMsgEthereumTx_ZeroGas verifies the msg_server bypass. When the context carries
// the zero-gas marker (as it would after the ante handler ran), RefundGas is skipped.
// This test does not run the ante. It injects the marker to simulate that. Sender with
// zero balance can execute a zero-value transfer. We assert no deduction and no refund
// (fee collector and sender stay at 0).
func (s *Suite) TestMsgEthereumTx_ZeroGas() {
	deps := evmtest.NewTestDeps()
	ethAcc := deps.Sender

	nonce := deps.EvmKeeper.GetAccNonce(deps.Ctx(), ethAcc.EthAddr)
	to := gethcommon.HexToAddress("0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed")
	ethTxMsg, err := evmtest.NewEthTxMsgFromTxData(
		&deps,
		gethcore.LegacyTxType,
		nil,
		nonce,
		&to,
		big.NewInt(0),
		gethparams.TxGas,
		nil,
	)
	s.Require().NoError(err)

	// Simulate ante having run. Same marker the ante sets, so the msg_server sees zero-gas and skips RefundGas.
	ctxWithMeta := deps.Ctx().WithValue(evm.CtxKeyZeroGasMeta, &evm.ZeroGasMeta{})

	resp, err := deps.App.EvmKeeper.EthereumTx(sdk.WrapSDKContext(ctxWithMeta), ethTxMsg)
	s.Require().NoError(err)
	s.Require().Empty(resp.VmError)

	txHash := gethcommon.HexToHash(ethTxMsg.Hash)
	sdbAfter := deps.EvmKeeper.NewSDB(
		deps.Ctx(),
		deps.EvmKeeper.TxConfig(deps.Ctx(), txHash),
	)
	feeCollectorBal := sdbAfter.GetBalance(evm.FEE_COLLECTOR_ADDR)
	senderBal := sdbAfter.GetBalance(ethAcc.EthAddr)

	// Bypass: no deduction, no refund. Fee collector and sender stay at 0.
	s.Require().Equal(0, feeCollectorBal.Cmp(uint256.NewInt(0)), "fee collector balance should be 0")
	s.Require().Equal(0, senderBal.Cmp(uint256.NewInt(0)), "sender balance should be 0")
}

// TestMsgEthereumTx_ZeroGas_WithRefund is like TestMsgEthereumTx_ZeroGas but uses a
// higher gas limit. With bypass, RefundGas is skipped so no refund occurs. Balances stay at 0.
// Same setup: msg_server only, zero-gas marker injected in context (ante not run).
func (s *Suite) TestMsgEthereumTx_ZeroGas_WithRefund() {
	deps := evmtest.NewTestDeps()
	ethAcc := deps.Sender

	gasLimit := uint64(100_000) // simple transfer uses 21000, remainder would be refunded but we skip
	nonce := deps.EvmKeeper.GetAccNonce(deps.Ctx(), ethAcc.EthAddr)
	to := gethcommon.HexToAddress("0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed")
	ethTxMsg, err := evmtest.NewEthTxMsgFromTxData(
		&deps,
		gethcore.LegacyTxType,
		nil,
		nonce,
		&to,
		big.NewInt(0),
		gasLimit,
		nil,
	)
	s.Require().NoError(err)

	// Simulate ante having run. Same marker the ante sets, so the msg_server sees zero-gas and skips RefundGas.
	ctxWithMeta := deps.Ctx().WithValue(evm.CtxKeyZeroGasMeta, &evm.ZeroGasMeta{})

	resp, err := deps.App.EvmKeeper.EthereumTx(sdk.WrapSDKContext(ctxWithMeta), ethTxMsg)
	s.Require().NoError(err)
	s.Require().Empty(resp.VmError)
	s.Require().Equal(uint64(21_000), resp.GasUsed, "simple transfer uses 21000 gas")

	txHash := gethcommon.HexToHash(ethTxMsg.Hash)
	sdbAfter := deps.EvmKeeper.NewSDB(
		deps.Ctx(),
		deps.EvmKeeper.TxConfig(deps.Ctx(), txHash),
	)
	feeCollectorBal := sdbAfter.GetBalance(evm.FEE_COLLECTOR_ADDR)
	senderBal := sdbAfter.GetBalance(ethAcc.EthAddr)

	s.Require().Equal(0, feeCollectorBal.Cmp(uint256.NewInt(0)), "fee collector balance should be 0")
	s.Require().Equal(0, senderBal.Cmp(uint256.NewInt(0)), "sender balance should be 0")
}

// TestMsgEthereumTx_ZeroGas_Reverted ensures zero-gas bypass works on reverted execution.
// No deduction, no refund. Fee collector unchanged from deploy, sender stays at 0.
// Context is given the zero-gas marker so we test msg_server behavior without running ante.
func (s *Suite) TestMsgEthereumTx_ZeroGas_Reverted() {
	deps := evmtest.NewTestDeps()
	ethAcc := deps.Sender

	err := testapp.FundModuleAccount(
		deps.App.BankKeeper,
		deps.Ctx(),
		authtypes.FeeCollectorName,
		sdk.NewCoins(sdk.NewCoin("unibi", sdkmath.NewInt(1000_000))),
	)
	s.Require().NoError(err)
	deployResp, err := evmtest.DeployContract(
		&deps, embeds.SmartContract_TestERC20,
	)
	s.Require().NoError(err)
	contractAddr := deployResp.ContractAddr
	testContract := embeds.SmartContract_TestERC20

	sdbAfterDeploy := deps.EvmKeeper.NewSDB(deps.Ctx(), deps.EvmKeeper.TxConfig(deps.Ctx(), gethcommon.Hash{}))
	feeCollectorBalAfterDeploy := sdbAfterDeploy.GetBalance(evm.FEE_COLLECTOR_ADDR)

	to := gethcommon.HexToAddress("0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed")
	amountExceedsBalance := new(big.Int).Exp(big.NewInt(10), big.NewInt(30), nil)
	input, err := testContract.ABI.Pack("transfer", to, amountExceedsBalance)
	s.Require().NoError(err)

	gasLimit := uint64(200_000)
	ethTxMsg, err := evmtest.NewMsgEthereumTx(evmtest.ArgsEthTx{
		ExecuteContract: &evmtest.ArgsExecuteContract{
			EthAcc:          ethAcc,
			EthChainIDInt:   deps.EvmKeeper.EthChainID(deps.Ctx()),
			GasPrice:        big.NewInt(1),
			Nonce:           deps.EvmKeeper.GetAccNonce(deps.Ctx(), ethAcc.EthAddr),
			GasLimit:        big.NewInt(int64(gasLimit)),
			ContractAddress: &contractAddr,
			Data:            input,
		},
	})
	s.Require().NoError(err)
	s.Require().NoError(ethTxMsg.ValidateBasic())

	// Simulate ante having run. Same marker the ante sets, so the msg_server sees zero-gas and skips RefundGas.
	ctxWithMeta := deps.Ctx().WithValue(evm.CtxKeyZeroGasMeta, &evm.ZeroGasMeta{})

	resp, err := deps.App.EvmKeeper.EthereumTx(sdk.WrapSDKContext(ctxWithMeta), ethTxMsg)
	s.Require().NoError(err, "handler must not return error on revert")
	s.Require().NotNil(resp)
	s.Require().True(resp.Failed(), "EVM execution must revert")
	s.Require().NotEmpty(resp.VmError)
	s.Contains(resp.VmError, "execution reverted")
	s.Require().NotZero(resp.GasUsed, "gas must be consumed on reverted execution")

	txHash := gethcommon.HexToHash(ethTxMsg.Hash)
	sdbAfter := deps.EvmKeeper.NewSDB(
		deps.Ctx(),
		deps.EvmKeeper.TxConfig(deps.Ctx(), txHash),
	)
	feeCollectorBal := sdbAfter.GetBalance(evm.FEE_COLLECTOR_ADDR)
	senderBal := sdbAfter.GetBalance(ethAcc.EthAddr)
	s.Require().Equal(0, feeCollectorBal.Cmp(feeCollectorBalAfterDeploy), "fee collector should be unchanged (bypass: no deduction for zero-gas tx)")
	s.Require().Equal(0, senderBal.Cmp(uint256.NewInt(0)), "sender balance should be 0")
}

func (s *Suite) TestEthereumTx_ABCI() {
	deps := evmtest.NewTestDeps()
	s.Require().NoError(testapp.FundAccount(
		deps.App.BankKeeper,
		deps.Ctx(),
		deps.Sender.NibiruAddr,
		sdk.NewCoins(sdk.NewCoin(evm.EVMBankDenom, sdk.NewInt(69_420))),
	))

	blockHeader := deps.Ctx().BlockHeader()
	// blockHeader := tmproto.Header{Height: deps.Ctx().BlockHeight()}
	deps.App.BeginBlock(abci.RequestBeginBlock{Header: blockHeader})
	to := evmtest.NewEthPrivAcc()
	evmTxMsg, err := evmtest.TxTransferWei{
		Deps:      &deps,
		To:        to.EthAddr,
		AmountWei: evm.NativeToWei(big.NewInt(420)),
	}.Build()
	s.NoError(err)

	txBuilder := deps.App.GetTxConfig().NewTxBuilder()
	blockTx, err := evmTxMsg.BuildTx(txBuilder, evm.EVMBankDenom)
	s.Require().NoError(err)

	txBz, err := deps.App.GetTxConfig().TxEncoder()(blockTx)
	s.Require().NoError(err)
	deliverTxResp := deps.App.DeliverTx(abci.RequestDeliverTx{Tx: txBz})
	s.Require().True(deliverTxResp.IsOK(), "%#v", deliverTxResp)
	deps.App.EndBlock(abci.RequestEndBlock{Height: deps.Ctx().BlockHeight()})

	{
		r := deliverTxResp
		s.EqualValuesf(21000, r.GasUsed, `gasUsed="%d", resp: %#v`, r.GasUsed, r)
		s.EqualValuesf(21000, r.GasWanted, `gasWanted="%d", resp: %#v`, r.GasWanted, r)
	}
	// Normal EVM tx (not zero-gas): context must not have ZeroGasMeta set.
	s.Require().False(evm.IsZeroGasEthTx(deps.Ctx()), "IsZeroGasEthTx should be false for normal EVM tx")
}
