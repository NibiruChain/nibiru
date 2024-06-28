package keeper_test

import (
	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/NibiruChain/nibiru/eth"
	"github.com/NibiruChain/nibiru/x/evm"
	"github.com/NibiruChain/nibiru/x/evm/evmtest"
)

func (s *KeeperSuite) TestInsertAndGet() {
	deps := evmtest.NewTestDeps()

	erc20Addr := gethcommon.HexToAddress("0xAEf9437FF23D48D73271a41a8A094DEc9ac71477")
	err := deps.K.FunTokens.SafeInsert(
		deps.Ctx,
		erc20Addr,
		"unibi",
		true,
	)
	s.Require().NoError(err)

	// test Get
	funToken, err := deps.K.FunTokens.Get(deps.Ctx, evm.NewFunTokenID(eth.NewHexAddr(erc20Addr), "unibi"))
	s.Require().NoError(err)
	s.Require().Equal(eth.HexAddr("0xAEf9437FF23D48D73271a41a8A094DEc9ac71477"), funToken.Erc20Addr)
	s.Require().Equal("unibi", funToken.BankDenom)
	s.Require().True(funToken.IsMadeFromCoin)

	// iter := deps.K.FunTokens.Indexes.BankDenom.ExactMatch(deps.Ctx, "unibi")
	// deps.K.FunTokens.Collect(ctx)
}

func (s *KeeperSuite) TestCollect() {
	deps := evmtest.NewTestDeps()

	erc20Addr := gethcommon.HexToAddress("0xAEf9437FF23D48D73271a41a8A094DEc9ac71477")
	err := deps.K.FunTokens.SafeInsert(
		deps.Ctx,
		erc20Addr,
		"unibi",
		true,
	)
	s.Require().NoError(err)

	// test Collect by bank denom
	iter := deps.K.FunTokens.Indexes.BankDenom.ExactMatch(deps.Ctx, "unibi")
	funTokens := deps.K.FunTokens.Collect(deps.Ctx, iter)
	s.Require().Len(funTokens, 1)
	s.Require().Equal(eth.HexAddr("0xAEf9437FF23D48D73271a41a8A094DEc9ac71477"), funTokens[0].Erc20Addr)
	s.Require().Equal("unibi", funTokens[0].BankDenom)
	s.Require().True(funTokens[0].IsMadeFromCoin)

	// test Collect by erc20 addr
	iter2 := deps.K.FunTokens.Indexes.ERC20Addr.ExactMatch(deps.Ctx, erc20Addr)
	funTokens = deps.K.FunTokens.Collect(deps.Ctx, iter2)
	s.Require().Len(funTokens, 1)
	s.Require().Equal(eth.HexAddr("0xAEf9437FF23D48D73271a41a8A094DEc9ac71477"), funTokens[0].Erc20Addr)
	s.Require().Equal("unibi", funTokens[0].BankDenom)
	s.Require().True(funTokens[0].IsMadeFromCoin)
}

func (s *KeeperSuite) TestDelete() {
	deps := evmtest.NewTestDeps()

	erc20Addr := gethcommon.HexToAddress("0xAEf9437FF23D48D73271a41a8A094DEc9ac71477")
	err := deps.K.FunTokens.SafeInsert(
		deps.Ctx,
		erc20Addr,
		"unibi",
		true,
	)
	s.Require().NoError(err)

	// test Delete
	err = deps.K.FunTokens.Delete(deps.Ctx, evm.NewFunTokenID(eth.NewHexAddr(erc20Addr), "unibi"))
	s.Require().NoError(err)

	// test Get
	_, err = deps.K.FunTokens.Get(deps.Ctx, evm.NewFunTokenID(eth.NewHexAddr(erc20Addr), "unibi"))
	s.Require().Error(err)
}
