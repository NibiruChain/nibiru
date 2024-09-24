// Copyright (c) 2023-2024 Nibi, Inc.
package keeper_test

import (
	"fmt"
	"math/big"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
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
	expectedERC20Addr := crypto.CreateAddress(deps.Sender.EthAddr, deps.StateDB().GetNonce(deps.Sender.EthAddr))
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

func (s *FunTokenFromErc20Suite) TestSendFromEvmToCosmos() {
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
		"mint",
		deps.Sender.EthAddr,
		big.NewInt(69_420),
	)
	s.Require().NoError(err)

	randomAcc := testutil.AccAddress()

	s.T().Log("send erc20 tokens to cosmos")
	_, err = deps.EvmKeeper.CallContract(
		deps.Ctx,
		embeds.SmartContract_FunToken.ABI,
		deps.Sender.EthAddr,
		&precompile.PrecompileAddr_FunToken,
		true,
		"bankSend",
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

	s.T().Log("sad: send too many erc20 tokens to cosmos")
	_, err = deps.EvmKeeper.CallContract(
		deps.Ctx,
		embeds.SmartContract_FunToken.ABI,
		deps.Sender.EthAddr,
		&precompile.PrecompileAddr_FunToken,
		true,
		"bankSend",
		deployResp.ContractAddr,
		big.NewInt(70_000),
		randomAcc.String(),
	)
	s.Require().Error(err)

	s.T().Log("send cosmos tokens back to erc20")
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

	s.T().Log("sad: send too many cosmos tokens back to erc20")
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

type FunTokenFromErc20Suite struct {
	suite.Suite
}

func TestFunTokenFromErc20Suite(t *testing.T) {
	suite.Run(t, new(FunTokenFromErc20Suite))
}
