package v0

import (
	"github.com/NibiruChain/nibiru/v2/bft/abci/example/kvstore"
	"github.com/NibiruChain/nibiru/v2/bft/config"
	mempl "github.com/NibiruChain/nibiru/v2/bft/mempool"
	mempoolv0 "github.com/NibiruChain/nibiru/v2/bft/mempool/v0"
	"github.com/NibiruChain/nibiru/v2/bft/proxy"
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
	mempool = mempoolv0.NewCListMempool(cfg, appConnMem, 0)
}

func Fuzz(data []byte) int {
	err := mempool.CheckTx(data, nil, mempl.TxInfo{})
	if err != nil {
		return 0
	}

	return 1
}
