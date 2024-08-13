package precompile_test

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/NibiruChain/nibiru/v2/x/common/testutil"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/evm/precompile"

	"github.com/stretchr/testify/suite"
)

type Suite struct {
	suite.Suite
}

// TestPrecompileSuite: Runs all the tests in the suite.
func TestSuite(t *testing.T) {
	s := new(Suite)
	suite.Run(t, s)
}

func (s *Suite) TestPrecompile_FunToken() {
	s.Run("PrecompileExists", s.FunToken_PrecompileExists)
	s.Run("HappyPath", s.FunToken_HappyPath)
}

// PrecompileExists: An integration test showing that a "PrecompileError" occurs
// when calling the FunToken
func (s *Suite) FunToken_PrecompileExists() {
	precompileAddr := precompile.PrecompileAddr_FuntokenGateway
	abi := embeds.SmartContract_FunToken.ABI
	deps := evmtest.NewTestDeps()

	codeResp, err := deps.EvmKeeper.Code(
		deps.GoCtx(),
		&evm.QueryCodeRequest{
			Address: precompileAddr.String(),
		},
	)
	s.NoError(err)
	s.Equal(string(codeResp.Code), "")

	s.True(deps.EvmKeeper.IsAvailablePrecompile(precompileAddr.ToAddr()),
		"did not see precompile address during \"InitPrecompiles\"")

	callArgs := []any{"nonsense", "args here", "to see if", "precompile is", "called"}
	methodName := string(precompile.FunTokenMethod_BankSend)
	packedArgs, err := abi.Pack(methodName, callArgs...)
	if err != nil {
		err = fmt.Errorf("failed to pack ABI args: %w", err) // easier to read
	}
	s.ErrorContains(
		err, fmt.Sprintf("argument count mismatch: got %d for 3", len(callArgs)),
		"callArgs: ", callArgs)

	fromEvmAddr := evm.EVM_MODULE_ADDRESS
	contractAddr := precompileAddr.ToAddr()
	commit := true
	bytecodeForCall := packedArgs
	_, err = deps.EvmKeeper.CallContractWithInput(
		deps.Ctx, fromEvmAddr, &contractAddr, commit,
		bytecodeForCall,
	)
	s.ErrorContains(err, "precompile error")
}

func (s *Suite) FunToken_HappyPath() {
	precompileAddr := precompile.PrecompileAddr_FuntokenGateway
	abi := embeds.SmartContract_FunToken.ABI
	deps := evmtest.NewTestDeps()

	theUser := deps.Sender.EthAddr
	theEvm := evm.EVM_MODULE_ADDRESS

	s.True(deps.EvmKeeper.IsAvailablePrecompile(precompileAddr.ToAddr()),
		"did not see precompile address during \"InitPrecompiles\"")

	s.T().Log("Create FunToken mapping and ERC20")
	bankDenom := "ibc/usdc"
	funtoken := evmtest.CreateFunTokenForBankCoin(&deps, bankDenom, &s.Suite)
	contract := funtoken.Erc20Addr.ToAddr()

	s.T().Log("Balances of the ERC20 should start empty")
	evmtest.AssertERC20BalanceEqual(s.T(), deps, contract, theUser, big.NewInt(0))
	evmtest.AssertERC20BalanceEqual(s.T(), deps, contract, theEvm, big.NewInt(0))

	s.T().Log("Mint tokens - Fail from non-owner")
	{
		from := theUser
		to := theUser
		input, err := embeds.SmartContract_ERC20Minter.ABI.Pack("mint", to, big.NewInt(69_420))
		s.NoError(err)
		_, err = evmtest.DoEthTx(&deps, contract, from, input)
		s.ErrorContains(err, "Ownable: caller is not the owner")
	}

	s.T().Log("Mint tokens - Success")
	{
		from := theEvm
		to := theUser
		input, err := embeds.SmartContract_ERC20Minter.ABI.Pack("mint", to, big.NewInt(69_420))
		s.NoError(err)

		_, err = evmtest.DoEthTx(&deps, contract, from, input)
		s.NoError(err)
		evmtest.AssertERC20BalanceEqual(s.T(), deps, contract, theUser, big.NewInt(69_420))
		evmtest.AssertERC20BalanceEqual(s.T(), deps, contract, theEvm, big.NewInt(0))
	}

	s.T().Log("Transfer - Success (sanity check)")
	randomAcc := testutil.AccAddress()
	{
		from := theUser
		to := theEvm
		_, err := deps.EvmKeeper.ERC20().Transfer(contract, from, to, big.NewInt(1), deps.Ctx)
		s.NoError(err)
		evmtest.AssertERC20BalanceEqual(s.T(), deps, contract, theUser, big.NewInt(69_419))
		evmtest.AssertERC20BalanceEqual(s.T(), deps, contract, theEvm, big.NewInt(1))
		s.Equal("0",
			deps.App.BankKeeper.GetBalance(deps.Ctx, randomAcc, funtoken.BankDenom).Amount.String(),
		)
	}

	s.T().Log("Send using precompile")
	amtToSend := int64(419)
	callArgs := precompile.ArgsFunTokenBankSend(contract, big.NewInt(amtToSend), randomAcc)
	methodName := string(precompile.FunTokenMethod_BankSend)
	input, err := abi.Pack(methodName, callArgs...)
	s.NoError(err)

	from := theUser
	_, err = evmtest.DoEthTx(&deps, precompileAddr.ToAddr(), from, input)
	s.Require().NoError(err)

	evmtest.AssertERC20BalanceEqual(s.T(), deps, contract, theUser, big.NewInt(69_419-amtToSend))
	evmtest.AssertERC20BalanceEqual(s.T(), deps, contract, theEvm, big.NewInt(1))
	s.Equal(fmt.Sprintf("%d", amtToSend),
		deps.App.BankKeeper.GetBalance(deps.Ctx, randomAcc, funtoken.BankDenom).Amount.String(),
	)

	evmtest.AssertERC20BalanceEqual(s.T(), deps, contract, theUser, big.NewInt(69_000))
	evmtest.AssertERC20BalanceEqual(s.T(), deps, contract, theEvm, big.NewInt(1))
	s.Equal("419",
		deps.App.BankKeeper.GetBalance(deps.Ctx, randomAcc, funtoken.BankDenom).Amount.String(),
	)
}
