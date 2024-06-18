package gosdk_test

import (
	"encoding/json"

	"github.com/MakeNowJust/heredoc/v2"
	cmtcoretypes "github.com/cometbft/cometbft/rpc/core/types"

	"github.com/NibiruChain/nibiru/gosdk"
)

// DoTestSequenceExpectations validates the behavior of account sequence numbers
// and transaction finalization in a blockchain network. It ensures that sequence
// numbers increment correctly with each transaction and that transactions can be
// queried successfully after the blocks are completed.
func (s *TestSuite) DoTestSequenceExpectations() {
	t := s.T()
	t.Log("Get sequence and block")
	// Go to next block
	_, err := s.network.WaitForNextBlockVerbose()
	s.Require().NoError(err)

	accAddr := s.val.Address
	getLatestAccNums := func() gosdk.AccountNumbers {
		accNums, err := s.nibiruSdk.GetAccountNumbers(accAddr.String())
		s.NoError(err)
		return accNums
	}
	seq := getLatestAccNums().Sequence

	t.Logf("starting sequence %v should not change from waiting a block", seq)
	s.NoError(s.network.WaitForNextBlock())
	newSeq := getLatestAccNums().Sequence
	s.EqualValues(seq, newSeq)

	t.Log("broadcast msg n times, expect sequence += n")
	numTxs := uint64(5)
	seqs := []uint64{}
	txResults := make(map[string]*cmtcoretypes.ResultTx)
	for broadcastCount := uint64(0); broadcastCount < numTxs; broadcastCount++ {
		s.NoError(s.network.WaitForNextBlock()) // Ensure block increment

		from, _, _, msgSend := s.msgSendVars()
		txResp, err := s.nibiruSdk.BroadcastMsgsGrpcWithSeq(
			from,
			seq+broadcastCount,
			msgSend,
		)
		s.NoError(err)
		txHashHex := s.AssertTxResponseSuccess(txResp)

		s.T().Log(heredoc.Docf(
			`Query for tx %v should fail b/c it's the same block and finalization 
			cannot have possibly occurred yet.`, broadcastCount))
		txResult, err := s.nibiruSdk.TxByHash(txHashHex)
		jsonBz, _ := json.MarshalIndent(txResp, "", "  ")
		s.Assert().Errorf(err, "txResp: %s", jsonBz)

		txResults[txHashHex] = txResult
		seqs = append(seqs, getLatestAccNums().Sequence)
	}

	s.T().Log("expect sequence += n")
	newNewSeq := getLatestAccNums().Sequence
	txResultsJson, _ := json.MarshalIndent(txResults, "", "  ")
	s.EqualValuesf(int(seq+numTxs-1), int(newNewSeq), "seqs: %v\ntxResults: %s", seqs, txResultsJson)

	s.T().Log("After the blocks are completed, tx queries by hash should work.")
	for times := 0; times < 2; times++ {
		s.NoError(s.network.WaitForNextBlock())
	}

	s.T().Log("Query each tx by hash (successfully)")
	for txHashHex := range txResults {
		_, err := s.nibiruSdk.TxByHash(txHashHex)
		s.Require().NoError(err)
	}
}
