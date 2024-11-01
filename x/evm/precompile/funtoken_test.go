package precompile_test

import (
	"math/big"
	"testing"

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

	s.T().Log("Create FunToken from coin")
	bankDenom := "unibi"
	s.funtoken = evmtest.CreateFunTokenForBankCoin(&s.deps, bankDenom, &s.Suite)
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
			methodName: string(precompile.FunTokenMethod_BankSend),
			callArgs:   []any{"nonsense", "args here", "to see if", "precompile is", "called"},
			wantError:  "argument count mismatch: got 5 for 3",
		},
		{
			name:       "wrong type for address",
			methodName: string(precompile.FunTokenMethod_BankSend),
			callArgs:   []any{"nonsense", "foo", "bar"},
			wantError:  "abi: cannot use string as type array as argument",
		},
		{
			name:       "wrong type for amount",
			methodName: string(precompile.FunTokenMethod_BankSend),
			callArgs:   []any{gethcommon.HexToAddress("0x7D4B7B8CA7E1a24928Bb96D59249c7a5bd1DfBe6"), "foo", testutil.AccAddress().String()},
			wantError:  "abi: cannot use string as type ptr as argument",
		},
		{
			name:       "wrong type for recipient",
			methodName: string(precompile.FunTokenMethod_BankSend),
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

func (s *FuntokenSuite) TestHappyPath() {
	deps := evmtest.NewTestDeps()

	s.T().Log("Create FunToken mapping and ERC20")
	bankDenom := "unibi"
	funtoken := evmtest.CreateFunTokenForBankCoin(&deps, bankDenom, &s.Suite)

	erc20 := funtoken.Erc20Addr.Address

	s.T().Log("Balances of the ERC20 should start empty")
	evmtest.AssertERC20BalanceEqual(s.T(), deps, erc20, deps.Sender.EthAddr, big.NewInt(0))
	evmtest.AssertERC20BalanceEqual(s.T(), deps, erc20, evm.EVM_MODULE_ADDRESS, big.NewInt(0))

	s.Require().NoError(testapp.FundAccount(
		deps.App.BankKeeper,
		deps.Ctx,
		deps.Sender.NibiruAddr,
		sdk.NewCoins(sdk.NewCoin(s.funtoken.BankDenom, sdk.NewInt(69_420))),
	))

	_, err := deps.EvmKeeper.ConvertCoinToEvm(
		sdk.WrapSDKContext(deps.Ctx),
		&evm.MsgConvertCoinToEvm{
			Sender:   deps.Sender.NibiruAddr.String(),
			BankCoin: sdk.NewCoin(s.funtoken.BankDenom, sdk.NewInt(69_420)),
			ToEthAddr: eth.EIP55Addr{
				Address: deps.Sender.EthAddr,
			},
		},
	)
	s.Require().NoError(err)

	s.T().Log("Mint tokens - Fail from non-owner")
	{
		input, err := embeds.SmartContract_ERC20Minter.ABI.Pack("mint", deps.Sender.EthAddr, big.NewInt(69_420))
		s.NoError(err)

		s.deps.ResetGasMeter()
		_, _, err = deps.EvmKeeper.CallContractWithInput(
			deps.Ctx, deps.Sender.EthAddr, &erc20, true, input, keeper.Erc20GasLimitQuery,
		)
		s.ErrorContains(err, "Ownable: caller is not the owner")
	}

	randomAcc := testutil.AccAddress()

	s.T().Log("Send NIBI (FunToken) using precompile")
	amtToSend := int64(420)
	callArgs := []any{erc20, big.NewInt(amtToSend), randomAcc.String()}
	input, err := embeds.SmartContract_FunToken.ABI.Pack(string(precompile.FunTokenMethod_BankSend), callArgs...)
	s.NoError(err)

	deps.ResetGasMeter()
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
	s.deps.ResetGasMeter()
	s.Require().NotNil(deps.EvmKeeper.Bank.StateDB)

	s.T().Log("Parse the response contract addr and response bytes")
	var sentAmt *big.Int
	err = embeds.SmartContract_FunToken.ABI.UnpackIntoInterface(
		&sentAmt,
		string(precompile.FunTokenMethod_BankSend),
		ethTxResp.Ret,
	)
	s.NoError(err)
	s.Require().Equal("420", sentAmt.String())
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

	s.T().Log("Fund sender's wallet")
	s.Require().NoError(testapp.FundAccount(
		deps.App.BankKeeper,
		deps.Ctx,
		deps.Sender.NibiruAddr,
		sdk.NewCoins(sdk.NewCoin(s.funtoken.BankDenom, sdk.NewInt(1000))),
	))

	s.T().Log("Fund contract with erc20 coins")
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

	s.deps.ResetGasMeter()

	s.T().Log("Happy: callBankSend with default gas")
	_, err = deps.EvmKeeper.CallContract(
		deps.Ctx,
		embeds.SmartContract_TestFunTokenPrecompileLocalGas.ABI,
		deps.Sender.EthAddr,
		&contractAddr,
		true,
		precompile.FunTokenGasLimitBankSend,
		"callBankSend",
		big.NewInt(1),
		randomAcc.String(),
	)
	s.Require().NoError(err)

	s.deps.ResetGasMeter()

	s.T().Log("Happy: callBankSend with local gas - sufficient gas amount")
	_, err = deps.EvmKeeper.CallContract(
		deps.Ctx,
		embeds.SmartContract_TestFunTokenPrecompileLocalGas.ABI,
		deps.Sender.EthAddr,
		&contractAddr,
		true,
		precompile.FunTokenGasLimitBankSend, // gasLimit for the entire call
		"callBankSendLocalGas",
		big.NewInt(1),      // erc20 amount
		randomAcc.String(), // to
		big.NewInt(int64(precompile.FunTokenGasLimitBankSend)), // customGas
	)
	s.Require().NoError(err)

	s.deps.ResetGasMeter()

	s.T().Log("Sad: callBankSend with local gas - insufficient gas amount")
	_, err = deps.EvmKeeper.CallContract(
		deps.Ctx,
		embeds.SmartContract_TestFunTokenPrecompileLocalGas.ABI,
		deps.Sender.EthAddr,
		&contractAddr,
		true,
		precompile.FunTokenGasLimitBankSend, // gasLimit for the entire call
		"callBankSendLocalGas",
		big.NewInt(1),      // erc20 amount
		randomAcc.String(), // to
		big.NewInt(50_000), // customGas - too small
	)
	s.Require().ErrorContains(err, "execution reverted")
}
