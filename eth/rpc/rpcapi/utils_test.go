package rpcapi_test

import (
	"fmt"

	"github.com/cometbft/cometbft/proto/tendermint/crypto"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"

	gethcrypto "github.com/ethereum/go-ethereum/crypto"

	"github.com/NibiruChain/nibiru/v2/eth/rpc/rpcapi"
)

func (s *BackendSuite) TestGetLogsFromBlockResults() {
	blockWithTx := s.SuccessfulTxDeployContract().BlockNumberRpc.Int64()
	blockResults, err := s.backend.TendermintBlockResultByNumber(&blockWithTx)
	s.Require().NoError(err)
	s.Require().NotNil(blockResults)

	logs, err := rpcapi.GetLogsFromBlockResults(blockResults)
	s.Require().NoError(err)
	s.Require().NotNil(logs)

	s.assertTxLogsMatch([]*gethcore.Log{
		{
			Address: testContractAddress,
			Topics: []gethcommon.Hash{
				gethcrypto.Keccak256Hash([]byte("Transfer(address,address,uint256)")),
				gethcommon.Address{}.Hash(),
				s.fundedAccEthAddr.Hash(),
			},
		},
	}, logs[0], "deploy contract tx")
}

func (s *BackendSuite) TestGetHexProofs() {
	defaultRes := []string{""}
	testCases := []struct {
		name  string
		proof *crypto.ProofOps
		exp   []string
	}{
		{
			"no proof provided",
			mockProofs(0, false),
			defaultRes,
		},
		{
			"no proof data provided",
			mockProofs(1, false),
			defaultRes,
		},
		{
			"valid proof provided",
			mockProofs(1, true),
			[]string{"0x0a190a034b4559120556414c55451a0b0801180120012a03000202"},
		},
	}
	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.name), func() {
			s.Require().Equal(tc.exp, rpcapi.GetHexProofs(tc.proof))
		})
	}
}

func mockProofs(num int, withData bool) *crypto.ProofOps {
	var proofOps *crypto.ProofOps
	if num > 0 {
		proofOps = new(crypto.ProofOps)
		for i := 0; i < num; i++ {
			proof := crypto.ProofOp{}
			if withData {
				proof.Data = []byte("\n\031\n\003KEY\022\005VALUE\032\013\010\001\030\001 \001*\003\000\002\002")
			}
			proofOps.Ops = append(proofOps.Ops, proof)
		}
	}
	return proofOps
}
