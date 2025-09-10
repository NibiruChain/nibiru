// Copyright (c) 2023-2024 Nibi, Inc.
package evm_test

import (
	"math/big"
	"strconv"
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bank "github.com/cosmos/cosmos-sdk/x/bank/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
)

type TestSuite struct {
	suite.Suite
}

func TestSuite_RunAll(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

func (s *TestSuite) TestFunToken() {
	for idx, tc := range []struct {
		bankDenom string
		input     string
		wantErr   bool
	}{
		{
			// sad: Invalid bank denom
			bankDenom: "",
			input:     "5aaeb6053f3e94c9b9a09f33669435e7ef1beaed",
			wantErr:   true,
		},
		{
			bankDenom: "unibi",
			input:     "5aaeb6053f3e94c9b9a09f33669435e7ef1beaed",
		},
		{
			bankDenom: "unibi",
			input:     "5AAEB6053F3E94C9B9A09F33669435E7EF1BEAED",
		},
		{
			bankDenom: "unibi",
			input:     "5aaeb6053f3e94c9b9a09f33669435e7ef1beaed",
		},

		{
			bankDenom: "ibc/AAA/BBB",
			input:     "0xE1aA1500b962528cBB42F05bD6d8A6032a85602f",
		},
		{
			bankDenom: "tf/contract-addr/subdenom",
			input:     "0x6B2e60f1030aFa69F584829f1d700b47eE5Fc74a",
		},
	} {
		s.Run(strconv.Itoa(idx), func() {
			eip55Addr, err := eth.NewEIP55AddrFromStr(tc.input)
			s.Require().NoError(err)

			funtoken := evm.FunToken{
				Erc20Addr: eip55Addr,
				BankDenom: tc.bankDenom,
			}
			if tc.wantErr {
				s.Require().Error(funtoken.Validate())
				return
			}

			s.Require().NoError(funtoken.Validate())
		})
	}

	for _, tc := range []struct {
		name string
		A    string
		B    string
	}{
		{
			name: "capital and lowercase match",
			A:    "5aaeb6053f3e94c9b9a09f33669435e7ef1beaed",
			B:    "5AAEB6053F3E94C9B9A09F33669435E7EF1BEAED",
		},
		{
			name: "0x prefix and no prefix match",
			A:    "5aaeb6053f3e94c9b9a09f33669435e7ef1beaed",
			B:    "0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed",
		},
		{
			name: "0x prefix and no prefix match",
			A:    "5aaeb6053f3e94c9b9a09f33669435e7ef1beaed",
			B:    "0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed",
		},
		{
			name: "mixed case compatibility",
			A:    "0x5Bdb32670a05Daa22Cb2E279B80044c37dc85e61",
			B:    "0x5BDB32670A05DAA22CB2E279B80044C37DC85E61",
		},
	} {
		s.Run(tc.name, func() {
			addrA, err := eth.NewEIP55AddrFromStr(tc.A)
			s.Require().NoError(err)

			addrB, err := eth.NewEIP55AddrFromStr(tc.B)
			s.Require().NoError(err)

			funA := evm.FunToken{Erc20Addr: addrA}
			funB := evm.FunToken{Erc20Addr: addrB}

			s.EqualValues(funA.Erc20Addr.Address, funB.Erc20Addr.Address)
		})
	}
}

func (s *TestSuite) TestModuleAddressEVM() {
	addr := evm.EVM_MODULE_ADDRESS
	s.Equal(addr.Hex(), "0x603871c2ddd41c26Ee77495E2E31e6De7f9957e0")

	// Sanity check
	nibiAddr := authtypes.NewModuleAddress(evm.ModuleName)
	evmModuleAddr := gethcommon.BytesToAddress(nibiAddr)
	s.Equal(addr.Hex(), evmModuleAddr.Hex())

	// EVM addr module acc and EVM address should be connected
	// EVM module should have mint perms
	deps := evmtest.NewTestDeps()
	{
		resp, err := deps.EvmKeeper.EthAccount(sdk.WrapSDKContext(deps.Ctx), &evm.QueryEthAccountRequest{
			Address: evmModuleAddr.Hex(),
		})
		s.NoError(err)
		s.Equal(nibiAddr.String(), resp.Bech32Address)
		s.Equal(evmModuleAddr.String(), resp.EthAddress)
	}
}

func (s *TestSuite) TestWeiConversion() {
	{
		unibiAmt := big.NewInt(420)
		s.Equal(
			unibiAmt,
			evm.WeiToNative(evm.NativeToWei(unibiAmt)),
			"native -> wei -> native should be an identity operation",
		)

		weiAmt := evm.NativeToWei(unibiAmt)
		want := "420" + strings.Repeat("0", 12)
		s.Equal(weiAmt.String(), want)
	}

	tenPow12 := new(big.Int).Exp(big.NewInt(10), big.NewInt(12), nil)
	for _, tc := range []struct {
		weiAmtIn  string
		want      *big.Int
		wantError string
	}{
		{
			//	Input  number:  123456789012345678901234567890
			//	Parsed number:  123456789012345678 * 10^12
			weiAmtIn:  "123456789012345678901234567890",
			want:      evm.NativeToWei(big.NewInt(123456789012345678)),
			wantError: "",
		},
		{
			weiAmtIn:  "123456789012345678901234567890",
			want:      evm.NativeToWei(big.NewInt(123456789012345678)),
			wantError: "",
		},
		{
			weiAmtIn:  "0",
			want:      big.NewInt(0),
			wantError: "",
		},
		{
			weiAmtIn:  "1",
			wantError: "cannot transfer less than 1 micronibi.",
		},
		{
			weiAmtIn: new(big.Int).Sub(
				tenPow12, big.NewInt(1),
			).String(),
			wantError: "cannot transfer less than 1 micronibi.",
		},
		{
			weiAmtIn:  "500",
			wantError: "cannot transfer less than 1 micronibi.",
		},
	} {
		weiAmtIn, _ := new(big.Int).SetString(tc.weiAmtIn, 10)
		got, err := evm.ParseWeiAsMultipleOfMicronibi(weiAmtIn)
		if tc.wantError != "" {
			s.Require().ErrorContains(err, tc.wantError)
			return
		}
		s.NoError(err)
		s.Require().Equal(tc.want.String(), got.String())
	}
}

func TestValidateFunTokenBankMetadata(t *testing.T) {
	// makeMD builds minimal bank metadata for tests.
	// Base denom is always the first unit (exp=0). Each exponent in exps adds a
	// display unit using the symbol. The last exponent is what ParseDecimalsFromBank
	// will return.
	makeMD := func(base, name, symbol string, exps ...uint32) bank.Metadata {
		units := []*bank.DenomUnit{{Denom: base, Exponent: 0}}
		for _, e := range exps {
			units = append(units, &bank.DenomUnit{Denom: symbol, Exponent: e})
		}
		display := symbol
		if display == "" {
			// Cosmos requires non-empty Display in Validate(), but the function under
			// test ignores Validate()’s error, so allow empty to exercise that path.
			display = ""
		}
		return bank.Metadata{
			DenomUnits: units,
			Base:       base,
			Display:    display,
			Name:       name,
			Symbol:     symbol,
		}
	}

	cases := []struct {
		name              string
		md                bank.Metadata
		allowZeroDecimals bool
		want              evm.ERC20Metadata
		// wantErr: if empty => expect no error; otherwise ErrorContains(wantErr)
		wantErr string
	}{
		{
			name: "happy: name/symbol set, decimals from last denom unit (8)",
			md:   makeMD("usome", "Some Token", "SOME", 8),
			want: evm.ERC20Metadata{Name: "Some Token", Symbol: "SOME", Decimals: 8},
		},
		{
			name: "happy: last unit wins (6, 9, 18 -> 18)",
			md:   makeMD("ufoo", "Foo", "FOO", 6, 9, 18),
			want: evm.ERC20Metadata{Name: "Foo", Symbol: "FOO", Decimals: 18},
		},
		{
			name:              "happy: zero decimals allowed",
			md:                makeMD("ubar", "Bar", "BAR" /* no extra units */),
			allowZeroDecimals: true,
			want:              evm.ERC20Metadata{Name: "Bar", Symbol: "BAR", Decimals: 0},
		},
		{
			name:    "sad: zero decimals not allowed",
			md:      makeMD("ubaz", "Baz", "BAZ" /* no extra units */),
			wantErr: `decimals = 0`, // matches: got ERC20.decimals = 0, which is consdiered an error unless "allow_zero_decimals" is true
		},
		{
			name:    "sad: empty name",
			md:      makeMD("uqqq", "" /*name*/, "QQQ", 18),
			wantErr: "empty name for ERC20",
		},
		{
			name:    "sad: empty symbol",
			md:      makeMD("urrr", "Rrr", "" /*symbol*/, 18),
			wantErr: "empty symbol for ERC20",
		},
		{
			name: "NOTE: bank.Metadata.Validate() error is ignored by function",
			// Uppercase base denom would fail bank.Metadata.Validate(), but
			// ValidateFunTokenBankMetadata ignores that error. Expect no error.
			md:                makeMD("BadDenomUpper", "Name", "SYM" /* no extra units */),
			allowZeroDecimals: true,
			want:              evm.ERC20Metadata{Name: "Name", Symbol: "SYM", Decimals: 0},
			wantErr:           "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := evm.ValidateFunTokenBankMetadata(tc.md, tc.allowZeroDecimals)

			if tc.wantErr != "" {
				assert.ErrorContains(t, err, tc.wantErr)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
