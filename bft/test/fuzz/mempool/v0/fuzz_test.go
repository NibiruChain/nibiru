package v0

import (
	"testing"

	"github.com/NibiruChain/nibiru/v2/bft/abci/example/kvstore"
	"github.com/NibiruChain/nibiru/v2/bft/config"
	"github.com/NibiruChain/nibiru/v2/bft/libs/log"
	mempl "github.com/NibiruChain/nibiru/v2/bft/mempool"
	mempoolv0 "github.com/NibiruChain/nibiru/v2/bft/mempool/v0"
	"github.com/NibiruChain/nibiru/v2/bft/proxy"
)

// FuzzMempoolV0 feeds arbitrary transactions to CheckTx on a mempool backed
// by the in-process kvstore app. Seeds live in testdata/fuzz/FuzzMempoolV0.
func FuzzMempoolV0(f *testing.F) {
	app := kvstore.NewApplication()
	cc := proxy.NewLocalClientCreator(app)
	appConn, err := cc.NewABCIClient()
	if err != nil {
		f.Fatal(err)
	}
	appConn.SetLogger(log.NewNopLogger())
	if err := appConn.Start(); err != nil {
		f.Fatal(err)
	}

	cfg := config.DefaultMempoolConfig()
	cfg.Broadcast = false
	mp := mempoolv0.NewCListMempool(cfg, appConn, 0)

	f.Fuzz(func(t *testing.T, data []byte) {
		// CheckTx errors (full mempool, cached or oversized tx) are expected.
		// Only a panic fails the test.
		_ = mp.CheckTx(data, nil, mempl.TxInfo{})

		// Keep the mempool below its size limit so later inputs still reach
		// the app instead of returning ErrMempoolIsFull.
		if mp.Size() >= cfg.Size {
			mp.Flush()
		}
	})
}
