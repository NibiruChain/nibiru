package backend_test

import (
	"fmt"

	"github.com/cometbft/cometbft/proto/tendermint/crypto"

	"github.com/NibiruChain/nibiru/v2/eth/rpc/backend"
)

func (s *BackendSuite) TestGetLogsFromBlockResults() {
	blockWithTx := transferTxBlockNumber.Int64()
	blockResults, err := s.backend.TendermintBlockResultByNumber(&blockWithTx)
	s.Require().NoError(err)
	s.Require().NotNil(blockResults)

	logs, err := backend.GetLogsFromBlockResults(blockResults)
	s.Require().NoError(err)
	s.Require().NotNil(logs)

	// TODO: ON: the structured event eth.evm.v1.EventTxLog is not emitted properly, so the logs are not retrieved
	// Add proper checks after implementing
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
			s.Require().Equal(tc.exp, backend.GetHexProofs(tc.proof))
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
