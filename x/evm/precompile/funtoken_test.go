package precompile_test

import (
	"fmt"
	"math/big"
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
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
func TestSuite(t *testing.T) {
	suite.Run(t, new(UtilsSuite))
	suite.Run(t, new(FuntokenSuite))
	suite.Run(t, new(WasmSuite))
}

type FuntokenSuite struct {
	suite.Suite
	deps evmtest.TestDeps
}

func (s *FuntokenSuite) SetupSuite() {
	s.deps = evmtest.NewTestDeps()
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
			return deps.EvmKeeper.CallContract(
				deps.Ctx,
				embeds.SmartContract_FunToken.ABI,
				deps.Sender.EthAddr,
				&precompile.PrecompileAddr_FunToken,
				false,
				keeper.Erc20GasLimitExecute,
				"whoAmI",
				[]any{
					arg, // who
				}...,
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

func (s *FuntokenSuite) TestHappyPath() {
	deps := evmtest.NewTestDeps()

	s.T().Log("Create FunToken mapping and ERC20")
	bankDenom := "anycoin"
	funtoken := evmtest.CreateFunTokenForBankCoin(&deps, bankDenom, &s.Suite)

	erc20 := funtoken.Erc20Addr.Address

	s.T().Log("Balances of the ERC20 should start empty")
	evmtest.AssertERC20BalanceEqual(s.T(), deps, erc20, deps.Sender.EthAddr, big.NewInt(0))
	evmtest.AssertERC20BalanceEqual(s.T(), deps, erc20, evm.EVM_MODULE_ADDRESS, big.NewInt(0))

	s.Require().NoError(testapp.FundAccount(
		deps.App.BankKeeper,
		deps.Ctx,
		deps.Sender.NibiruAddr,
		sdk.NewCoins(sdk.NewCoin(funtoken.BankDenom, sdk.NewInt(69_420))),
	))

	s.Run("IFunToken.bankBalance", func() {
		s.Require().NotEmpty(funtoken.BankDenom)
		evmResp, err := deps.EvmKeeper.CallContract(
			deps.Ctx,
			embeds.SmartContract_FunToken.ABI,
			deps.Sender.EthAddr,
			&precompile.PrecompileAddr_FunToken,
			false,
			keeper.Erc20GasLimitExecute,
			"bankBalance",
			[]any{
				deps.Sender.EthAddr, // who
				funtoken.BankDenom,  // bankDenom
			}...,
		)
		s.Require().NoError(err, evmResp)

		bal, ethAddr, bech32Addr, err := new(FunTokenBankBalanceReturn).ParseFromResp(evmResp)
		s.NoError(err)
		s.Require().Equal("69420", bal.String())
		s.Equal(deps.Sender.EthAddr.Hex(), ethAddr.Hex())
		s.Equal(deps.Sender.NibiruAddr.String(), bech32Addr)
	})

	_, err := deps.EvmKeeper.ConvertCoinToEvm(
		sdk.WrapSDKContext(deps.Ctx),
		&evm.MsgConvertCoinToEvm{
			Sender:   deps.Sender.NibiruAddr.String(),
			BankCoin: sdk.NewCoin(funtoken.BankDenom, sdk.NewInt(69_420)),
			ToEthAddr: eth.EIP55Addr{
				Address: deps.Sender.EthAddr,
			},
		},
	)
	s.Require().NoError(err)

	s.T().Log("Mint tokens - Fail from non-owner")
	{
		_, err = deps.EvmKeeper.CallContract(deps.Ctx, embeds.SmartContract_ERC20Minter.ABI, deps.Sender.EthAddr, &erc20, true, keeper.Erc20GasLimitExecute, "mint", deps.Sender.EthAddr, big.NewInt(69_420))
		s.ErrorContains(err, "Ownable: caller is not the owner")
	}

	randomAcc := testutil.AccAddress()

	s.T().Log("Send NIBI (FunToken) using precompile")
	amtToSend := int64(420)
	callArgs := []any{erc20, big.NewInt(amtToSend), randomAcc.String()}
	input, err := embeds.SmartContract_FunToken.ABI.Pack(string(precompile.FunTokenMethod_sendToBank), callArgs...)
	s.NoError(err)

	s.Require().NoError(testapp.FundFeeCollector(deps.App.BankKeeper, deps.Ctx, sdkmath.NewInt(20)))
	_, ethTxResp, err := evmtest.CallContractTx(
		&deps,
		precompile.PrecompileAddr_FunToken,
		input,
		deps.Sender,
	)
	s.Require().NoError(err)
	s.Require().Empty(ethTxResp.VmError)
	s.True(deps.App.BankKeeper == deps.App.EvmKeeper.Bank)

	evmtest.AssertERC20BalanceEqual(
		s.T(), deps, erc20, deps.Sender.EthAddr, big.NewInt(69_000),
	)
	evmtest.AssertERC20BalanceEqual(
		s.T(), deps, erc20, evm.EVM_MODULE_ADDRESS, big.NewInt(0),
	)
	s.Equal(sdk.NewInt(420).String(),
		deps.App.BankKeeper.GetBalance(deps.Ctx, randomAcc, funtoken.BankDenom).Amount.String(),
	)
	s.Require().NotNil(deps.EvmKeeper.Bank.StateDB)

	s.T().Log("Parse the response contract addr and response bytes")
	var sentAmt *big.Int
	err = embeds.SmartContract_FunToken.ABI.UnpackIntoInterface(
		&sentAmt,
		string(precompile.FunTokenMethod_sendToBank),
		ethTxResp.Ret,
	)
	s.NoError(err)
	s.Require().Equal("420", sentAmt.String())

	s.Run("IFuntoken.balance", func() {
		evmResp, err := deps.EvmKeeper.CallContract(
			deps.Ctx,
			embeds.SmartContract_FunToken.ABI,
			deps.Sender.EthAddr,
			&precompile.PrecompileAddr_FunToken,
			false,
			keeper.Erc20GasLimitExecute,
			"balance",
			[]any{
				deps.Sender.EthAddr, // who
				erc20,               // funtoken
			}...,
		)
		s.Require().NoError(err, evmResp)

		bals, err := new(FunTokenBalanceReturn).ParseFromResp(evmResp)
		s.NoError(err)
		s.Equal(funtoken.Erc20Addr, bals.FunToken.Erc20Addr)
		s.Equal(funtoken.BankDenom, bals.FunToken.BankDenom)
		s.Equal(deps.Sender.EthAddr, bals.Account)
		s.Equal("0", bals.BalanceBank.String())
		s.Equal("69000", bals.BalanceERC20.String())
	})
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

func (s *FuntokenSuite) TestPrecompileLocalGas() {
	deps := s.deps
	randomAcc := testutil.AccAddress()
	bankDenom := "unibi"
	funtoken := evmtest.CreateFunTokenForBankCoin(&s.deps, bankDenom, &s.Suite)

	deployResp, err := evmtest.DeployContract(
		&deps, embeds.SmartContract_TestFunTokenPrecompileLocalGas,
		funtoken.Erc20Addr.Address,
	)
	s.Require().NoError(err)
	contractAddr := deployResp.ContractAddr

	s.T().Log("Fund sender's wallet")
	s.Require().NoError(testapp.FundAccount(
		deps.App.BankKeeper,
		deps.Ctx,
		deps.Sender.NibiruAddr,
		sdk.NewCoins(sdk.NewCoin(funtoken.BankDenom, sdk.NewInt(1000))),
	))

	s.T().Log("Fund contract with erc20 coins")
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

	s.T().Log("Happy: callBankSend with default gas")
	_, err = deps.EvmKeeper.CallContract(
		deps.Ctx,
		embeds.SmartContract_TestFunTokenPrecompileLocalGas.ABI,
		deps.Sender.EthAddr,
		&contractAddr,
		true,
		evmtest.FunTokenGasLimitSendToEvm,
		"callBankSend",
		big.NewInt(1),
		randomAcc.String(),
	)
	s.Require().NoError(err)

	s.T().Log("Happy: callBankSend with local gas - sufficient gas amount")
	_, err = deps.EvmKeeper.CallContract(
		deps.Ctx,
		embeds.SmartContract_TestFunTokenPrecompileLocalGas.ABI,
		deps.Sender.EthAddr,
		&contractAddr,
		true,
		evmtest.FunTokenGasLimitSendToEvm, // gasLimit for the entire call
		"callBankSendLocalGas",
		big.NewInt(1),      // erc20 amount
		randomAcc.String(), // to
		big.NewInt(int64(evmtest.FunTokenGasLimitSendToEvm)), // customGas
	)
	s.Require().NoError(err)

	s.T().Log("Sad: callBankSend with local gas - insufficient gas amount")
	_, err = deps.EvmKeeper.CallContract(
		deps.Ctx,
		embeds.SmartContract_TestFunTokenPrecompileLocalGas.ABI,
		deps.Sender.EthAddr,
		&contractAddr,
		true,
		evmtest.FunTokenGasLimitSendToEvm, // gasLimit for the entire call
		"callBankSendLocalGas",
		big.NewInt(1),      // erc20 amount
		randomAcc.String(), // to
		big.NewInt(50_000), // customGas - too small
	)
	s.Require().ErrorContains(err, "execution reverted")
}

func (s *FuntokenSuite) TestSendToEvm() {
	deps := evmtest.NewTestDeps()
	s.Require().NoError(testapp.FundFeeCollector(deps.App.BankKeeper, deps.Ctx, sdkmath.NewInt(20)))

	s.T().Log("1) Create a new FunToken from coin 'ulibi'")
	bankDenom := "ulibi"
	funtoken := evmtest.CreateFunTokenForBankCoin(&deps, bankDenom, &s.Suite)
	fmt.Println(funtoken)
	erc20Addr := funtoken.Erc20Addr.Address

	s.T().Log("2) Fund the sender with some ulibi on the bank side")
	err := testapp.FundAccount(
		deps.App.BankKeeper,
		deps.Ctx,
		deps.Sender.NibiruAddr,
		sdk.NewCoins(sdk.NewCoin(bankDenom, sdk.NewInt(1234))),
	)
	s.Require().NoError(err)

	s.T().Log("Check the user starts with 0 ERC20 tokens")
	evmtest.AssertERC20BalanceEqual(s.T(), deps, erc20Addr, deps.Sender.EthAddr, big.NewInt(0))

	s.T().Log("3) Call the new method: sendToEvm(string bankDenom, uint256 amount, string to)")
	callArgs := []any{
		bankDenom,
		big.NewInt(1000),          // amount
		deps.Sender.EthAddr.Hex(), // 'to' can be bech32 or hex
	}

	input, err := embeds.SmartContract_FunToken.ABI.Pack(
		"sendToEvm",
		callArgs...,
	)
	s.Require().NoError(err)

	_, ethTxResp, err := evmtest.CallContractTx(
		&deps,
		precompile.PrecompileAddr_FunToken,
		input,
		deps.Sender,
	)
	s.Require().NoError(err)
	s.Require().Empty(ethTxResp.VmError, "sendToEvm VMError")

	// 1000 tokens are escrowed on module address
	s.EqualValues(1000, deps.App.BankKeeper.GetBalance(deps.Ctx, evm.EVM_MODULE_ADDRESS[:], bankDenom).Amount.BigInt().Int64())

	s.T().Log("4) The response returns the actual minted/unescrowed amount")
	var actualMinted *big.Int
	err = embeds.SmartContract_FunToken.ABI.UnpackIntoInterface(
		&actualMinted, "sendToEvm", ethTxResp.Ret,
	)
	s.Require().NoError(err)
	s.Require().EqualValues(1000, actualMinted.Int64(), "expect 1000 minted to EVM")

	s.T().Log("Check the user lost 1000 ulibi in bank")
	wantBank := big.NewInt(234) // 1234 - 1000 => 234
	bankBal := deps.App.BankKeeper.GetBalance(deps.Ctx, deps.Sender.NibiruAddr, bankDenom).Amount.BigInt()
	s.EqualValues(wantBank, bankBal, "did user lose 1000 ulibi from bank?")

	// check the evm module account balance
	wantEvm := big.NewInt(1000)
	evmBal := deps.App.BankKeeper.GetBalance(deps.Ctx, evm.EVM_MODULE_ADDRESS[:], bankDenom).Amount.BigInt()
	s.EqualValues(wantEvm, evmBal, "did evm module properly mint ulibi?")

	s.T().Log("Check the user gained 1000 in ERC20 representation")
	evmtest.AssertERC20BalanceEqual(s.T(), deps, erc20Addr, deps.Sender.EthAddr, big.NewInt(1000))

	//-----------------------------------------------------------------------
	// 5) Now send some tokens *back* to the bank via `sendToBank`.
	//-----------------------------------------------------------------------
	// We'll pick a brand new random account to receive them.
	recipient := testutil.AccAddress()
	s.T().Logf("5) Sending 400 tokens back from EVM to Cosmos bank => recipient: %s", recipient)

	sendBackArgs := []any{
		erc20Addr,          // address erc20
		big.NewInt(400),    // amount
		recipient.String(), // to
	}

	inputSendBack, err := embeds.SmartContract_FunToken.ABI.Pack(
		string(precompile.FunTokenMethod_sendToBank),
		sendBackArgs...,
	)
	s.Require().NoError(err)

	_, ethTxResp2, err := evmtest.CallContractTx(
		&deps,
		precompile.PrecompileAddr_FunToken,
		inputSendBack,
		deps.Sender,
	)
	s.Require().NoError(err)
	s.Require().Empty(ethTxResp2.VmError, "sendToBank VMError")

	s.T().Log("Parse the returned amount from `sendToBank`")
	var actualSentBack *big.Int
	err = embeds.SmartContract_FunToken.ABI.UnpackIntoInterface(
		&actualSentBack, string(precompile.FunTokenMethod_sendToBank),
		ethTxResp2.Ret,
	)
	s.Require().NoError(err)
	s.Require().EqualValues(400, actualSentBack.Int64(), "expect 400 minted back to bank")

	s.T().Log("Check sender's EVM balance has decreased by 400")
	// The sender started with 1000 after the first sendToEvm
	evmtest.AssertERC20BalanceEqual(
		s.T(),
		deps,
		erc20Addr,
		deps.Sender.EthAddr,
		big.NewInt(600), // 1000 - 400
	)

	s.T().Log("Check the bank side got 400 more")
	recipientBal := deps.App.BankKeeper.GetBalance(deps.Ctx, recipient, bankDenom).Amount.BigInt()
	s.Require().EqualValues(400, recipientBal.Int64(), "did the recipient get 400?")

	s.T().Log("Confirm module account doesn't keep them (burn or escrow) for bank-based tokens")
	moduleBal := deps.App.BankKeeper.GetBalance(deps.Ctx, evm.EVM_MODULE_ADDRESS[:], bankDenom).Amount.BigInt()
	s.Require().EqualValues(600, moduleBal.Int64(), "module should now have 600 left escrowed")

	s.T().Log("Done! We sent tokens to EVM, then back to the bank, verifying the final balances.")
}

func bigTokens(n int64) *big.Int {
	e18 := big.NewInt(1e18) // 1e18
	return new(big.Int).Mul(big.NewInt(n), e18)
}

func (s *FuntokenSuite) TestSendToEvm_NotMadeFromCoin() {
	// Create ERC20 token

	// EVM Transfer - Send 500 tokens to Bob (EVM)

	// sendToBank -  Send 100 tokens from bob to alice's bank balance (EVM -> Cosmos)
	// 	- mint cosmos token
	// 	- escrow erc20 token

	// sendToEVM - Send 100 tokens from alice to bob's EVM address (Cosmos -> EVM)
	// 	- burn cosmos token
	// 	- unescrow erc20 token

	deps := evmtest.NewTestDeps()
	s.Require().NoError(testapp.FundFeeCollector(deps.App.BankKeeper, deps.Ctx, sdkmath.NewInt(20)))

	bob := evmtest.NewEthPrivAcc()
	alice := evmtest.NewEthPrivAcc()

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
	evmtest.AssertERC20BalanceEqual(s.T(), deps, erc20Addr, deps.Sender.EthAddr, bigTokens(1000000))

	// create fun token from that erc20
	_, err = deps.EvmKeeper.CreateFunToken(sdk.WrapSDKContext(deps.Ctx), &evm.MsgCreateFunToken{
		Sender:    deps.Sender.NibiruAddr.String(),
		FromErc20: &eth.EIP55Addr{Address: erc20Addr},
	})
	s.Require().NoError(err)

	// Transfer 500 tokens to bob => 500 * 10^18 raw
	deployerAddr := gethcommon.HexToAddress(erc20Resp.EthTxMsg.From)
	_, err = deps.EvmKeeper.CallContract(
		deps.Ctx,
		embeds.SmartContract_TestERC20.ABI,
		deployerAddr,
		&erc20Addr,
		true,
		keeper.Erc20GasLimitExecute,
		"transfer",
		bob.EthAddr,
		bigTokens(500), // 500 in human sense
	)
	s.Require().NoError(err)

	// Now user should have 500 tokens => raw is 500 * 10^18
	evmtest.AssertERC20BalanceEqual(s.T(), deps, erc20Addr, bob.EthAddr, bigTokens(500))

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
	evmtest.AssertERC20BalanceEqual(s.T(), deps, erc20Addr, bob.EthAddr, bigTokens(400))

	// 100 tokens are escrowed
	evmtest.AssertERC20BalanceEqual(s.T(), deps, erc20Addr, evm.EVM_MODULE_ADDRESS, bigTokens(100))

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
	evmtest.AssertERC20BalanceEqual(s.T(), deps, erc20Addr, bob.EthAddr, bigTokens(500))

	// check evm module account's balance, it should have escrowed some tokens
	// unescrow the tokens
	evmtest.AssertERC20BalanceEqual(s.T(), deps, erc20Addr, evm.EVM_MODULE_ADDRESS, bigTokens(0))

	// burns the bank tokens
	evmBal2 := deps.App.BankKeeper.GetBalance(deps.Ctx, evm.EVM_MODULE_ADDRESS[:], bankBal.Denom).Amount.BigInt()
	s.Require().EqualValues(bigTokens(0), evmBal2)

	// user has 500 tokens again => 500 * 1e18
	evmtest.AssertERC20BalanceEqual(s.T(), deps, erc20Addr, bob.EthAddr, bigTokens(500))
}
