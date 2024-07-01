// Copyright (c) 2023-2024 Nibi, Inc.
package keeper_test

import (
	"fmt"

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
		},
	)
	s.Error(err)
}
