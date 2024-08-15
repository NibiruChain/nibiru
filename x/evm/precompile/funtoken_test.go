package precompile_test

import (
	"fmt"
	"math/big"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/evm/precompile"
)

type Suite struct {
	suite.Suite
}

// TestPrecompileSuite: Runs all the tests in the suite.
func TestSuite(t *testing.T) {
	suite.Run(t, new(Suite))
}

// PrecompileExists: An integration test showing that a "PrecompileError" occurs
// when calling the FunToken
func (s *Suite) TestPrecompileExists() {
	abi := embeds.SmartContract_FunToken.ABI
	deps := evmtest.NewTestDeps()

	codeResp, err := deps.EvmKeeper.Code(
		sdk.WrapSDKContext(deps.Ctx),
		&evm.QueryCodeRequest{
			Address: precompile.PrecompileAddr_FunToken.String(),
		},
	)
	s.Require().NoError(err)
	s.Equal(string(codeResp.Code), "")

	s.True(deps.EvmKeeper.IsAvailablePrecompile(precompile.PrecompileAddr_FunToken),
		"did not see precompile address during \"InitPrecompiles\"")

	callArgs := []any{"nonsense", "args here", "to see if", "precompile is", "called"}
	input, err := abi.Pack(string(precompile.FunTokenMethod_BankSend), callArgs...)
	s.Require().ErrorContains(
		err, fmt.Sprintf("argument count mismatch: got %d for 3", len(callArgs)),
		"callArgs: ", callArgs)
	s.Require().Nil(input)

	_, err = deps.EvmKeeper.CallContractWithInput(
		deps.Ctx, evm.EVM_MODULE_ADDRESS, &precompile.PrecompileAddr_FunToken, true,
		input,
	)
	s.ErrorContains(err, "precompile error")
}

func (s *Suite) TestHappyPath() {
	deps := evmtest.NewTestDeps()

	s.True(deps.EvmKeeper.IsAvailablePrecompile(precompile.PrecompileAddr_FunToken),
		"did not see precompile address during \"InitPrecompiles\"")

	s.T().Log("Create FunToken mapping and ERC20")
	bankDenom := "unibi"
	funtoken := evmtest.CreateFunTokenForBankCoin(&deps, bankDenom, &s.Suite)
	contract := funtoken.Erc20Addr.Addr()

	s.T().Log("Balances of the ERC20 should start empty")
	evmtest.AssertERC20BalanceEqual(s.T(), deps, contract, deps.Sender.EthAddr, big.NewInt(0))
	evmtest.AssertERC20BalanceEqual(s.T(), deps, contract, evm.EVM_MODULE_ADDRESS, big.NewInt(0))

	s.Require().NoError(testapp.FundAccount(
		deps.App.BankKeeper,
		deps.Ctx,
		deps.Sender.NibiruAddr,
		sdk.NewCoins(sdk.NewCoin(bankDenom, sdk.NewInt(69_420))),
	))

	_, err := deps.EvmKeeper.ConvertCoinToEvm(
		sdk.WrapSDKContext(deps.Ctx),
		&evm.MsgConvertCoinToEvm{
			Sender:    deps.Sender.NibiruAddr.String(),
			BankCoin:  sdk.NewCoin(bankDenom, sdk.NewInt(69_420)),
			ToEthAddr: eth.NewHexAddr(deps.Sender.EthAddr),
		},
	)
	s.Require().NoError(err)

	s.T().Log("Mint tokens - Fail from non-owner")
	{
		input, err := embeds.SmartContract_ERC20Minter.ABI.Pack("mint", deps.Sender.EthAddr, big.NewInt(69_420))
		s.NoError(err)
		_, err = deps.EvmKeeper.CallContractWithInput(
			deps.Ctx, deps.Sender.EthAddr, &contract, true, input,
		)
		s.ErrorContains(err, "Ownable: caller is not the owner")
	}

	randomAcc := testutil.AccAddress()

	s.T().Log("Send using precompile")
	amtToSend := int64(420)
	callArgs := []any{contract, big.NewInt(amtToSend), randomAcc.String()}
	input, err := embeds.SmartContract_FunToken.ABI.Pack(string(precompile.FunTokenMethod_BankSend), callArgs...)
	s.NoError(err)

	_, err = deps.EvmKeeper.CallContractWithInput(
		deps.Ctx, deps.Sender.EthAddr, &precompile.PrecompileAddr_FunToken, true, input,
	)
	s.Require().NoError(err)

	evmtest.AssertERC20BalanceEqual(s.T(), deps, contract, deps.Sender.EthAddr, big.NewInt(69_000))
	evmtest.AssertERC20BalanceEqual(s.T(), deps, contract, evm.EVM_MODULE_ADDRESS, big.NewInt(0))
	s.Equal(sdk.NewInt(420),
		deps.App.BankKeeper.GetBalance(deps.Ctx, randomAcc, funtoken.BankDenom).Amount,
	)
}
