// Copyright (c) 2023-2024 Nibi, Inc.
package keeper_test

import (
	"fmt"
	"math/big"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	bank "github.com/cosmos/cosmos-sdk/x/bank/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"
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

func (s *FunTokenFromErc20Suite) TestCreateFunTokenFromERC20() {
	deps := evmtest.NewTestDeps()

	// assert that the ERC20 contract is not deployed
	expectedERC20Addr := crypto.CreateAddress(deps.Sender.EthAddr, deps.NewStateDB().GetNonce(deps.Sender.EthAddr))
	_, err := deps.EvmKeeper.FindERC20Metadata(deps.Ctx, expectedERC20Addr)
	s.Error(err)

	s.T().Log("Deploy ERC20")
	metadata := keeper.ERC20Metadata{
		Name:     "erc20name",
		Symbol:   "TOKEN",
		Decimals: 18,
	}
	deployResp, err := evmtest.DeployContract(
		&deps, embeds.SmartContract_ERC20Minter,
		metadata.Name, metadata.Symbol, metadata.Decimals,
	)
	s.Require().NoError(err)
	s.Require().Equal(expectedERC20Addr, deployResp.ContractAddr)

	info, err := deps.EvmKeeper.FindERC20Metadata(deps.Ctx, deployResp.ContractAddr)
	s.Require().NoError(err)
	s.Require().Equal(metadata, *info)

	_, err = deps.EvmKeeper.Code(deps.Ctx, &evm.QueryCodeRequest{
		Address: expectedERC20Addr.String(),
	})
	s.Require().NoError(err)

	erc20Addr := eth.EIP55Addr{
		Address: deployResp.ContractAddr,
	}

	s.T().Log("sad: insufficient funds to create FunToken mapping")
	_, err = deps.EvmKeeper.CreateFunToken(
		sdk.WrapSDKContext(deps.Ctx),
		&evm.MsgCreateFunToken{
			FromErc20: &erc20Addr,
			Sender:    deps.Sender.NibiruAddr.String(),
		},
	)
	s.Require().ErrorContains(err, "insufficient funds")

	s.T().Log("happy: CreateFunToken for the ERC20")
	s.Require().NoError(testapp.FundAccount(
		deps.App.BankKeeper,
		deps.Ctx,
		deps.Sender.NibiruAddr,
		deps.EvmKeeper.FeeForCreateFunToken(deps.Ctx),
	))

	resp, err := deps.EvmKeeper.CreateFunToken(
		sdk.WrapSDKContext(deps.Ctx),
		&evm.MsgCreateFunToken{
			FromErc20: &erc20Addr,
			Sender:    deps.Sender.NibiruAddr.String(),
		},
	)
	s.Require().NoError(err, "erc20 %s", erc20Addr)

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
	bankMetadata, _ := deps.App.BankKeeper.GetDenomMetaData(deps.Ctx, expectedBankDenom)
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
	}, bankMetadata)

	s.T().Log("sad: CreateFunToken for the ERC20: already registered")
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

	s.T().Log("sad: CreateFunToken for the ERC20: invalid sender")

	_, err = deps.EvmKeeper.CreateFunToken(
		sdk.WrapSDKContext(deps.Ctx),
		&evm.MsgCreateFunToken{
			FromErc20: &erc20Addr,
		},
	)
	s.ErrorContains(err, "invalid sender")

	s.T().Log("sad: CreateFunToken for the ERC20: missing erc20 address")

	_, err = deps.EvmKeeper.CreateFunToken(
		sdk.WrapSDKContext(deps.Ctx),
		&evm.MsgCreateFunToken{
			FromErc20:     nil,
			FromBankDenom: "",
			Sender:        deps.Sender.NibiruAddr.String(),
		},
	)
	s.ErrorContains(err, "either the \"from_erc20\" or \"from_bank_denom\" must be set")
}

func (s *FunTokenFromErc20Suite) TestSendFromEvmToBank() {
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
		&deps, embeds.SmartContract_ERC20Minter,
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

	s.T().Logf("mint erc20 tokens to %s", deps.Sender.EthAddr.String())
	evmCfg := deps.EvmKeeper.GetEVMConfig(deps.Ctx)
	stateDB := deps.EvmKeeper.NewStateDB(deps.Ctx, deps.EvmKeeper.TxConfig(deps.Ctx, gethcommon.BigToHash(big.NewInt(0))))
	input, err := embeds.SmartContract_ERC20Minter.ABI.Pack("mint", deps.Sender.EthAddr, big.NewInt(69_420))
	s.Require().NoError(err)
	evmMsg := gethcore.NewMessage(
		deps.Sender.EthAddr,
		&deployResp.ContractAddr,
		deps.EvmKeeper.GetAccNonce(deps.Ctx, deps.Sender.EthAddr),
		big.NewInt(0),
		keeper.Erc20GasLimitExecute,
		big.NewInt(0),
		big.NewInt(0),
		big.NewInt(0),
		input,
		gethcore.AccessList{},
		false,
	)
	evmObj := deps.EvmKeeper.NewEVM(deps.Ctx, evmMsg, evmCfg, nil /*tracer*/, stateDB)
	_, err = deps.EvmKeeper.CallContractWithInput(
		deps.Ctx,
		evmObj,
		evm.EVM_MODULE_ADDRESS,
		&deployResp.ContractAddr,
		false,
		input,
		keeper.Erc20GasLimitExecute,
	)
	s.Require().NoError(err)

	randomAcc := testutil.AccAddress()

	deps.ResetGasMeter()

	s.T().Log("happy: send erc20 tokens to Bank")
	input, err = embeds.SmartContract_FunToken.ABI.Pack("sendToBank", deployResp.ContractAddr, big.NewInt(1), randomAcc.String())
	s.Require().NoError(err)
	evmMsg = gethcore.NewMessage(
		deps.Sender.EthAddr,
		&precompile.PrecompileAddr_FunToken,
		deps.EvmKeeper.GetAccNonce(deps.Ctx, deps.Sender.EthAddr),
		big.NewInt(0),
		keeper.Erc20GasLimitExecute,
		big.NewInt(0),
		big.NewInt(0),
		big.NewInt(0),
		input,
		gethcore.AccessList{},
		false,
	)
	evmObj = deps.EvmKeeper.NewEVM(deps.Ctx, evmMsg, evmCfg, nil /*tracer*/, stateDB)
	_, err = deps.EvmKeeper.CallContractWithInput(
		deps.Ctx,
		evmObj,
		evm.EVM_MODULE_ADDRESS,
		&precompile.PrecompileAddr_FunToken,
		true,
		input,
		evmtest.FunTokenGasLimitSendToEvm,
	)
	s.Require().NoError(err)

	s.T().Log("check balances")
	evmtest.AssertERC20BalanceEqual(s.T(), deps, deployResp.ContractAddr, deps.Sender.EthAddr, big.NewInt(69_419))
	evmtest.AssertERC20BalanceEqual(s.T(), deps, deployResp.ContractAddr, evm.EVM_MODULE_ADDRESS, big.NewInt(1))
	s.Require().Equal(sdk.NewInt(1),
		deps.App.BankKeeper.GetBalance(deps.Ctx, randomAcc, bankDemon).Amount,
	)

	deps.ResetGasMeter()

	s.T().Log("sad: send too many erc20 tokens to Bank")
	input, err = embeds.SmartContract_FunToken.ABI.Pack("sendToBank", deployResp.ContractAddr, big.NewInt(70_000), randomAcc.String())
	s.Require().NoError(err)
	evmMsg = gethcore.NewMessage(
		deps.Sender.EthAddr,
		&precompile.PrecompileAddr_FunToken,
		deps.EvmKeeper.GetAccNonce(deps.Ctx, deps.Sender.EthAddr),
		big.NewInt(0),
		evmtest.FunTokenGasLimitSendToEvm,
		big.NewInt(0),
		big.NewInt(0),
		big.NewInt(0),
		input,
		gethcore.AccessList{},
		false,
	)
	evmObj = deps.EvmKeeper.NewEVM(deps.Ctx, evmMsg, evmCfg, nil /*tracer*/, stateDB)
	evmResp, err := deps.EvmKeeper.CallContractWithInput(
		deps.Ctx,
		evmObj,
		evm.EVM_MODULE_ADDRESS,
		&precompile.PrecompileAddr_FunToken,
		true,
		input,
		evmtest.FunTokenGasLimitSendToEvm,
	)
	s.T().Log("check balances")
	s.Require().Error(err, evmResp.String())

	deps.ResetGasMeter()

	s.T().Log("send Bank tokens back to erc20")
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

	s.T().Log("check balances")
	evmtest.AssertERC20BalanceEqual(s.T(), deps, deployResp.ContractAddr, deps.Sender.EthAddr, big.NewInt(69_420))
	evmtest.AssertERC20BalanceEqual(s.T(), deps, deployResp.ContractAddr, evm.EVM_MODULE_ADDRESS, big.NewInt(0))
	s.Require().True(
		deps.App.BankKeeper.GetBalance(deps.Ctx, randomAcc, bankDemon).Amount.Equal(sdk.NewInt(0)),
	)

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
func (s *FunTokenFromErc20Suite) TestCreateFunTokenFromERC20MaliciousName() {
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
func (s *FunTokenFromErc20Suite) TestFunTokenFromERC20MaliciousTransfer() {
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

	deps.ResetGasMeter()

	s.T().Log("send erc20 tokens to cosmos")
	input, err := embeds.SmartContract_FunToken.ABI.Pack("sendToBank", deployResp.ContractAddr, big.NewInt(1), randomAcc.String())
	s.Require().NoError(err)
	evmMsg := gethcore.NewMessage(
		deps.Sender.EthAddr,
		&precompile.PrecompileAddr_FunToken,
		deps.EvmKeeper.GetAccNonce(deps.Ctx, deps.Sender.EthAddr),
		big.NewInt(0),
		evmtest.FunTokenGasLimitSendToEvm,
		big.NewInt(0),
		big.NewInt(0),
		big.NewInt(0),
		input,
		gethcore.AccessList{},
		false,
	)
	evmCfg := deps.EvmKeeper.GetEVMConfig(deps.Ctx)
	stateDB := deps.EvmKeeper.NewStateDB(deps.Ctx, deps.EvmKeeper.TxConfig(deps.Ctx, gethcommon.BigToHash(big.NewInt(0))))
	evmObj := deps.EvmKeeper.NewEVM(deps.Ctx, evmMsg, evmCfg, nil /*tracer*/, stateDB)
	_, err = deps.EvmKeeper.CallContractWithInput(
		deps.Ctx,
		evmObj,
		evm.EVM_MODULE_ADDRESS,
		&precompile.PrecompileAddr_FunToken,
		true,
		input,
		evmtest.FunTokenGasLimitSendToEvm,
	)
	s.Require().ErrorContains(err, "gas required exceeds allowance")
}

// TestFunTokenInfiniteRecursionERC20 creates a funtoken from a contract
// with a malicious recursive balanceOf() and transfer() functions.
func (s *FunTokenFromErc20Suite) TestFunTokenInfiniteRecursionERC20() {
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

	deps.ResetGasMeter()

	s.T().Log("happy: call attackBalance()")
	input, err := embeds.SmartContract_TestInfiniteRecursionERC20.ABI.Pack("attackBalance")
	s.Require().NoError(err)
	evmMsg := gethcore.NewMessage(
		deps.Sender.EthAddr,
		&erc20Addr.Address,
		deps.EvmKeeper.GetAccNonce(deps.Ctx, deps.Sender.EthAddr),
		big.NewInt(0),
		evmtest.FunTokenGasLimitSendToEvm,
		big.NewInt(0),
		big.NewInt(0),
		big.NewInt(0),
		input,
		gethcore.AccessList{},
		false,
	)
	stateDB := deps.EvmKeeper.NewStateDB(deps.Ctx, deps.EvmKeeper.TxConfig(deps.Ctx, gethcommon.BigToHash(big.NewInt(0))))
	evmCfg := deps.EvmKeeper.GetEVMConfig(deps.Ctx)
	evmObj := deps.EvmKeeper.NewEVM(deps.Ctx, evmMsg, evmCfg, nil /*tracer*/, stateDB)
	_, err = deps.EvmKeeper.CallContractWithInput(
		deps.Ctx,
		evmObj,
		evm.EVM_MODULE_ADDRESS,
		&erc20Addr.Address,
		true,
		input,
		evmtest.FunTokenGasLimitSendToEvm,
	)
	s.Require().NoError(err)

	s.T().Log("sad: call attackBalance()")
	_, err = deps.EvmKeeper.CallContractWithInput(
		deps.Ctx,
		evmObj,
		evm.EVM_MODULE_ADDRESS,
		&erc20Addr.Address,
		true,
		input,
		evmtest.FunTokenGasLimitSendToEvm,
	)
	s.Require().ErrorContains(err, "execution reverted")
}

// TestSendERC20WithFee creates a funtoken from a malicious contract which charges a 10% fee on any transfer.
// Test ensures that after sending ERC20 token to coin and back, all bank coins are burned.
func (s *FunTokenFromErc20Suite) TestSendERC20WithFee() {
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

	deps.ResetGasMeter()

	s.T().Log("send erc20 tokens to Bank")
	input, err := embeds.SmartContract_FunToken.ABI.Pack("sendToBank", deployResp.ContractAddr, big.NewInt(100), randomAcc.String())
	s.Require().NoError(err)
	evmMsg := gethcore.NewMessage(
		deps.Sender.EthAddr,
		&precompile.PrecompileAddr_FunToken,
		deps.EvmKeeper.GetAccNonce(deps.Ctx, deps.Sender.EthAddr),
		big.NewInt(0),
		evmtest.FunTokenGasLimitSendToEvm,
		big.NewInt(0),
		big.NewInt(0),
		big.NewInt(0),
		input,
		gethcore.AccessList{},
		false,
	)
	evmCfg := deps.EvmKeeper.GetEVMConfig(deps.Ctx)
	stateDB := deps.EvmKeeper.NewStateDB(deps.Ctx, deps.EvmKeeper.TxConfig(deps.Ctx, gethcommon.BigToHash(big.NewInt(0))))
	evmObj := deps.EvmKeeper.NewEVM(deps.Ctx, evmMsg, evmCfg, nil /*tracer*/, stateDB)
	_, err = deps.EvmKeeper.CallContractWithInput(
		deps.Ctx,
		evmObj,
		evm.EVM_MODULE_ADDRESS,
		&precompile.PrecompileAddr_FunToken,
		true,
		input,
		evmtest.FunTokenGasLimitSendToEvm,
	)
	s.Require().NoError(err)

	s.T().Log("check balances")
	evmtest.AssertERC20BalanceEqual(s.T(), deps, deployResp.ContractAddr, deps.Sender.EthAddr, big.NewInt(900))
	evmtest.AssertERC20BalanceEqual(s.T(), deps, deployResp.ContractAddr, deployResp.ContractAddr, big.NewInt(10))
	evmtest.AssertERC20BalanceEqual(s.T(), deps, deployResp.ContractAddr, evm.EVM_MODULE_ADDRESS, big.NewInt(90))
	s.Require().Equal(sdk.NewInt(90), deps.App.BankKeeper.GetBalance(deps.Ctx, randomAcc, bankDemon).Amount)

	deps.ResetGasMeter()

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
	evmtest.AssertERC20BalanceEqual(s.T(), deps, deployResp.ContractAddr, deps.Sender.EthAddr, big.NewInt(981))
	evmtest.AssertERC20BalanceEqual(s.T(), deps, deployResp.ContractAddr, deployResp.ContractAddr, big.NewInt(19))
	evmtest.AssertERC20BalanceEqual(s.T(), deps, deployResp.ContractAddr, evm.EVM_MODULE_ADDRESS, big.NewInt(0))
	s.Require().True(deps.App.BankKeeper.GetBalance(deps.Ctx, randomAcc, bankDemon).Amount.Equal(sdk.NewInt(0)))
	s.Require().True(deps.App.BankKeeper.GetBalance(deps.Ctx, evm.EVM_MODULE_ADDRESS_NIBI, bankDemon).Amount.Equal(sdk.NewInt(0)))
}

type FunTokenFromErc20Suite struct {
	suite.Suite
}

func TestFunTokenFromErc20Suite(t *testing.T) {
	suite.Run(t, new(FunTokenFromErc20Suite))
}
