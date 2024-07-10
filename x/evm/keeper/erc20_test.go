// Copyright (c) 2023-2024 Nibi, Inc.
package keeper_test

import (
	"fmt"
	"math/big"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	bank "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/NibiruChain/nibiru/eth"
	"github.com/NibiruChain/nibiru/x/evm"
	"github.com/NibiruChain/nibiru/x/evm/embeds"
	"github.com/NibiruChain/nibiru/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/x/evm/keeper"
)

func (s *Suite) TestCreateFunTokenFromERC20() {
	deps := evmtest.NewTestDeps()

	// Compute contract address. FindERC20 should fail
	nonce := deps.StateDB().GetNonce(deps.Sender.EthAddr)
	contractAddress := crypto.CreateAddress(deps.Sender.EthAddr, nonce)
	_, err := deps.K.FindERC20Metadata(deps.Ctx, contractAddress)
	s.Error(err)

	s.T().Log("Case 1: Deploy and invoke ERC20 for info")
	{
		metadata := keeper.ERC20Metadata{
			Name:     "erc20name",
			Symbol:   "TOKEN",
			Decimals: 18,
		}
		deployResp, err := evmtest.DeployContract(
			&deps, embeds.SmartContract_ERC20Minter, s.T(),
			metadata.Name, metadata.Symbol, metadata.Decimals,
		)
		s.NoError(err)
		s.Equal(contractAddress, deployResp.ContractAddr)

		info, err := deps.K.FindERC20Metadata(deps.Ctx, deployResp.ContractAddr)
		s.NoError(err, info)
		s.Equal(metadata, info)
	}

	s.T().Log("Case 2: Deploy and invoke ERC20 for info")
	{
		metadata := keeper.ERC20Metadata{
			Name:     "gwei",
			Symbol:   "GWEI",
			Decimals: 9,
		}
		deployResp, err := evmtest.DeployContract(
			&deps, embeds.SmartContract_ERC20Minter, s.T(),
			metadata.Name, metadata.Symbol, metadata.Decimals,
		)
		s.NoError(err)
		s.NotEqual(contractAddress, deployResp.ContractAddr)

		info, err := deps.K.FindERC20Metadata(deps.Ctx, deployResp.ContractAddr)
		s.NoError(err, info)
		s.Equal(metadata, info)
	}

	s.T().Log("happy: CreateFunToken for the ERC20")

	erc20Addr := eth.NewHexAddr(contractAddress)
	queryCodeReq := &evm.QueryCodeRequest{
		Address: erc20Addr.String(),
	}
	_, err = deps.K.Code(deps.Ctx, queryCodeReq)
	s.NoError(err)

	createFuntokenResp, err := deps.K.CreateFunToken(
		deps.GoCtx(),
		&evm.MsgCreateFunToken{
			FromErc20: &erc20Addr,
			Sender:    deps.Sender.NibiruAddr.String(),
		},
	)
	s.NoError(err, "erc20 %s", erc20Addr)
	s.Equal(
		createFuntokenResp.FuntokenMapping,
		evm.FunToken{
			Erc20Addr:      erc20Addr,
			BankDenom:      fmt.Sprintf("erc20/%s", erc20Addr.String()),
			IsMadeFromCoin: false,
		})

	s.T().Log("sad: CreateFunToken for the ERC20: already registered")
	_, err = deps.K.CreateFunToken(
		deps.GoCtx(),
		&evm.MsgCreateFunToken{
			FromErc20: &erc20Addr,
			Sender:    deps.Sender.NibiruAddr.String(),
		},
	)
	s.ErrorContains(err, "Funtoken mapping already created")

	s.T().Log("sad: CreateFunToken for the ERC20: invalid sender")
	_, err = deps.K.CreateFunToken(
		deps.GoCtx(),
		&evm.MsgCreateFunToken{
			FromErc20: &erc20Addr,
		},
	)
	s.ErrorContains(err, "invalid sender")
}

func (s *Suite) TestDeployERC20ForBankCoin() {
	deps := evmtest.NewTestDeps()

	// Compute contract address. FindERC20 should fail
	nonce := deps.StateDB().GetNonce(deps.Sender.EthAddr)
	contractAddress := crypto.CreateAddress(deps.Sender.EthAddr, nonce)
	_, err := deps.K.FindERC20Metadata(deps.Ctx, contractAddress)
	s.Error(err)

	s.T().Log("Case 1: Deploy and invoke ERC20 for info")
	bankDenom := "sometoken"
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
	erc20Addr, err := deps.K.DeployERC20ForBankCoin(
		deps.Ctx, bankMetadata,
	)
	s.NoError(err)
	s.NotEqual(contractAddress, erc20Addr,
		"address derived from before call should differ since the contract deployment succeeds")

	s.T().Log("Expect ERC20 metadata on contract")
	metadata := keeper.ERC20Metadata{
		Name:     bankDenom,
		Symbol:   "TOKEN",
		Decimals: 0,
	}
	info, err := deps.K.FindERC20Metadata(deps.Ctx, erc20Addr)
	s.NoError(err, info)
	s.Equal(metadata, info)
}

func (s *Suite) TestCreateFunTokenFromCoin() {
	deps := evmtest.NewTestDeps()

	// Compute contract address. FindERC20 should fail
	nonce := deps.StateDB().GetNonce(deps.Sender.EthAddr)
	contractAddress := crypto.CreateAddress(deps.Sender.EthAddr, nonce)
	_, err := deps.K.FindERC20Metadata(deps.Ctx, contractAddress)
	s.Error(err)

	s.T().Log("Setup: Create a coin in the bank state")
	bankDenom := "sometoken"

	setBankDenomMetadata(deps.Ctx, deps.Chain.BankKeeper, bankDenom)

	s.T().Log("happy: CreateFunToken for the bank coin")
	createFuntokenResp, err := deps.K.CreateFunToken(
		deps.GoCtx(),
		&evm.MsgCreateFunToken{
			FromBankDenom: bankDenom,
			Sender:        deps.Sender.NibiruAddr.String(),
		},
	)
	s.NoError(err, "bankDenom %s", bankDenom)
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
		Address: erc20Addr.String(),
	}
	_, err = deps.K.Code(deps.Ctx, queryCodeReq)
	s.NoError(err)

	s.T().Log("Expect ERC20 metadata on contract")
	metadata := keeper.ERC20Metadata{
		Name:     bankDenom,
		Symbol:   "TOKEN",
		Decimals: 0,
	}
	info, err := deps.K.FindERC20Metadata(deps.Ctx, erc20Addr)
	s.NoError(err, info)
	s.Equal(metadata, info)

	s.T().Log("sad: CreateFunToken for the bank coin: already registered")
	_, err = deps.K.CreateFunToken(
		deps.GoCtx(),
		&evm.MsgCreateFunToken{
			FromBankDenom: bankDenom,
			Sender:        deps.Sender.NibiruAddr.String(),
		},
	)
	s.ErrorContains(err, "Funtoken mapping already created")
}

// TestSendFunTokenToEvm executes sending fun tokens from bank coin to erc20 and checks the results:
// - sender balance should be reduced by sendAmount
// - erc-20 balance should be increased by sendAmount
// - evm module account should hold sender's coins
func (s *Suite) TestSendFunTokenToEvm() {
	for _, tc := range []struct {
		name                string
		bankDenom           string
		senderBalanceBefore math.Int
		amountToSend        math.Int
		wantErr             string
	}{
		{
			name:                "happy: proper sending",
			bankDenom:           "unibi",
			senderBalanceBefore: math.NewInt(100),
			amountToSend:        math.NewInt(10),
			wantErr:             "",
		},
		{
			name:                "sad: not registered bank denom",
			bankDenom:           "not-registered-denom",
			senderBalanceBefore: math.NewInt(100),
			amountToSend:        math.NewInt(10),
			wantErr:             "does not exist",
		},
		{
			name:                "sad: insufficient balance",
			bankDenom:           "unibi",
			senderBalanceBefore: math.NewInt(10),
			amountToSend:        math.NewInt(100),
			wantErr:             "insufficient funds",
		},
	} {
		s.Run(tc.name, func() {
			deps := evmtest.NewTestDeps()
			bankDenom := "unibi"
			recipientEVMAddr := eth.MustNewHexAddrFromStr("0x1234500000000000000000000000000000000000")
			evmModuleAddr := deps.Chain.AccountKeeper.GetModuleAddress(evm.ModuleName)
			spendableAmount := tc.senderBalanceBefore.Int64()
			spendableCoins := sdk.NewCoins(sdk.NewInt64Coin(bankDenom, spendableAmount))

			ctx := deps.GoCtx()
			setBankDenomMetadata(deps.Ctx, deps.Chain.BankKeeper, bankDenom)

			// Fund sender's wallet
			err := deps.Chain.BankKeeper.MintCoins(deps.Ctx, evm.ModuleName, spendableCoins)
			s.Require().NoError(err)
			err = deps.Chain.BankKeeper.SendCoinsFromModuleToAccount(
				deps.Ctx, evm.ModuleName, deps.Sender.NibiruAddr, spendableCoins,
			)
			s.Require().NoError(err)

			// Create fun token from coin
			createFunTokenResp, err := deps.K.CreateFunToken(
				ctx,
				&evm.MsgCreateFunToken{
					FromBankDenom: bankDenom,
					Sender:        deps.Sender.NibiruAddr.String(),
				},
			)
			s.Require().NoError(err)
			funTokenErc20Addr := createFunTokenResp.FuntokenMapping.Erc20Addr.ToAddr()

			// Send fun token to ERC-20 contract
			_, err = deps.K.SendFunTokenToEvm(
				ctx,
				&evm.MsgSendFunTokenToEvm{
					Sender:    deps.Sender.NibiruAddr.String(),
					BankCoin:  sdk.Coin{Denom: tc.bankDenom, Amount: tc.amountToSend},
					ToEthAddr: recipientEVMAddr,
				},
			)
			if tc.wantErr != "" {
				s.Require().ErrorContains(err, tc.wantErr)
				return
			}
			s.Require().NoError(err)

			// Check 1: coins are stored on a module balance
			moduleBalance, err := deps.Chain.BankKeeper.Balance(ctx, &bank.QueryBalanceRequest{
				Address: evmModuleAddr.String(),
				Denom:   bankDenom,
			})
			s.Require().NoError(err)
			s.Equal(tc.amountToSend, moduleBalance.Balance.Amount)

			// Check 2: Sender balance reduced by send amount
			senderBalance, err := deps.Chain.BankKeeper.Balance(ctx, &bank.QueryBalanceRequest{
				Address: deps.Sender.NibiruAddr.String(),
				Denom:   bankDenom,
			})
			s.Require().NoError(err)
			s.Equal(tc.senderBalanceBefore.Sub(tc.amountToSend), senderBalance.Balance.Amount)

			// Check 3: erc-20 balance equals to send amount
			recipientERC20Balance, err := deps.K.CallContract(
				deps.Ctx,
				embeds.Contract_ERC20Minter.ABI,
				evm.ModuleAddressEVM(),
				&funTokenErc20Addr,
				false,
				"balanceOf",
				recipientEVMAddr.ToAddr(),
			)
			s.Require().NoError(err)
			res, err := embeds.Contract_ERC20Minter.ABI.Unpack(
				"balanceOf", recipientERC20Balance.Ret,
			)
			s.Require().NoError(err)
			s.Equal(1, len(res))
			s.Equal(tc.amountToSend.BigInt(), res[0])
		})
	}
}

// setBankDenomMetadata utility method to set bank denom metadata required for working with coin
func setBankDenomMetadata(ctx sdk.Context, bankKeeper bankkeeper.Keeper, bankDenom string) {
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
	bankKeeper.SetDenomMetaData(ctx, bankMetadata)
}

func (s *Suite) TestERC20Calls() {
	deps := evmtest.NewTestDeps()
	bankDenom := "ibc/btc"
	funtoken := evmtest.CreateFunTokenForBankCoin(&deps, bankDenom, &s.Suite)
	contract := funtoken.Erc20Addr.ToAddr()

	theUser := deps.Sender.EthAddr
	theEvm := evm.ModuleAddressEVM()

	s.T().Log("Mint tokens - Fail from non-owner")
	{
		from := theUser
		to := theUser
		_, err := deps.K.ERC20().Mint(contract, from, to, big.NewInt(69_420), deps.Ctx)
		s.ErrorContains(err, evm.ErrOwnable)
	}

	s.T().Log("Mint tokens - Success")
	{
		from := theEvm
		to := theEvm

		_, err := deps.K.ERC20().Mint(contract, from, to, big.NewInt(69_420), deps.Ctx)
		s.NoError(err)

		evmtest.AssertERC20BalanceEqual(s.T(), deps, contract, theUser, big.NewInt(0))
		evmtest.AssertERC20BalanceEqual(s.T(), deps, contract, theEvm, big.NewInt(69_420))
	}

	s.T().Log("Transfer - Not enough funds")
	{
		from := theUser
		to := theEvm
		_, err := deps.K.ERC20().Transfer(contract, from, to, big.NewInt(9_420), deps.Ctx)
		s.ErrorContains(err, "ERC20: transfer amount exceeds balance")
		// balances unchanged
		evmtest.AssertERC20BalanceEqual(s.T(), deps, contract, theUser, big.NewInt(0))
		evmtest.AssertERC20BalanceEqual(s.T(), deps, contract, theEvm, big.NewInt(69_420))
	}

	s.T().Log("Transfer - Success (sanity check)")
	{
		from := theEvm
		to := theUser
		_, err := deps.K.ERC20().Transfer(contract, from, to, big.NewInt(9_420), deps.Ctx)
		s.NoError(err)
		evmtest.AssertERC20BalanceEqual(s.T(), deps, contract, theUser, big.NewInt(9_420))
		evmtest.AssertERC20BalanceEqual(s.T(), deps, contract, theEvm, big.NewInt(60_000))
	}

	s.T().Log("Burn tokens - Allowed as non-owner")
	{
		from := theUser
		_, err := deps.K.ERC20().Burn(contract, from, big.NewInt(420), deps.Ctx)
		s.NoError(err)

		from = theEvm
		_, err = deps.K.ERC20().Burn(contract, from, big.NewInt(6_000), deps.Ctx)
		s.NoError(err)

		evmtest.AssertERC20BalanceEqual(s.T(), deps, contract, theUser, big.NewInt(9_000))
		evmtest.AssertERC20BalanceEqual(s.T(), deps, contract, theEvm, big.NewInt(54_000))
	}
}
