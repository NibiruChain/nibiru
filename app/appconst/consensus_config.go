package appconst

import (
	tmcfg "github.com/cometbft/cometbft/config"
	"time"
)

// NewDefaultTendermintConfig returns a consensus "Config" (CometBFT) with new
// default values for the "consensus" and "db_backend" fields to be enforced upon
// node initialization. See the "nibiru/cmd/nibid/cmd/InitCmd" function for more
// information.
func NewDefaultTendermintConfig() *tmcfg.Config {
	cfg := tmcfg.DefaultConfig()

	// Overwrite consensus config
	ms := func(n time.Duration) time.Duration {
		return n * time.Millisecond
	}
	cfg.Consensus.TimeoutPropose = ms(3_000)
	cfg.Consensus.TimeoutProposeDelta = ms(500)
	cfg.Consensus.TimeoutPrevote = ms(1_000)
	cfg.Consensus.TimeoutPrevoteDelta = ms(500)
	cfg.Consensus.TimeoutPrecommit = ms(1_000)
	cfg.Consensus.TimeoutPrecommitDelta = ms(500)
	cfg.Consensus.TimeoutCommit = ms(1_000)

	cfg.DBBackend = string(DefaultDBBackend)
	return cfg
}
