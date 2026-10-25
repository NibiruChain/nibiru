package mock_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	tmproto "github.com/NibiruChain/nibiru/v2/bft/proto/tendermint/types"
	cmttypes "github.com/NibiruChain/nibiru/v2/bft/types"

	"github.com/NibiruChain/nibiru/v2/lib/ibc-go/testing/mock"
)

const chainID = "testChain"

func TestGetPubKey(t *testing.T) {
	pv := mock.NewPV()
	pk, err := pv.GetPubKey()
	require.NoError(t, err)
	require.Equal(t, "ed25519", pk.Type())
}

func TestSignVote(t *testing.T) {
	pv := mock.NewPV()
	pk, _ := pv.GetPubKey()

	vote := &tmproto.Vote{Height: 2}
	err := pv.SignVote(chainID, vote)
	require.NoError(t, err)

	msg := cmttypes.VoteSignBytes(chainID, vote)
	ok := pk.VerifySignature(msg, vote.Signature)
	require.True(t, ok)
}

func TestSignProposal(t *testing.T) {
	pv := mock.NewPV()
	pk, _ := pv.GetPubKey()

	proposal := &tmproto.Proposal{Round: 2}
	err := pv.SignProposal(chainID, proposal)
	require.NoError(t, err)

	msg := cmttypes.ProposalSignBytes(chainID, proposal)
	ok := pk.VerifySignature(msg, proposal.Signature)
	require.True(t, ok)
}
