package eth_test

import (
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/v2/eth"
)

// mustNewEIP55AddrFromStr is the same as [NewEIP55AddrFromStr], except it panics
// when there's an error.
func mustNewEIP55AddrFromStr(input string) eth.EIP55Addr {
	addr, err := eth.NewEIP55AddrFromStr(input)
	if err != nil {
		panic(err)
	}
	return addr
}

func (s *EIP55AddrSuite) TestEquivalence() {
	expectedGethAddr := gethcommon.HexToAddress("0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed")
	expectedEIP55Addr := mustNewEIP55AddrFromStr("0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed")

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

func (s *EIP55AddrSuite) TestJsonEncoding() {
	for tcIdx, tc := range []struct {
		input        eth.EIP55Addr
		expectedJson json.RawMessage
		wantErr      string
	}{
		{
			input:        mustNewEIP55AddrFromStr("0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed"),
			expectedJson: []byte("\"0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed\""),
		},
		{
			input:        mustNewEIP55AddrFromStr("0xAe967917c465db8578ca9024c205720b1a3651A9"),
			expectedJson: []byte("\"0xAe967917c465db8578ca9024c205720b1a3651A9\""),
		},
		{
			input:        mustNewEIP55AddrFromStr("0x1111111111111111111112222222222223333323"),
			expectedJson: []byte("\"0x1111111111111111111112222222222223333323\""),
		},
	} {
		s.Run(strconv.Itoa(tcIdx), func() {
			jsonBz, err := tc.input.MarshalJSON()
			s.Require().NoError(err)
			s.Require().EqualValues(tc.expectedJson, jsonBz)

			eip55Addr := new(eth.EIP55Addr)
			s.Require().NoError(eip55Addr.UnmarshalJSON(jsonBz))
			s.Require().EqualValues(tc.input, *eip55Addr)
		})
	}
}

func (s *EIP55AddrSuite) TestProtobufEncoding() {
	for tcIdx, tc := range []struct {
		input           eth.EIP55Addr
		expectedProtoBz []byte
		wantErr         string
	}{
		{
			input:           mustNewEIP55AddrFromStr("0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed"),
			expectedProtoBz: []byte{90, 174, 182, 5, 63, 62, 148, 201, 185, 160, 159, 51, 102, 148, 53, 231, 239, 27, 234, 237},
		},
		{
			input:           mustNewEIP55AddrFromStr("0xAe967917c465db8578ca9024c205720b1a3651A9"),
			expectedProtoBz: []byte{174, 150, 121, 23, 196, 101, 219, 133, 120, 202, 144, 36, 194, 5, 114, 11, 26, 54, 81, 169},
		},
		{
			input:           mustNewEIP55AddrFromStr("0x1111111111111111111112222222222223333323"),
			expectedProtoBz: []byte{17, 17, 17, 17, 17, 17, 17, 17, 17, 17, 18, 34, 34, 34, 34, 34, 35, 51, 51, 35},
		},
	} {
		s.Run(strconv.Itoa(tcIdx), func() {
			bz, err := tc.input.Marshal()
			s.Require().NoError(err)
			s.Require().EqualValues(tc.expectedProtoBz, bz)

			eip55Addr := new(eth.EIP55Addr)
			s.Require().NoError(eip55Addr.Unmarshal(bz))
			s.Require().Equal(tc.input.Address, eip55Addr.Address)
		})
	}
}

func (s *EIP55AddrSuite) TestSize() {
	for idx, tc := range []struct {
		input        eth.EIP55Addr
		expectedSize int
		wantErr      string
	}{
		{
			input:        mustNewEIP55AddrFromStr("0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed"),
			expectedSize: 20,
		},
		{
			input:        mustNewEIP55AddrFromStr("0xAe967917c465db8578ca9024c205720b1a3651A9"),
			expectedSize: 20,
		},
		{
			input:        mustNewEIP55AddrFromStr("0x1111111111111111111112222222222223333323"),
			expectedSize: 20,
		},
	} {
		s.Run(strconv.Itoa(idx), func() {
			s.Require().EqualValues(tc.expectedSize, tc.input.Size())
		})
	}
}

// showcases how geth checks for valid hex addresses and treats invalid inputs
func (s *EIP55AddrSuite) TestHexAddress() {
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

func (s *EIP55AddrSuite) TestStringEncoding() {
	addrHex := "0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed"
	addr, err := eth.NewEIP55AddrFromStr(addrHex)
	s.Require().NoError(err)
	s.Require().Equal(addrHex, addr.Address.Hex())

	addrBz, err := addr.Marshal()
	s.Require().NoError(err)
	s.Require().EqualValues(addr.Bytes(), addrBz)

	bz, err := addr.MarshalJSON()
	s.Require().NoError(err)
	s.Require().Equal(fmt.Sprintf(`"%s"`, addrHex), string(bz))

	newAddr := new(eth.EIP55Addr)
	err = newAddr.UnmarshalJSON(bz)
	s.Require().NoError(err)
	s.Require().EqualValues(addrHex, newAddr.Hex())
}
