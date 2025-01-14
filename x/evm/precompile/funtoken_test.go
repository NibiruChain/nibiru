package precompile_test

import (
	"fmt"
	"math/big"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/evm/keeper"
	"github.com/NibiruChain/nibiru/v2/x/evm/precompile"
	"github.com/NibiruChain/nibiru/v2/x/evm/statedb"
)

// TestSuite: Runs all the tests in the suite.
func TestSuite(t *testing.T) {
	suite.Run(t, new(UtilsSuite))
	suite.Run(t, new(FuntokenSuite))
	suite.Run(t, new(WasmSuite))
}

type FuntokenSuite struct {
	suite.Suite

	deps     evmtest.TestDeps
	funtoken evm.FunToken
}

func (s *FuntokenSuite) SetupSuite() {
	s.deps = evmtest.NewTestDeps()
	s.funtoken = evmtest.CreateFunTokenForBankCoin(s.deps, "unibi", &s.Suite)
}

func (s *FuntokenSuite) TestFailToPackABI() {
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

	abi := embeds.SmartContract_FunToken.ABI

	for _, tc := range testcases {
		s.Run(tc.name, func() {
			input, err := abi.Pack(tc.methodName, tc.callArgs...)
			s.ErrorContains(err, tc.wantError)
			s.Nil(input)
		})
	}
}

func (s *FuntokenSuite) TestWhoAmI() {
	deps := evmtest.NewTestDeps()

	for accIdx, acc := range []evmtest.EthPrivKeyAcc{
		deps.Sender, evmtest.NewEthPrivAcc(),
	} {
		s.T().Logf("test account %d, use both address formats", accIdx)
		callWhoAmIWithArg := func(arg string) (evmResp *evm.MsgEthereumTxResponse, err error) {
			fmt.Printf("arg: %s", arg)
			contractInput, err := embeds.SmartContract_FunToken.ABI.Pack("whoAmI", arg)
			s.Require().NoError(err)
			evmCfg := deps.EvmKeeper.GetEVMConfig(deps.Ctx)
			txConfig := deps.EvmKeeper.TxConfig(deps.Ctx, gethcommon.BigToHash(big.NewInt(0)))
			stateDB := deps.EvmKeeper.NewStateDB(deps.Ctx, txConfig)
			evmMsg := gethcore.NewMessage(
				evm.EVM_MODULE_ADDRESS,              /*from*/
				&precompile.PrecompileAddr_FunToken, /*to*/
				deps.App.EvmKeeper.GetAccNonce(deps.Ctx, evm.EVM_MODULE_ADDRESS),
				big.NewInt(0),                     /*value*/
				evmtest.FunTokenGasLimitSendToEvm, /*gasLimit*/
				big.NewInt(0),                     /*gasPrice*/
				big.NewInt(0),                     /*gasFeeCap*/
				big.NewInt(0),                     /*gasTipCap*/
				contractInput,                     /*data*/
				gethcore.AccessList{},             /*accessList*/
				false,                             /*isFake*/
			)
			evmObj := deps.EvmKeeper.NewEVM(deps.Ctx, evmMsg, evmCfg, nil /*tracer*/, stateDB)
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
		for _, arg := range []string{acc.NibiruAddr.String(), acc.EthAddr.Hex()} {
			evmResp, err := callWhoAmIWithArg(arg)
			s.Require().NoError(err, evmResp)
			gotAddrEth, gotAddrBech32, err := new(FunTokenWhoAmIReturn).ParseFromResp(evmResp)
			s.NoError(err)
			s.Equal(acc.EthAddr.Hex(), gotAddrEth.Hex())
			s.Equal(acc.NibiruAddr.String(), gotAddrBech32)
		}
		// Sad path check
		evmResp, err := callWhoAmIWithArg("not_an_address")
		s.Require().ErrorContains(
			err, "could not parse address as Nibiru Bech32 or Ethereum hexadecimal", evmResp,
		)
	}
}

func (s *FuntokenSuite) TestHappyPath() {
	deps := evmtest.NewTestDeps()

	s.T().Log("Create FunToken mapping and ERC20")
	bankDenom := "unibi"
	funtoken := evmtest.CreateFunTokenForBankCoin(deps, bankDenom, &s.Suite)
	erc20 := funtoken.Erc20Addr.Address

	s.Require().NoError(testapp.FundAccount(
		deps.App.BankKeeper,
		deps.Ctx,
		deps.Sender.NibiruAddr,
		sdk.NewCoins(sdk.NewCoin(bankDenom, sdk.NewInt(69_420))),
	))

	s.T().Log("set up evmObj")
	stateDB := deps.EvmKeeper.NewStateDB(deps.Ctx, statedb.NewEmptyTxConfig(gethcommon.BytesToHash(deps.Ctx.HeaderHash())))
	evmObj := deps.EvmKeeper.NewEVM(deps.Ctx, gethcore.Message{}, deps.EvmKeeper.GetEVMConfig(deps.Ctx), evm.NewNoOpTracer(), stateDB)

	s.T().Log("Call IFunToken.bankBalance()")
	s.Run("IFunToken.bankBalance()", func() {
		contractInput, err := embeds.SmartContract_FunToken.ABI.Pack("bankBalance", deps.Sender.EthAddr, funtoken.BankDenom)
		s.Require().NoError(err)
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

	_, err := deps.EvmKeeper.ConvertCoinToEvm(
		sdk.WrapSDKContext(deps.Ctx),
		&evm.MsgConvertCoinToEvm{
			Sender:   deps.Sender.NibiruAddr.String(),
			BankCoin: sdk.NewCoin(bankDenom, sdk.NewInt(69_420)),
			ToEthAddr: eth.EIP55Addr{
				Address: deps.Sender.EthAddr,
			},
		},
	)
	s.Require().NoError(err)

	s.Run("Mint tokens - Fail from non-owner", func() {
		s.deps.ResetGasMeter()
		contractInput, err := embeds.SmartContract_ERC20Minter.ABI.Pack("mint", deps.Sender.EthAddr, big.NewInt(69_420))
		s.Require().NoError(err)
		_, err = deps.EvmKeeper.CallContractWithInput(
			deps.Ctx,
			evmObj,
			deps.Sender.EthAddr,
			&precompile.PrecompileAddr_FunToken,
			false,
			contractInput,
			evmtest.FunTokenGasLimitSendToEvm,
		)
		s.ErrorContains(err, "Ownable: caller is not the owner")
	})

	randomAcc := testutil.AccAddress()

	s.Run("IFunToken.sendToBank()", func() {
		deps.ResetGasMeter()

		input, err := embeds.SmartContract_FunToken.ABI.Pack(string(precompile.FunTokenMethod_sendToBank), erc20, big.NewInt(420), randomAcc.String())
		s.NoError(err)

		// err = testapp.FundFeeCollector(deps.App.BankKeeper, deps.Ctx,
		// 	sdkmath.NewInt(70_000),
		// )
		// s.NoError(err)

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
		s.True(deps.App.BankKeeper == deps.App.EvmKeeper.Bank)

		evmtest.AssertERC20BalanceEqualWithDescription(
			s.T(), deps, evmObj, erc20, deps.Sender.EthAddr, big.NewInt(69_000), "expect 69000 balance",
		)
		evmtest.AssertERC20BalanceEqualWithDescription(
			s.T(), deps, evmObj, erc20, evm.EVM_MODULE_ADDRESS, big.NewInt(0), "expect 0 balance",
		)
		s.Require().True(deps.App.BankKeeper.GetBalance(deps.Ctx, randomAcc, funtoken.BankDenom).Amount.Equal(sdk.NewInt(420)))
		s.Require().NotNil(deps.EvmKeeper.Bank.StateDB)

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
		contractInput, err := embeds.SmartContract_FunToken.ABI.Pack("balance", deps.Sender.EthAddr, erc20)
		s.Require().NoError(err)
		evmResp, err := deps.EvmKeeper.CallContractWithInput(
			deps.Ctx,
			evmObj,
			deps.Sender.EthAddr,
			&precompile.PrecompileAddr_FunToken,
			false,
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
	deps := s.deps
	randomAcc := testutil.AccAddress()
	deployResp, err := evmtest.DeployContract(
		&deps, embeds.SmartContract_TestFunTokenPrecompileLocalGas,
		s.funtoken.Erc20Addr.Address,
	)
	s.Require().NoError(err)
	contractAddr := deployResp.ContractAddr

	s.T().Log("create evmObj")
	stateDB := deps.EvmKeeper.NewStateDB(deps.Ctx, statedb.NewEmptyTxConfig(gethcommon.BytesToHash(deps.Ctx.HeaderHash())))
	evmObj := deps.EvmKeeper.NewEVM(deps.Ctx, gethcore.Message{}, deps.EvmKeeper.GetEVMConfig(deps.Ctx), nil /*tracer*/, stateDB)

	s.T().Log("Fund sender's wallet")
	s.Require().NoError(testapp.FundAccount(
		deps.App.BankKeeper,
		deps.Ctx,
		deps.Sender.NibiruAddr,
		sdk.NewCoins(sdk.NewCoin(s.funtoken.BankDenom, sdk.NewInt(1000))),
	))

	s.Run("Fund contract with erc20 coins", func() {
		_, err = deps.EvmKeeper.ConvertCoinToEvm(
			sdk.WrapSDKContext(deps.Ctx),
			&evm.MsgConvertCoinToEvm{
				Sender:   deps.Sender.NibiruAddr.String(),
				BankCoin: sdk.NewCoin(s.funtoken.BankDenom, sdk.NewInt(1000)),
				ToEthAddr: eth.EIP55Addr{
					Address: contractAddr,
				},
			},
		)
		s.Require().NoError(err)
	})

	s.Run("Happy: callBankSend with default gas", func() {
		s.deps.ResetGasMeter()
		contractInput, err := embeds.SmartContract_TestFunTokenPrecompileLocalGas.ABI.Pack(
			"callBankSend",
			big.NewInt(1),
			randomAcc.String(),
		)
		s.Require().NoError(err)
		_, err = deps.EvmKeeper.CallContractWithInput(
			deps.Ctx,
			evmObj,
			deps.Sender.EthAddr,
			&contractAddr,
			true,
			contractInput,
			evmtest.FunTokenGasLimitSendToEvm,
		)
		s.Require().NoError(err)
	})

	s.Run("Happy: callBankSend with local gas - sufficient gas amount", func() {
		s.deps.ResetGasMeter()
		contractInput, err := embeds.SmartContract_TestFunTokenPrecompileLocalGas.ABI.Pack(
			"callBankSendLocalGas",
			big.NewInt(1),
			randomAcc.String(),
			big.NewInt(int64(evmtest.FunTokenGasLimitSendToEvm)),
		)
		s.Require().NoError(err)
		_, err = deps.EvmKeeper.CallContractWithInput(
			deps.Ctx,
			evmObj,
			deps.Sender.EthAddr,
			&contractAddr,
			true,
			contractInput,
			evmtest.FunTokenGasLimitSendToEvm, // gasLimit for the entire call
		)
		s.Require().NoError(err)
	})

	s.Run("Sad: callBankSend with local gas - insufficient gas amount", func() {
		s.deps.ResetGasMeter()
		contractInput, err := embeds.SmartContract_TestFunTokenPrecompileLocalGas.ABI.Pack(
			"callBankSendLocalGas",
			big.NewInt(1),
			randomAcc.String(),
			big.NewInt(int64(evmtest.FunTokenGasLimitSendToEvm)),
		)
		s.Require().NoError(err)
		_, err = deps.EvmKeeper.CallContractWithInput(
			deps.Ctx,
			evmObj,
			deps.Sender.EthAddr,
			&contractAddr,
			true,
			contractInput,
			evmtest.FunTokenGasLimitSendToEvm, // gasLimit for the entire call
		)
		s.Require().ErrorContains(err, "execution reverted")
	})
}

func (s *FuntokenSuite) TestSendToEvm_MadeFromCoin() {
	deps := evmtest.NewTestDeps()

	s.T().Log("create evmObj")
	stateDB := deps.EvmKeeper.NewStateDB(deps.Ctx, statedb.NewEmptyTxConfig(gethcommon.BytesToHash(deps.Ctx.HeaderHash())))
	evmObj := deps.EvmKeeper.NewEVM(deps.Ctx, gethcore.Message{}, deps.EvmKeeper.GetEVMConfig(deps.Ctx), nil /*tracer*/, stateDB)

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
			"sendToEvm",
			bankDenom,
			big.NewInt(1000),
			deps.Sender.EthAddr.Hex(),
		)
		s.Require().NoError(err)

		deps.ResetGasMeter()
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

		s.T().Log("4) The response returns the actual minted/unescrowed amount")
		var amountSent *big.Int
		err = embeds.SmartContract_FunToken.ABI.UnpackIntoInterface(
			&amountSent, "sendToEvm", ethTxResp.Ret,
		)
		s.Require().NoError(err)
		s.Require().EqualValues(1000, amountSent.Int64(), "expect 1000 minted to EVM")

		s.T().Log("Check the user lost 1000 ulibi in bank")
		bankBal := deps.App.BankKeeper.GetBalance(deps.Ctx, deps.Sender.NibiruAddr, bankDenom).Amount.BigInt()
		s.EqualValues(big.NewInt(234), bankBal, "did user lose 1000 ulibi from bank?")

		// check the evm module account balance
		s.EqualValues(big.NewInt(1000), deps.App.BankKeeper.GetBalance(deps.Ctx, evm.EVM_MODULE_ADDRESS_NIBI, bankDenom).Amount.BigInt())

		s.T().Log("Check the user gained 1000 in ERC20 representation")
		evmtest.AssertERC20BalanceEqualWithDescription(s.T(), deps, evmObj, erc20Addr, deps.Sender.EthAddr, big.NewInt(1000), "expect 1000 balance")
	})

	//-----------------------------------------------------------------------
	// 5) Now send some tokens *back* to the bank via `sendToBank`.
	//-----------------------------------------------------------------------
	// We'll pick a brand new random account to receive them.
	recipient := testutil.AccAddress()

	s.Run("Sending 400 tokens back from EVM to Cosmos bank => recipient:", func() {
		deps.ResetGasMeter()
		contractInput, err := embeds.SmartContract_FunToken.ABI.Pack(
			string(precompile.FunTokenMethod_sendToBank),
			erc20Addr,
			big.NewInt(400),
			recipient.String(),
		)
		s.Require().NoError(err)

		ethExResp, err := deps.EvmKeeper.CallContractWithInput(
			deps.Ctx,
			evmObj,
			deps.Sender.EthAddr,
			&precompile.PrecompileAddr_FunToken,
			true,
			contractInput,
			evmtest.FunTokenGasLimitSendToEvm,
		)
		s.Require().NoError(err)
		s.Require().Empty(ethExResp.VmError, "sendToBank VMError")

		s.T().Log("Parse the returned amount from `sendToBank`")
		var actualSent *big.Int
		err = embeds.SmartContract_FunToken.ABI.UnpackIntoInterface(
			&actualSent, string(precompile.FunTokenMethod_sendToBank),
			ethExResp.Ret,
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
		s.Require().EqualValues(big.NewInt(400), deps.App.BankKeeper.GetBalance(deps.Ctx, recipient, bankDenom).Amount.BigInt(), "did the recipient get 400?")

		s.T().Log("Confirm module account doesn't keep them (burn or escrow) for bank-based tokens")
		s.Require().EqualValues(big.NewInt(600), deps.App.BankKeeper.GetBalance(deps.Ctx, evm.EVM_MODULE_ADDRESS[:], bankDenom).Amount.BigInt(), "module should now have 600 left escrowed")
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
	// 	- mint cosmos token
	// 	- escrow erc20 token

	// sendToEVM - Send 100 tokens from alice to bob's EVM address (Cosmos -> EVM)
	// 	- burn cosmos token
	// 	- unescrow erc20 token

	deps := evmtest.NewTestDeps()
	alice := evmtest.NewEthPrivAcc()
	bob := evmtest.NewEthPrivAcc()
	stateDB := deps.EvmKeeper.NewStateDB(deps.Ctx, statedb.NewEmptyTxConfig(gethcommon.Hash(deps.Ctx.HeaderHash())))
	evmObj := deps.EvmKeeper.NewEVM(deps.Ctx, gethcore.Message{}, deps.EvmKeeper.GetEVMConfig(deps.Ctx), evm.NewNoOpTracer(), stateDB)

	// Fund user so they can create funtoken from an ERC20
	createFunTokenFee := deps.EvmKeeper.FeeForCreateFunToken(deps.Ctx)
	s.Require().NoError(testapp.FundAccount(
		deps.App.BankKeeper, deps.Ctx, deps.Sender.NibiruAddr,
		createFunTokenFee,
	))

	// Deploy an ERC20 with 18 decimals
	erc20Resp, err := evmtest.DeployContract(&deps, embeds.SmartContract_TestERC20)

	s.Require().NoError(err, "failed to deploy test ERC20")
	erc20Addr := erc20Resp.ContractAddr

	// the initial supply was sent to the deployer
	evmtest.AssertERC20BalanceEqualWithDescription(s.T(), deps, evmObj, erc20Addr, deps.Sender.EthAddr, bigTokens(1000000), "expect nonzero balance")

	// create fun token from that erc20
	_, err = deps.EvmKeeper.CreateFunToken(sdk.WrapSDKContext(deps.Ctx), &evm.MsgCreateFunToken{
		Sender:    deps.Sender.NibiruAddr.String(),
		FromErc20: &eth.EIP55Addr{Address: erc20Addr},
	})
	s.Require().NoError(err)

	// Transfer 500 tokens to bob => 500 * 10^18 raw
	deployerAddr := gethcommon.HexToAddress(erc20Resp.EthTxMsg.From)
	contractInput, err := embeds.SmartContract_TestERC20.ABI.Pack(
		"transfer",
		bob.EthAddr,
		bigTokens(500), // 500 in human sense
	)
	s.Require().NoError(err)
	_, err = deps.EvmKeeper.CallContractWithInput(
		deps.Ctx,
		evmObj,
		deployerAddr,
		&erc20Addr,
		true,
		contractInput,
		keeper.Erc20GasLimitExecute,
	)
	s.Require().NoError(err)

	// Now user should have 500 tokens => raw is 500 * 10^18
	evmtest.AssertERC20BalanceEqualWithDescription(s.T(), deps, evmObj, erc20Addr, bob.EthAddr, bigTokens(500), "expect nonzero balance")

	// sendToBank: e.g. 100 tokens => 100 * 1e18 raw
	// expects to escrow on EVM side and mint on cosmos side
	input, err := embeds.SmartContract_FunToken.ABI.Pack(
		string(precompile.FunTokenMethod_sendToBank),
		[]any{
			erc20Addr, // address
			bigTokens(100),
			alice.NibiruAddr.String(),
		}...,
	)
	s.Require().NoError(err)
	_, resp, err := evmtest.CallContractTx(&deps, precompile.PrecompileAddr_FunToken, input, bob)
	s.Require().NoError(err)
	s.Require().Empty(resp.VmError)

	// Bank side should see 100
	bankBal := deps.App.BankKeeper.GetBalance(deps.Ctx, alice.NibiruAddr, "erc20/"+erc20Addr.Hex())
	s.Require().EqualValues(bigTokens(100), bankBal.Amount.BigInt())

	// Expect user to have 400 tokens => 400 * 10^18
	evmtest.AssertERC20BalanceEqualWithDescription(s.T(), deps, evmObj, erc20Addr, bob.EthAddr, bigTokens(400), "expect nonzero balance")

	// 100 tokens are escrowed
	evmtest.AssertERC20BalanceEqualWithDescription(s.T(), deps, evmObj, erc20Addr, evm.EVM_MODULE_ADDRESS, bigTokens(100), "expect nonzero balance")

	// Finally sendToEvm(100) -> (expects to burn on cosmos side and unescrow in the EVM side)
	input2, err := embeds.SmartContract_FunToken.ABI.Pack(
		"sendToEvm",
		[]any{
			bankBal.Denom,
			bigTokens(100),
			bob.EthAddr.Hex(),
		}...,
	)
	s.Require().NoError(err)
	_, resp2, err := evmtest.CallContractTx(&deps, precompile.PrecompileAddr_FunToken, input2, alice)
	s.Require().NoError(err)
	s.Require().Empty(resp2.VmError)

	// no bank side left for alice
	balAfter := deps.App.BankKeeper.GetBalance(deps.Ctx, alice.NibiruAddr, bankBal.Denom).Amount.BigInt()
	s.Require().EqualValues(bigTokens(0), balAfter)

	// check bob has 500 tokens again => 500 * 1e18
	evmtest.AssertERC20BalanceEqualWithDescription(s.T(), deps, evmObj, erc20Addr, bob.EthAddr, bigTokens(500), "expect nonzero balance")

	// check evm module account's balance, it should have escrowed some tokens
	// unescrow the tokens
	evmtest.AssertERC20BalanceEqualWithDescription(s.T(), deps, evmObj, erc20Addr, evm.EVM_MODULE_ADDRESS, bigTokens(0), "expect zero balance")

	// burns the bank tokens
	evmBal2 := deps.App.BankKeeper.GetBalance(deps.Ctx, evm.EVM_MODULE_ADDRESS[:], bankBal.Denom).Amount.BigInt()
	s.Require().EqualValues(bigTokens(0), evmBal2)

	// user has 500 tokens again => 500 * 1e18
	evmtest.AssertERC20BalanceEqualWithDescription(s.T(), deps, evmObj, erc20Addr, bob.EthAddr, bigTokens(500), "expect nonzero balance")
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
