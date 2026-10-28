package consensus

import (
	"errors"
	"time"

	cmttime "github.com/NibiruChain/nibiru/v2/bft/types/time"
)

// CommitTimeoutFunc lets the application set the commit wait that Nibiru BFT
// would otherwise read from config.toml `consensus.timeout_commit`.
//
// The commit wait is the time between this node's commit decision (the
// moment it holds precommits for one block from more than two-thirds of
// voting power, recorded as CommitTime) and the scheduled start of the next
// height. Block execution happens inside that interval. Letting the chain
// own the value means every validator paces blocks the same way, instead of
// each operator choosing a local setting.
//
// The engine calls the function with the height it is about to start. It
// returns the wait and ok=true when the chain has a value. It returns
// ok=false when the chain has no value, and the engine then uses
// `timeout_commit` from config.toml. A negative duration is treated like
// ok=false.
//
// The engine calls the function while holding the consensus state mutex, at
// least once per height (at startup it can run twice for the same height), so
// it must be idempotent. It must be cheap and non-blocking: read a cached
// value, never query application state, take application locks, or do I/O. A slow
// function delays consensus for every peer that waits on this node.
type CommitTimeoutFunc func(height int64) (wait time.Duration, ok bool)

// errCommitTimeoutWhileRunning is returned by SetCommitTimeoutFunc when the
// consensus state has already started.
var errCommitTimeoutWhileRunning = errors.New(
	"cannot set the commit timeout func after consensus has started",
)

// WithCommitTimeoutFunc returns a StateOption that installs fn as the source
// of the commit wait. NewState applies options before it computes the start
// time of the first height, so the value takes effect from that height. A nil
// fn keeps the config.toml behavior.
func WithCommitTimeoutFunc(fn CommitTimeoutFunc) StateOption {
	return func(cs *State) {
		cs.commitTimeoutFn = fn
	}
}

// SetCommitTimeoutFunc installs fn on a State that was already constructed
// but not started, and recomputes the start time of the current height with
// it. It exists for callers that receive the State from a constructor they do
// not control, such as node.NewNode, where StateOptions cannot be passed.
// Round 0 of the first height is scheduled only in OnStart, so calling this
// before Start has the same effect as passing WithCommitTimeoutFunc to
// NewState. It returns an error once the State is running, because changing
// the hook under a live height would race with the scheduled timeout.
func (cs *State) SetCommitTimeoutFunc(fn CommitTimeoutFunc) error {
	cs.mtx.Lock()
	defer cs.mtx.Unlock()
	if cs.IsRunning() {
		return errCommitTimeoutWhileRunning
	}
	cs.commitTimeoutFn = fn
	cs.StartTime = cs.nextHeightStartTime(cs.Height)
	return nil
}

// nextHeightStartTime returns when round 0 of height should start: the
// commit decision time of the previous height plus the commit wait. On a
// fresh start there is no commit decision yet (CommitTime is zero), so the
// wait counts from now, which gives the first block time to gather
// transactions. Callers must hold cs.mtx or be inside NewState.
func (cs *State) nextHeightStartTime(height int64) time.Time {
	from := cs.CommitTime
	if from.IsZero() {
		from = cmttime.Now()
	}
	return from.Add(cs.commitTimeout(height))
}

// commitTimeout returns the commit wait for height: the hook's value when it
// reports one, otherwise config.toml `timeout_commit`. It also logs when the
// hook's value starts to differ from config, so operators know their local
// setting is ignored. Callers must hold cs.mtx or be inside NewState.
func (cs *State) commitTimeout(height int64) time.Duration {
	configured := cs.config.TimeoutCommit
	if cs.commitTimeoutFn == nil {
		return configured
	}
	wait, ok := cs.commitTimeoutFn(height)
	if ok && wait < 0 {
		ok = false
	}
	cs.logCommitTimeoutChange(height, wait, ok)
	if !ok {
		return configured
	}
	return wait
}

// logCommitTimeoutChange logs once per change of the hook's value, not once
// per block. A value that differs from config.toml gets a warning naming both
// values. A value equal to config.toml is recorded without a log line. The
// hook going back to unset gets an info line. During NewState the logger is
// not set yet, so nothing is recorded and the first height after SetLogger
// logs instead.
func (cs *State) logCommitTimeoutChange(
	height int64, wait time.Duration, ok bool,
) {
	if cs.Logger == nil {
		return
	}
	last := cs.lastCommitTimeoutOverride
	switch {
	case !ok && last.set:
		cs.Logger.Info(
			"chain commit wait is unset; using config.toml timeout_commit",
			"height", height,
			"timeout_commit", cs.config.TimeoutCommit,
		)
	case ok && (!last.set || last.wait != wait) &&
		wait != cs.config.TimeoutCommit:
		// The Logger interface has no warn level. Info is visible at the
		// default log level, and the message says it is a warning.
		cs.Logger.Info(
			"warning: config.toml timeout_commit is ignored; the chain value "+
				"min_block_commit_duration_ms sets the commit wait",
			"height", height,
			"min_block_commit_duration", wait,
			"timeout_commit", cs.config.TimeoutCommit,
		)
	}
	cs.lastCommitTimeoutOverride = commitTimeoutOverride{wait: wait, set: ok}
}

// commitTimeoutOverride remembers the last value the hook reported, so the
// override is logged once per change instead of once per block.
type commitTimeoutOverride struct {
	wait time.Duration
	set  bool
}
