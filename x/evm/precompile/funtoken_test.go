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
	"github.com/NibiruChain/nibiru/v2/x/evm/precompile"
)

// TestSuite: Runs all the tests in the suite.
func TestSuite(t *testing.T) {
	suite.Run(t, new(FuntokenSuite))
	suite.Run(t, new(WasmSuite))
}

type FuntokenSuite struct {
	suite.Suite
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
		sdk.NewCoins(sdk.NewCoin(bankDenom, sdk.NewInt(69_420))),
	))

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

	s.T().Log("Mint tokens - Fail from non-owner")
	{
		input, err := embeds.SmartContract_ERC20Minter.ABI.Pack("mint", deps.Sender.EthAddr, big.NewInt(69_420))
		s.NoError(err)
		_, err = deps.EvmKeeper.CallContractWithInput(
			deps.Ctx, deps.Sender.EthAddr, &erc20, true, input,
		)
		s.ErrorContains(err, "Ownable: caller is not the owner")
	}

	randomAcc := testutil.AccAddress()

	s.T().Log("Send using precompile")
	amtToSend := int64(420)
	callArgs := []any{erc20, big.NewInt(amtToSend), randomAcc.String()}
	input, err := embeds.SmartContract_FunToken.ABI.Pack(string(precompile.FunTokenMethod_BankSend), callArgs...)
	s.NoError(err)

	_, resp, err := evmtest.CallContractTx(
		&deps,
		precompile.PrecompileAddr_FunToken,
		input,
		deps.Sender,
	)
	s.Require().NoError(err)
	s.Require().Empty(resp.VmError)

	evmtest.AssertERC20BalanceEqual(s.T(), deps, erc20, deps.Sender.EthAddr, big.NewInt(69_000))
	evmtest.AssertERC20BalanceEqual(s.T(), deps, erc20, evm.EVM_MODULE_ADDRESS, big.NewInt(0))
	s.Equal(sdk.NewInt(420).String(),
		deps.App.BankKeeper.GetBalance(deps.Ctx, randomAcc, funtoken.BankDenom).Amount.String(),
	)
}
