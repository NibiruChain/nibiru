package eth_test

import (
	"fmt"
	"strconv"
	"strings"

	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/NibiruChain/nibiru/eth"
	"github.com/NibiruChain/nibiru/x/common/set"
)

var threeValidAddrs []eth.HexAddr = []eth.HexAddr{
	eth.MustNewHexAddrFromStr("0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed"),
	eth.MustNewHexAddrFromStr("0xAe967917c465db8578ca9024c205720b1a3651A9"),
	eth.MustNewHexAddrFromStr("0x1111111111111111111112222222222223333323"),
}

func (s *Suite) TestHexAddr_UniqueMapping() {
	type CorrectAnswer struct {
		gethAddrOut gethcommon.Address
		hexAddrOut  eth.HexAddr
	}

	for tcIdx, tc := range []struct {
		equivSet set.Set[string]
	}{
		{
			equivSet: set.New(
				"0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed",
				"0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed",
				"0x5AAEB6053F3E94C9B9A09F33669435E7EF1BEAED",
				"5aaeb6053f3e94c9b9a09f33669435e7ef1beaed",
				"0X5AAEB6053F3E94C9B9A09F33669435E7EF1BEAED",
			),
		},
	} {
		s.Run(strconv.Itoa(tcIdx), func() {
			s.T().Log("Show that each member of the set is equivalent")
			var answer CorrectAnswer
			for idx, equivHexAddrString := range tc.equivSet.ToSlice() {
				gethAddrOut := gethcommon.HexToAddress(equivHexAddrString)
				hexAddrOut, err := eth.NewHexAddrFromStr(equivHexAddrString)
				s.NoError(err)
				if idx == 0 {
					answer = CorrectAnswer{
						gethAddrOut: gethAddrOut,
						hexAddrOut:  hexAddrOut,
					}
					continue
				}

				s.Equal(answer.gethAddrOut, gethAddrOut)
				s.Equal(answer.gethAddrOut, hexAddrOut.ToAddr())
				s.Equal(answer.hexAddrOut, hexAddrOut)
			}
		})
	}
}

// TestHexAddr_NewHexAddr: Test to showcase the flexibility of inputs that can be
// passed to `eth.NewHexAddrFromStr` and result in a "valid" `HexAddr` that preserves
// bijectivity with `gethcommon.Address` and has a canonical string
// representation.
//
// We only want to store valid `HexAddr` strings in state. Hex addresses that
// include or remove the prefix, or change the letters to and from lower and
// upper case will all produce the same `HexAddr` when passed to
// `eth.NewHexAddrFromStr`.
func (s *Suite) TestHexAddr_NewHexAddr() {
	// InputAddrVariation: An instance of a "hexAddr" that derives to the
	// expected Ethereum address and results in the same string representation.
	type InputAddrVariation struct {
		hexAddr      string
		testCaseName string
		wantNotEqual bool
	}

	for _, tcGroup := range []struct {
		want     eth.HexAddr
		hexAddrs []InputAddrVariation
	}{
		{
			want: threeValidAddrs[0],
			hexAddrs: []InputAddrVariation{
				{
					hexAddr:      "0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed",
					testCaseName: "happy: no-op (sanity check to show constructor doesn't break a valid input)",
				},
				{
					hexAddr:      "0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed",
					testCaseName: "happy: lower case is valid",
				},
				{
					hexAddr:      "0x5AAEB6053F3E94C9B9A09F33669435E7EF1BEAED",
					testCaseName: "happy: upper case is valid",
				},
				{
					hexAddr:      "5aaeb6053f3e94c9b9a09f33669435e7ef1beaed",
					testCaseName: "happy: 0x prefix: missing",
				},
				{
					hexAddr:      "nibi1zaa12312312aacbcbeabea123",
					testCaseName: "sad: bech32 is not hex",
					wantNotEqual: true,
				},
			},
		},
		{
			want: threeValidAddrs[1],
			hexAddrs: []InputAddrVariation{
				{
					hexAddr:      "0XAe967917c465db8578ca9024c205720b1a3651A9",
					testCaseName: "happy: 0x prefix: typo",
				},
				{
					hexAddr:      "0xae967917c465db8578ca9024c205720b1a3651A9",
					testCaseName: "happy: mixed case checksum not valid according to ERC55",
				},
				{
					hexAddr:      "not-a-hex-addr",
					testCaseName: "sad: sanity check: clearly not a hex addr",
					wantNotEqual: true,
				},
			},
		},
		{
			want: threeValidAddrs[2],
			hexAddrs: []InputAddrVariation{
				{
					hexAddr:      "0x1111111111111111111112222222222223333323",
					testCaseName: "happy",
				},
			},
		},
	} {
		want := tcGroup.want
		for _, tc := range tcGroup.hexAddrs {
			tcName := fmt.Sprintf("want %s, %s", want, tc.testCaseName)
			s.Run(tcName, func() {
				got, err := eth.NewHexAddrFromStr(tc.hexAddr)

				// gethcommon.Address input should give the same thing
				got2 := eth.NewHexAddr(gethcommon.HexToAddress(tc.hexAddr))
				if tc.wantNotEqual {
					s.NotEqual(want, got)
					s.NotEqual(want.ToAddr(), got.ToAddr())
					s.NotEqual(want, got2)
					s.NotEqual(want.ToAddr(), got2.ToAddr())
					s.Require().Error(err)
					return
				} else {
					// string input should give the canonical HexAddr
					s.Equal(want, got)
					s.Equal(want.ToAddr(), got.ToAddr())

					// gethcommon.Address input should give the same thing
					got2 := eth.NewHexAddr(gethcommon.HexToAddress(tc.hexAddr))
					s.Equal(want, got2)
					s.Equal(want.ToAddr(), got2.ToAddr())
				}

				s.Require().NoError(err)
			})
		}
	}
}

// TestHexAddr_Valid: Test that demonstrates
func (s *Suite) TestHexAddr_Valid() {
	for _, tc := range []struct {
		name    string
		hexAddr string
		wantErr string
	}{
		{
			name:    "happy 0",
			hexAddr: threeValidAddrs[0].String(),
		},
		{
			name:    "happy 1",
			hexAddr: threeValidAddrs[1].String(),
		},
		{
			name:    "happy 2",
			hexAddr: threeValidAddrs[2].String(),
		},
		{
			name:    "0x prefix: missing",
			hexAddr: "5aaeb6053f3e94c9b9a09f33669435e7ef1beaed",
			wantErr: eth.HexAddrError,
		},
		{
			name:    "0x prefix: typo",
			hexAddr: "0XAe967917c465db8578ca9024c205720b1a3651A9",
			wantErr: eth.HexAddrError,
		},
		{
			name:    "mixed case checksum not valid according to ERC55",
			hexAddr: "0xae967917c465db8578ca9024c205720b1a3651A9",
			wantErr: eth.HexAddrError,
		},
		{
			name:    "sad 1",
			hexAddr: "0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed",
			wantErr: eth.HexAddrError,
		},
	} {
		s.Run(tc.name, func() {
			err := eth.HexAddr(tc.hexAddr).Valid()
			if tc.wantErr != "" {
				s.Require().ErrorContains(err, tc.wantErr)
				return
			}
			s.Require().NoError(err)
		})
	}
}

func withQuotes(s string) string { return fmt.Sprintf("\"%s\"", s) }

func withoutQuotes(s string) string {
	return strings.TrimPrefix(strings.TrimSuffix(s, "\""), "\"")
}

func (s *Suite) TestProtobufEncoding() {
	for tcIdx, tc := range []struct {
		given   eth.HexAddr
		json    string
		wantErr string
	}{
		{
			given: threeValidAddrs[0],
			json:  withQuotes(threeValidAddrs[0].String()),
		},
		{
			given: threeValidAddrs[1],
			json:  withQuotes(threeValidAddrs[1].String()),
		},
		{
			given: threeValidAddrs[2],
			json:  withQuotes(threeValidAddrs[2].String()),
		},
	} {
		s.Run(strconv.Itoa(tcIdx), func() {
			givenMut := tc.given
			jsonBz, err := givenMut.MarshalJSON()
			s.NoError(err)
			s.Equal(tc.json, string(jsonBz))

			err = (&givenMut).UnmarshalJSON(jsonBz)
			s.NoError(err)
			s.Equal(givenMut, tc.given,
				"Given -> MarshalJSON -> UnmarshalJSON returns a different value than the given when it should be an identity operation (no-op). test case #%d", tcIdx)

			bz, err := tc.given.Marshal()
			s.NoError(err)
			jsonBzWithoutQuotes := withoutQuotes(tc.json)
			s.Equal(jsonBzWithoutQuotes, string(bz),
				"Marshaling to bytes gives different value than the test case specifies. test case #%d", tcIdx)

			err = (&givenMut).Unmarshal(bz)
			s.NoError(err)
			s.Equal(tc.given, givenMut,
				"Given -> Marshal -> Unmarshal returns a different value than the given when it should be an identity operation (no-op). test case #%d", tcIdx)

			s.Equal(len(tc.given), tc.given.Size())
			s.Equal(len(tc.json), tc.given.Size()+2)
		})
	}
}

func (s *Suite) TestHexAddrToString() {
	hexAddr := eth.HexAddr("0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed")
	s.Equal("0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed", hexAddr.String())
	s.Equal("0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed", string(hexAddr))

	ethAddr := gethcommon.HexToAddress("0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed")
	s.Equal("0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed", ethAddr.String())
}
