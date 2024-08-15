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
	nonce := deps.StateDB().GetNonce(deps.Sender.EthAddr)
	contractAddress := crypto.CreateAddress(deps.Sender.EthAddr, nonce)
	metadata, err := deps.EvmKeeper.FindERC20Metadata(deps.Ctx, contractAddress)
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
	info, err := deps.EvmKeeper.FindERC20Metadata(deps.Ctx, erc20Addr.Addr())
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
	alice := evmtest.NewEthPrivAcc()
	bankDenom := "unibi"

	s.T().Log("Setup: Create a coin in the bank state")
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

	s.T().Log("Give the sender funds")
	s.Require().NoError(testapp.FundAccount(
		deps.App.BankKeeper,
		deps.Ctx,
		deps.Sender.NibiruAddr,
		deps.EvmKeeper.FeeForCreateFunToken(deps.Ctx).Add(sdk.NewCoin(bankDenom, sdk.NewInt(100))),
	))

	s.T().Log("Create FunToken mapping and ERC20")
	createFunTokenResp, err := deps.EvmKeeper.CreateFunToken(
		sdk.WrapSDKContext(deps.Ctx),
		&evm.MsgCreateFunToken{
			FromBankDenom: bankDenom,
			Sender:        deps.Sender.NibiruAddr.String(),
		},
	)
	s.Require().NoError(err)

	funTokenErc20Addr := createFunTokenResp.FuntokenMapping.Erc20Addr

	s.T().Log("Convert bank coin to erc-20")
	_, err = deps.EvmKeeper.ConvertCoinToEvm(
		sdk.WrapSDKContext(deps.Ctx),
		&evm.MsgConvertCoinToEvm{
			Sender:    deps.Sender.NibiruAddr.String(),
			BankCoin:  sdk.NewCoin(bankDenom, sdk.NewInt(10)),
			ToEthAddr: eth.NewHexAddr(alice.EthAddr),
		},
	)
	s.Require().NoError(err)

	s.T().Log("Check typed event")
	testutil.RequireContainsTypedEvent(
		s.T(),
		deps.Ctx,
		&evm.EventConvertCoinToEvm{
			Sender:               deps.Sender.NibiruAddr.String(),
			Erc20ContractAddress: funTokenErc20Addr.String(),
			ToEthAddr:            alice.EthAddr.String(),
			BankCoin:             sdk.NewCoin(bankDenom, sdk.NewInt(10)),
		},
	)

	// Check 1: module balance
	moduleBalance := deps.App.BankKeeper.GetBalance(deps.Ctx, authtypes.NewModuleAddress(evm.ModuleName), bankDenom)
	s.Require().Equal(sdk.NewInt(10), moduleBalance.Amount)

	// Check 2: Sender balance
	senderBalance := deps.App.BankKeeper.GetBalance(deps.Ctx, deps.Sender.NibiruAddr, bankDenom)
	s.Require().Equal(sdk.NewInt(90), senderBalance.Amount)

	// Check 3: erc-20 balance
	balance, err := deps.EvmKeeper.ERC20().BalanceOf(funTokenErc20Addr.Addr(), alice.EthAddr, deps.Ctx)
	s.Require().NoError(err)
	s.Require().Zero(balance.Cmp(big.NewInt(10)))

	s.T().Log("sad: Convert more bank coin to erc-20, insufficient funds")
	_, err = deps.EvmKeeper.ConvertCoinToEvm(
		sdk.WrapSDKContext(deps.Ctx),
		&evm.MsgConvertCoinToEvm{
			Sender:    deps.Sender.NibiruAddr.String(),
			BankCoin:  sdk.NewCoin(bankDenom, sdk.NewInt(100)),
			ToEthAddr: eth.NewHexAddr(alice.EthAddr),
		},
	)
	s.Require().ErrorContains(err, "insufficient funds")

	s.T().Log("Convert erc-20 to back to bank coin")
	_, err = deps.EvmKeeper.CallContract(
		deps.Ctx,
		embeds.SmartContract_FunToken.ABI,
		alice.EthAddr,
		&precompile.PrecompileAddr_FunToken,
		true,
		"bankSend",
		funTokenErc20Addr.Addr(),
		big.NewInt(10),
		deps.Sender.NibiruAddr.String(),
	)
	s.Require().NoError(err)

	// Check 1: module balance
	moduleBalance = deps.App.BankKeeper.GetBalance(deps.Ctx, authtypes.NewModuleAddress(evm.ModuleName), bankDenom)
	s.Require().True(moduleBalance.Amount.Equal(sdk.ZeroInt()))

	// Check 2: Sender balance
	senderBalance = deps.App.BankKeeper.GetBalance(deps.Ctx, deps.Sender.NibiruAddr, bankDenom)
	s.Require().Equal(sdk.NewInt(100), senderBalance.Amount)

	// Check 3: erc-20 balance
	balance, err = deps.EvmKeeper.ERC20().BalanceOf(funTokenErc20Addr.Addr(), alice.EthAddr, deps.Ctx)
	s.Require().NoError(err)
	s.Require().Zero(balance.Cmp(big.NewInt(0)))

	s.T().Log("sad: Convert more erc-20 to back to bank coin, insufficient funds")
	_, err = deps.EvmKeeper.CallContract(
		deps.Ctx,
		embeds.SmartContract_FunToken.ABI,
		alice.EthAddr,
		&precompile.PrecompileAddr_FunToken,
		true,
		"bankSend",
		funTokenErc20Addr.Addr(),
		big.NewInt(10),
		deps.Sender.NibiruAddr.String(),
	)
	s.Require().ErrorContains(err, "transfer amount exceeds balance")
}

type FunTokenFromCoinSuite struct {
	suite.Suite
}

func TestFunTokenFromCoinSuite(t *testing.T) {
	suite.Run(t, new(FunTokenFromCoinSuite))
}
