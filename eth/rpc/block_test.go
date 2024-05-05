package rpc

import (
	"fmt"
	"math/big"
	"testing"

	grpctypes "github.com/cosmos/cosmos-sdk/types/grpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/metadata"
)

type BlockSuite struct {
	suite.Suite
}

func TestBlockSuite(t *testing.T) {
	suite.Run(t, new(BlockSuite))
}

func (s *BlockSuite) TestNewBlockNumber() {
	bigInt := big.NewInt(1)
	bn := NewBlockNumber(bigInt)
	bnInt64 := bn.Int64()
	bnTmHeight := bn.TmHeight()
	s.EqualValues(bnInt64, *bnTmHeight)
	s.EqualValues(bigInt.Int64(), bnInt64)
}

func (s *BlockSuite) TestNewContextWithHeight() {
	// Test with zero height
	ctxZero := NewContextWithHeight(0)
	_, ok := metadata.FromOutgoingContext(ctxZero)
	s.False(ok, "No metadata should be present for height 0")

	// Test with non-zero height
	height := int64(10)
	ctxTen := NewContextWithHeight(height)
	md, ok := metadata.FromOutgoingContext(ctxTen)
	s.True(ok, "Metadata should be present for non-zero height")
	s.NotEmpty(md, "Metadata should not be empty")

	heightStr, ok := md[grpctypes.GRPCBlockHeightHeader]
	s.True(ok, grpctypes.GRPCBlockHeightHeader, " metadata should be present")
	s.Require().Len(heightStr, 1,
		fmt.Sprintf("There should be exactly one %s value", grpctypes.GRPCBlockHeightHeader))
	s.Equal(fmt.Sprintf("%d", height), heightStr[0],
		"The height value in metadata should match the provided height")
}

func (s *BlockSuite) TestUnmarshalBlockNumberOrHash() {
	bnh := new(BlockNumberOrHash)

	testCases := []struct {
		msg      string
		input    []byte
		malleate func()
		expPass  bool
	}{
		{
			msg:   "JSON input with block hash",
			input: []byte("{\"blockHash\": \"0x579917054e325746fda5c3ee431d73d26255bc4e10b51163862368629ae19739\"}"),
			malleate: func() {
				s.Equal(*bnh.BlockHash, common.HexToHash("0x579917054e325746fda5c3ee431d73d26255bc4e10b51163862368629ae19739"))
				s.Nil(bnh.BlockNumber)
			},
			expPass: true,
		},
		{
			"JSON input with block number",
			[]byte("{\"blockNumber\": \"0x35\"}"),
			func() {
				s.Equal(*bnh.BlockNumber, BlockNumber(0x35))
				s.Nil(bnh.BlockHash)
			},
			true,
		},
		{
			"JSON input with block number latest",
			[]byte("{\"blockNumber\": \"latest\"}"),
			func() {
				s.Equal(*bnh.BlockNumber, EthLatestBlockNumber)
				s.Nil(bnh.BlockHash)
			},
			true,
		},
		{
			"JSON input with both block hash and block number",
			[]byte("{\"blockHash\": \"0x579917054e325746fda5c3ee431d73d26255bc4e10b51163862368629ae19739\", \"blockNumber\": \"0x35\"}"),
			func() {
			},
			false,
		},
		{
			"String input with block hash",
			[]byte("\"0x579917054e325746fda5c3ee431d73d26255bc4e10b51163862368629ae19739\""),
			func() {
				s.Equal(*bnh.BlockHash, common.HexToHash("0x579917054e325746fda5c3ee431d73d26255bc4e10b51163862368629ae19739"))
				s.Nil(bnh.BlockNumber)
			},
			true,
		},
		{
			"String input with block number",
			[]byte("\"0x35\""),
			func() {
				s.Equal(*bnh.BlockNumber, BlockNumber(0x35))
				s.Nil(bnh.BlockHash)
			},
			true,
		},
		{
			"String input with block number latest",
			[]byte("\"latest\""),
			func() {
				s.Equal(*bnh.BlockNumber, EthLatestBlockNumber)
				s.Nil(bnh.BlockHash)
			},
			true,
		},
		{
			"String input with block number overflow",
			[]byte("\"0xffffffffffffffffffffffffffffffffffffff\""),
			func() {
			},
			false,
		},
	}

	for _, tc := range testCases {
		fmt.Printf("Case %s", tc.msg)
		// reset input
		bnh = new(BlockNumberOrHash)
		err := bnh.UnmarshalJSON(tc.input)
		tc.malleate()
		if tc.expPass {
			s.NoError(err)
		} else {
			s.Error(err)
		}
	}
}
