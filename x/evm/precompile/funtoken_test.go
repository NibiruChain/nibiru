package precompile_test

import (
	"fmt"
	"math/big"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
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

// TestSuite: Runs all the tests in the suite.
type FuntokenSuite struct {
	suite.Suite
}

func TestFuntokenSuite(t *testing.T) {
	suite.Run(t, new(FuntokenSuite))
}

func TestFailToPackABI(t *testing.T) {
	testcases := []struct {
		name       string
		methodName string
		callArgs   []any
		wantError  string
	}{
		{
			name:       "wrong amount of call args",
			methodName: string(precompile.FunTokenMethod_sendToBank),
			callArgs:   []any{"nonsense", "args here", "to see if", "precompile is", "called"},
			wantError:  "argument count mismatch: got 5 for 3",
		},
		{
			name:       "wrong type for address",
			methodName: string(precompile.FunTokenMethod_sendToBank),
			callArgs:   []any{"nonsense", "foo", "bar"},
			wantError:  "abi: cannot use string as type array as argument",
		},
		{
			name:       "wrong type for amount",
			methodName: string(precompile.FunTokenMethod_sendToBank),
			callArgs:   []any{gethcommon.HexToAddress("0x7D4B7B8CA7E1a24928Bb96D59249c7a5bd1DfBe6"), "foo", testutil.AccAddress().String()},
			wantError:  "abi: cannot use string as type ptr as argument",
		},
		{
			name:       "wrong type for recipient",
			methodName: string(precompile.FunTokenMethod_sendToBank),
			callArgs:   []any{gethcommon.HexToAddress("0x7D4B7B8CA7E1a24928Bb96D59249c7a5bd1DfBe6"), big.NewInt(1), 111},
			wantError:  "abi: cannot use int as type string as argument",
		},
		{
			name:       "invalid method name",
			methodName: "foo",
			callArgs:   []any{gethcommon.HexToAddress("0x7D4B7B8CA7E1a24928Bb96D59249c7a5bd1DfBe6"), big.NewInt(1), testutil.AccAddress().String()},
			wantError:  "method 'foo' not found",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			input, err := embeds.SmartContract_FunToken.ABI.Pack(tc.methodName, tc.callArgs...)
			require.ErrorContains(t, err, tc.wantError)
			require.Nil(t, input)
		})
	}
}

func TestWhoAmI(t *testing.T) {
	deps := evmtest.NewTestDeps()

	callWhoAmI := func(arg string) (evmResp *evm.MsgEthereumTxResponse, err error) {
		fmt.Printf("arg: %s", arg)
		contractInput, err := embeds.SmartContract_FunToken.ABI.Pack("whoAmI", arg)
		require.NoError(t, err)
		evmObj, _ := deps.NewEVM()
		return deps.EvmKeeper.CallContractWithInput(
			deps.Ctx,
			evmObj,
			deps.Sender.EthAddr,
			&precompile.PrecompileAddr_FunToken,
			false,
			contractInput,
			evmtest.FunTokenGasLimitSendToEvm,
		)
	}

	for accIdx, acc := range []evmtest.EthPrivKeyAcc{
		deps.Sender, evmtest.NewEthPrivAcc(),
	} {
		t.Logf("test account %d, use both address formats", accIdx)

		for _, arg := range []string{acc.NibiruAddr.String(), acc.EthAddr.Hex()} {
			evmResp, err := callWhoAmI(arg)
			require.NoError(t, err)
			gotAddrEth, gotAddrBech32, err := new(FunTokenWhoAmIReturn).ParseFromResp(evmResp)
			require.NoError(t, err)
			require.Equal(t, acc.EthAddr.Hex(), gotAddrEth.Hex())
			require.Equal(t, acc.NibiruAddr.String(), gotAddrBech32)
		}
		// Sad path check
		_, err := callWhoAmI("not_an_address")
		require.ErrorContains(t, err, "could not parse address as Nibiru Bech32 or Ethereum hexadecimal")
	}
}

func (s *FuntokenSuite) TestHappyPath() {
	deps := evmtest.NewTestDeps()

	s.T().Log("Create FunToken mapping and ERC20")
	funtoken := evmtest.CreateFunTokenForBankCoin(deps, evm.EVMBankDenom, &s.Suite)
	erc20 := funtoken.Erc20Addr.Address

	s.Require().NoError(testapp.FundAccount(
		deps.App.BankKeeper,
		deps.Ctx,
		deps.Sender.NibiruAddr,
		sdk.NewCoins(sdk.NewCoin(evm.EVMBankDenom, sdk.NewInt(69_420))),
	))

	s.Run("IFunToken.bankBalance()", func() {
		contractInput, err := embeds.SmartContract_FunToken.ABI.Pack("bankBalance", deps.Sender.EthAddr, funtoken.BankDenom)
		s.Require().NoError(err)
		evmObj, _ := deps.NewEVM()
		evmResp, err := deps.EvmKeeper.CallContractWithInput(
			deps.Ctx,
			evmObj,
			deps.Sender.EthAddr,
			&precompile.PrecompileAddr_FunToken,
			false,
			contractInput,
			evmtest.FunTokenGasLimitSendToEvm,
		)
		s.Require().NoError(err, evmResp)

		bal, ethAddr, bech32Addr, err := new(FunTokenBankBalanceReturn).ParseFromResp(evmResp)
		s.NoError(err)
		s.Require().Zero(bal.Cmp(big.NewInt(69_420)))
		s.Equal(deps.Sender.EthAddr.Hex(), ethAddr.Hex())
		s.Equal(deps.Sender.NibiruAddr.String(), bech32Addr)
	})

	s.Run("ConvertCoinToEvm", func() {
		_, err := deps.EvmKeeper.ConvertCoinToEvm(
			sdk.WrapSDKContext(deps.Ctx),
			&evm.MsgConvertCoinToEvm{
				Sender:   deps.Sender.NibiruAddr.String(),
				BankCoin: sdk.NewCoin(evm.EVMBankDenom, sdk.NewInt(69_420)),
				ToEthAddr: eth.EIP55Addr{
					Address: deps.Sender.EthAddr,
				},
			},
		)
		s.Require().NoError(err)
		evmObj, _ := deps.NewEVM()
		evmtest.AssertERC20BalanceEqualWithDescription(
			s.T(), deps, evmObj, erc20, deps.Sender.EthAddr, big.NewInt(69_420), "expect 69420 balance",
		)
		evmtest.AssertBankBalanceEqualWithDescription(s.T(), deps, evm.EVMBankDenom, deps.Sender.EthAddr, big.NewInt(0), "expect the sender to have zero balance")
		evmtest.AssertBankBalanceEqualWithDescription(s.T(), deps, evm.EVMBankDenom, evm.EVM_MODULE_ADDRESS, big.NewInt(69_420), "expect x/evm module to escrow all tokens")
	})

	s.Run("Mint tokens - Fail from non-owner", func() {
		contractInput, err := embeds.SmartContract_ERC20MinterWithMetadataUpdates.ABI.Pack("mint", deps.Sender.EthAddr, big.NewInt(69_420))
		evmObj, _ := deps.NewEVM()
		s.Require().NoError(err)
		_, err = deps.EvmKeeper.CallContractWithInput(
			deps.Ctx,
			evmObj,
			deps.Sender.EthAddr,
			&erc20,
			false,
			contractInput,
			evmtest.FunTokenGasLimitSendToEvm,
		)
		s.ErrorContains(err, "Ownable: caller is not the owner")
	})

	s.Run("IFunToken.sendToBank()", func() {
		randomAcc := testutil.AccAddress()

		input, err := embeds.SmartContract_FunToken.ABI.Pack(string(precompile.FunTokenMethod_sendToBank), erc20, big.NewInt(420), randomAcc.String())
		s.NoError(err)

		evmObj, _ := deps.NewEVM()
		ethTxResp, err := deps.EvmKeeper.CallContractWithInput(
			deps.Ctx,
			evmObj,
			deps.Sender.EthAddr,
			&precompile.PrecompileAddr_FunToken,
			true, /*commit*/
			input,
			keeper.Erc20GasLimitExecute,
		)
		s.Require().NoError(err)
		s.Require().Empty(ethTxResp.VmError)

		evmtest.AssertERC20BalanceEqualWithDescription(
			s.T(), deps, evmObj, erc20, deps.Sender.EthAddr, big.NewInt(69_000), "expect 69000 balance remaining",
		)
		evmtest.AssertERC20BalanceEqualWithDescription(
			s.T(), deps, evmObj, erc20, evm.EVM_MODULE_ADDRESS, big.NewInt(0), "expect 0 balance",
		)
		evmtest.AssertBankBalanceEqualWithDescription(
			s.T(), deps, evm.EVMBankDenom, eth.NibiruAddrToEthAddr(randomAcc), big.NewInt(420), "expect 420 balance",
		)
		evmtest.AssertBankBalanceEqualWithDescription(
			s.T(), deps, evm.EVMBankDenom, evm.EVM_MODULE_ADDRESS, big.NewInt(69_000), "expect 69000 balance",
		)

		s.T().Log("Parse the response contract addr and response bytes")
		var sentAmt *big.Int
		s.NoError(embeds.SmartContract_FunToken.ABI.UnpackIntoInterface(
			&sentAmt,
			string(precompile.FunTokenMethod_sendToBank),
			ethTxResp.Ret,
		))
		s.Require().Zero(sentAmt.Cmp(big.NewInt(420)))
	})

	s.Run("IFuntoken.balance", func() {
		contractInput, err := embeds.SmartContract_FunToken.ABI.Pack(string(precompile.FunTokenMethod_balance), deps.Sender.EthAddr, erc20)
		s.Require().NoError(err)

		evmObj, _ := deps.NewEVM()
		evmResp, err := deps.EvmKeeper.CallContractWithInput(
			deps.Ctx,
			evmObj,
			deps.Sender.EthAddr,                 // from
			&precompile.PrecompileAddr_FunToken, // to
			false,                               // commit
			contractInput,
			keeper.Erc20GasLimitQuery,
		)
		s.Require().NoError(err, evmResp)

		bals, err := new(FunTokenBalanceReturn).ParseFromResp(evmResp)
		s.NoError(err)
		s.Equal(funtoken.Erc20Addr, bals.FunToken.Erc20Addr)
		s.Equal(funtoken.BankDenom, bals.FunToken.BankDenom)
		s.Equal(deps.Sender.EthAddr, bals.Account)
		s.Zero(bals.BalanceBank.Cmp(big.NewInt(0)))
		s.Zero(bals.BalanceERC20.Cmp(big.NewInt(69_000)))
	})
}

func (s *FuntokenSuite) TestPrecompileLocalGas() {
	deps := evmtest.NewTestDeps()
	funtoken := evmtest.CreateFunTokenForBankCoin(deps, evm.EVMBankDenom, &s.Suite)
	randomAcc := testutil.AccAddress()

	deployResp, err := evmtest.DeployContract(
		&deps, embeds.SmartContract_TestFunTokenPrecompileLocalGas,
		funtoken.Erc20Addr.Address,
	)
	s.Require().NoError(err)
	contractAddr := deployResp.ContractAddr

	s.Run("Fund sender's wallet", func() {
		s.Require().NoError(testapp.FundAccount(
			deps.App.BankKeeper,
			deps.Ctx,
			deps.Sender.NibiruAddr,
			sdk.NewCoins(sdk.NewCoin(funtoken.BankDenom, sdk.NewInt(1000))),
		))
	})

	s.Run("Fund contract with erc20 coins", func() {
		_, err = deps.EvmKeeper.ConvertCoinToEvm(
			sdk.WrapSDKContext(deps.Ctx),
			&evm.MsgConvertCoinToEvm{
				Sender:   deps.Sender.NibiruAddr.String(),
				BankCoin: sdk.NewCoin(funtoken.BankDenom, sdk.NewInt(1000)),
				ToEthAddr: eth.EIP55Addr{
					Address: contractAddr,
				},
			},
		)
		s.Require().NoError(err)
	})

	s.Run("Happy: callBankSend with default gas", func() {
		contractInput, err := embeds.SmartContract_TestFunTokenPrecompileLocalGas.ABI.Pack(
			"callBankSend",
			big.NewInt(1),
			randomAcc.String(),
		)
		s.Require().NoError(err)
		evmObj, _ := deps.NewEVM()
		resp, err := deps.EvmKeeper.CallContractWithInput(
			deps.Ctx,
			evmObj,
			deps.Sender.EthAddr,
			&contractAddr,
			true,
			contractInput,
			evmtest.FunTokenGasLimitSendToEvm,
		)
		s.Require().NoError(err)
		s.Require().NotZero(resp.GasUsed)
	})

	s.Run("Happy: callBankSend with local gas - sufficient gas amount", func() {
		contractInput, err := embeds.SmartContract_TestFunTokenPrecompileLocalGas.ABI.Pack(
			"callBankSendLocalGas",
			big.NewInt(1),
			randomAcc.String(),
			big.NewInt(int64(evmtest.FunTokenGasLimitSendToEvm)),
		)
		s.Require().NoError(err)
		evmObj, _ := deps.NewEVM()
		resp, err := deps.EvmKeeper.CallContractWithInput(
			deps.Ctx,
			evmObj,
			deps.Sender.EthAddr,
			&contractAddr,
			true,
			contractInput,
			evmtest.FunTokenGasLimitSendToEvm, // gasLimit for the entire call
		)
		s.Require().NoError(err)
		s.Require().NotZero(resp.GasUsed)
	})

	s.Run("Sad: callBankSend with local gas - insufficient gas amount", func() {
		contractInput, err := embeds.SmartContract_TestFunTokenPrecompileLocalGas.ABI.Pack(
			"callBankSendLocalGas",
			big.NewInt(1),
			randomAcc.String(),
			big.NewInt(50_000), // customGas - too small
		)
		s.Require().NoError(err)
		evmObj, _ := deps.NewEVM()
		resp, err := deps.EvmKeeper.CallContractWithInput(
			deps.Ctx,
			evmObj,
			deps.Sender.EthAddr,
			&contractAddr,
			true,
			contractInput,
			evmtest.FunTokenGasLimitSendToEvm, // gasLimit for the entire call
		)
		s.Require().ErrorContains(err, "execution reverted")
		s.Require().NotZero(resp.GasUsed)
	})
}

func (s *FuntokenSuite) TestSendToEvm_MadeFromCoin() {
	deps := evmtest.NewTestDeps()

	s.T().Log("create evmObj")
	evmObj, _ := deps.NewEVM()

	s.T().Log("1) Create a new FunToken from coin 'ulibi'")
	bankDenom := "ulibi"
	funtoken := evmtest.CreateFunTokenForBankCoin(deps, bankDenom, &s.Suite)
	erc20Addr := funtoken.Erc20Addr.Address

	s.T().Log("2) Fund the sender with some ulibi on the bank side")
	err := testapp.FundAccount(
		deps.App.BankKeeper,
		deps.Ctx,
		deps.Sender.NibiruAddr,
		sdk.NewCoins(sdk.NewCoin(bankDenom, sdk.NewInt(1234))),
	)
	s.Require().NoError(err)

	s.Run("Call sendToEvm(string bankDenom, uint256 amount, string to)", func() {
		contractInput, err := embeds.SmartContract_FunToken.ABI.Pack(
			string(precompile.FunTokenMethod_sendToEvm),
			bankDenom,
			big.NewInt(1000),
			deps.Sender.EthAddr.Hex(),
		)
		s.Require().NoError(err)

		ethTxResp, err := deps.EvmKeeper.CallContractWithInput(
			deps.Ctx,
			evmObj,
			deps.Sender.EthAddr,
			&precompile.PrecompileAddr_FunToken,
			true,
			contractInput,
			evmtest.FunTokenGasLimitSendToEvm,
		)
		s.Require().NoError(err)
		s.Require().Empty(ethTxResp.VmError, "sendToEvm VMError")

		s.T().Log("The response returns the actual minted/unescrowed amount")
		var amountSent *big.Int
		err = embeds.SmartContract_FunToken.ABI.UnpackIntoInterface(
			&amountSent, string(precompile.FunTokenMethod_sendToEvm), ethTxResp.Ret,
		)
		s.Require().NoError(err)
		s.Require().EqualValues(1000, amountSent.Int64(), "expect 1000 minted to EVM")

		s.T().Log("Check the user lost 1000 ulibi in bank")
		evmtest.AssertBankBalanceEqualWithDescription(s.T(), deps, bankDenom, deps.Sender.EthAddr, big.NewInt(234), "did user lose 1000 ulibi from bank?")

		s.T().Log("Check the module account has 1000 ulibi")
		evmtest.AssertBankBalanceEqualWithDescription(s.T(), deps, bankDenom, evm.EVM_MODULE_ADDRESS, big.NewInt(1000), "expect 1000 balance")

		s.T().Log("Check the user gained 1000 in ERC20 representation")
		evmtest.AssertERC20BalanceEqualWithDescription(s.T(), deps, evmObj, erc20Addr, deps.Sender.EthAddr, big.NewInt(1000), "expect 1000 balance")
	})

	//-----------------------------------------------------------------------
	// 5) Now send some tokens *back* to the bank via `sendToBank`.
	//-----------------------------------------------------------------------
	// We'll pick a brand new random account to receive them.

	s.Run("Sending 400 tokens back from EVM to Cosmos bank => recipient:", func() {
		randomRecipient := testutil.AccAddress()

		contractInput, err := embeds.SmartContract_FunToken.ABI.Pack(
			string(precompile.FunTokenMethod_sendToBank),
			erc20Addr,
			big.NewInt(400),
			randomRecipient.String(),
		)
		s.Require().NoError(err)

		ethTxResp, err := deps.EvmKeeper.CallContractWithInput(
			deps.Ctx,
			evmObj,
			deps.Sender.EthAddr,
			&precompile.PrecompileAddr_FunToken,
			true,
			contractInput,
			evmtest.FunTokenGasLimitSendToEvm,
		)
		s.Require().NoError(err)
		s.Require().Empty(ethTxResp.VmError, "sendToBank VMError")

		s.T().Log("Parse the returned amount from `sendToBank`")
		var actualSent *big.Int
		err = embeds.SmartContract_FunToken.ABI.UnpackIntoInterface(
			&actualSent, string(precompile.FunTokenMethod_sendToBank),
			ethTxResp.Ret,
		)
		s.Require().NoError(err)
		s.Require().EqualValues(big.NewInt(400), actualSent, "expect 400 minted back to bank")

		s.T().Log("Check sender's EVM balance has decreased by 400")
		// The sender started with 1000 after the first sendToEvm
		evmtest.AssertERC20BalanceEqualWithDescription(
			s.T(),
			deps,
			evmObj,
			erc20Addr,
			deps.Sender.EthAddr,
			big.NewInt(600), // 1000 - 400
			"expect 600 balance",
		)

		s.T().Log("Check the bank side got 400 more")
		evmtest.AssertBankBalanceEqualWithDescription(
			s.T(),
			deps,
			bankDenom,
			eth.NibiruAddrToEthAddr(randomRecipient),
			big.NewInt(400),
			"did the recipient get 400?",
		)

		s.T().Log("Confirm module account doesn't keep them (burn or escrow) for bank-based tokens")
		evmtest.AssertBankBalanceEqualWithDescription(
			s.T(),
			deps,
			bankDenom,
			evm.EVM_MODULE_ADDRESS,
			big.NewInt(600),
			"module should now have 600 left escrowed",
		)
	})
}

func bigTokens(n int64) *big.Int {
	e18 := big.NewInt(1e18) // 1e18
	return new(big.Int).Mul(big.NewInt(n), e18)
}

func (s *FuntokenSuite) TestSendToEvm_MadeFromERC20() {
	// Create ERC20 token

	// EVM Transfer - Send 500 tokens to Bob (EVM)

	// sendToBank -  Send 100 tokens from bob to alice's bank balance (EVM -> Cosmos)
	// 	- escrow erc20 token
	// 	- mint cosmos token

	// sendToEVM - Send 100 tokens from alice to bob's EVM address (Cosmos -> EVM)
	// 	- burn cosmos token
	// 	- unescrow erc20 token

	deps := evmtest.NewTestDeps()

	alice := evmtest.NewEthPrivAcc()
	bob := evmtest.NewEthPrivAcc()

	// Fund user so they can create funtoken from an ERC20
	s.Require().NoError(testapp.FundAccount(
		deps.App.BankKeeper, deps.Ctx, deps.Sender.NibiruAddr,
		deps.EvmKeeper.FeeForCreateFunToken(deps.Ctx),
	))

	// Deploy an ERC20 with 18 decimals
	erc20Resp, err := evmtest.DeployContract(&deps, embeds.SmartContract_TestERC20)
	s.Require().NoError(err, "failed to deploy test ERC20")
	erc20Addr := erc20Resp.ContractAddr

	// create fun token from that erc20
	_, err = deps.EvmKeeper.CreateFunToken(
		sdk.WrapSDKContext(deps.Ctx),
		&evm.MsgCreateFunToken{
			Sender:    deps.Sender.NibiruAddr.String(),
			FromErc20: &eth.EIP55Addr{Address: erc20Addr},
		},
	)
	s.Require().NoError(err)

	// Transfer 500 tokens to bob => 500 * 10^18 raw
	s.Run("Transfer 500 tokens to bob", func() {
		contractInput, err := embeds.SmartContract_TestERC20.ABI.Pack(
			"transfer",
			bob.EthAddr,
			bigTokens(500), // 500 in human sense
		)
		s.Require().NoError(err)
		evmObj, _ := deps.NewEVM()
		_, err = deps.EvmKeeper.CallContractWithInput(
			deps.Ctx,
			evmObj,
			deps.Sender.EthAddr,
			&erc20Addr,
			true,
			contractInput,
			keeper.Erc20GasLimitExecute,
		)
		s.Require().NoError(err)

		// Now user should have 500 tokens => raw is 500 * 10^18
		evmtest.AssertERC20BalanceEqualWithDescription(s.T(), deps, evmObj, erc20Addr, bob.EthAddr, bigTokens(500), "expect nonzero balance")
	})

	// sendToBank: e.g. 100 tokens => 100 * 1e18 raw
	// expects to escrow on EVM side and mint on cosmos side
	s.Run("send 100 tokens to alice", func() {
		contractInput, err := embeds.SmartContract_FunToken.ABI.Pack(
			string(precompile.FunTokenMethod_sendToBank),
			erc20Addr, // address
			bigTokens(100),
			alice.NibiruAddr.String(),
		)
		s.Require().NoError(err)
		evmObj, _ := deps.NewEVM()
		resp, err := deps.EvmKeeper.CallContractWithInput(
			deps.Ctx,
			evmObj,
			bob.EthAddr,                         /* from */
			&precompile.PrecompileAddr_FunToken, /* to */
			true,                                /* commit */
			contractInput,
			evmtest.FunTokenGasLimitSendToEvm, /* gasLimit */
		)
		s.Require().NoError(err)
		s.Require().Empty(resp.VmError)

		// Bank side should see 100
		evmtest.AssertBankBalanceEqualWithDescription(s.T(), deps, "erc20/"+erc20Addr.Hex(), alice.EthAddr, bigTokens(100), "expect 100 balance")

		// Expect user to have 400 tokens => 400 * 10^18
		evmtest.AssertERC20BalanceEqualWithDescription(s.T(), deps, evmObj, erc20Addr, bob.EthAddr, bigTokens(400), "expect Bob's balance to be 400")

		// 100 tokens are escrowed
		evmtest.AssertERC20BalanceEqualWithDescription(s.T(), deps, evmObj, erc20Addr, evm.EVM_MODULE_ADDRESS, bigTokens(100), "expect EVM module to escrow 100 tokens")
	})

	// Finally sendToEvm(100) -> (expects to burn on cosmos side and unescrow in the EVM side)
	s.Run("send 100 tokens back to Bob", func() {
		contractInput, err := embeds.SmartContract_FunToken.ABI.Pack(
			"sendToEvm",
			"erc20/"+erc20Addr.Hex(),
			bigTokens(100),
			bob.EthAddr.Hex(),
		)
		s.Require().NoError(err)
		evmObj, _ := deps.NewEVM()
		resp, err := deps.EvmKeeper.CallContractWithInput(
			deps.Ctx,
			evmObj,
			alice.EthAddr,
			&precompile.PrecompileAddr_FunToken,
			true,
			contractInput,
			evmtest.FunTokenGasLimitSendToEvm,
		)
		s.Require().NoError(err)
		s.Require().Empty(resp.VmError)

		// no bank side left for alice
		evmtest.AssertBankBalanceEqualWithDescription(s.T(), deps, "erc20/"+erc20Addr.Hex(), alice.EthAddr, bigTokens(0), "expect 0 balance")

		// check bob has 500 tokens again => 500 * 1e18
		evmtest.AssertERC20BalanceEqualWithDescription(s.T(), deps, evmObj, erc20Addr, bob.EthAddr, bigTokens(500), "expect nonzero balance")

		// check evm module account's balance, it should have escrowed some tokens
		// unescrow the tokens
		evmtest.AssertERC20BalanceEqualWithDescription(s.T(), deps, evmObj, erc20Addr, evm.EVM_MODULE_ADDRESS, bigTokens(0), "expect zero balance")

		// burns the bank tokens
		evmtest.AssertBankBalanceEqualWithDescription(s.T(), deps, "erc20/"+erc20Addr.Hex(), evm.EVM_MODULE_ADDRESS, bigTokens(0), "expect 0 balance")
	})
}

// FunTokenWhoAmIReturn holds the return values from the "IFuntoken.whoAmI"
// method. The return bytes from successful calls of that method can be ABI
// unpacked into this struct.
type FunTokenWhoAmIReturn struct {
	NibiruAcc struct {
		EthAddr    gethcommon.Address `abi:"ethAddr"`
		Bech32Addr string             `abi:"bech32Addr"`
	} `abi:"whoAddrs"`
}

func (out FunTokenWhoAmIReturn) ParseFromResp(
	evmResp *evm.MsgEthereumTxResponse,
) (ethAddr gethcommon.Address, bech32Addr string, err error) {
	err = embeds.SmartContract_FunToken.ABI.UnpackIntoInterface(
		&out,
		"whoAmI",
		evmResp.Ret,
	)
	if err != nil {
		return
	}
	return out.NibiruAcc.EthAddr, out.NibiruAcc.Bech32Addr, nil
}

// FunTokenBalanceReturn holds the return values from the "IFuntoken.balance"
// method. The return bytes from successful calls of that method can be ABI
// unpacked into this struct.
type FunTokenBalanceReturn struct {
	Erc20Bal *big.Int `abi:"erc20Balance"`
	BankBal  *big.Int `abi:"bankBalance"`
	Token    struct {
		Erc20     gethcommon.Address `abi:"erc20"`
		BankDenom string             `abi:"bankDenom"`
	} `abi:"token"`
	NibiruAcc struct {
		EthAddr    gethcommon.Address `abi:"ethAddr"`
		Bech32Addr string             `abi:"bech32Addr"`
	} `abi:"whoAddrs"`
}

func (out FunTokenBalanceReturn) ParseFromResp(
	evmResp *evm.MsgEthereumTxResponse,
) (bals evmtest.FunTokenBalanceAssert, err error) {
	err = embeds.SmartContract_FunToken.ABI.UnpackIntoInterface(
		&out,
		"balance",
		evmResp.Ret,
	)
	if err != nil {
		return
	}
	return evmtest.FunTokenBalanceAssert{
		FunToken: evm.FunToken{
			Erc20Addr: eth.EIP55Addr{Address: out.Token.Erc20},
			BankDenom: out.Token.BankDenom,
		},
		Account:      out.NibiruAcc.EthAddr,
		BalanceBank:  out.BankBal,
		BalanceERC20: out.Erc20Bal,
	}, nil
}

// FunTokenBankBalanceReturn holds the return values from the
// "IFuntoken.bankBalance" method. The return bytes from successful calls of that
// method can be ABI unpacked into this struct.
type FunTokenBankBalanceReturn struct {
	BankBal   *big.Int `abi:"bankBalance"`
	NibiruAcc struct {
		EthAddr    gethcommon.Address `abi:"ethAddr"`
		Bech32Addr string             `abi:"bech32Addr"`
	} `abi:"whoAddrs"`
}

func (out FunTokenBankBalanceReturn) ParseFromResp(
	evmResp *evm.MsgEthereumTxResponse,
) (bal *big.Int, ethAddr gethcommon.Address, bech32Addr string, err error) {
	err = embeds.SmartContract_FunToken.ABI.UnpackIntoInterface(
		&out,
		"bankBalance",
		evmResp.Ret,
	)
	if err != nil {
		return
	}
	return out.BankBal, out.NibiruAcc.EthAddr, out.NibiruAcc.Bech32Addr, nil
}
