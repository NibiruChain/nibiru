package precompile_test

import (
	"fmt"
	"math/big"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/eth"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/NibiruChain/nibiru/x/evm"
	"github.com/NibiruChain/nibiru/x/evm/embeds"
	"github.com/NibiruChain/nibiru/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/x/evm/precompile"

	"github.com/stretchr/testify/suite"
)

type Suite struct {
	suite.Suite
}

// TestPrecompileSuite: Runs all the tests in the suite.
func TestPrecompileSuite(t *testing.T) {
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
	abi := embeds.Contract_Funtoken.ABI
	deps := evmtest.NewTestDeps()

	codeResp, err := deps.K.Code(
		deps.GoCtx(),
		&evm.QueryCodeRequest{
			Address: precompileAddr.String(),
		},
	)
	s.NoError(err)
	s.Equal(string(codeResp.Code), "")

	s.True(deps.K.PrecompileSet().Has(precompileAddr.ToAddr()),
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

	fromEvmAddr := evm.ModuleAddressEVM()
	contractAddr := precompileAddr.ToAddr()
	commit := true
	bytecodeForCall := packedArgs
	_, err = deps.K.CallContractWithInput(
		deps.Ctx, fromEvmAddr, &contractAddr, commit,
		bytecodeForCall,
	)
	s.ErrorContains(err, "precompile error")
}

// FunToken_HappyPath: runs a whole process of sending fun token from bank to evm and back
// Steps:
// 1. Mint a new bank coin: ibc/usdc
// 2. Create fungible token from the bank coin
// 3. Send coins from bank to evm account
// 4. Send some coins between evm accounts
// 5. Send part of the tokens from evm to a bank account
func (s *Suite) FunToken_HappyPath() {
	precompileAddr := precompile.PrecompileAddr_FuntokenGateway
	abi := embeds.Contract_Funtoken.ABI
	deps := evmtest.NewTestDeps()

	bankDenom := "ibc/usdc"
	amountToSendToEvm := int64(69_420)
	amountToSendWithinEvm := int64(1)
	amountToSendFromEvm := int64(419)

	theUser := deps.Sender.EthAddr
	theEvm := evm.ModuleAddressEVM()

	s.True(deps.K.PrecompileSet().Has(precompileAddr.ToAddr()),
		"did not see precompile address during \"InitPrecompiles\"")

	// Create new fungible token
	s.T().Log("Create FunToken mapping and ERC20")
	funtoken := evmtest.CreateFunTokenForBankCoin(&deps, bankDenom, &s.Suite)
	contract := funtoken.Erc20Addr.ToAddr()

	// Mint coins for the sender
	spendableCoins := sdk.NewCoins(sdk.NewInt64Coin(bankDenom, amountToSendToEvm))
	err := deps.Chain.BankKeeper.MintCoins(deps.Ctx, evm.ModuleName, spendableCoins)
	s.Require().NoError(err)
	err = deps.Chain.BankKeeper.SendCoinsFromModuleToAccount(
		deps.Ctx, evm.ModuleName, deps.Sender.NibiruAddr, spendableCoins,
	)
	s.Require().NoError(err)

	s.T().Log("Balances of the ERC20 should start empty")
	evmtest.AssertERC20BalanceEqual(s.T(), &deps, contract, theUser, big.NewInt(0))
	evmtest.AssertERC20BalanceEqual(s.T(), &deps, contract, theEvm, big.NewInt(0))

	// Send fungible tokens from bank to EVM
	_, err = deps.K.SendFunTokenToEvm(
		deps.Ctx,
		&evm.MsgSendFunTokenToEvm{
			Sender:    deps.Sender.NibiruAddr.String(),
			BankCoin:  sdk.Coin{Denom: bankDenom, Amount: math.NewInt(amountToSendToEvm)},
			ToEthAddr: eth.MustNewHexAddrFromStr(theUser.String()),
		},
	)
	s.Require().NoError(err, "failed to send FunToken to EVM")

	evmtest.AssertERC20BalanceEqual(s.T(), &deps, contract, theUser, big.NewInt(amountToSendToEvm))
	evmtest.AssertERC20BalanceEqual(s.T(), &deps, contract, theEvm, big.NewInt(0))

	// Transfer tokens to another account within EVM
	s.T().Log("Transfer - Success (sanity check)")
	randomAcc := testutil.AccAddress()
	amountRemaining := amountToSendToEvm - amountToSendWithinEvm
	{
		from := theUser
		to := theEvm
		_, err := deps.K.ERC20().Transfer(contract, from, to, big.NewInt(amountToSendWithinEvm), deps.Ctx)
		s.NoError(err)
		evmtest.AssertERC20BalanceEqual(s.T(), &deps, contract, theUser, big.NewInt(amountRemaining))
		evmtest.AssertERC20BalanceEqual(s.T(), &deps, contract, theEvm, big.NewInt(amountToSendWithinEvm))
		s.Equal(
			int64(0),
			deps.Chain.BankKeeper.GetBalance(deps.Ctx, randomAcc, funtoken.BankDenom).Amount.Int64(),
		)
	}

	// Send fungible token from EVM to bank
	s.T().Log("Send using precompile")

	callArgs := precompile.ArgsFunTokenBankSend(contract, big.NewInt(amountToSendFromEvm), randomAcc)
	methodName := string(precompile.FunTokenMethod_BankSend)
	input, err := abi.Pack(methodName, callArgs...)
	s.NoError(err)

	from := theUser
	_, err = evmtest.DoEthTx(&deps, precompileAddr.ToAddr(), from, input)
	s.Require().NoError(err)

	evmtest.AssertERC20BalanceEqual(
		s.T(),
		&deps,
		contract,
		theUser,
		big.NewInt(amountRemaining-amountToSendFromEvm),
	)
	evmtest.AssertERC20BalanceEqual(s.T(), &deps, contract, theEvm, big.NewInt(amountToSendWithinEvm))
	s.Equal(fmt.Sprintf("%d", amountToSendFromEvm),
		deps.Chain.BankKeeper.GetBalance(deps.Ctx, randomAcc, funtoken.BankDenom).Amount.String(),
	)
	evmtest.AssertERC20BalanceEqual(s.T(), &deps, contract, theUser, big.NewInt(amountRemaining-amountToSendFromEvm))
	evmtest.AssertERC20BalanceEqual(s.T(), &deps, contract, theEvm, big.NewInt(amountToSendWithinEvm))
	s.Equal(
		amountToSendFromEvm,
		deps.Chain.BankKeeper.GetBalance(deps.Ctx, randomAcc, funtoken.BankDenom).Amount.Int64(),
	)
}
