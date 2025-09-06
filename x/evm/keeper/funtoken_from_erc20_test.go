// Copyright (c) 2023-2024 Nibi, Inc.
package keeper_test

import (
	"encoding/hex"
	"fmt"
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
	"github.com/NibiruChain/nibiru/v2/x/evm/precompile"
)

func (s *SuiteFunToken) TestCreateFunTokenFromERC20() {
	deps := evmtest.NewTestDeps()

	// assert that the ERC20 contract is not deployed
	expectedERC20Addr := crypto.CreateAddress(deps.Sender.EthAddr, deps.NewStateDB().GetNonce(deps.Sender.EthAddr))

	s.T().Log("Deploy ERC20")
	metadata := keeper.ERC20Metadata{
		Name:     "erc20name",
		Symbol:   "TOKEN",
		Decimals: 18,
	}
	deployResp, err := evmtest.DeployContract(
		&deps, embeds.SmartContract_ERC20MinterWithMetadataUpdates,
		metadata.Name, metadata.Symbol, metadata.Decimals,
	)
	s.Require().NoError(err)
	s.Require().Equal(expectedERC20Addr, deployResp.ContractAddr)

	evmObj, _ := deps.NewEVM()
	actualMetadata, err := deps.EvmKeeper.FindERC20Metadata(deps.Ctx, evmObj, deployResp.ContractAddr, nil)
	s.Require().NoError(err)
	s.Require().Equal(metadata, *actualMetadata)

	_, err = deps.EvmKeeper.Code(deps.Ctx, &evm.QueryCodeRequest{
		Address: expectedERC20Addr.String(),
	})
	s.Require().NoError(err)

	erc20Addr := eth.EIP55Addr{
		Address: deployResp.ContractAddr,
	}

	s.Run("sad: insufficient funds to create FunToken mapping", func() {
		_, err = deps.EvmKeeper.CreateFunToken(
			sdk.WrapSDKContext(deps.Ctx),
			&evm.MsgCreateFunToken{
				FromErc20: &erc20Addr,
				Sender:    deps.Sender.NibiruAddr.String(),
			},
		)
		s.Require().ErrorContains(err, "insufficient funds")
	})

	s.Run("happy: CreateFunToken for the ERC20", func() {
		s.Require().NoError(testapp.FundAccount(
			deps.App.BankKeeper,
			deps.Ctx,
			deps.Sender.NibiruAddr,
			deps.EvmKeeper.FeeForCreateFunToken(deps.Ctx),
		))

		deps.Ctx = deps.Ctx.WithGasMeter(sdk.NewInfiniteGasMeter())
		resp, err := deps.EvmKeeper.CreateFunToken(
			sdk.WrapSDKContext(deps.Ctx),
			&evm.MsgCreateFunToken{
				FromErc20: &erc20Addr,
				Sender:    deps.Sender.NibiruAddr.String(),
			},
		)
		s.Require().NoError(err, "erc20 %s", erc20Addr)
		s.Require().NotZero(deps.Ctx.GasMeter().GasConsumed())

		expectedBankDenom := fmt.Sprintf("erc20/%s", expectedERC20Addr.String())
		s.Equal(
			resp.FuntokenMapping,
			evm.FunToken{
				Erc20Addr:      erc20Addr,
				BankDenom:      expectedBankDenom,
				IsMadeFromCoin: false,
			})

		// Event "EventFunTokenCreated" must present
		testutil.RequireContainsTypedEvent(
			s.T(),
			deps.Ctx,
			&evm.EventFunTokenCreated{
				BankDenom:            expectedBankDenom,
				Erc20ContractAddress: erc20Addr.String(),
				Creator:              deps.Sender.NibiruAddr.String(),
				IsMadeFromCoin:       false,
			},
		)

		bankDenomMetadata, _ := deps.App.BankKeeper.GetDenomMetaData(deps.Ctx, expectedBankDenom)
		s.Require().Equal(bank.Metadata{
			Description: fmt.Sprintf(
				"ERC20 token \"%s\" represented as a Bank Coin with a corresponding FunToken mapping", erc20Addr.String(),
			),
			DenomUnits: []*bank.DenomUnit{
				{Denom: expectedBankDenom, Exponent: 0},
				{Denom: metadata.Symbol, Exponent: uint32(metadata.Decimals)},
			},
			Base:    expectedBankDenom,
			Display: metadata.Symbol,
			Name:    metadata.Name,
			Symbol:  metadata.Symbol,
			URI:     "",
			URIHash: "",
		}, bankDenomMetadata)
	})

	s.Run("sad: CreateFunToken for the ERC20: already registered", func() {
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
				FromErc20: &erc20Addr,
				Sender:    deps.Sender.NibiruAddr.String(),
			},
		)
		s.ErrorContains(err, "funtoken mapping already created")
	})

	s.Run("sad: CreateFunToken for the ERC20: invalid sender", func() {
		_, err = deps.EvmKeeper.CreateFunToken(
			sdk.WrapSDKContext(deps.Ctx),
			&evm.MsgCreateFunToken{
				FromErc20: &erc20Addr,
			},
		)
		s.ErrorContains(err, "invalid sender")
	})

	s.Run("sad: CreateFunToken for the ERC20: missing erc20 address", func() {
		_, err = deps.EvmKeeper.CreateFunToken(
			sdk.WrapSDKContext(deps.Ctx),
			&evm.MsgCreateFunToken{
				FromErc20:     nil,
				FromBankDenom: "",
				Sender:        deps.Sender.NibiruAddr.String(),
			},
		)
		s.ErrorContains(err, "either the \"from_erc20\" or \"from_bank_denom\" must be set")
	})
}

func (s *SuiteFunToken) TestSendFromEvmToBank_MadeFromErc20() {
	deps := evmtest.NewTestDeps()
	s.Require().NoError(testapp.FundAccount(
		deps.App.BankKeeper,
		deps.Ctx,
		deps.Sender.NibiruAddr,
		deps.EvmKeeper.FeeForCreateFunToken(deps.Ctx),
	))

	s.T().Log("Deploy ERC20")
	metadata := keeper.ERC20Metadata{
		Name:     "erc20name",
		Symbol:   "TOKEN",
		Decimals: 18,
	}
	deployResp, err := evmtest.DeployContract(
		&deps, embeds.SmartContract_ERC20MinterWithMetadataUpdates,
		metadata.Name, metadata.Symbol, metadata.Decimals,
	)
	s.Require().NoError(err)

	s.T().Log("CreateFunToken for the ERC20")
	resp, err := deps.EvmKeeper.CreateFunToken(
		sdk.WrapSDKContext(deps.Ctx),
		&evm.MsgCreateFunToken{
			FromErc20: &eth.EIP55Addr{
				Address: deployResp.ContractAddr,
			},
			Sender: deps.Sender.NibiruAddr.String(),
		},
	)
	s.Require().NoError(err, "erc20 %s", deployResp.ContractAddr)
	bankDemon := resp.FuntokenMapping.BankDenom

	s.Run("happy: mint erc20 tokens", func() {
		contractInput, err := embeds.SmartContract_ERC20MinterWithMetadataUpdates.ABI.Pack("mint", deps.Sender.EthAddr, big.NewInt(69_420))
		s.Require().NoError(err)
		deps.Ctx = deps.Ctx.WithGasMeter(sdk.NewInfiniteGasMeter())
		evmObj, _ := deps.NewEVM()
		evmResp, err := deps.EvmKeeper.CallContractWithInput(
			deps.Ctx,
			evmObj,
			deps.Sender.EthAddr,      /*from*/
			&deployResp.ContractAddr, /*to*/
			true,                     /*commit*/
			contractInput,
			keeper.Erc20GasLimitExecute,
			nil,
		)
		s.Require().NoError(err)
		s.Require().NotZero(deps.Ctx.GasMeter().GasConsumed())
		s.Require().NotZero(evmResp.GasUsed)
		s.Require().Greater(deps.Ctx.GasMeter().GasConsumed(), evmResp.GasUsed)
	})

	randomAcc := testutil.AccAddress()
	s.Run("happy: send erc20 tokens to Bank", func() {
		contractInput, err := embeds.SmartContract_FunToken.ABI.Pack("sendToBank", deployResp.ContractAddr, big.NewInt(1), randomAcc.String())
		s.Require().NoError(err)
		deps.Ctx = deps.Ctx.WithGasMeter(sdk.NewInfiniteGasMeter())
		evmObj, _ := deps.NewEVM()
		evmResp, err := deps.EvmKeeper.CallContractWithInput(
			deps.Ctx,
			evmObj,
			deps.Sender.EthAddr,                 /*from*/
			&precompile.PrecompileAddr_FunToken, /*to*/
			true,                                /*commit*/
			contractInput,
			evmtest.FunTokenGasLimitSendToEvm,
			nil,
		)
		s.Require().NoError(err)
		s.Require().NotZero(deps.Ctx.GasMeter().GasConsumed())
		s.Require().NotZero(evmResp.GasUsed)
		s.Require().Greater(deps.Ctx.GasMeter().GasConsumed(), evmResp.GasUsed)
	})

	s.Run("happy: check balances", func() {
		evmObj, _ := deps.NewEVM()
		evmtest.AssertERC20BalanceEqualWithDescription(s.T(), deps, evmObj, deployResp.ContractAddr, deps.Sender.EthAddr, big.NewInt(69_419), "expect nonzero balance")
		evmtest.AssertERC20BalanceEqualWithDescription(s.T(), deps, evmObj, deployResp.ContractAddr, evm.EVM_MODULE_ADDRESS, big.NewInt(1), "expect nonzero balance")
		s.Require().Equal(sdk.NewInt(1),
			deps.App.BankKeeper.GetBalance(deps.Ctx, randomAcc, bankDemon).Amount,
		)
	})

	s.Run("sad: send too many erc20 tokens to Bank", func() {
		contractInput, err := embeds.SmartContract_FunToken.ABI.Pack("sendToBank", deployResp.ContractAddr, big.NewInt(70_000), randomAcc.String())
		s.Require().NoError(err)
		deps.Ctx = deps.Ctx.WithGasMeter(sdk.NewInfiniteGasMeter())
		evmObj, _ := deps.NewEVM()
		evmResp, err := deps.EvmKeeper.CallContractWithInput(
			deps.Ctx,
			evmObj,
			deps.Sender.EthAddr,                 /*from*/
			&precompile.PrecompileAddr_FunToken, /*to*/
			true,                                /*commit*/
			contractInput,
			evmtest.FunTokenGasLimitSendToEvm,
			nil,
		)
		s.Require().Error(err, evmResp.String())
		s.Require().NotZero(deps.Ctx.GasMeter().GasConsumed())
		s.Require().NotZero(evmResp.GasUsed)
		s.Require().Greater(deps.Ctx.GasMeter().GasConsumed(), evmResp.GasUsed)
	})

	s.Run("happy: send Bank tokens back to erc20", func() {
		deps.Ctx = deps.Ctx.WithGasMeter(sdk.NewInfiniteGasMeter()).WithEventManager(sdk.NewEventManager())
		_, err = deps.EvmKeeper.ConvertCoinToEvm(sdk.WrapSDKContext(deps.Ctx),
			&evm.MsgConvertCoinToEvm{
				ToEthAddr: eth.EIP55Addr{
					Address: deps.Sender.EthAddr,
				},
				Sender:   randomAcc.String(),
				BankCoin: sdk.NewCoin(bankDemon, sdk.NewInt(1)),
			},
		)
		s.Require().NoError(err)
		s.Require().NotZero(deps.Ctx.GasMeter().GasConsumed())

		// Event "EventConvertCoinToEvm" must present
		testutil.RequireContainsTypedEvent(
			s.T(),
			deps.Ctx,
			&evm.EventConvertCoinToEvm{
				Sender:               randomAcc.String(),
				Erc20ContractAddress: deployResp.ContractAddr.Hex(),
				ToEthAddr:            deps.Sender.EthAddr.String(),
				BankCoin: sdk.Coin{
					Denom:  bankDemon,
					Amount: sdk.NewInt(1),
				},
			},
		)

		// Event "EventTxLog" must present with OwnershipTransferred event
		emptyHash := gethcommon.BytesToHash(make([]byte, 32)).Hex()
		signature := crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)")).Hex()
		fromAddress := gethcommon.BytesToHash(evm.EVM_MODULE_ADDRESS.Bytes()).Hex()
		toAddress := gethcommon.BytesToHash(deps.Sender.EthAddr.Bytes()).Hex()
		amountBase64 := gethcommon.LeftPadBytes(big.NewInt(1).Bytes(), 32)

		testutil.RequireContainsTypedEvent(
			s.T(),
			deps.Ctx,
			&evm.EventTxLog{
				Logs: []evm.Log{
					{
						Address: deployResp.ContractAddr.Hex(),
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
						Index:       0,
						Removed:     false,
					},
				},
			},
		)
	})

	s.T().Log("check balances")
	s.Run("happy: check balances", func() {
		evmObj, _ := deps.NewEVM()
		evmtest.AssertERC20BalanceEqualWithDescription(s.T(), deps, evmObj, deployResp.ContractAddr, deps.Sender.EthAddr, big.NewInt(69_420), "expect nonzero balance")
		evmtest.AssertERC20BalanceEqualWithDescription(s.T(), deps, evmObj, deployResp.ContractAddr, evm.EVM_MODULE_ADDRESS, big.NewInt(0), "expect nonzero balance")
		s.Require().True(
			deps.App.BankKeeper.GetBalance(deps.Ctx, randomAcc, bankDemon).Amount.Equal(sdk.NewInt(0)),
		)
	})

	s.T().Log("sad: send too many Bank tokens back to erc20")
	_, err = deps.EvmKeeper.ConvertCoinToEvm(sdk.WrapSDKContext(deps.Ctx),
		&evm.MsgConvertCoinToEvm{
			ToEthAddr: eth.EIP55Addr{
				Address: deps.Sender.EthAddr,
			},
			Sender:   randomAcc.String(),
			BankCoin: sdk.NewCoin(bankDemon, sdk.NewInt(1)),
		},
	)
	s.Require().Error(err)
}

// TestCreateFunTokenFromERC20MaliciousName tries to create funtoken from a contract
// with a malicious (gas intensive) name() function.
// Fun token should fail creation with "out of gas"
func (s *SuiteFunToken) TestCreateFunTokenFromERC20MaliciousName() {
	deps := evmtest.NewTestDeps()

	s.T().Log("Deploy ERC20MaliciousName")
	metadata := keeper.ERC20Metadata{
		Name:     "erc20name",
		Symbol:   "TOKEN",
		Decimals: 18,
	}
	deployResp, err := evmtest.DeployContract(
		&deps, embeds.SmartContract_TestERC20MaliciousName,
		metadata.Name, metadata.Symbol, metadata.Decimals,
	)
	s.Require().NoError(err)

	erc20Addr := eth.EIP55Addr{
		Address: deployResp.ContractAddr,
	}

	s.T().Log("sad: CreateFunToken for ERC20 with malicious name")
	s.Require().NoError(testapp.FundAccount(
		deps.App.BankKeeper,
		deps.Ctx,
		deps.Sender.NibiruAddr,
		deps.EvmKeeper.FeeForCreateFunToken(deps.Ctx),
	))

	_, err = deps.EvmKeeper.CreateFunToken(
		sdk.WrapSDKContext(deps.Ctx),
		&evm.MsgCreateFunToken{
			FromErc20: &erc20Addr,
			Sender:    deps.Sender.NibiruAddr.String(),
		},
	)
	s.Require().ErrorContains(err, "gas required exceeds allowance")
}

// TestFunTokenFromERC20MaliciousTransfer creates a funtoken from a contract
// with a malicious (gas intensive) transfer() function.
// Fun token should be created but sending from erc20 to bank should fail with out of gas
func (s *SuiteFunToken) TestFunTokenFromERC20MaliciousTransfer() {
	deps := evmtest.NewTestDeps()
	s.Require().NoError(testapp.FundAccount(
		deps.App.BankKeeper,
		deps.Ctx,
		deps.Sender.NibiruAddr,
		deps.EvmKeeper.FeeForCreateFunToken(deps.Ctx),
	))

	s.T().Log("Deploy ERC20MaliciousTransfer")
	metadata := keeper.ERC20Metadata{
		Name:     "erc20name",
		Symbol:   "TOKEN",
		Decimals: 18,
	}
	deployResp, err := evmtest.DeployContract(
		&deps, embeds.SmartContract_TestERC20MaliciousTransfer,
		metadata.Name, metadata.Symbol, metadata.Decimals,
	)
	s.Require().NoError(err)

	erc20Addr := eth.EIP55Addr{
		Address: deployResp.ContractAddr,
	}

	s.T().Log("happy: CreateFunToken for ERC20 with malicious transfer")
	_, err = deps.EvmKeeper.CreateFunToken(
		sdk.WrapSDKContext(deps.Ctx),
		&evm.MsgCreateFunToken{
			FromErc20: &erc20Addr,
			Sender:    deps.Sender.NibiruAddr.String(),
		},
	)
	s.Require().NoError(err)
	randomAcc := testutil.AccAddress()

	s.T().Log("send erc20 tokens to cosmos")
	input, err := embeds.SmartContract_FunToken.ABI.Pack("sendToBank", deployResp.ContractAddr, big.NewInt(1), randomAcc.String())
	s.Require().NoError(err)
	deps.Ctx = deps.Ctx.WithGasMeter(sdk.NewInfiniteGasMeter())
	evmObj, _ := deps.NewEVM()
	evmResp, err := deps.EvmKeeper.CallContractWithInput(
		deps.Ctx,
		evmObj,
		evm.EVM_MODULE_ADDRESS,
		&precompile.PrecompileAddr_FunToken,
		true,
		input,
		evmtest.FunTokenGasLimitSendToEvm,
		nil,
	)
	s.Require().ErrorContains(err, "gas required exceeds allowance")
	s.Require().NotZero(evmResp.GasUsed)
	s.Require().NotZero(deps.Ctx.GasMeter().GasConsumed())
	s.Require().Greater(deps.Ctx.GasMeter().GasConsumed(), evmResp.GasUsed)
}

// TestFunTokenInfiniteRecursionERC20 creates a funtoken from a contract
// with a malicious recursive balanceOf() and transfer() functions.
func (s *SuiteFunToken) TestFunTokenInfiniteRecursionERC20() {
	deps := evmtest.NewTestDeps()
	s.Require().NoError(testapp.FundAccount(
		deps.App.BankKeeper,
		deps.Ctx,
		deps.Sender.NibiruAddr,
		deps.EvmKeeper.FeeForCreateFunToken(deps.Ctx),
	))

	s.T().Log("Deploy InfiniteRecursionERC20")
	metadata := keeper.ERC20Metadata{
		Name:     "erc20name",
		Symbol:   "TOKEN",
		Decimals: 18,
	}
	deployResp, err := evmtest.DeployContract(
		&deps, embeds.SmartContract_TestInfiniteRecursionERC20,
		metadata.Name, metadata.Symbol, metadata.Decimals,
	)
	s.Require().NoError(err)

	erc20Addr := eth.EIP55Addr{
		Address: deployResp.ContractAddr,
	}

	s.T().Log("happy: CreateFunToken for ERC20 with infinite recursion")
	_, err = deps.EvmKeeper.CreateFunToken(
		sdk.WrapSDKContext(deps.Ctx),
		&evm.MsgCreateFunToken{
			FromErc20: &erc20Addr,
			Sender:    deps.Sender.NibiruAddr.String(),
		},
	)
	s.Require().NoError(err)

	s.T().Log("happy: call attackBalance()")
	contractInput, err := embeds.SmartContract_TestInfiniteRecursionERC20.ABI.Pack("attackBalance")
	s.Require().NoError(err)
	evmObj, _ := deps.NewEVM()
	evmResp, err := deps.EvmKeeper.CallContractWithInput(
		deps.Ctx,
		evmObj,
		deps.Sender.EthAddr, /*from*/
		&erc20Addr.Address,  /*to*/
		false,               /*commit*/
		contractInput,
		10_000_000,
		nil,
	)
	s.Require().NoError(err)
	s.Require().NotZero(evmResp.GasUsed)
	s.Require().NotZero(deps.Ctx.GasMeter().GasConsumed())
	s.Require().Greater(deps.Ctx.GasMeter().GasConsumed(), evmResp.GasUsed)

	s.T().Log("sad: call attackTransfer()")
	contractInput, err = embeds.SmartContract_TestInfiniteRecursionERC20.ABI.Pack("attackTransfer")
	s.Require().NoError(err)
	evmObj, _ = deps.NewEVM()
	evmResp, err = deps.EvmKeeper.CallContractWithInput(
		deps.Ctx,
		evmObj,
		deps.Sender.EthAddr, /*from*/
		&erc20Addr.Address,  /*to*/
		true,                /*commit*/
		contractInput,
		10_000_000,
		nil,
	)
	s.Require().ErrorContains(err, "execution reverted")
	s.Require().NotZero(evmResp.GasUsed)
	s.Require().NotZero(deps.Ctx.GasMeter().GasConsumed())
	s.Require().Greater(deps.Ctx.GasMeter().GasConsumed(), evmResp.GasUsed)
}

// TestSendERC20WithFee creates a funtoken from a malicious contract which charges a 10% fee on any transfer.
// Test ensures that after sending ERC20 token to coin and back, all bank coins are burned.
func (s *SuiteFunToken) TestSendERC20WithFee() {
	deps := evmtest.NewTestDeps()
	s.Require().NoError(testapp.FundAccount(
		deps.App.BankKeeper,
		deps.Ctx,
		deps.Sender.NibiruAddr,
		deps.EvmKeeper.FeeForCreateFunToken(deps.Ctx),
	))

	s.T().Log("Deploy ERC20")
	metadata := keeper.ERC20Metadata{
		Name:   "erc20name",
		Symbol: "TOKEN",
	}
	deployResp, err := evmtest.DeployContract(
		&deps, embeds.SmartContract_TestERC20TransferWithFee,
		metadata.Name, metadata.Symbol,
	)
	s.Require().NoError(err)

	s.T().Log("CreateFunToken for the ERC20 with fee")
	resp, err := deps.EvmKeeper.CreateFunToken(
		sdk.WrapSDKContext(deps.Ctx),
		&evm.MsgCreateFunToken{
			FromErc20: &eth.EIP55Addr{
				Address: deployResp.ContractAddr,
			},
			Sender: deps.Sender.NibiruAddr.String(),
		},
	)
	s.Require().NoError(err, "erc20 %s", deployResp.ContractAddr)
	bankDemon := resp.FuntokenMapping.BankDenom

	randomAcc := testutil.AccAddress()

	s.T().Log("send erc20 tokens to Bank")
	contractInput, err := embeds.SmartContract_FunToken.ABI.Pack(
		"sendToBank",
		deployResp.ContractAddr, /*erc20Addr*/
		big.NewInt(100),         /*amount*/
		randomAcc.String(),      /*to*/
	)
	s.Require().NoError(err)
	deps.Ctx = deps.Ctx.WithGasMeter(sdk.NewInfiniteGasMeter())
	evmObj, _ := deps.NewEVM()
	evmResp, err := deps.EvmKeeper.CallContractWithInput(
		deps.Ctx,
		evmObj,
		deps.Sender.EthAddr,                 /*from*/
		&precompile.PrecompileAddr_FunToken, /*to*/
		true,                                /*commit*/
		contractInput,
		evmtest.FunTokenGasLimitSendToEvm,
		nil,
	)
	s.Require().NoError(err)
	s.Require().NotZero(deps.Ctx.GasMeter().GasConsumed())
	s.Require().NotZero(evmResp.GasUsed)
	s.Require().Greater(deps.Ctx.GasMeter().GasConsumed(), evmResp.GasUsed)

	s.T().Log("check balances")
	evmtest.AssertERC20BalanceEqualWithDescription(s.T(), deps, evmObj, deployResp.ContractAddr, deps.Sender.EthAddr, big.NewInt(900), "expect 900 balance")
	evmtest.AssertERC20BalanceEqualWithDescription(s.T(), deps, evmObj, deployResp.ContractAddr, deployResp.ContractAddr, big.NewInt(10), "expect 10 balance")
	evmtest.AssertERC20BalanceEqualWithDescription(s.T(), deps, evmObj, deployResp.ContractAddr, evm.EVM_MODULE_ADDRESS, big.NewInt(90), "expect 90 balance")

	s.Require().Equal(sdk.NewInt(90), deps.App.BankKeeper.GetBalance(deps.Ctx, randomAcc, bankDemon).Amount)

	s.T().Log("send Bank tokens back to erc20")
	_, err = deps.EvmKeeper.ConvertCoinToEvm(sdk.WrapSDKContext(deps.Ctx),
		&evm.MsgConvertCoinToEvm{
			ToEthAddr: eth.EIP55Addr{
				Address: deps.Sender.EthAddr,
			},
			Sender:   randomAcc.String(),
			BankCoin: sdk.NewCoin(bankDemon, sdk.NewInt(90)),
		},
	)
	s.Require().NoError(err)

	s.T().Log("check balances")
	evmObj, _ = deps.NewEVM()
	evmtest.AssertERC20BalanceEqualWithDescription(s.T(), deps, evmObj, deployResp.ContractAddr, deps.Sender.EthAddr, big.NewInt(981), "expect 981 balance")
	evmtest.AssertERC20BalanceEqualWithDescription(s.T(), deps, evmObj, deployResp.ContractAddr, deployResp.ContractAddr, big.NewInt(19), "expect 19 balance")
	evmtest.AssertERC20BalanceEqualWithDescription(s.T(), deps, evmObj, deployResp.ContractAddr, evm.EVM_MODULE_ADDRESS, big.NewInt(0), "expect 0 balance")
	s.Require().True(deps.App.BankKeeper.GetBalance(deps.Ctx, randomAcc, bankDemon).Amount.Equal(sdk.NewInt(0)))
	s.Require().True(deps.App.BankKeeper.GetBalance(deps.Ctx, evm.EVM_MODULE_ADDRESS_NIBI, bankDemon).Amount.Equal(sdk.NewInt(0)))
}

type MkrMetadata struct {
	Symbol [32]byte
}

func (s *SuiteFunToken) TestFindMKRMetadata() {
	deps := evmtest.NewTestDeps()

	s.T().Log("Deploy MKR")

	byteSlice, err := hex.DecodeString("4d4b520000000000000000000000000000000000000000000000000000000000")
	s.Require().NoError(err)
	var byteArray [32]byte
	copy(byteArray[:], byteSlice)

	metadata := MkrMetadata{
		Symbol: byteArray,
	}
	deployResp, err := evmtest.DeployContract(
		&deps, embeds.SmartContract_TestBytes32Metadata,
		metadata.Symbol,
	)
	s.Require().NoError(err)

	s.T().Log("set name")

	byteSlice, err = hex.DecodeString("4d616b6572000000000000000000000000000000000000000000000000000000")
	s.Require().NoError(err)
	copy(byteArray[:], byteSlice)

	contractInput, err := embeds.SmartContract_TestBytes32Metadata.ABI.Pack(
		"setName",
		byteArray,
	)
	s.Require().NoError(err)

	deps.Ctx = deps.Ctx.WithGasMeter(sdk.NewInfiniteGasMeter())
	evmObj, _ := deps.NewEVM()
	evmResp, err := deps.EvmKeeper.CallContractWithInput(
		deps.Ctx,
		evmObj,
		deps.Sender.EthAddr,
		&deployResp.ContractAddr,
		true,
		contractInput,
		evmtest.FunTokenGasLimitSendToEvm,
		nil,
	)
	s.Require().NoError(err)
	s.Require().NotZero(deps.Ctx.GasMeter().GasConsumed())
	s.Require().NotZero(evmResp.GasUsed)
	s.Require().Greater(deps.Ctx.GasMeter().GasConsumed(), evmResp.GasUsed)

	info, err := deps.EvmKeeper.FindERC20Metadata(deps.Ctx, evmObj, deployResp.ContractAddr, embeds.SmartContract_TestBytes32Metadata.ABI)
	s.Require().NoError(err)

	actualMetadata := keeper.ERC20Metadata{
		Name:     "Maker",
		Symbol:   "MKR",
		Decimals: 18,
	}
	s.Require().Equal(actualMetadata, *info)
}
