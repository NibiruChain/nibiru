package consensus

import (
	"bytes"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/bft/abci/example/kvstore"
	cfg "github.com/NibiruChain/nibiru/v2/bft/config"
	"github.com/NibiruChain/nibiru/v2/bft/libs/log"
	"github.com/NibiruChain/nibiru/v2/bft/types"
	cmttime "github.com/NibiruChain/nibiru/v2/bft/types/time"
)

// newCommitTimeoutTestState builds a one-validator State whose first height
// start time is computed with the given options, and returns the window of
// wall-clock time in which NewState ran.
func newCommitTimeoutTestState(
	t *testing.T, options ...StateOption,
) (cs *State, config *cfg.Config, before, after time.Time) {
	t.Helper()
	config = cfg.ResetTestRoot("consensus_commit_timeout_test")
	state, privVals := randGenesisState(1, false, 10)
	before = cmttime.Now()
	cs = newStateWithConfig(
		config, state, privVals[0], kvstore.NewApplication(), options...,
	)
	after = cmttime.Now()
	return cs, config, before, after
}

// requireStartWithin asserts that StartTime is the NewState time plus wait.
func requireStartWithin(
	t *testing.T, cs *State, before, after time.Time, wait time.Duration,
) {
	t.Helper()
	require.False(t, cs.StartTime.Before(before.Add(wait)),
		"start %v is before %v", cs.StartTime, before.Add(wait))
	require.False(t, cs.StartTime.After(after.Add(wait)),
		"start %v is after %v", cs.StartTime, after.Add(wait))
}

func TestCommitTimeoutNilHookUsesConfig(t *testing.T) {
	cs, config, before, after := newCommitTimeoutTestState(t)
	configured := config.Consensus.TimeoutCommit
	requireStartWithin(t, cs, before, after, configured)

	commitTime := cmttime.Now()
	cs.CommitTime = commitTime
	require.Equal(t, commitTime.Add(configured), cs.nextHeightStartTime(2))
	require.Equal(t, config.Consensus.Commit(commitTime),
		cs.nextHeightStartTime(2))
}

func TestCommitTimeoutHookOverridesStartTime(t *testing.T) {
	const wait = 7 * time.Second
	var heights []int64
	hook := func(height int64) (time.Duration, bool) {
		heights = append(heights, height)
		return wait, true
	}
	cs, _, before, after := newCommitTimeoutTestState(
		t, WithCommitTimeoutFunc(hook),
	)
	// The option applies inside NewState, so the first height uses it.
	requireStartWithin(t, cs, before, after, wait)
	require.Equal(t, []int64{cs.Height}, heights)

	// Later heights count the wait from the commit decision.
	commitTime := cmttime.Now()
	cs.CommitTime = commitTime
	require.Equal(t, commitTime.Add(wait), cs.nextHeightStartTime(5))
	require.Equal(t, int64(5), heights[len(heights)-1])
}

func TestCommitTimeoutHookUnsetUsesConfig(t *testing.T) {
	testCases := []struct {
		name string
		wait time.Duration
		ok   bool
	}{
		{name: "ok=false", wait: 7 * time.Second, ok: false},
		{name: "negative wait", wait: -time.Second, ok: true},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hook := func(int64) (time.Duration, bool) { return tc.wait, tc.ok }
			cs, config, before, after := newCommitTimeoutTestState(
				t, WithCommitTimeoutFunc(hook),
			)
			configured := config.Consensus.TimeoutCommit
			requireStartWithin(t, cs, before, after, configured)

			commitTime := cmttime.Now()
			cs.CommitTime = commitTime
			require.Equal(t, commitTime.Add(configured),
				cs.nextHeightStartTime(2))
		})
	}
}

func TestSetCommitTimeoutFuncRecomputesStartTime(t *testing.T) {
	cs, config, before, _ := newCommitTimeoutTestState(t)
	requireStartWithin(t, cs, before, cmttime.Now(),
		config.Consensus.TimeoutCommit)

	const wait = 9 * time.Second
	setBefore := cmttime.Now()
	err := cs.SetCommitTimeoutFunc(
		func(int64) (time.Duration, bool) { return wait, true },
	)
	require.NoError(t, err)
	requireStartWithin(t, cs, setBefore, cmttime.Now(), wait)

	// A nil fn restores the config value.
	setBefore = cmttime.Now()
	require.NoError(t, cs.SetCommitTimeoutFunc(nil))
	requireStartWithin(t, cs, setBefore, cmttime.Now(),
		config.Consensus.TimeoutCommit)
}

func TestCommitTimeoutWarnsOncePerChange(t *testing.T) {
	cs, config, _, _ := newCommitTimeoutTestState(t)
	configured := config.Consensus.TimeoutCommit

	var buf bytes.Buffer
	cs.SetLogger(log.NewTMLogger(log.NewSyncWriter(&buf)))

	type reply struct {
		wait time.Duration
		ok   bool
	}
	var current reply
	cs.commitTimeoutFn = func(int64) (time.Duration, bool) {
		return current.wait, current.ok
	}
	steps := []reply{
		{0, false},              // unset: no log
		{2 * time.Second, true}, // warn 1
		{2 * time.Second, true}, // same value: no log
		{2 * time.Second, true},
		{3 * time.Second, true}, // warn 2
		{3 * time.Second, true},
		{0, false}, // back to config: one info line
		{0, false},
		{3 * time.Second, true}, // warn 3
		{configured, true},      // equals config: no warning
		{configured, true},
	}
	for i, step := range steps {
		current = step
		cs.commitTimeout(int64(i + 2))
	}

	out := buf.String()
	require.Equal(t, 3,
		strings.Count(out, "timeout_commit is ignored"), out)
	require.Equal(t, 1,
		strings.Count(out, "chain commit wait is unset"), out)
	require.Contains(t, out, "min_block_commit_duration=2s")
	require.Contains(t, out, "min_block_commit_duration=3s")
	require.Contains(t, out, "timeout_commit="+configured.String())
}

// TestCommitTimeoutHookPacesRunningConsensus runs a one-validator chain and
// checks that the next height is scheduled at CommitTime plus the hook value,
// not config.toml timeout_commit.
func TestCommitTimeoutHookPacesRunningConsensus(t *testing.T) {
	const wait = 50 * time.Millisecond
	var (
		mu      sync.Mutex
		heights []int64
	)
	hook := func(height int64) (time.Duration, bool) {
		mu.Lock()
		defer mu.Unlock()
		heights = append(heights, height)
		return wait, true
	}

	config := cfg.ResetTestRoot("consensus_commit_timeout_test")
	// Honor the commit wait even with every precommit in hand.
	config.Consensus.SkipTimeoutCommit = false
	require.NotEqual(t, wait, config.Consensus.TimeoutCommit)
	state, privVals := randGenesisState(1, false, 10)
	cs := newStateWithConfig(
		config, state, privVals[0], kvstore.NewApplication(),
		WithCommitTimeoutFunc(hook),
	)
	t.Cleanup(func() { _ = cs.Stop() })

	newRoundCh := subscribe(cs.eventBus, types.EventQueryNewRound)
	height := cs.Height
	startTestRound(cs, height, 0)
	ensureNewRound(newRoundCh, height, 0)
	ensureNewRound(newRoundCh, height+1, 0)

	rs := cs.GetRoundState()
	require.False(t, rs.CommitTime.IsZero())
	require.Equal(t, wait, rs.StartTime.Sub(rs.CommitTime))

	mu.Lock()
	defer mu.Unlock()
	require.Contains(t, heights, height+1)
}
