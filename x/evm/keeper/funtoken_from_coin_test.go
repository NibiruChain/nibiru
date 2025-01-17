// Copyright (c) 2023-2024 Nibi, Inc.
package keeper_test

import (
	"math/big"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bank "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/evm/keeper"
	"github.com/NibiruChain/nibiru/v2/x/evm/precompile"
)

func (s *FunTokenFromCoinSuite) TestCreateFunTokenFromCoin() {
	deps := evmtest.NewTestDeps()

	// Compute contract address. FindERC20 should fail
	nonce := deps.NewStateDB().GetNonce(deps.Sender.EthAddr)
	contractAddress := crypto.CreateAddress(deps.Sender.EthAddr, nonce)
	evmObj, _ := deps.NewEVM()
	metadata, err := deps.EvmKeeper.FindERC20Metadata(deps.Ctx, evmObj, contractAddress)
	s.Require().Error(err)
	s.Require().Nil(metadata)

	s.T().Log("Setup: Create a coin in the bank state")
	bankDenom := "sometoken"
	deps.App.BankKeeper.SetDenomMetaData(deps.Ctx, bank.Metadata{
		DenomUnits: []*bank.DenomUnit{
			{
				Denom:    bankDenom,
				Exponent: 0,
				Aliases:  nil,
			},
		},
		Base:    bankDenom,
		Display: bankDenom,
		Name:    bankDenom,
		Symbol:  "TOKEN",
	})

	s.T().Log("sad: not enough funds to create fun token")
	_, err = deps.EvmKeeper.CreateFunToken(
		sdk.WrapSDKContext(deps.Ctx),
		&evm.MsgCreateFunToken{
			FromBankDenom: bankDenom,
			Sender:        deps.Sender.NibiruAddr.String(),
		},
	)
	s.Require().ErrorContains(err, "insufficient funds")

	// Give the sender funds for the fee
	s.Require().NoError(testapp.FundAccount(
		deps.App.BankKeeper,
		deps.Ctx,
		deps.Sender.NibiruAddr,
		deps.EvmKeeper.FeeForCreateFunToken(deps.Ctx),
	))

	s.T().Log("sad: invalid bank denom")
	_, err = deps.EvmKeeper.CreateFunToken(
		sdk.WrapSDKContext(deps.Ctx),
		&evm.MsgCreateFunToken{
			FromBankDenom: "doesn't exist",
			Sender:        deps.Sender.NibiruAddr.String(),
		},
	)
	s.Require().Error(err)

	s.T().Log("happy: CreateFunToken for the bank coin")
	s.Require().NoError(testapp.FundAccount(
		deps.App.BankKeeper,
		deps.Ctx,
		deps.Sender.NibiruAddr,
		deps.EvmKeeper.FeeForCreateFunToken(deps.Ctx),
	))
	createFuntokenResp, err := deps.EvmKeeper.CreateFunToken(
		sdk.WrapSDKContext(deps.Ctx),
		&evm.MsgCreateFunToken{
			FromBankDenom: bankDenom,
			Sender:        deps.Sender.NibiruAddr.String(),
		},
	)
	s.Require().NoError(err)

	erc20Addr := createFuntokenResp.FuntokenMapping.Erc20Addr

	s.Equal(
		createFuntokenResp.FuntokenMapping,
		evm.FunToken{
			Erc20Addr:      erc20Addr,
			BankDenom:      bankDenom,
			IsMadeFromCoin: true,
		},
	)

	s.T().Log("Expect ERC20 to be deployed")
	_, err = deps.EvmKeeper.Code(deps.Ctx, &evm.QueryCodeRequest{
		Address: erc20Addr.String(),
	})
	s.Require().NoError(err)

	s.T().Log("Expect ERC20 metadata on contract")
	info, err := deps.EvmKeeper.FindERC20Metadata(deps.Ctx, evmObj, erc20Addr.Address)
	s.Require().NoError(err, info)
	s.Equal(
		keeper.ERC20Metadata{
			Name:     bankDenom,
			Symbol:   "TOKEN",
			Decimals: 0,
		}, *info,
	)

	// Event "EventFunTokenCreated" must present
	testutil.RequireContainsTypedEvent(
		s.T(),
		deps.Ctx,
		&evm.EventFunTokenCreated{
			BankDenom:            bankDenom,
			Erc20ContractAddress: erc20Addr.String(),
			Creator:              deps.Sender.NibiruAddr.String(),
			IsMadeFromCoin:       true,
		},
	)

	s.T().Log("sad: CreateFunToken for the bank coin: already registered")
	// Give the sender funds for the fee
	s.Require().NoError(testapp.FundAccount(
		deps.App.BankKeeper,
		deps.Ctx,
		deps.Sender.NibiruAddr,
		deps.EvmKeeper.FeeForCreateFunToken(deps.Ctx),
	))
	_, err = deps.EvmKeeper.CreateFunToken(
		sdk.WrapSDKContext(deps.Ctx),
		&evm.MsgCreateFunToken{
			FromBankDenom: bankDenom,
			Sender:        deps.Sender.NibiruAddr.String(),
		},
	)
	s.Require().ErrorContains(err, "funtoken mapping already created")

	s.T().Log("sad: bank denom metadata not registered")
	// Give the sender funds for the fee
	s.Require().NoError(testapp.FundAccount(
		deps.App.BankKeeper,
		deps.Ctx,
		deps.Sender.NibiruAddr,
		deps.EvmKeeper.FeeForCreateFunToken(deps.Ctx),
	))
	_, err = deps.EvmKeeper.CreateFunToken(
		sdk.WrapSDKContext(deps.Ctx),
		&evm.MsgCreateFunToken{
			FromBankDenom: "some random denom",
			Sender:        deps.Sender.NibiruAddr.String(),
		},
	)
	s.Require().ErrorContains(err, "bank coin denom should have bank metadata for denom")
}

func (s *FunTokenFromCoinSuite) TestConvertCoinToEvmAndBack() {
	deps := evmtest.NewTestDeps()
	evmObj, _ := deps.NewEVM()
	alice := evmtest.NewEthPrivAcc()

	// Initial setup
	funToken := s.fundAndCreateFunToken(deps, 100)

	s.T().Log("Convert bank coin to erc-20")
	_, err := deps.EvmKeeper.ConvertCoinToEvm(
		sdk.WrapSDKContext(deps.Ctx),
		&evm.MsgConvertCoinToEvm{
			Sender:   deps.Sender.NibiruAddr.String(),
			BankCoin: sdk.NewCoin(evm.EVMBankDenom, sdk.NewInt(10)),
			ToEthAddr: eth.EIP55Addr{
				Address: alice.EthAddr,
			},
		},
	)
	s.Require().NoError(err)

	s.T().Log("Check typed event")
	testutil.RequireContainsTypedEvent(
		s.T(),
		deps.Ctx,
		&evm.EventConvertCoinToEvm{
			Sender:               deps.Sender.NibiruAddr.String(),
			Erc20ContractAddress: funToken.Erc20Addr.String(),
			ToEthAddr:            alice.EthAddr.String(),
			BankCoin:             sdk.NewCoin(evm.EVMBankDenom, sdk.NewInt(10)),
		},
	)

	// Check 1: module balance
	moduleBalance := deps.App.BankKeeper.GetBalance(deps.Ctx, authtypes.NewModuleAddress(evm.ModuleName), evm.EVMBankDenom)
	s.Require().Equal(sdk.NewInt(10), moduleBalance.Amount)

	// Check 2: Sender balance
	senderBalance := deps.App.BankKeeper.GetBalance(deps.Ctx, deps.Sender.NibiruAddr, evm.EVMBankDenom)
	s.Require().Equal(sdk.NewInt(90), senderBalance.Amount)

	// Check 3: erc-20 balance
	balance, err := deps.EvmKeeper.ERC20().BalanceOf(funToken.Erc20Addr.Address, alice.EthAddr, deps.Ctx, evmObj)
	s.Require().NoError(err)
	s.Require().Zero(balance.Cmp(big.NewInt(10)))

	s.Run("sad: Convert more bank coin to erc-20, insufficient funds", func() {
		_, err = deps.EvmKeeper.ConvertCoinToEvm(
			sdk.WrapSDKContext(deps.Ctx),
			&evm.MsgConvertCoinToEvm{
				Sender:   deps.Sender.NibiruAddr.String(),
				BankCoin: sdk.NewCoin(evm.EVMBankDenom, sdk.NewInt(100)),
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
	evmObj, _ = deps.NewEVM()
	_, err = deps.EvmKeeper.CallContractWithInput(
		deps.Ctx,
		evmObj,
		alice.EthAddr,                       // from
		&precompile.PrecompileAddr_FunToken, // to
		true,                                // commit
		contractInput,
		evmtest.FunTokenGasLimitSendToEvm,
	)
	s.Require().NoError(err)

	// Check 1: module balance
	moduleBalance = deps.App.BankKeeper.GetBalance(deps.Ctx, authtypes.NewModuleAddress(evm.ModuleName), evm.EVMBankDenom)
	s.Require().True(moduleBalance.Amount.Equal(sdk.ZeroInt()))

	// Check 2: Sender balance
	senderBalance = deps.App.BankKeeper.GetBalance(deps.Ctx, deps.Sender.NibiruAddr, evm.EVMBankDenom)
	s.Require().Equal(sdk.NewInt(100), senderBalance.Amount)

	// Check 3: erc-20 balance
	balance, err = deps.EvmKeeper.ERC20().BalanceOf(funToken.Erc20Addr.Address, alice.EthAddr, deps.Ctx, evmObj)
	s.Require().NoError(err)
	s.Require().Equal("0", balance.String())

	s.T().Log("sad: Convert more erc-20 to back to bank coin, insufficient funds")
	evmObj, _ = deps.NewEVM()
	_, err = deps.EvmKeeper.CallContractWithInput(
		deps.Ctx,
		evmObj,
		alice.EthAddr,                       // from
		&precompile.PrecompileAddr_FunToken, // to
		true,                                // commit
		contractInput,
		evmtest.FunTokenGasLimitSendToEvm,
	)
	s.Require().ErrorContains(err, "transfer amount exceeds balance")
}

// TestNativeSendThenPrecompileSend tests a race condition where the state DB
// commit may overwrite the state after the precompile execution, potentially
// causing a loss of funds.
//
// The order of operations is to:
//  1. Create a funtoken mapping from NIBI, a bank coin.
//  2. Use a test Solidity contract to perform two transfers in a single call: a
//     transfer of NIBI with native send and a precompile "IFunToken.sendToBank"
//     transfer for the same asset.
//
// INITIAL STATE:
// - Test contract funds: 10 NIBI, 10 WNIBI
// CONTRACT CALL:
// - Sends 10 NIBI natively and 10 WNIBI -> NIBI to Alice using precompile
// EXPECTED:
// - Test contract funds: 0 NIBI, 0 WNIBI
// - Alice: 20 NIBI
// - Module account: 0 NIBI escrowed
func (s *FunTokenFromCoinSuite) TestNativeSendThenPrecompileSend() {
	deps := evmtest.NewTestDeps()
	evmObj, _ := deps.NewEVM()
	bankDenom := evm.EVMBankDenom

	// Initial setup
	sendAmt := big.NewInt(10)
	funtoken := s.fundAndCreateFunToken(deps, sendAmt.Int64())

	s.T().Log("Deploy Test Contract")
	deployResp, err := evmtest.DeployContract(
		&deps,
		embeds.SmartContract_TestNativeSendThenPrecompileSendJson,
		funtoken.Erc20Addr.Address,
	)
	s.Require().NoError(err)

	testContractAddr := deployResp.ContractAddr
	testContractNibiAddr := eth.EthAddrToNibiruAddr(testContractAddr)

	s.T().Log("Give the test contract 10 NIBI (native)")
	s.Require().NoError(testapp.FundAccount(
		deps.App.BankKeeper,
		deps.Ctx,
		testContractNibiAddr,
		sdk.NewCoins(sdk.NewCoin(bankDenom, sdk.NewIntFromBigInt(sendAmt)))),
	)
	evmtest.AssertBankBalanceEqualWithDescription(
		s.T(), deps, bankDenom, testContractAddr, sendAmt, "expect 10 balance",
	)
	evmtest.AssertBankBalanceEqualWithDescription(
		s.T(), deps, bankDenom, evm.EVM_MODULE_ADDRESS, big.NewInt(0), "expect 0 balance",
	)

	s.T().Log("Convert bank coin to erc-20: give test contract 10 WNIBI (erc20)")
	_, err = deps.EvmKeeper.ConvertCoinToEvm(
		sdk.WrapSDKContext(deps.Ctx),
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
		FunToken:     funtoken,
		Account:      alice.EthAddr,
		BalanceBank:  big.NewInt(0),
		BalanceERC20: big.NewInt(0),
	}.Assert(s.T(), deps, evmObj)

	s.T().Log("call test contract")
	newSendAmtSendToBank := new(big.Int).Quo(sendAmt, big.NewInt(2))
	newSendAmtEvmTransfer := evm.NativeToWei(newSendAmtSendToBank)

	contractInput, err := embeds.SmartContract_TestNativeSendThenPrecompileSendJson.ABI.Pack(
		"nativeSendThenPrecompileSend",
		alice.EthAddr,             /*to*/
		newSendAmtEvmTransfer,     /*amount*/
		alice.NibiruAddr.String(), /*to*/
		newSendAmtSendToBank,      /*amount*/
	)
	s.Require().NoError(err)
	evmObj, _ = deps.NewEVM()
	evmResp, err := deps.EvmKeeper.CallContractWithInput(
		deps.Ctx,
		evmObj,
		deps.Sender.EthAddr,
		&testContractAddr,
		true,
		contractInput,
		evmtest.FunTokenGasLimitSendToEvm,
	)
	s.Require().NoError(err)
	s.Empty(evmResp.VmError)
	gasUsedFor2Ops := evmResp.GasUsed

	evmtest.FunTokenBalanceAssert{
		FunToken: funtoken,
		Account:  alice.EthAddr,
		BalanceBank: new(big.Int).Mul(
			newSendAmtSendToBank, big.NewInt(2)),
		BalanceERC20: big.NewInt(0),
	}.Assert(s.T(), deps, evmObj)

	evmtest.FunTokenBalanceAssert{
		FunToken:     funtoken,
		Account:      testContractAddr,
		BalanceBank:  big.NewInt(5),
		BalanceERC20: big.NewInt(5),
	}.Assert(s.T(), deps, evmObj)

	evmtest.FunTokenBalanceAssert{
		FunToken:     funtoken,
		Account:      evm.EVM_MODULE_ADDRESS,
		BalanceBank:  big.NewInt(5),
		BalanceERC20: big.NewInt(0),
	}.Assert(s.T(), deps, evmObj)

	contractInput, err = embeds.SmartContract_TestNativeSendThenPrecompileSendJson.ABI.Pack(
		"justPrecompileSend",
		alice.NibiruAddr.String(), /*to*/
		newSendAmtSendToBank,      /*amount*/
	)
	s.Require().NoError(err)
	evmObj, _ = deps.NewEVM()
	evmResp, err = deps.EvmKeeper.CallContractWithInput(
		deps.Ctx,
		evmObj,
		deps.Sender.EthAddr,
		&testContractAddr,
		true,
		contractInput,
		evmtest.DefaultEthCallGasLimit,
	)
	s.Require().NoError(err)
	s.Empty(evmResp.VmError)
	gasUsedFor1Op := evmResp.GasUsed

	evmtest.FunTokenBalanceAssert{
		FunToken: funtoken,
		Account:  alice.EthAddr,
		BalanceBank: new(big.Int).Mul(
			newSendAmtSendToBank, big.NewInt(3)),
		BalanceERC20: big.NewInt(0),
	}.Assert(s.T(), deps, evmObj)

	evmtest.FunTokenBalanceAssert{
		FunToken:     funtoken,
		Account:      testContractAddr,
		BalanceBank:  big.NewInt(5),
		BalanceERC20: big.NewInt(0),
	}.Assert(s.T(), deps, evmObj)

	evmtest.FunTokenBalanceAssert{
		FunToken:     funtoken,
		Account:      evm.EVM_MODULE_ADDRESS,
		BalanceBank:  big.NewInt(0),
		BalanceERC20: big.NewInt(0),
	}.Assert(s.T(), deps, evmObj)
	s.Require().Greater(gasUsedFor2Ops, gasUsedFor1Op, "2 operations should consume more gas")
}

// TestERC20TransferThenPrecompileSend
// 1. Creates a funtoken from coin.
// 2. Using the test contract, performs two sends in a single call: a erc20
// transfer and a precompile sendToBank.
// It tests a race condition where the state DB commit may overwrite the state after the precompile execution,
// potentially causing an infinite minting of funds.
//
// INITIAL STATE:
// - Test contract funds: 10 WNIBI
// CONTRACT CALL:
// - Sends 1 WNIBI to Alice using erc20 transfer and 9 WNIBI -> NIBI to Alice using precompile
// EXPECTED:
// - Test contract funds: 0 WNIBI
// - Alice: 1 WNIBI, 9 NIBI
// - Module account: 1 NIBI escrowed (which Alice holds as 1 WNIBI)
func (s *FunTokenFromCoinSuite) TestERC20TransferThenPrecompileSend() {
	deps := evmtest.NewTestDeps()
	evmObj, _ := deps.NewEVM()

	funToken := s.fundAndCreateFunToken(deps, 10e6)

	s.T().Log("Deploy Test Contract")
	deployResp, err := evmtest.DeployContract(
		&deps,
		embeds.SmartContract_TestERC20TransferThenPrecompileSend,
		funToken.Erc20Addr.Address,
	)
	s.Require().NoError(err)
	testContractAddr := deployResp.ContractAddr

	s.T().Log("Convert bank coin to erc-20: give test contract 10 WNIBI (erc20)")
	_, err = deps.EvmKeeper.ConvertCoinToEvm(
		sdk.WrapSDKContext(deps.Ctx),
		&evm.MsgConvertCoinToEvm{
			Sender:    deps.Sender.NibiruAddr.String(),
			BankCoin:  sdk.NewCoin(evm.EVMBankDenom, sdk.NewInt(10e6)),
			ToEthAddr: eth.EIP55Addr{Address: testContractAddr},
		},
	)
	s.Require().NoError(err)

	// check balances
	evmtest.FunTokenBalanceAssert{
		FunToken:     funToken,
		Account:      testContractAddr,
		BalanceBank:  big.NewInt(0),
		BalanceERC20: big.NewInt(10e6),
	}.Assert(s.T(), deps, evmObj)

	evmtest.FunTokenBalanceAssert{
		FunToken:     funToken,
		Account:      evm.EVM_MODULE_ADDRESS,
		BalanceBank:  big.NewInt(10e6),
		BalanceERC20: big.NewInt(0),
	}.Assert(s.T(), deps, evmObj)

	evmtest.FunTokenBalanceAssert{
		FunToken:     funToken,
		Account:      deps.Sender.EthAddr,
		BalanceBank:  big.NewInt(0),
		BalanceERC20: big.NewInt(0),
	}.Assert(s.T(), deps, evmObj)

	// Alice hex and Alice bech32 is the same address in different representation
	alice := evmtest.NewEthPrivAcc()

	s.T().Log("call test contract")
	contractInput, err := embeds.SmartContract_TestERC20TransferThenPrecompileSend.ABI.Pack(
		"erc20TransferThenPrecompileSend",
		alice.EthAddr,             /*to*/
		big.NewInt(1e6),           /*amount*/
		alice.NibiruAddr.String(), /*to*/
		big.NewInt(9e6),           /*amount*/
	)
	s.Require().NoError(err)
	evmObj, _ = deps.NewEVM()
	_, err = deps.EvmKeeper.CallContractWithInput(
		deps.Ctx,
		evmObj,
		deps.Sender.EthAddr, // from
		&testContractAddr,   // to
		true,                // commit
		contractInput,
		10_000_000, // gas limit
	)
	s.Require().NoError(err)

	evmtest.FunTokenBalanceAssert{
		FunToken:     funToken,
		Account:      alice.EthAddr,
		BalanceBank:  big.NewInt(9e6),
		BalanceERC20: big.NewInt(1e6),
		Description:  "Alice has 9 NIBI / 1 WNIBI",
	}.Assert(s.T(), deps, evmObj)

	evmtest.FunTokenBalanceAssert{
		FunToken:     funToken,
		Account:      testContractAddr,
		BalanceBank:  big.NewInt(0),
		BalanceERC20: big.NewInt(0),
		Description:  "Test contract 0 NIBI / 0 WNIBI",
	}.Assert(s.T(), deps, evmObj)

	evmtest.FunTokenBalanceAssert{
		FunToken:     funToken,
		Account:      evm.EVM_MODULE_ADDRESS,
		BalanceBank:  big.NewInt(1e6),
		BalanceERC20: big.NewInt(0e6),
		Description:  "Module account has 1 NIBI escrowed",
	}.Assert(s.T(), deps, evmObj)
}

// TestPrecompileSelfCallRevert
//  1. Creates a funtoken from coin.
//  2. Using the test contract, creates another instance of itself, calls the precompile method and then force reverts.
//     It tests a race condition where the state DB commit
//     may save the wrong state before the precompile execution, not revert it entirely,
//     potentially causing an infinite mint of funds.
//
// INITIAL STATE:
// - Test contract funds: 10 NIBI, 10 WNIBI
// CONTRACT CALL:
// - Sends 1 NIBI to Alice using native send and 1 WNIBI -> NIBI to Charles using precompile
// EXPECTED:
// - all changes reverted
// - Test contract funds: 10 NIBI, 10 WNIBI
// - Alice: 0 NIBI
// - Charles: 0 NIBI
// - Module account: 10 NIBI escrowed (which Test contract holds as 10 WNIBI)
func (s *FunTokenFromCoinSuite) TestPrecompileSelfCallRevert() {
	deps := evmtest.NewTestDeps()
	evmObj, _ := deps.NewEVM()

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

	s.T().Log("Convert bank coin to erc-20: give test contract 10 WNIBI (erc20)")
	_, err = deps.EvmKeeper.ConvertCoinToEvm(
		sdk.WrapSDKContext(deps.Ctx),
		&evm.MsgConvertCoinToEvm{
			Sender:    deps.Sender.NibiruAddr.String(),
			BankCoin:  sdk.NewCoin(evm.EVMBankDenom, sdk.NewInt(10e6)),
			ToEthAddr: eth.EIP55Addr{Address: testContractAddr},
		},
	)
	s.Require().NoError(err)

	s.T().Log("Give the test contract 10 NIBI (native)")
	s.Require().NoError(testapp.FundAccount(
		deps.App.BankKeeper,
		deps.Ctx,
		eth.EthAddrToNibiruAddr(testContractAddr),
		sdk.NewCoins(sdk.NewCoin(evm.EVMBankDenom, sdk.NewInt(10e6))),
	))

	evmtest.FunTokenBalanceAssert{
		FunToken:     funToken,
		Account:      testContractAddr,
		BalanceBank:  big.NewInt(10e6),
		BalanceERC20: big.NewInt(10e6),
		Description:  "Initial contract state sanity check: 10 NIBI / 10 WNIBI",
	}.Assert(s.T(), deps, evmObj)

	// Create Alice and Charles. Contract will try to send Alice native coins and
	// send Charles tokens via sendToBank
	alice := evmtest.NewEthPrivAcc()
	charles := evmtest.NewEthPrivAcc()

	s.T().Log("call test contract")
	contractInput, err := embeds.SmartContract_TestPrecompileSelfCallRevert.ABI.Pack(
		"selfCallTransferFunds",
		alice.EthAddr,
		evm.NativeToWei(big.NewInt(1e6)),
		charles.NibiruAddr.String(),
		big.NewInt(9e6),
	)
	s.Require().NoError(err)
	_, err = deps.EvmKeeper.CallContractWithInput(
		deps.Ctx,
		evmObj,
		deps.Sender.EthAddr,
		&testContractAddr,
		true,
		contractInput,
		evmtest.FunTokenGasLimitSendToEvm,
	)
	s.Require().NoError(err)

	evmtest.FunTokenBalanceAssert{
		FunToken:     funToken,
		Account:      alice.EthAddr,
		BalanceBank:  big.NewInt(0),
		BalanceERC20: big.NewInt(0),
		Description:  "Alice has 0 NIBI / 0 WNIBI",
	}.Assert(s.T(), deps, evmObj)

	evmtest.FunTokenBalanceAssert{
		FunToken:     funToken,
		Account:      charles.EthAddr,
		BalanceBank:  big.NewInt(0),
		BalanceERC20: big.NewInt(0),
		Description:  "Charles has 0 NIBI / 0 WNIBI",
	}.Assert(s.T(), deps, evmObj)

	evmtest.FunTokenBalanceAssert{
		FunToken:     funToken,
		Account:      testContractAddr,
		BalanceBank:  big.NewInt(10e6),
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

// fundAndCreateFunToken creates initial setup for tests
func (s *FunTokenFromCoinSuite) fundAndCreateFunToken(deps evmtest.TestDeps, unibiAmount int64) evm.FunToken {
	bankDenom := evm.EVMBankDenom

	s.T().Log("Setup: Create a coin in the bank state")
	deps.App.BankKeeper.SetDenomMetaData(deps.Ctx, bank.Metadata{
		DenomUnits: []*bank.DenomUnit{
			{
				Denom:    bankDenom,
				Exponent: 0,
			},
			{
				Denom:    "NIBI",
				Exponent: 6,
			},
		},
		Base:    bankDenom,
		Display: "NIBI",
		Name:    "NIBI",
		Symbol:  "NIBI",
	})

	s.T().Log("Give the sender funds for funtoken creation and funding test contract")
	s.Require().NoError(testapp.FundAccount(
		deps.App.BankKeeper,
		deps.Ctx,
		deps.Sender.NibiruAddr,
		deps.EvmKeeper.FeeForCreateFunToken(deps.Ctx).Add(sdk.NewCoin(bankDenom, sdk.NewInt(unibiAmount))),
	))

	s.T().Log("Create FunToken from coin")
	createFunTokenResp, err := deps.EvmKeeper.CreateFunToken(
		sdk.WrapSDKContext(deps.Ctx),
		&evm.MsgCreateFunToken{
			FromBankDenom: bankDenom,
			Sender:        deps.Sender.NibiruAddr.String(),
		},
	)
	s.Require().NoError(err)

	return createFunTokenResp.FuntokenMapping
}

type FunTokenFromCoinSuite struct {
	suite.Suite
}

func TestFunTokenFromCoinSuite(t *testing.T) {
	suite.Run(t, new(FunTokenFromCoinSuite))
}
