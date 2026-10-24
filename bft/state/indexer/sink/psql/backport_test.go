package psql

import (
	"github.com/NibiruChain/nibiru/v2/bft/state/indexer"
	"github.com/NibiruChain/nibiru/v2/bft/state/txindex"
)

var (
	_ indexer.BlockIndexer = BackportBlockIndexer{}
	_ txindex.TxIndexer    = BackportTxIndexer{}
)
