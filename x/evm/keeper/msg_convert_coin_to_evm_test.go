// Copyright (c) 2023-2024 Nibi, Inc.
package keeper_test

import (
	"math/big"

	sdkmath "cosmossdk.io/math"
	"github.com/MakeNowJust/heredoc/v2"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/evm/precompile"
)

func (s *SuiteFunToken) TestConvertCoinToEvmAndBack() {
	deps := evmtest.NewTestDeps()
	evmObj, _ := deps.NewEVM()
	alice := evmtest.NewEthPrivAcc()

	// Initial setup
	funToken := s.fundAndCreateFunToken(deps, 100)

	s.T().Log("Convert bank coin to erc-20")
	deps.Ctx = deps.Ctx.WithGasMeter(sdk.NewInfiniteGasMeter()).WithEventManager(sdk.NewEventManager())
	bankDenom := funToken.BankDenom
	_, err := deps.EvmKeeper.ConvertCoinToEvm(
		deps.GoCtx(),
		&evm.MsgConvertCoinToEvm{
			Sender:   deps.Sender.NibiruAddr.String(),
			BankCoin: sdk.NewCoin(bankDenom, sdk.NewInt(10)),
			ToEthAddr: eth.EIP55Addr{
				Address: alice.EthAddr,
			},
		},
	)
	s.Require().NoError(err)
	s.Require().NotZero(deps.Ctx.GasMeter().GasConsumed())

	s.T().Log("Check typed event ConvertCoinToEvm")
	testutil.RequireContainsTypedEvent(
		s.T(),
		deps.Ctx,
		&evm.EventConvertCoinToEvm{
			Sender:               deps.Sender.NibiruAddr.String(),
			Erc20ContractAddress: funToken.Erc20Addr.String(),
			ToEthAddr:            alice.EthAddr.String(),
			BankCoin:             sdk.NewCoin(funToken.BankDenom, sdk.NewInt(10)),
		},
	)

	s.T().Log("Check typed event EventTxLog with Transfer event")
	emptyHash := gethcommon.BytesToHash(make([]byte, 32)).Hex()
	signature := crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)")).Hex()
	fromAddress := emptyHash // Mint
	toAddress := gethcommon.BytesToHash(alice.EthAddr.Bytes()).Hex()
	amountBase64 := gethcommon.LeftPadBytes(big.NewInt(10).Bytes(), 32)

	testutil.RequireContainsTypedEvent(
		s.T(),
		deps.Ctx,
		&evm.EventTxLog{
			Logs: []evm.Log{
				{
					Address: funToken.Erc20Addr.Hex(),
					Topics: []string{
						signature,
						fromAddress,
						toAddress,
					},
					Data:        amountBase64,
					BlockNumber: 1, // we are in simulation, no real block numbers or tx hashes
					TxHash:      emptyHash,
					TxIndex:     0,
					BlockHash:   emptyHash,
					Index:       1,
					Removed:     false,
				},
			},
		},
	)

	// Check 1: module balance
	moduleBalance := deps.App.BankKeeper.GetBalance(deps.Ctx, authtypes.NewModuleAddress(evm.ModuleName), funToken.BankDenom)
	s.Require().Equal(sdk.NewInt(10), moduleBalance.Amount)

	// Check 2: Sender balance
	senderBalance := deps.App.BankKeeper.GetBalance(deps.Ctx, deps.Sender.NibiruAddr, funToken.BankDenom)
	s.Require().Equal(sdk.NewInt(90), senderBalance.Amount)

	// Check 3: erc-20 balance
	balance, err := deps.EvmKeeper.ERC20().BalanceOf(funToken.Erc20Addr.Address, alice.EthAddr, deps.Ctx, evmObj)
	s.Require().NoError(err)
	s.Require().Zero(balance.Cmp(big.NewInt(10)))

	s.Run("sad: Convert more bank coin to erc-20, insufficient funds", func() {
		_, err = deps.EvmKeeper.ConvertCoinToEvm(
			deps.GoCtx(),
			&evm.MsgConvertCoinToEvm{
				Sender:   deps.Sender.NibiruAddr.String(),
				BankCoin: sdk.NewCoin(funToken.BankDenom, sdk.NewInt(100)),
				ToEthAddr: eth.EIP55Addr{
					Address: alice.EthAddr,
				},
			},
		)
		s.Require().ErrorContains(err, "insufficient funds")
	})

	s.T().Log("Convert erc-20 to back to bank coin")
	contractInput, err := embeds.SmartContract_FunToken.ABI.Pack(
		"sendToBank",
		funToken.Erc20Addr.Address,
		big.NewInt(10),
		deps.Sender.NibiruAddr.String(),
	)
	s.Require().NoError(err)
	deps.Ctx = deps.Ctx.WithGasMeter(sdk.NewInfiniteGasMeter())
	evmObj, _ = deps.NewEVM()
	_, err = deps.EvmKeeper.CallContractWithInput(
		deps.Ctx,
		evmObj,
		alice.EthAddr,                       // from
		&precompile.PrecompileAddr_FunToken, // to
		true,                                // commit
		contractInput,
		evmtest.FunTokenGasLimitSendToEvm,
		nil,
	)
	s.Require().NoError(err)
	s.Require().NotZero(deps.Ctx.GasMeter().GasConsumed())

	// Check 1: module balance
	moduleBalance = deps.App.BankKeeper.GetBalance(deps.Ctx, authtypes.NewModuleAddress(evm.ModuleName), funToken.BankDenom)
	s.Require().True(moduleBalance.Amount.Equal(sdk.ZeroInt()))

	// Check 2: Sender balance
	senderBalance = deps.App.BankKeeper.GetBalance(deps.Ctx, deps.Sender.NibiruAddr, funToken.BankDenom)
	s.Require().Equal(sdk.NewInt(100), senderBalance.Amount)

	// Check 3: erc-20 balance
	balance, err = deps.EvmKeeper.ERC20().BalanceOf(funToken.Erc20Addr.Address, alice.EthAddr, deps.Ctx, evmObj)
	s.Require().NoError(err)
	s.Require().Equal("0", balance.String())

	s.T().Log("sad: Convert more erc-20 to back to bank coin, insufficient funds")
	deps.Ctx = deps.Ctx.WithGasMeter(sdk.NewInfiniteGasMeter())
	evmObj, _ = deps.NewEVM()
	_, err = deps.EvmKeeper.CallContractWithInput(
		deps.Ctx,
		evmObj,
		alice.EthAddr,                       // from
		&precompile.PrecompileAddr_FunToken, // to
		true,                                // commit
		contractInput,
		evmtest.FunTokenGasLimitSendToEvm,
		nil,
	)
	s.Require().ErrorContains(err, "transfer amount exceeds balance")
	s.Require().NotZero(deps.Ctx.GasMeter().GasConsumed())
}

// TestNativeSendThenPrecompileSend tests a race condition where the state DB
// commit may overwrite the state after the precompile execution, potentially
// causing a loss of funds.
//
// The order of operations is to:
//  1. Create a funtoken mapping from "testfuntoken", a bank coin.
//  2. Use a test Solidity contract to perform two transfers in a single call: a
//     transfer of NIBI with native send and a precompile "IFunToken.sendToBank"
//     transfer for the same asset.
//
// INITIAL STATE: BC means Bank Coin, EVM means ERC20
//   - Test contract funds: 10 BC, 10 EVM
//
// CONTRACT CALL:
//   - Sends 10 BC natively and 10 EVM -> BC to Alice using precompile
//
// EXPECTED:
//   - Test contract funds: 0 BC, 0 EVM
//   - Alice: 20 BC
//   - Module account: 0 BC escrowed
func (s *SuiteFunToken) TestNativeSendThenPrecompileSend() {
	deps := evmtest.NewTestDeps()
	err := deps.DeployWNIBI(&s.Suite)
	s.Require().NoError(err)
	evmObj, _ := deps.NewEVM()

	// Initial setup
	var (
		sendAmt = big.NewInt(10)
		// Amount passed as an argument for "IFunToken.sendToBank" inside the
		// contract logic of NativeSendThenPrecompileSend.
		newSendAmtSendToBank  = big.NewInt(5)
		newSendAmtEvmTransfer = evm.NativeToWei(newSendAmtSendToBank)

		// A fungible token mapping TEST, not NIBI.
		funtoken = s.fundAndCreateFunToken(deps, sendAmt.Int64())

		// Bank coin denom of the FunToken mapping used in the test.
		bankDenom = funtoken.BankDenom
	)

	s.T().Log("Deploy Test Contract")
	deployResp, err := evmtest.DeployContract(
		&deps,
		embeds.SmartContract_TestNativeSendThenPrecompileSendJson,
		funtoken.Erc20Addr.Address,
	)
	s.Require().NoError(err)

	testContractAddr := deployResp.ContractAddr
	testContractNibiAddr := eth.EthAddrToNibiruAddr(testContractAddr)

	s.T().Log("Give the test contract 10 microNIBi (native), 10 BC (TEST)")
	s.Require().NoError(
		testapp.FundAccount(
			deps.App.BankKeeper,
			deps.Ctx,
			testContractNibiAddr,
			sdk.NewCoins(
				sdk.NewCoin(evm.EVMBankDenom, sdk.NewIntFromBigInt(sendAmt)),
				sdk.NewCoin(bankDenom, sdk.NewIntFromBigInt(sendAmt)),
			),
		),
	)
	evmtest.AssertBankBalanceEqualWithDescription(
		s.T(), deps, bankDenom, testContractAddr, sendAmt, "expect 10 balance",
	)
	evmtest.AssertBankBalanceEqualWithDescription(
		s.T(), deps, evm.EVMBankDenom, testContractAddr, sendAmt, "expect 10 balance",
	)
	evmtest.AssertBankBalanceEqualWithDescription(
		s.T(), deps, bankDenom, evm.EVM_MODULE_ADDRESS, big.NewInt(0), "expect 0 balance",
	)
	evmtest.AssertBankBalanceEqualWithDescription(
		s.T(), deps, evm.EVMBankDenom, evm.EVM_MODULE_ADDRESS, big.NewInt(0), "expect 0 balance",
	)

	s.T().Log("Convert bank coin to erc-20: give test contract 10 ERC20s")
	_, err = deps.EvmKeeper.ConvertCoinToEvm(
		deps.GoCtx(),
		&evm.MsgConvertCoinToEvm{
			Sender:    deps.Sender.NibiruAddr.String(),
			BankCoin:  sdk.NewCoin(bankDenom, sdk.NewIntFromBigInt(sendAmt)),
			ToEthAddr: eth.EIP55Addr{Address: testContractAddr},
		},
	)
	s.Require().NoError(err)
	evmtest.FunTokenBalanceAssert{
		FunToken:     funtoken,
		Account:      testContractAddr,
		BalanceBank:  sendAmt,
		BalanceERC20: sendAmt,
	}.Assert(s.T(), deps, evmObj)
	evmtest.FunTokenBalanceAssert{
		FunToken:     funtoken,
		Account:      evm.EVM_MODULE_ADDRESS,
		BalanceBank:  sendAmt,
		BalanceERC20: big.NewInt(0),
	}.Assert(s.T(), deps, evmObj)

	// Alice hex and Alice bech32 is the same address in different representation,
	// so funds are expected to be available in Alice's bank wallet
	alice := evmtest.NewEthPrivAcc()
	evmtest.FunTokenBalanceAssert{
		Account:      alice.EthAddr,
		FunToken:     funtoken,
		BalanceBank:  big.NewInt(0),
		BalanceERC20: big.NewInt(0),
		Description:  "sanity check on Alice, must start empty",
	}.Assert(s.T(), deps, evmObj)

	s.T().Log("call test contract and fund wei")

	s.Require().NoError(testapp.FundAccount(
		deps.App.BankKeeper,
		deps.Ctx,
		deps.Sender.NibiruAddr,
		sdk.NewCoins(sdk.NewCoin(
			evm.EVMBankDenom, sdkmath.NewIntFromBigInt(newSendAmtSendToBank),
		)),
	))
	senderBalMicronibi := deps.App.BankKeeper.GetBalance(deps.Ctx, deps.Sender.NibiruAddr, evm.EVMBankDenom)
	s.Require().GreaterOrEqual(
		senderBalMicronibi.Amount.BigInt().Cmp(newSendAmtSendToBank),
		0,
		"expect sender to have enough NIBI: senderBalMicronibi: %s, newSendAmtEvmTransfer: %s, newSendAmtSendToBank: %s", senderBalMicronibi, newSendAmtEvmTransfer, newSendAmtSendToBank,
	)
	evmObj, _ = deps.NewEVM()
	senderBalWei := evmObj.StateDB.GetBalance(deps.Sender.EthAddr)
	s.Require().GreaterOrEqual(
		senderBalWei.ToBig().Cmp(newSendAmtEvmTransfer),
		0,
		"expect sender to have enough NIBI: senderBalWei: %s, newSendAmtEvmTransfer: %s, newSendAmtSendToBank: %s", senderBalWei, newSendAmtEvmTransfer, newSendAmtSendToBank,
	)

	contractInput, err := embeds.SmartContract_TestNativeSendThenPrecompileSendJson.ABI.Pack(
		"nativeSendThenPrecompileSend",
		alice.EthAddr,             /*to*/
		newSendAmtEvmTransfer,     /*amount*/
		alice.NibiruAddr.String(), /*to*/
		newSendAmtSendToBank,      /*amount*/
	)
	s.Require().NoError(err)
	deps.Ctx = deps.Ctx.WithGasMeter(sdk.NewInfiniteGasMeter())
	evmObj, _ = deps.NewEVM()
	evmResp, err := deps.EvmKeeper.CallContractWithInput(
		deps.Ctx,
		evmObj,              /*evmObj*/
		deps.Sender.EthAddr, /*fromAcc*/
		&testContractAddr,   /*contract*/
		true,                /*commit*/
		contractInput,
		evmtest.FunTokenGasLimitSendToEvm,
		nil,
	)
	s.Require().NoError(err)
	s.Require().NotZero(deps.Ctx.GasMeter().GasConsumed())
	s.Require().NotZero(evmResp.GasUsed)
	s.Require().Greaterf(deps.Ctx.GasMeter().GasConsumed(), evmResp.GasUsed, "total gas consumed on cosmos context should be greater than gas used by EVM")
	s.Empty(evmResp.VmError)
	gasUsedFor2Ops := evmResp.GasUsed

	s.T().Log(heredoc.Doc(
		`Summary - Before the tx, sender has [only enough NIBI for the native send, 10 BC], while the test contract has [10 microNIBI, 10 BC]
Sender calls "nativeSendThenPrecompileSend".
- Expect 5 microNIBI to go from the sender to Alice (nativeRecipient.send)
- Expect 5 ERC20s from the testContract balance to go to Alice as BCs (FUNTOKEN_PRECOMPILE.sendToBank)
`,
	))
	// Assertions for Alice
	evmtest.FunTokenBalanceAssert{
		Account:      alice.EthAddr,
		FunToken:     funtoken,
		BalanceBank:  big.NewInt(5), // 0 -> 5 - Because testContract sends 5 to Alice by calling IFunToken.sendToBank
		BalanceERC20: big.NewInt(0), // Unaffected
	}.Assert(s.T(), deps, evmObj)
	evmtest.BalanceAssertNIBI{
		Account:      alice.EthAddr,
		BalanceBank:  big.NewInt(5), // 0 -> 5 - Because of native send from testContract to Alice
		BalanceERC20: big.NewInt(0), // Unaffected
		EvmObj:       evmObj,
	}.Assert(s.T(), deps)

	// Assertions for testContract
	evmtest.FunTokenBalanceAssert{
		Account:      testContractAddr,
		FunToken:     funtoken,
		BalanceBank:  big.NewInt(10), // Unaffected
		BalanceERC20: big.NewInt(5),  // 10 -> 5 - Because testContract sends 5 to Alice by calling IFunToken.sendToBank
	}.Assert(s.T(), deps, evmObj)
	evmtest.BalanceAssertNIBI{
		Account:      testContractAddr,
		BalanceBank:  big.NewInt(5), // 10 -> 5 - Because of native send from testContract to Alice
		BalanceERC20: big.NewInt(0), // Unaffected
	}.Assert(s.T(), deps)

	// Assertions for EVM Module
	evmtest.FunTokenBalanceAssert{
		Account:  evm.EVM_MODULE_ADDRESS,
		FunToken: funtoken,
		// 10 -> 5 - Because calling IFunToken.sendToBank removes tokens from
		// escrow for a coin-originated FunToken.
		BalanceBank:  big.NewInt(5),
		BalanceERC20: big.NewInt(0), // Unaffected
	}.Assert(s.T(), deps, evmObj)
	evmtest.BalanceAssertNIBI{
		Account:      evm.EVM_MODULE_ADDRESS,
		BalanceBank:  big.NewInt(0), // Unaffected
		BalanceERC20: big.NewInt(0), // Unaffected
		EvmObj:       evmObj,
	}.Assert(s.T(), deps)

	contractInput, err = embeds.SmartContract_TestNativeSendThenPrecompileSendJson.ABI.Pack(
		"justPrecompileSend",
		alice.NibiruAddr.String(), /*to*/
		newSendAmtSendToBank,      /*amount*/
	)
	s.Require().NoError(err)
	deps.Ctx = deps.Ctx.WithGasMeter(sdk.NewInfiniteGasMeter())
	evmObj, _ = deps.NewEVM()
	evmResp, err = deps.EvmKeeper.CallContractWithInput(
		deps.Ctx,
		evmObj,
		deps.Sender.EthAddr,
		&testContractAddr,
		true,
		contractInput,
		evmtest.DefaultEthCallGasLimit,
		nil,
	)
	s.Require().NoError(err)
	s.Require().NotZero(deps.Ctx.GasMeter().GasConsumed())
	s.Require().NotZero(evmResp.GasUsed)
	s.Require().Greaterf(deps.Ctx.GasMeter().GasConsumed(), evmResp.GasUsed, "total gas consumed on cosmos context should be greater than gas used by EVM")
	s.Empty(evmResp.VmError)
	gasUsedFor1Op := evmResp.GasUsed

	// Assertions for Alice
	evmtest.FunTokenBalanceAssert{
		Account:      alice.EthAddr,
		FunToken:     funtoken,
		BalanceBank:  big.NewInt(10), // 5 -> 10 - Because testContract sends 5 to Alice by calling IFunToken.sendToBank
		BalanceERC20: big.NewInt(0),  // Unaffected
	}.Assert(s.T(), deps, evmObj)
	evmtest.BalanceAssertNIBI{
		Account:      alice.EthAddr,
		BalanceBank:  big.NewInt(5), // Unaffected
		BalanceERC20: big.NewInt(0), // Unaffected
		EvmObj:       evmObj,
	}.Assert(s.T(), deps)

	// Assertions for testContract
	evmtest.FunTokenBalanceAssert{
		Account:      testContractAddr,
		FunToken:     funtoken,
		BalanceBank:  big.NewInt(10), // Unaffected
		BalanceERC20: big.NewInt(0),  // 5 -> 0 - Because testContract sends 5 to Alice by calling IFunToken.sendToBank
	}.Assert(s.T(), deps, evmObj)
	evmtest.BalanceAssertNIBI{
		Account:      testContractAddr,
		BalanceBank:  big.NewInt(5), // Unaffected
		BalanceERC20: big.NewInt(0), // Unaffected
		EvmObj:       evmObj,
	}.Assert(s.T(), deps)

	// Assertions for EVM Module
	evmtest.FunTokenBalanceAssert{
		Account:  evm.EVM_MODULE_ADDRESS,
		FunToken: funtoken,
		// 5 -> 0 - Because calling IFunToken.sendToBank removes tokens from
		// escrow for a coin-originated FunToken.
		BalanceBank:  big.NewInt(0),
		BalanceERC20: big.NewInt(0), // Unaffected
	}.Assert(s.T(), deps, evmObj)
	evmtest.BalanceAssertNIBI{
		Account:      evm.EVM_MODULE_ADDRESS,
		BalanceBank:  big.NewInt(0), // Unaffected
		BalanceERC20: big.NewInt(0), // Unaffected
		EvmObj:       evmObj,
	}.Assert(s.T(), deps)

	s.Require().Greater(gasUsedFor2Ops, gasUsedFor1Op, "2 operations should consume more gas")
}

// TestPrecompileSendToBankThenErc20Transfer
//  1. Creates a funtoken from coin.
//  2. Calls the test contract
//     a. sendToBank
//     b. erc20 transfer
//
// INITIAL STATE:
//   - Test contract funds: 10 ERC20
//
// CONTRACT CALL:
//   - Sends 10 ERC20 to Alice, and try to send 1 BC to Bob
//
// EXPECTED:
//   - all changes reverted because of not enough balance
//   - Test contract funds: 10 ERC20
//   - Alice: 10 ERC20
//   - Bob: 0 BC
//   - Module account: 10 BC escrowed (which Test contract holds as 10 ERC20)
func (s *SuiteFunToken) TestPrecompileSendToBankThenErc20Transfer() {
	deps := evmtest.NewTestDeps()

	// Initial setup
	funToken := s.fundAndCreateFunToken(deps, 10e6)

	s.T().Log("Deploy Test Contract")
	deployResp, err := evmtest.DeployContract(
		&deps,
		embeds.SmartContract_TestPrecompileSendToBankThenERC20Transfer,
		funToken.Erc20Addr.Address,
		deps.Sender.NibiruAddr.String(),
	)
	s.Require().NoError(err)
	testContractAddr := deployResp.ContractAddr

	s.T().Log("Convert bank coin to erc-20: give test contract 10 WNIBI (erc20)")
	_, err = deps.EvmKeeper.ConvertCoinToEvm(
		deps.GoCtx(),
		&evm.MsgConvertCoinToEvm{
			Sender:    deps.Sender.NibiruAddr.String(),
			BankCoin:  sdk.NewCoin(funToken.BankDenom, sdk.NewInt(10e6)),
			ToEthAddr: eth.EIP55Addr{Address: testContractAddr},
		},
	)
	s.Require().NoError(err)

	// Create Alice and Bob. Contract will try to send Alice native coins and
	// send Bob ERC20 tokens.
	alice := evmtest.NewEthPrivAcc()
	bob := evmtest.NewEthPrivAcc()

	s.T().Log("call test contract")
	contractInput, err := embeds.SmartContract_TestPrecompileSendToBankThenERC20Transfer.ABI.Pack(
		"attack",
	)
	s.Require().NoError(err)
	deps.Ctx = deps.Ctx.WithGasMeter(sdk.NewInfiniteGasMeter())
	evmObj, _ := deps.NewEVM()
	evpResp, err := deps.EvmKeeper.CallContractWithInput(
		deps.Ctx,
		evmObj,
		deps.Sender.EthAddr,
		&testContractAddr,
		true,
		contractInput,
		evmtest.FunTokenGasLimitSendToEvm,
		nil,
	)
	s.Require().ErrorContains(err, "execution reverted")
	s.Require().NotZero(deps.Ctx.GasMeter().GasConsumed())
	s.Require().NotZero(evpResp.GasUsed)
	s.Require().Greaterf(deps.Ctx.GasMeter().GasConsumed(), evpResp.GasUsed, "total gas consumed on cosmos context should be greater than gas used by EVM")

	evmtest.FunTokenBalanceAssert{
		FunToken:     funToken,
		Account:      alice.EthAddr,
		BalanceBank:  big.NewInt(0),
		BalanceERC20: big.NewInt(0),
		Description:  "Alice has 0 NIBI / 0 WNIBI",
	}.Assert(s.T(), deps, evmObj)

	evmtest.FunTokenBalanceAssert{
		FunToken:     funToken,
		Account:      bob.EthAddr,
		BalanceBank:  big.NewInt(0),
		BalanceERC20: big.NewInt(0),
		Description:  "Bob has 0 NIBI / 0 WNIBI",
	}.Assert(s.T(), deps, evmObj)

	evmtest.FunTokenBalanceAssert{
		FunToken:     funToken,
		Account:      testContractAddr,
		BalanceBank:  big.NewInt(0),
		BalanceERC20: big.NewInt(10e6),
		Description:  "Test contract has 10 NIBI / 10 WNIBI",
	}.Assert(s.T(), deps, evmObj)

	evmtest.FunTokenBalanceAssert{
		FunToken:     funToken,
		Account:      evm.EVM_MODULE_ADDRESS,
		BalanceBank:  big.NewInt(10e6),
		BalanceERC20: big.NewInt(0),
		Description:  "Module account has 10 NIBI escrowed",
	}.Assert(s.T(), deps, evmObj)
}

// TestPrecompileSelfCallRevert
//  1. Creates a funtoken from coin.
//  2. Using the test contract, creates another instance of itself, calls the precompile method and then force reverts.
//     It tests a race condition where the state DB commit
//     may save the wrong state before the precompile execution, not revert it entirely,
//     potentially causing an infinite mint of funds.
//
// INITIAL STATE: BC means Bank Coin, E20 means ERC20
//   - Test contract funds: 10 BC, 10 E20
//
// CONTRACT CALL:
//   - Sends 1 BC to Alice using native send and 1 E20 -> BC to Charles using precompile
//
// EXPECTED:
//   - all changes reverted
//   - Test contract funds: 10 BC, 10 E20
//   - Alice: 0 BC
//   - Charles: 0 BC
//   - Module account: 10 BC escrowed (which Test contract holds as 10 E20)
func (s *SuiteFunToken) TestPrecompileSelfCallRevert() {
	deps := evmtest.NewTestDeps()
	err := deps.DeployWNIBI(&s.Suite)
	s.Require().NoError(err)

	// Initial setup
	funToken := s.fundAndCreateFunToken(deps, 10e6)

	s.T().Log("Deploy Test Contract")
	deployResp, err := evmtest.DeployContract(
		&deps,
		embeds.SmartContract_TestPrecompileSelfCallRevert,
		funToken.Erc20Addr.Address,
	)
	s.Require().NoError(err)
	testContractAddr := deployResp.ContractAddr

	s.T().Log("Convert bank coin to erc-20: give test contract 10 ERC20")
	_, err = deps.EvmKeeper.ConvertCoinToEvm(
		deps.GoCtx(),
		&evm.MsgConvertCoinToEvm{
			Sender:    deps.Sender.NibiruAddr.String(),
			BankCoin:  sdk.NewCoin(funToken.BankDenom, sdk.NewInt(10e6)),
			ToEthAddr: eth.EIP55Addr{Address: testContractAddr},
		},
	)
	s.Require().NoError(err)

	s.T().Log("Give the test contract 10 BC (native)")
	s.Require().NoError(testapp.FundAccount(
		deps.App.BankKeeper,
		deps.Ctx,
		eth.EthAddrToNibiruAddr(testContractAddr),
		sdk.NewCoins(sdk.NewCoin(funToken.BankDenom, sdk.NewInt(10e6))),
	))

	evmObj, _ := deps.NewEVM()
	evmtest.FunTokenBalanceAssert{
		FunToken:     funToken,
		Account:      testContractAddr,
		BalanceBank:  big.NewInt(10e6),
		BalanceERC20: big.NewInt(10e6),
		Description:  "Initial contract state sanity check: 10 BC, 10 ERC20",
	}.Assert(s.T(), deps, evmObj)
	evmtest.FunTokenBalanceAssert{
		FunToken:     funToken,
		Account:      deps.Sender.EthAddr,
		BalanceBank:  big.NewInt(0),
		BalanceERC20: big.NewInt(0),
		Description:  "Initial sender state sanity check: 0 BC, 0 ERC20",
	}.Assert(s.T(), deps, evmObj)

	// Create Alice and Charles. Contract will try to send Alice native coins and
	// send Charles tokens via sendToBank
	alice := evmtest.NewEthPrivAcc()
	charles := evmtest.NewEthPrivAcc()

	s.T().Log("call test contract")
	deps.Ctx = deps.Ctx.WithGasMeter(sdk.NewInfiniteGasMeter())
	evmObj, _ = deps.NewEVM()
	contractInput, err := embeds.SmartContract_TestPrecompileSelfCallRevert.ABI.Pack(
		"selfCallTransferFunds",
		alice.EthAddr,
		evm.NativeToWei(big.NewInt(1e6)),
		charles.NibiruAddr.String(),
		big.NewInt(9e6),
	)
	s.Require().NoError(err)
	evpResp, err := deps.EvmKeeper.CallContractWithInput(
		deps.Ctx,
		evmObj,
		deps.Sender.EthAddr,
		&testContractAddr,
		true,
		contractInput,
		evmtest.FunTokenGasLimitSendToEvm,
		nil,
	)
	s.Require().NoError(err)
	s.Require().NotZero(deps.Ctx.GasMeter().GasConsumed())
	s.Require().NotZero(evpResp.GasUsed)
	s.Require().Greaterf(deps.Ctx.GasMeter().GasConsumed(), evpResp.GasUsed, "total gas consumed on cosmos context should be greater than gas used by EVM")

	evmtest.FunTokenBalanceAssert{
		FunToken:     funToken,
		Account:      alice.EthAddr,
		BalanceBank:  big.NewInt(0),
		BalanceERC20: big.NewInt(0),
		Description:  "Alice has 0 BC, 0 ERC20",
	}.Assert(s.T(), deps, evmObj)

	evmtest.FunTokenBalanceAssert{
		FunToken:     funToken,
		Account:      charles.EthAddr,
		BalanceBank:  big.NewInt(0),
		BalanceERC20: big.NewInt(0),
		Description:  "Charles has 0 BC, 0 ERC20",
	}.Assert(s.T(), deps, evmObj)

	evmtest.FunTokenBalanceAssert{
		FunToken:     funToken,
		Account:      testContractAddr,
		BalanceBank:  big.NewInt(10e6),
		BalanceERC20: big.NewInt(10e6),
		Description:  "Test contract has 10 BC, 10 ERC20",
	}.Assert(s.T(), deps, evmObj)

	evmtest.FunTokenBalanceAssert{
		FunToken:     funToken,
		Account:      evm.EVM_MODULE_ADDRESS,
		BalanceBank:  big.NewInt(10e6),
		BalanceERC20: big.NewInt(0),
		Description:  "Module account has 10 BC escrowed",
	}.Assert(s.T(), deps, evmObj)
}

func (s *SuiteFunToken) TestConvertCoinToEvmForWNIBI() {
	var (
		// All of this test pertains only to NIBI.
		bankDenom   = evm.EVMBankDenom
		someoneElse = evmtest.NewEthPrivAcc()
	)

	unibi := func(amt *big.Int) sdk.Coin {
		return sdk.NewCoin(bankDenom, sdk.NewIntFromBigInt(amt))
	}

	s.Run("Nibiru mainnet - WNIBI live", func() {
		// Version: For versions v2.7.0+
		// On mainnet, WNIBI.sol exists already at address
		// "0x0CaCF669f8446BeCA826913a3c6B96aCD4b02a97"

		deps := evmtest.NewTestDeps()
		err := deps.DeployWNIBI(&s.Suite)
		s.Require().NoError(err)
		erc20Addr := deps.EvmKeeper.GetParams(deps.Ctx).CanonicalWnibi

		s.T().Log("happy path. Sender has NIBI and gets the correct amount")
		err = testapp.FundAccount(
			deps.App.BankKeeper,
			deps.Ctx,
			deps.Sender.NibiruAddr,
			sdk.NewCoins(unibi(big.NewInt(42069))),
		)
		s.NoError(err)

		_, err = deps.EvmKeeper.ConvertCoinToEvm(
			deps.GoCtx(),
			&evm.MsgConvertCoinToEvm{
				Sender:    deps.Sender.NibiruAddr.String(),
				BankCoin:  unibi(big.NewInt(69)),
				ToEthAddr: eth.EIP55Addr{Address: someoneElse.EthAddr},
			},
		)
		s.Require().NoError(err)

		s.T().Log("Validate total supply and balances")
		{
			evmObj, _ := deps.NewEVM()
			totalSupply, err := deps.EvmKeeper.ERC20().TotalSupply(erc20Addr.Address, deps.Ctx, evmObj)
			s.NoError(err)
			s.Equal(evm.NativeToWei(big.NewInt(69)).String(), totalSupply.String())
		}

		evmtest.BalanceAssertNIBI{
			Account:      deps.Sender.EthAddr,
			BalanceBank:  big.NewInt(42_000),
			BalanceERC20: big.NewInt(0),
		}.Assert(s.T(), deps)
		evmtest.BalanceAssertNIBI{
			Account:      someoneElse.EthAddr,
			BalanceBank:  big.NewInt(0),
			BalanceERC20: evm.NativeToWei(big.NewInt(69)),
		}.Assert(s.T(), deps)

		testutil.RequireContainsTypedEvent(
			s.T(),
			deps.Ctx,
			&evm.EventConvertCoinToEvm{
				Sender:               deps.Sender.NibiruAddr.String(),
				Erc20ContractAddress: erc20Addr.Hex(),
				ToEthAddr:            someoneElse.EthAddr.Hex(),
				BankCoin:             unibi(big.NewInt(69)),
			},
		)

		s.T().Log("sad: sender has insufficient funds, exit before EVM call")
		_, err = deps.EvmKeeper.ConvertCoinToEvm(
			deps.GoCtx(),
			&evm.MsgConvertCoinToEvm{
				Sender:    deps.Sender.NibiruAddr.String(),
				BankCoin:  unibi(big.NewInt(69420)),
				ToEthAddr: eth.EIP55Addr{Address: someoneElse.EthAddr},
			},
		)
		s.Require().ErrorContains(err, "ConvertCoinToEvm: insufficient funds to convert NIBI into WNIBI")
	})

	s.Run("WNIBI not deployed (other networks)", func() {
		deps := evmtest.NewTestDeps()

		s.T().Log("Set WNIBI as the zero address in the evm params")
		evmParams := deps.EvmKeeper.GetParams(deps.Ctx)
		zeroAddr := gethcommon.Address{}
		s.Require().Equal(
			gethcommon.BigToAddress(big.NewInt(0)),
			zeroAddr,
			"sanity check zero address as default struct",
		)
		evmParams.CanonicalWnibi = eth.EIP55Addr{Address: zeroAddr}
		s.NoError(
			deps.EvmKeeper.SetParams(deps.Ctx, evmParams),
		)

		s.T().Log("sad: try converting NIBI to WNIBI when WNIBI doesn't exist")
		err := testapp.FundAccount(
			deps.App.BankKeeper,
			deps.Ctx,
			deps.Sender.NibiruAddr,
			sdk.NewCoins(unibi(big.NewInt(42069))),
		)
		s.NoError(err)

		_, err = deps.EvmKeeper.ConvertCoinToEvm(
			deps.GoCtx(),
			&evm.MsgConvertCoinToEvm{
				Sender:    deps.Sender.NibiruAddr.String(),
				BankCoin:  unibi(big.NewInt(69)),
				ToEthAddr: eth.EIP55Addr{Address: someoneElse.EthAddr},
			},
		)
		s.Require().ErrorContains(err, evm.ErrCanonicalWnibi)
	})
}
