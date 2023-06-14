package cmd

import (
	"time"

	db "github.com/cometbft/cometbft-db"
	tmcfg "github.com/cometbft/cometbft/config"
)

const (
	// FlagOverwrite defines a flag to overwrite an existing genesis JSON file.
	FlagOverwrite = "overwrite"

	// FlagSeed defines a flag to initialize the private validator key from a specific seed.
	FlagRecover = "recover"
)

func customTendermintConfig() *tmcfg.Config {
	cfg := tmcfg.DefaultConfig()

	ms := func(n time.Duration) time.Duration {
		return n * time.Millisecond
	}
	consensus := cfg.Consensus
	consensus.TimeoutPropose = ms(3_000)
	consensus.TimeoutProposeDelta = ms(500)
	consensus.TimeoutPrevote = ms(1_000)
	consensus.TimeoutPrevoteDelta = ms(500)
	consensus.TimeoutPrecommit = ms(1_000)
	consensus.TimeoutPrecommitDelta = ms(500)
	consensus.TimeoutCommit = ms(1_000)

	cfg.DBBackend = string(db.RocksDBBackend)

	return cfg
}
