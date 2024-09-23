// Copyright (c) 2023-2024 Nibi, Inc.
package server

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/NibiruChain/nibiru/v2/eth/indexer"

	tmnode "github.com/cometbft/cometbft/node"
	sm "github.com/cometbft/cometbft/state"
	tmstore "github.com/cometbft/cometbft/store"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/server"
)

func NewEVMTxIndexCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "evm-tx-index [minBlockNumber|last-indexed] [maxBlockNumber|latest]",
		Short: "Index historical evm blocks and transactions",
		Long: `Command is useful for catching up if the node experienced a period
with EVMTxIndexer turned off or was stopped without proper closing/flushing EVMIndexerDB.
Processes blocks from minBlockNumber to maxBlockNumber, indexes evm txs.

- minBlockNumber: min block to start indexing. Supply "last-indexed" to start with the latest block available in EVMIndexerDB.
- maxBlockNumber: max block, could be a number or "latest".

Default run before the full node/archive node start should be:

nibid evm-tx-index last-indexed latest
		`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			serverCtx := server.GetServerContextFromCmd(cmd)
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			cfg := serverCtx.Config
			logger := serverCtx.Logger
			evmIndexerDB, err := OpenIndexerDB(cfg.RootDir, server.GetAppDBBackend(serverCtx.Viper))
			if err != nil {
				logger.Error("failed to open evm indexer DB", "error", err.Error())
				return err
			}

			evmTxIndexer := indexer.NewEVMTxIndexer(evmIndexerDB, logger.With("module", "evmindex"), clientCtx)

			tmdb, err := tmnode.DefaultDBProvider(&tmnode.DBContext{ID: "blockstore", Config: cfg})
			if err != nil {
				return err
			}
			blockStore := tmstore.NewBlockStore(tmdb)
			minAvailableHeight := blockStore.Base()
			maxAvailableHeight := blockStore.Height()
			fmt.Printf("Block range available on the node: %d - %d\n", minAvailableHeight, maxAvailableHeight)

			var fromBlock int64
			var toBlock int64

			// FROM block could be one of two:
			// - int64 number - replaced with minAvailableHeight if too low
			// - last-indexed - latest available block in EVMIndexerDB, 0 if nothing is indexed
			if args[0] == "last-indexed" {
				fromBlock, err = evmTxIndexer.LastIndexedBlock()
				if err != nil || fromBlock < 0 {
					fromBlock = 0
				}
			} else {
				fromBlock, err = strconv.ParseInt(args[1], 10, 64)
				if err != nil {
					return fmt.Errorf("cannot parse min block number: %s", args[1])
				}
				if fromBlock > maxAvailableHeight {
					return fmt.Errorf("maximum available block is: %d", maxAvailableHeight)
				}
			}
			if fromBlock < minAvailableHeight {
				fromBlock = minAvailableHeight
			}

			// TO block could be one of two:
			// - int64 number - replaced with maxAvailableHeight if too high
			// - latest - latest available block in the node
			if args[1] == "latest" {
				toBlock = maxAvailableHeight
			} else {
				toBlock, err = strconv.ParseInt(args[1], 10, 64)
				if err != nil {
					return fmt.Errorf("cannot parse max block number: %s", args[1])
				}
				if toBlock > maxAvailableHeight {
					toBlock = maxAvailableHeight
				}
			}
			if fromBlock > toBlock {
				return fmt.Errorf("minBlockNumber must be less or equal to maxBlockNumber")
			}
			stateDB, err := tmnode.DefaultDBProvider(&tmnode.DBContext{ID: "state", Config: cfg})
			if err != nil {
				return err
			}
			stateStore := sm.NewStore(stateDB, sm.StoreOptions{
				DiscardABCIResponses: cfg.Storage.DiscardABCIResponses,
			})

			fmt.Printf("Indexing blocks from %d to %d\n", fromBlock, toBlock)
			for height := fromBlock; height <= toBlock; height++ {
				block := blockStore.LoadBlock(height)
				if block == nil {
					return fmt.Errorf("block not found %d", height)
				}
				blockResults, err := stateStore.LoadABCIResponses(height)
				if err != nil {
					return err
				}
				if err := evmTxIndexer.IndexBlock(block, blockResults.DeliverTxs); err != nil {
					return err
				}
				fmt.Println(height)
			}
			err = evmTxIndexer.CloseDBAndExit()
			if err != nil {
				return err
			}
			fmt.Println("Indexing complete")
			return nil
		},
	}
	return cmd
}
