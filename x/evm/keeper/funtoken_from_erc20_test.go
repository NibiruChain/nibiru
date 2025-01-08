// Copyright (c) 2023-2024 Nibi, Inc.
package keeper_test

import (
	"fmt"
	"math/big"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
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
	s.Require().NoError(testapp.FundAccount(
		deps.App.BankKeeper,
		deps.Ctx,
		deps.Sender.NibiruAddr,
		deps.EvmKeeper.FeeForCreateFunToken(deps.Ctx),
	))

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
	_, err = deps.EvmKeeper.CallContract(
		deps.Ctx,
		embeds.SmartContract_ERC20Minter.ABI,
		deps.Sender.EthAddr,
		&deployResp.ContractAddr,
		true,
		keeper.Erc20GasLimitExecute,
		"mint",
		deps.Sender.EthAddr,
		big.NewInt(69_420),
	)
	s.Require().NoError(err)

	randomAcc := testutil.AccAddress()

	deps.ResetGasMeter()

	s.T().Log("send erc20 tokens to Bank")
	_, err = deps.EvmKeeper.CallContract(
		deps.Ctx,
		embeds.SmartContract_FunToken.ABI,
		deps.Sender.EthAddr,
		&precompile.PrecompileAddr_FunToken,
		true,
		evmtest.FunTokenGasLimitSendToEvm,
		"sendToBank",
		deployResp.ContractAddr,
		big.NewInt(1),
		randomAcc.String(),
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
	evmResp, err := deps.EvmKeeper.CallContract(
		deps.Ctx,
		embeds.SmartContract_FunToken.ABI,
		deps.Sender.EthAddr,
		&precompile.PrecompileAddr_FunToken,
		true,
		evmtest.FunTokenGasLimitSendToEvm,
		"sendToBank",
		deployResp.ContractAddr,
		big.NewInt(70_000),
		randomAcc.String(),
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
	s.Require().NoError(err)
	randomAcc := testutil.AccAddress()

	deps.ResetGasMeter()

	s.T().Log("send erc20 tokens to cosmos")
	_, err = deps.EvmKeeper.CallContract(
		deps.Ctx,
		embeds.SmartContract_FunToken.ABI,
		deps.Sender.EthAddr,
		&precompile.PrecompileAddr_FunToken,
		true,
		evmtest.FunTokenGasLimitSendToEvm,
		"sendToBank",
		deployResp.ContractAddr,
		big.NewInt(1),
		randomAcc.String(),
	)
	s.Require().ErrorContains(err, "gas required exceeds allowance")
}

// TestFunTokenInfiniteRecursionERC20 creates a funtoken from a contract
// with a malicious recursive balanceOf() and transfer() functions.
func (s *FunTokenFromErc20Suite) TestFunTokenInfiniteRecursionERC20() {
	deps := evmtest.NewTestDeps()

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
	s.Require().NoError(err)

	deps.ResetGasMeter()

	s.T().Log("happy: call attackBalance()")
	res, err := deps.EvmKeeper.CallContract(
		deps.Ctx,
		embeds.SmartContract_TestInfiniteRecursionERC20.ABI,
		deps.Sender.EthAddr,
		&erc20Addr.Address,
		false,
		10_000_000,
		"attackBalance",
	)
	s.Require().NoError(err)
	s.Require().NotNil(res)
	s.Require().Empty(res.VmError)

	s.T().Log("sad: call attackBalance()")
	_, err = deps.EvmKeeper.CallContract(
		deps.Ctx,
		embeds.SmartContract_TestInfiniteRecursionERC20.ABI,
		deps.Sender.EthAddr,
		&erc20Addr.Address,
		true,
		10_000_000,
		"attackTransfer",
	)
	s.Require().ErrorContains(err, "execution reverted")
}

// TestSendERC20WithFee creates a funtoken from a malicious contract which charges a 10% fee on any transfer.
// Test ensures that after sending ERC20 token to coin and back, all bank coins are burned.
func (s *FunTokenFromErc20Suite) TestSendERC20WithFee() {
	deps := evmtest.NewTestDeps()

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
	s.Require().NoError(testapp.FundAccount(
		deps.App.BankKeeper,
		deps.Ctx,
		deps.Sender.NibiruAddr,
		deps.EvmKeeper.FeeForCreateFunToken(deps.Ctx),
	))

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
	_, err = deps.EvmKeeper.CallContract(
		deps.Ctx,
		embeds.SmartContract_FunToken.ABI,
		deps.Sender.EthAddr,
		&precompile.PrecompileAddr_FunToken,
		true,
		evmtest.FunTokenGasLimitSendToEvm,
		"sendToBank",
		deployResp.ContractAddr,
		big.NewInt(100),
		randomAcc.String(),
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
