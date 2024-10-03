package eth_test

import (
	"strconv"
	"testing"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/v2/eth"
)

// MustNewEIP55AddrFromStr is the same as [NewEIP55AddrFromStr], except it panics
// when there's an error.
func MustNewEIP55AddrFromStr(input string) eth.EIP55Addr {
	addr, err := eth.NewEIP55AddrFromStr(input)
	if err != nil {
		panic(err)
	}
	return addr
}

var threeValidAddrs []eth.EIP55Addr = []eth.EIP55Addr{
	MustNewEIP55AddrFromStr("0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed"),
	MustNewEIP55AddrFromStr("0xAe967917c465db8578ca9024c205720b1a3651A9"),
	MustNewEIP55AddrFromStr("0x1111111111111111111112222222222223333323"),
}

func (s *EIP55AddrSuite) TestEquivalence() {
	expectedGethAddr := gethcommon.HexToAddress("0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed")
	expectedEIP55Addr := MustNewEIP55AddrFromStr("0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed")

	equivalentAddrs := []string{
		"0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed",
		"0x5AAEB6053F3E94C9B9A09F33669435E7EF1BEAED",
		"5aaeb6053f3e94c9b9a09f33669435e7ef1beaed",
		"0X5AAEB6053F3E94C9B9A09F33669435E7EF1BEAED",
	}

	for _, addr := range equivalentAddrs {
		eip55Addr, err := eth.NewEIP55AddrFromStr(addr)
		s.Require().NoError(err)

		s.Equal(expectedEIP55Addr, eip55Addr)
		s.Equal(expectedGethAddr, eip55Addr.Address)
	}
}

// TestEIP55Addr_NewEIP55Addr: Test to showcase the flexibility of inputs that can be
// passed to `eth.NewEIP55AddrFromStr` and result in a "valid" `EIP55Addr` that preserves
// bijectivity with `gethcommon.Address` and has a canonical string
// representation.
//
// We only want to store valid `EIP55Addr` strings in state. Hex addresses that
// include or remove the prefix, or change the letters to and from lower and
// upper case will all produce the same `EIP55Addr` when passed to
// `eth.NewEIP55AddrFromStr`.
func (s *EIP55AddrSuite) TestNewEIP55Addr() {
	// TestCase: An instance of a "EIP55Addr" that derives to the
	// expected Ethereum address and results in the same string representation.
	type TestCase struct {
		input   string
		name    string
		wantErr bool
	}

	want := "0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed"

	for _, tc := range []TestCase{
		{
			input: "0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed",
			name:  "happy: no-op (sanity check to show constructor doesn't break a valid input)",
		},
		{
			input: "0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed",
			name:  "happy: lower case is valid",
		},
		{
			input: "0x5AAEB6053F3E94C9B9A09F33669435E7EF1BEAED",
			name:  "happy: upper case is valid",
		},
		{
			input: "5aaeb6053f3e94c9b9a09f33669435e7ef1beaed",
			name:  "happy: 0x prefix: missing",
		},
		{
			input: "0X5aaeb6053f3e94c9b9a09f33669435e7ef1beaed",
			name:  "happy: 0X prefix: typo",
		},
		{
			input:   "nibi1zaa12312312aacbcbeabea123",
			name:    "sad: bech32 is not hex",
			wantErr: true,
		},
	} {
		tc := tc
		s.Run(tc.name, func() {
			got, err := eth.NewEIP55AddrFromStr(tc.input)
			if tc.wantErr {
				s.Require().Error(err)
				return
			}

			// string input should give the canonical EIP55Addr
			s.Equal(want, got.String())
			s.Equal(gethcommon.HexToAddress(tc.input), got.Address)
		})
	}
}

func (s *EIP55AddrSuite) TestProtobufEncoding() {
	for tcIdx, tc := range []struct {
		input        eth.EIP55Addr
		expectedJson string
		wantErr      string
	}{
		{
			input:        threeValidAddrs[0],
			expectedJson: "\"0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed\"",
		},
		{
			input:        threeValidAddrs[1],
			expectedJson: "\"0xAe967917c465db8578ca9024c205720b1a3651A9\"",
		},
		{
			input:        threeValidAddrs[2],
			expectedJson: "\"0x1111111111111111111112222222222223333323\"",
		},
	} {
		s.Run(strconv.Itoa(tcIdx), func() {
			givenMut := tc.input
			jsonBz, err := givenMut.MarshalJSON()
			s.NoError(err)
			s.Equal(tc.expectedJson, string(jsonBz))

			eip55Addr := new(eth.EIP55Addr)
			s.NoError(eip55Addr.UnmarshalJSON(jsonBz))
			s.Equal(givenMut, tc.input,
				"Given -> MarshalJSON -> UnmarshalJSON returns a different value than the given when it should be an identity operation (no-op). test case #%d", tcIdx)

			bz, err := tc.input.Marshal()
			s.NoError(err)
			s.Equal(tc.input.Bytes(), bz,
				"Marshaling to bytes gives different value than the test case specifies. test case #%d", tcIdx)

			err = eip55Addr.Unmarshal(bz)
			s.NoError(err)
			s.Equal(tc.input.Address, eip55Addr.Address,
				"Given -> Marshal -> Unmarshal returns a different value than the given when it should be an identity operation (no-op). test case #%d", tcIdx)

			s.Equal(len(tc.input.Bytes()), tc.input.Size())
		})
	}
}

// showcases how geth checks for valid hex addresses and treats invalid inputs
func (s *EIP55AddrSuite) TestIsEIP55Address() {
	s.True(gethcommon.IsHexAddress("0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed"))
	s.True(gethcommon.IsHexAddress("0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAED"))
	s.False(gethcommon.IsHexAddress("0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed1234"))
	s.False(gethcommon.IsHexAddress("0x5aAeb6053F3E94C9b9A09f33669435E7Ef1B"))

	s.Equal("0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed", gethcommon.HexToAddress("0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed").Hex())
	s.Equal("0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed", gethcommon.HexToAddress("0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAED").Hex())
	s.Equal("0xb6053f3e94c9B9a09f33669435e7eF1BEAEd1234", gethcommon.HexToAddress("0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed1234").Hex())
	s.Equal("0x00005AaEb6053f3e94c9b9A09f33669435e7Ef1b", gethcommon.HexToAddress("0x5aAeb6053F3E94C9b9A09f33669435E7Ef1B").Hex())
}

type EIP55AddrSuite struct {
	suite.Suite
}

func TestEIP55AddrSuite(t *testing.T) {
	suite.Run(t, new(EIP55AddrSuite))
}
