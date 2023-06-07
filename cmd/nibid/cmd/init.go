package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	db "github.com/cometbft/cometbft-db"
	tmcfg "github.com/cometbft/cometbft/config"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// FlagOverwrite defines a flag to overwrite an existing genesis JSON file.
	FlagOverwrite = "overwrite"

	// FlagSeed defines a flag to initialize the private validator key from a specific seed.
	FlagRecover = "recover"
)

type printInfo struct {
	Moniker    string          `json:"moniker" yaml:"moniker"`
	ChainID    string          `json:"chain_id" yaml:"chain_id"`
	NodeID     string          `json:"node_id" yaml:"node_id"`
	GenTxsDir  string          `json:"gentxs_dir" yaml:"gentxs_dir"`
	AppMessage json.RawMessage `json:"app_message" yaml:"app_message"`
}

func newPrintInfo(moniker, chainID, nodeID, genTxsDir string, appMessage json.RawMessage) printInfo {
	return printInfo{
		Moniker:    moniker,
		ChainID:    chainID,
		NodeID:     nodeID,
		GenTxsDir:  genTxsDir,
		AppMessage: appMessage,
	}
}

func displayInfo(info printInfo) error {
	out, err := json.MarshalIndent(info, "", " ")
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(os.Stderr, "%s\n", string(sdk.MustSortJSON(out)))

	return err
}

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
