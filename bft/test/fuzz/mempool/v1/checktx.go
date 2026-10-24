package v1

import (
	"github.com/NibiruChain/nibiru/v2/bft/abci/example/kvstore"
	"github.com/NibiruChain/nibiru/v2/bft/config"
	"github.com/NibiruChain/nibiru/v2/bft/libs/log"
	mempl "github.com/NibiruChain/nibiru/v2/bft/mempool"
	"github.com/NibiruChain/nibiru/v2/bft/proxy"

	mempoolv1 "github.com/NibiruChain/nibiru/v2/bft/mempool/v1" //nolint:staticcheck // SA1019 Priority mempool deprecated but still supported in this release.
)

var mempool mempl.Mempool

func init() {
	app := kvstore.NewApplication()
	cc := proxy.NewLocalClientCreator(app)
	appConnMem, _ := cc.NewABCIClient()
	err := appConnMem.Start()
	if err != nil {
		panic(err)
	}
	cfg := config.DefaultMempoolConfig()
	cfg.Broadcast = false
	log := log.NewNopLogger()
	mempool = mempoolv1.NewTxMempool(log, cfg, appConnMem, 0)
}

func Fuzz(data []byte) int {

	err := mempool.CheckTx(data, nil, mempl.TxInfo{})
	if err != nil {
		return 0
	}

	return 1
}
