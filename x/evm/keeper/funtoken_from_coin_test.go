// Copyright (c) 2023-2024 Nibi, Inc.
package keeper_test

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	bank "github.com/cosmos/cosmos-sdk/x/bank/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/evm/keeper"
)

func (s *SuiteFunToken) TestCreateFunTokenFromCoin() {
	deps := evmtest.NewTestDeps()
	s.Run("Compute contract address. FindERC20 should fail", func() {
		evmObj, _ := deps.NewEVM()
		metadata, err := deps.EvmKeeper.FindERC20Metadata(
			deps.Ctx,
			evmObj,
			crypto.CreateAddress(
				evm.EVM_MODULE_ADDRESS, deps.EvmKeeper.GetAccNonce(deps.Ctx, evm.EVM_MODULE_ADDRESS)),
			nil,
		)
		s.Require().Error(err)
		s.Require().Nil(metadata)
	})

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

	s.Run("insufficient funds to create funtoken", func() {
		s.T().Log("sad: not enough funds to create fun token")
		_, err := deps.EvmKeeper.CreateFunToken(
			sdk.WrapSDKContext(deps.Ctx),
			&evm.MsgCreateFunToken{
				FromBankDenom: bankDenom,
				Sender:        deps.Sender.NibiruAddr.String(),
			},
		)
		s.Require().ErrorContains(err, "insufficient funds")
	})

	s.Run("invalid bank denom", func() {
		s.Require().NoError(testapp.FundAccount(
			deps.App.BankKeeper,
			deps.Ctx,
			deps.Sender.NibiruAddr,
			deps.EvmKeeper.FeeForCreateFunToken(deps.Ctx),
		))
		_, err := deps.EvmKeeper.CreateFunToken(
			sdk.WrapSDKContext(deps.Ctx),
			&evm.MsgCreateFunToken{
				FromBankDenom: "doesn't exist",
				Sender:        deps.Sender.NibiruAddr.String(),
			},
		)
		s.Require().Error(err)
	})

	s.Run("happy: CreateFunToken for the bank coin", func() {
		deps.Ctx = deps.Ctx.WithGasMeter(sdk.NewInfiniteGasMeter())
		s.Require().NoError(testapp.FundAccount(
			deps.App.BankKeeper,
			deps.Ctx,
			deps.Sender.NibiruAddr,
			deps.EvmKeeper.FeeForCreateFunToken(deps.Ctx),
		))
		expectedErc20Addr := crypto.CreateAddress(evm.EVM_MODULE_ADDRESS, deps.EvmKeeper.GetAccNonce(deps.Ctx, evm.EVM_MODULE_ADDRESS))
		createFuntokenResp, err := deps.EvmKeeper.CreateFunToken(
			sdk.WrapSDKContext(deps.Ctx),
			&evm.MsgCreateFunToken{
				FromBankDenom: bankDenom,
				Sender:        deps.Sender.NibiruAddr.String(),
			},
		)
		s.Require().NoError(err)
		s.Require().NotZero(deps.Ctx.GasMeter().GasConsumed())

		s.Equal(
			createFuntokenResp.FuntokenMapping,
			evm.FunToken{
				Erc20Addr:      eth.EIP55Addr{Address: expectedErc20Addr},
				BankDenom:      bankDenom,
				IsMadeFromCoin: true,
			},
		)
		actualErc20Addr := createFuntokenResp.FuntokenMapping.Erc20Addr

		s.T().Log("Expect ERC20 to be deployed")
		_, err = deps.EvmKeeper.Code(deps.Ctx, &evm.QueryCodeRequest{
			Address: actualErc20Addr.String(),
		})
		s.Require().NoError(err)

		s.T().Log("Expect ERC20 metadata on contract")
		evmObj, _ := deps.NewEVM()
		info, err := deps.EvmKeeper.FindERC20Metadata(deps.Ctx, evmObj, actualErc20Addr.Address, nil)
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
				Erc20ContractAddress: actualErc20Addr.String(),
				Creator:              deps.Sender.NibiruAddr.String(),
				IsMadeFromCoin:       true,
			},
		)

		// Event "EventTxLog" must present with OwnershipTransferred event
		emptyHash := gethcommon.BytesToHash(make([]byte, 32)).Hex()
		signature := crypto.Keccak256Hash([]byte("OwnershipTransferred(address,address)")).Hex()
		ownershipFrom := emptyHash
		ownershipTo := gethcommon.BytesToHash(evm.EVM_MODULE_ADDRESS.Bytes()).Hex()

		testutil.RequireContainsTypedEvent(
			s.T(),
			deps.Ctx,
			&evm.EventTxLog{
				Logs: []evm.Log{
					{
						Address: actualErc20Addr.Hex(),
						Topics: []string{
							signature,
							ownershipFrom,
							ownershipTo,
						},
						Data:        nil,
						BlockNumber: 1, // we are in simulation, no real block numbers or tx hashes
						TxHash:      emptyHash,
						TxIndex:     0,
						BlockHash:   emptyHash,
						Index:       0,
						Removed:     false,
					},
				},
			},
		)
	})

	s.Run("sad: CreateFunToken for the bank coin: already registered", func() {
		// Give the sender funds for the fee
		s.Require().NoError(testapp.FundAccount(
			deps.App.BankKeeper,
			deps.Ctx,
			deps.Sender.NibiruAddr,
			deps.EvmKeeper.FeeForCreateFunToken(deps.Ctx),
		))
		_, err := deps.EvmKeeper.CreateFunToken(
			sdk.WrapSDKContext(deps.Ctx),
			&evm.MsgCreateFunToken{
				FromBankDenom: bankDenom,
				Sender:        deps.Sender.NibiruAddr.String(),
			},
		)
		s.Require().ErrorContains(err, "funtoken mapping already created")
	})
}

// TestERC20TransferThenPrecompileSend
// 1. Creates a funtoken from coin.
// 2. Using the test contract, performs two sends in a single call: a erc20
// transfer and a precompile sendToBank.
// It tests a race condition where the state DB commit may overwrite the state after the precompile execution,
// potentially causing an infinite minting of funds.
//
// INITIAL STATE:
//   - Test contract funds: 10_000_000 TEST (Bank)
//
// CONTRACT CALL:
//   - Sends 1e6 TEST to Alice using erc20 transfer
//   - and send 9e6 TEST to Alice using precompile
//
// EXPECTED:
//   - Test contract funds: 0 EVM
//   - Alice: 1 EVM, 9 BC
//   - Module account: 1 BC escrowed (which Alice holds as 1 EVM)
func (s *SuiteFunToken) TestERC20TransferThenPrecompileSend() {
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

	s.T().Logf("Convert bank coin to erc-20: give test contract %d %s (erc20)", int64(10e6), funToken.BankDenom)
	_, err = deps.EvmKeeper.ConvertCoinToEvm(
		sdk.WrapSDKContext(deps.Ctx),
		&evm.MsgConvertCoinToEvm{
			Sender:    deps.Sender.NibiruAddr.String(),
			BankCoin:  sdk.NewCoin(funToken.BankDenom, sdk.NewInt(10e6)),
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
	deps.Ctx = deps.Ctx.WithGasMeter(sdk.NewInfiniteGasMeter())
	evmObj, _ = deps.NewEVM()
	evmResp, err := deps.EvmKeeper.CallContractWithInput(
		deps.Ctx,
		evmObj,
		deps.Sender.EthAddr, // from
		&testContractAddr,   // to
		true,                // commit
		contractInput,
		10_000_000, // gas limit
		nil,
	)
	s.Require().NoError(err)
	s.Require().NotZero(deps.Ctx.GasMeter().GasConsumed())
	s.Require().NotZero(evmResp.GasUsed)
	s.Require().Greaterf(deps.Ctx.GasMeter().GasConsumed(), evmResp.GasUsed, "total gas consumed on cosmos context should be greater than gas used by EVM")

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

// fundAndCreateFunToken creates initial setup for tests
func (s *SuiteFunToken) fundAndCreateFunToken(deps evmtest.TestDeps, bankAmount int64) evm.FunToken {
	bankDenom := "testfuntoken"

	s.T().Log("Setup: Create a coin in the bank state")
	deps.App.BankKeeper.SetDenomMetaData(deps.Ctx, bank.Metadata{
		DenomUnits: []*bank.DenomUnit{
			{
				Denom:    bankDenom,
				Exponent: 0,
			},
			{
				Denom:    "TEST",
				Exponent: 6,
			},
		},
		Base:    bankDenom,
		Display: "TEST",
		Name:    "TEST",
		Symbol:  "TEST",
	})

	s.T().Log("Give the sender funds for funtoken creation and funding test contract")
	tokensToFund := deps.EvmKeeper.FeeForCreateFunToken(deps.Ctx).Add(
		sdk.NewCoin(bankDenom, sdk.NewInt(bankAmount)),
	)
	s.Require().NoError(testapp.FundAccount(
		deps.App.BankKeeper,
		deps.Ctx,
		deps.Sender.NibiruAddr,
		tokensToFund,
	))
	s.T().Logf("Funded %s", tokensToFund)

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
