// Copyright (c) 2023-2024 Nibi, Inc.
package evm_test

import (
	"math/big"
	"strconv"
	"strings"
	"testing"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/eth"
	"github.com/NibiruChain/nibiru/x/evm"
	"github.com/NibiruChain/nibiru/x/evm/evmtest"
)

type TestSuite struct {
	suite.Suite
}

func TestSuite_RunAll(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

func (s *TestSuite) TestFunToken() {
	for testIdx, tc := range []struct {
		bankDenom string
		erc20Addr eth.HexAddr
		wantErr   string
	}{
		{
			// sad: Invalid bank denom
			bankDenom: "",
			erc20Addr: eth.HexAddr(""),
			wantErr:   "FunTokenError",
		},
		{
			bankDenom: "unibi",
			erc20Addr: eth.MustNewHexAddrFromStr("5aaeb6053f3e94c9b9a09f33669435e7ef1beaed"),
			wantErr:   "",
		},
		{
			bankDenom: "unibi",
			erc20Addr: eth.MustNewHexAddrFromStr("5AAEB6053F3E94C9B9A09F33669435E7EF1BEAED"),
			wantErr:   "",
		},
		{
			// NOTE: notice how this one errors using the same happy path
			// input as above because an unsafe constructor was used.
			// Naked type overrides should not be used with eth.HexAddr.
			// Always use NewHexAddr, NewHexAddrFromStr, or MustNewHexAddr...
			bankDenom: "unibi",
			erc20Addr: eth.HexAddr("5aaeb6053f3e94c9b9a09f33669435e7ef1beaed"),
			wantErr:   "not encoded as expected",
		},

		{
			bankDenom: "ibc/AAA/BBB",
			erc20Addr: eth.MustNewHexAddrFromStr("0xE1aA1500b962528cBB42F05bD6d8A6032a85602f"),
			wantErr:   "",
		},
		{
			bankDenom: "tf/contract-addr/subdenom",
			erc20Addr: eth.MustNewHexAddrFromStr("0x6B2e60f1030aFa69F584829f1d700b47eE5Fc74a"),
			wantErr:   "",
		},
	} {
		s.Run(strconv.Itoa(testIdx), func() {
			funtoken := evm.FunToken{
				Erc20Addr: tc.erc20Addr,
				BankDenom: tc.bankDenom,
			}
			err := funtoken.Validate()
			if tc.wantErr != "" {
				s.Require().Error(err, "funtoken %s", funtoken)
				return
			}
			s.Require().NoError(err)
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
			funA := evm.FunToken{Erc20Addr: eth.HexAddr(tc.A)}
			funB := evm.FunToken{Erc20Addr: eth.HexAddr(tc.B)}

			s.EqualValues(funA.Erc20Addr.ToAddr(), funB.Erc20Addr.ToAddr())
		})
	}
}

func (s *TestSuite) TestModuleAddressEVM() {
	addr := evm.ModuleAddressEVM()
	s.Equal(addr.Hex(), "0x603871c2ddd41c26Ee77495E2E31e6De7f9957e0")

	// Sanity check
	nibiAddr := authtypes.NewModuleAddress(evm.ModuleName)
	evmModuleAddr := gethcommon.BytesToAddress(nibiAddr)
	s.Equal(addr.Hex(), evmModuleAddr.Hex())

	// EVM addr module acc and EVM address should be connected
	// EVM module should have mint perms
	deps := evmtest.NewTestDeps()
	{
		resp, err := deps.EvmKeeper.EthAccount(deps.GoCtx(), &evm.QueryEthAccountRequest{
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
