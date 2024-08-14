// Copyright (c) 2023-2024 Nibi, Inc.
package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bank "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/evm/keeper"
)

func (s *FunTokenFromCoinSuite) TestDeployERC20ForBankCoin() {
	deps := evmtest.NewTestDeps()

	// Compute contract address. FindERC20 should fail
	nonce := deps.StateDB().GetNonce(evm.EVM_MODULE_ADDRESS)
	expectedERC20Addr := crypto.CreateAddress(evm.EVM_MODULE_ADDRESS, nonce)
	_, err := deps.EvmKeeper.FindERC20Metadata(deps.Ctx, expectedERC20Addr)
	s.Error(err)

	s.T().Log("Case 1: Deploy and invoke ERC20 for info")
	const bankDenom = "sometoken"
	bankMetadata := bank.Metadata{
		DenomUnits: []*bank.DenomUnit{
			{
				Denom:    bankDenom,
				Exponent: 0,
			},
		},
		Base:    bankDenom,
		Display: bankDenom,
		Name:    bankDenom,
		Symbol:  "TOKEN",
	}
	erc20Addr, err := deps.EvmKeeper.DeployERC20ForBankCoin(
		deps.Ctx, bankMetadata,
	)
	s.Require().NoError(err)
	s.Equal(expectedERC20Addr, erc20Addr)

	s.T().Log("Expect ERC20 metadata on contract")
	info, err := deps.EvmKeeper.FindERC20Metadata(deps.Ctx, erc20Addr)
	s.NoError(err)
	s.Equal(keeper.ERC20Metadata{
		Name:     bankDenom,
		Symbol:   "TOKEN",
		Decimals: 0,
	}, info)
}

func (s *FunTokenFromCoinSuite) TestCreateFunTokenFromCoin() {
	deps := evmtest.NewTestDeps()

	// Compute contract address. FindERC20 should fail
	nonce := deps.StateDB().GetNonce(deps.Sender.EthAddr)
	contractAddress := crypto.CreateAddress(deps.Sender.EthAddr, nonce)
	_, err := deps.EvmKeeper.FindERC20Metadata(deps.Ctx, contractAddress)
	s.Error(err)

	s.T().Log("Setup: Create a coin in the bank state")
	bankDenom := "sometoken"

	setBankDenomMetadata(deps.Ctx, deps.App.BankKeeper, bankDenom)

	s.T().Log("happy: CreateFunToken for the bank coin")
	// Give the sender funds for the fee
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
	s.Require().NoError(err, "bankDenom %s", bankDenom)
	erc20 := createFuntokenResp.FuntokenMapping.Erc20Addr
	erc20Addr := erc20.ToAddr()
	s.Equal(
		createFuntokenResp.FuntokenMapping,
		evm.FunToken{
			Erc20Addr:      erc20,
			BankDenom:      bankDenom,
			IsMadeFromCoin: true,
		})

	s.T().Log("Expect ERC20 to be deployed")
	queryCodeReq := &evm.QueryCodeRequest{
		Address: erc20.String(),
	}
	_, err = deps.EvmKeeper.Code(deps.Ctx, queryCodeReq)
	s.Require().NoError(err)

	s.T().Log("Expect ERC20 metadata on contract")
	metadata := keeper.ERC20Metadata{
		Name:     bankDenom,
		Symbol:   "TOKEN",
		Decimals: 0,
	}
	info, err := deps.EvmKeeper.FindERC20Metadata(deps.Ctx, erc20Addr)
	s.NoError(err, info)
	s.Equal(metadata, info)

	// Event "EventFunTokenCreated" must present
	// Event "EventFunTokenCreated" must present
	testutil.RequireContainsTypedEvent(
		s.T(),
		deps.Ctx,
		&evm.EventFunTokenCreated{
			BankDenom:            bankDenom,
			Erc20ContractAddress: erc20.String(),
			Creator:              deps.Sender.NibiruAddr.String(),
			IsMadeFromCoin:       true,
		},
	)

	s.T().Log("sad: CreateFunToken for the bank coin: already registered")
	// Give the sender funds for the fee
	err = testapp.FundAccount(
		deps.App.BankKeeper,
		deps.Ctx,
		deps.Sender.NibiruAddr,
		deps.EvmKeeper.FeeForCreateFunToken(deps.Ctx),
	)
	s.Require().NoError(err)
	_, err = deps.EvmKeeper.CreateFunToken(
		sdk.WrapSDKContext(deps.Ctx),
		&evm.MsgCreateFunToken{
			FromBankDenom: bankDenom,
			Sender:        deps.Sender.NibiruAddr.String(),
		},
	)
	s.Require().ErrorContains(err, "funtoken mapping already created")
}

// TestSendFunTokenToEvm executes sending fun tokens from bank coin to erc20 and checks the results:
// - sender balance should be reduced by sendAmount
// - erc-20 balance should be increased by sendAmount
// - evm module account should hold sender's coins
func (s *FunTokenFromCoinSuite) TestSendFunTokenToEvm() {
	for _, tc := range []struct {
		name           string
		bankDenom      string
		initialBalance math.Int
		amountToSend   math.Int
		wantErr        string
	}{
		{
			name:           "happy: proper sending",
			bankDenom:      "unibi",
			initialBalance: math.NewInt(100),
			amountToSend:   math.NewInt(10),
			wantErr:        "",
		},
		{
			name:           "sad: not registered bank denom",
			bankDenom:      "not-registered-denom",
			initialBalance: math.NewInt(100),
			amountToSend:   math.NewInt(10),
			wantErr:        "does not exist",
		},
		{
			name:           "sad: insufficient balance",
			bankDenom:      "unibi",
			initialBalance: math.NewInt(10),
			amountToSend:   math.NewInt(100),
			wantErr:        "insufficient funds",
		},
	} {
		s.Run(tc.name, func() {
			deps := evmtest.NewTestDeps()
			bankDenom := "unibi"
			recipientEVMAddr := eth.MustNewHexAddrFromStr("0x1234500000000000000000000000000000000000")
			evmModuleAddr := deps.App.AccountKeeper.GetModuleAddress(evm.ModuleName)

			ctx := sdk.WrapSDKContext(deps.Ctx)
			setBankDenomMetadata(deps.Ctx, deps.App.BankKeeper, bankDenom)

			// Give the sender funds
			s.Require().NoError(testapp.FundAccount(
				deps.App.BankKeeper,
				deps.Ctx,
				deps.Sender.NibiruAddr,
				deps.EvmKeeper.FeeForCreateFunToken(deps.Ctx).Add(sdk.NewCoin(bankDenom, tc.initialBalance)),
			))

			// Create fun token from coin
			createFunTokenResp, err := deps.EvmKeeper.CreateFunToken(
				ctx,
				&evm.MsgCreateFunToken{
					FromBankDenom: bankDenom,
					Sender:        deps.Sender.NibiruAddr.String(),
				},
			)
			s.Require().NoError(err)
			funTokenErc20Addr := createFunTokenResp.FuntokenMapping.Erc20Addr.ToAddr()

			// Send fun token to ERC-20 contract
			bankCoin := sdk.NewCoin(tc.bankDenom, tc.amountToSend)
			_, err = deps.EvmKeeper.ConvertCoinToEvm(
				ctx,
				&evm.MsgSendFunTokenToEvm{
					Sender:    deps.Sender.NibiruAddr.String(),
					BankCoin:  bankCoin,
					ToEthAddr: recipientEVMAddr,
				},
			)
			if tc.wantErr != "" {
				s.Require().ErrorContains(err, tc.wantErr)
				return
			}
			s.Require().NoError(err)

			// Event "EventSendFunTokenToEvm" must present
			testutil.RequireContainsTypedEvent(
				s.T(),
				deps.Ctx,
				&evm.EventSendFunTokenToEvm{
					Sender:               deps.Sender.NibiruAddr.String(),
					Erc20ContractAddress: funTokenErc20Addr.String(),
					ToEthAddr:            recipientEVMAddr.String(),
					BankCoin:             bankCoin,
				},
			)

			// Check 1: coins are stored on a module balance
			moduleBalance := deps.App.BankKeeper.GetBalance(deps.Ctx, evmModuleAddr, bankDenom)
			s.Require().Equal(tc.amountToSend, moduleBalance.Amount)

			// Check 2: Sender balance reduced by send amount
			senderBalance := deps.App.BankKeeper.GetBalance(deps.Ctx, deps.Sender.NibiruAddr, bankDenom)
			s.Require().Equal(tc.initialBalance.Sub(tc.amountToSend), senderBalance.Amount)

			// Check 3: erc-20 balance equals to send amount
			balance, err := deps.EvmKeeper.ERC20().BalanceOf(funTokenErc20Addr, recipientEVMAddr.ToAddr(), deps.Ctx)
			s.Require().NoError(err)
			s.Require().Zero(balance.Cmp(tc.amountToSend.BigInt()))
		})
	}
}

type FunTokenFromCoinSuite struct {
	suite.Suite
}

func TestFunTokenFromCoinSuite(t *testing.T) {
	suite.Run(t, new(FunTokenFromCoinSuite))
}
