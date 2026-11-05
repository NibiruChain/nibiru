package store

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/bft/internal/test"
	"github.com/NibiruChain/nibiru/v2/bft/libs/log"
	"github.com/NibiruChain/nibiru/v2/bft/types"
	cmttime "github.com/NibiruChain/nibiru/v2/bft/types/time"
)

// TestLoadBlockExtendedCommit tests loading the extended commit for a previously
// saved block. The load method should return nil when only a commit was saved and
// return the extended commit otherwise.
func BenchmarkRepeatedLoadSeenCommitSameBlock(b *testing.B) {
	state, bs, cleanup := makeStateAndBlockStore(log.NewTMLogger(new(bytes.Buffer)))
	defer cleanup()
	h := bs.Height() + 1
	block, err := state.MakeBlock(h, test.MakeNTxs(h, 10), new(types.Commit), nil, state.Validators.GetProposer().Address)
	require.NoError(b, err)
	seenCommit := makeTestCommit(block.Header.Height, cmttime.Now())
	ps, err := block.MakePartSet(types.BlockPartSizeBytes)
	require.NoError(b, err)
	bs.SaveBlock(block, ps, seenCommit)

	// sanity check
	res := bs.LoadSeenCommit(block.Height)
	require.Equal(b, seenCommit, res)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res := bs.LoadSeenCommit(block.Height)
		require.NotNil(b, res)
	}
}
