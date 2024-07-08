// Copyright (c) 2023-2024 Nibi, Inc.
package keeper_test

import (
	"fmt"

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
			FromErc20: erc20Addr,
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
			FromErc20: erc20Addr,
			Sender:    deps.Sender.NibiruAddr.String(),
		},
	)
	s.ErrorContains(err, "Funtoken mapping already created")

	s.T().Log("sad: CreateFunToken for the ERC20: invalid sender")
	_, err = deps.K.CreateFunToken(
		deps.GoCtx(),
		&evm.MsgCreateFunToken{
			FromErc20: erc20Addr,
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
	deps.Chain.BankKeeper.SetDenomMetaData(deps.Ctx, bankMetadata)

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
