package keeper_test

import (
	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/NibiruChain/nibiru/eth"
	"github.com/NibiruChain/nibiru/x/evm"
	"github.com/NibiruChain/nibiru/x/evm/evmtest"
)

func (s *Suite) TestInsertAndGet() {
	deps := evmtest.NewTestDeps()

	erc20Addr := gethcommon.HexToAddress("0xAEf9437FF23D48D73271a41a8A094DEc9ac71477")
	err := deps.EvmKeeper.FunTokens.SafeInsert(
		deps.Ctx,
		erc20Addr,
		"unibi",
		true,
	)
	s.Require().NoError(err)

	// test Get
	funToken, err := deps.EvmKeeper.FunTokens.Get(deps.Ctx, evm.NewFunTokenID(eth.NewHexAddr(erc20Addr), "unibi"))
	s.Require().NoError(err)
	s.Require().Equal(eth.HexAddr("0xAEf9437FF23D48D73271a41a8A094DEc9ac71477"), funToken.Erc20Addr)
	s.Require().Equal("unibi", funToken.BankDenom)
	s.Require().True(funToken.IsMadeFromCoin)

	// iter := deps.K.FunTokens.Indexes.BankDenom.ExactMatch(deps.Ctx, "unibi")
	// deps.K.FunTokens.Collect(ctx)
}

func (s *Suite) TestCollect() {
	deps := evmtest.NewTestDeps()

	erc20Addr := gethcommon.HexToAddress("0xAEf9437FF23D48D73271a41a8A094DEc9ac71477")
	err := deps.EvmKeeper.FunTokens.SafeInsert(
		deps.Ctx,
		erc20Addr,
		"unibi",
		true,
	)
	s.Require().NoError(err)

	// test Collect by bank denom
	iter := deps.EvmKeeper.FunTokens.Indexes.BankDenom.ExactMatch(deps.Ctx, "unibi")
	funTokens := deps.EvmKeeper.FunTokens.Collect(deps.Ctx, iter)
	s.Require().Len(funTokens, 1)
	s.Require().Equal(eth.HexAddr("0xAEf9437FF23D48D73271a41a8A094DEc9ac71477"), funTokens[0].Erc20Addr)
	s.Require().Equal("unibi", funTokens[0].BankDenom)
	s.Require().True(funTokens[0].IsMadeFromCoin)

	// test Collect by erc20 addr
	iter2 := deps.EvmKeeper.FunTokens.Indexes.ERC20Addr.ExactMatch(deps.Ctx, erc20Addr)
	funTokens = deps.EvmKeeper.FunTokens.Collect(deps.Ctx, iter2)
	s.Require().Len(funTokens, 1)
	s.Require().Equal(eth.HexAddr("0xAEf9437FF23D48D73271a41a8A094DEc9ac71477"), funTokens[0].Erc20Addr)
	s.Require().Equal("unibi", funTokens[0].BankDenom)
	s.Require().True(funTokens[0].IsMadeFromCoin)
}

func (s *Suite) TestDelete() {
	deps := evmtest.NewTestDeps()

	erc20Addr := gethcommon.HexToAddress("0xAEf9437FF23D48D73271a41a8A094DEc9ac71477")
	err := deps.EvmKeeper.FunTokens.SafeInsert(
		deps.Ctx,
		erc20Addr,
		"unibi",
		true,
	)
	s.Require().NoError(err)

	// test Delete
	err = deps.EvmKeeper.FunTokens.Delete(deps.Ctx, evm.NewFunTokenID(eth.NewHexAddr(erc20Addr), "unibi"))
	s.Require().NoError(err)

	// test Get
	_, err = deps.EvmKeeper.FunTokens.Get(deps.Ctx, evm.NewFunTokenID(eth.NewHexAddr(erc20Addr), "unibi"))
	s.Require().Error(err)
}
