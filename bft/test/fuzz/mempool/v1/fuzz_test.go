package v1

import (
	"testing"

	"github.com/NibiruChain/nibiru/v2/bft/abci/example/kvstore"
	"github.com/NibiruChain/nibiru/v2/bft/config"
	"github.com/NibiruChain/nibiru/v2/bft/libs/log"
	mempl "github.com/NibiruChain/nibiru/v2/bft/mempool"
	// The priority mempool is deprecated but still supported in v0.37.
	mempoolv1 "github.com/NibiruChain/nibiru/v2/bft/mempool/v1" //nolint:staticcheck
	"github.com/NibiruChain/nibiru/v2/bft/proxy"
)

// FuzzMempoolV1 feeds arbitrary transactions to CheckTx on a mempool backed
// by the in-process kvstore app. Seeds live in testdata/fuzz/FuzzMempoolV1.
func FuzzMempoolV1(f *testing.F) {
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
	mp := mempoolv1.NewTxMempool(log.NewNopLogger(), cfg, appConn, 0)

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
