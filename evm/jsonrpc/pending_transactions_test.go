package jsonrpc_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http/httptest"
	"testing"

	gethcommon "github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/app/appconst"
	abci "github.com/NibiruChain/nibiru/v2/bft/abci/types"
	cmtbytes "github.com/NibiruChain/nibiru/v2/bft/libs/bytes"
	cmtlog "github.com/NibiruChain/nibiru/v2/bft/libs/log"
	cmtclient "github.com/NibiruChain/nibiru/v2/bft/rpc/client"
	cmttypes "github.com/NibiruChain/nibiru/v2/bft/rpc/core/types"
	cmttx "github.com/NibiruChain/nibiru/v2/bft/types"
	"github.com/NibiruChain/nibiru/v2/evm"
	"github.com/NibiruChain/nibiru/v2/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/evm/jsonrpc"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/client"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/server"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	banktypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/bank/types"
)

// pendingSnapshotClient supplies a fixed CometBFT mempool view to the real RPC
// backend. It does not run consensus, so the transactions cannot become mined
// between a broadcast and an assertion. Unused client methods intentionally have
// no implementation; an unexpected dependency fails the test instead of hiding it.
type pendingSnapshotClient struct {
	client.TendermintRPC
	cmtclient.MempoolClient
	txs cmttx.Txs
	err error
}

// UnconfirmedTxs returns the controlled snapshot or the injected transport error.
func (c *pendingSnapshotClient) UnconfirmedTxs(context.Context, *int) (*cmttypes.ResultUnconfirmedTxs, error) {
	if c.err != nil {
		return nil, c.err
	}
	return &cmttypes.ResultUnconfirmedTxs{Txs: c.txs, Count: len(c.txs), Total: len(c.txs)}, nil
}

// ABCIQueryWithOptions supplies chain params used when formatting EVM messages.
// Encoding, transaction decoding, EVM filtering and JSON output remain production
// code, rather than reimplementations of eth_getPendingTransactions in the test.
func (c *pendingSnapshotClient) ABCIQueryWithOptions(_ context.Context, path string, _ cmtbytes.HexBytes, _ cmtclient.ABCIQueryOptions) (*cmttypes.ResultABCIQuery, error) {
	if path != "/eth.evm.v1.Query/Params" {
		return nil, fmt.Errorf("unexpected query: %s", path)
	}
	value, err := (&evm.QueryParamsResponse{Params: evm.DefaultParams()}).Marshal()
	if err != nil {
		return nil, err
	}
	return &cmttypes.ResultABCIQuery{Response: abci.ResponseQuery{Value: value, Height: 1}}, nil
}

// pendingRPCFixture runs the registered EthAPI over HTTP with a real SDK codec
// and a controlled upstream mempool. No node keyring or managed accounts are set.
func pendingRPCFixture(t *testing.T, snapshot *pendingSnapshotClient, enc app.EncodingConfig) *gethrpc.Client {
	t.Helper()
	ctx := client.Context{}.WithChainID("nibiru-localnet-0").WithClient(snapshot).
		WithTxConfig(enc.TxConfig).WithInterfaceRegistry(enc.InterfaceRegistry).WithCodec(enc.Codec)
	backend := jsonrpc.NewBackend(server.NewDefaultContext(), cmtlog.NewNopLogger(), ctx, false, nil)
	api := jsonrpc.NewImplEthAPI(cmtlog.NewNopLogger(), backend)
	rpcServer := gethrpc.NewServer()
	require.NoError(t, rpcServer.RegisterName("eth", api))
	t.Cleanup(rpcServer.Stop)
	httpServer := httptest.NewServer(rpcServer)
	t.Cleanup(httpServer.Close)
	rpcClient, err := gethrpc.DialHTTP(httpServer.URL)
	require.NoError(t, err)
	t.Cleanup(rpcClient.Close)
	return rpcClient
}

// TestPendingTransactionsRPCSnapshot proves genuinely unconfirmed transactions
// are represented correctly without racing live block production. A Cosmos-only
// transaction shares the mempool with signed EVM transactions, but is omitted
// from Ethereum RPC. Pending block metadata must be JSON null, not zero values.
func TestPendingTransactionsRPCSnapshot(t *testing.T) {
	enc := app.MakeEncodingConfig()
	sender := evmtest.NewEthPrivAcc()
	recipient := gethcommon.HexToAddress("0x1234")
	chainID := big.NewInt(appconst.ETH_CHAIN_ID_DEFAULT)
	encode := func(msg sdk.Msg) cmttx.Tx {
		builder := enc.TxConfig.NewTxBuilder()
		require.NoError(t, builder.SetMsgs(msg))
		bz, err := enc.TxConfig.TxEncoder()(builder.GetTx())
		require.NoError(t, err)
		return bz
	}

	bankTx := encode(banktypes.NewMsgSend(sender.NibiruAddr, sender.NibiruAddr, sdk.NewCoins(sdk.NewInt64Coin("unibi", 1))))
	wantHashes := make([]gethcommon.Hash, 0, 2)
	txs := cmttx.Txs{bankTx}
	for nonce := uint64(0); nonce < 2; nonce++ {
		msg := evm.NewTx(&evm.EvmTxArgs{ChainID: chainID, Nonce: nonce, To: &recipient, GasLimit: 21_000, GasPrice: big.NewInt(1), Amount: big.NewInt(123)})
		msg.From = sender.EthAddr.Hex()
		require.NoError(t, msg.Sign(gethcore.LatestSignerForChainID(chainID), sender.KeyringSigner))
		txs = append(txs, encode(msg))
		wantHashes = append(wantHashes, msg.AsTransaction().Hash())
	}

	rpcClient := pendingRPCFixture(t, &pendingSnapshotClient{txs: txs}, enc)
	var pending []map[string]json.RawMessage
	require.NoError(t, rpcClient.Call(&pending, "eth_getPendingTransactions"))
	require.Len(t, pending, 2)
	gotHashes := make([]gethcommon.Hash, 0, len(pending))
	for _, tx := range pending {
		var hash gethcommon.Hash
		var from gethcommon.Address
		require.NoError(t, json.Unmarshal(tx["hash"], &hash))
		require.NoError(t, json.Unmarshal(tx["from"], &from))
		gotHashes = append(gotHashes, hash)
		require.Equal(t, sender.EthAddr, from, "unmanaged sender must remain visible")
		for _, field := range []string{"blockHash", "blockNumber", "transactionIndex"} {
			require.JSONEq(t, "null", string(tx[field]), field)
		}
	}
	require.ElementsMatch(t, wantHashes, gotHashes, "pending RPC must preserve transaction identity without promising an order")
}

// TestPendingTransactionsRPCEmptyAndErrors distinguishes an empty mempool from a
// failed query or malformed Cosmos transaction. Returning a successful empty or
// partial list on either error would hide upstream failures from RPC clients.
func TestPendingTransactionsRPCEmptyAndErrors(t *testing.T) {
	enc := app.MakeEncodingConfig()
	for _, tc := range []struct {
		name      string
		snapshot  pendingSnapshotClient
		wantError bool
	}{
		{name: "empty"},
		{name: "upstream error", snapshot: pendingSnapshotClient{err: errors.New("mempool unavailable")}, wantError: true},
		{name: "malformed Cosmos transaction", snapshot: pendingSnapshotClient{txs: cmttx.Txs{{0xff}}}, wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rpcClient := pendingRPCFixture(t, &tc.snapshot, enc)
			var pending json.RawMessage
			err := rpcClient.Call(&pending, "eth_getPendingTransactions")
			if tc.wantError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.JSONEq(t, "[]", string(pending), "empty mempool must serialize as an empty array")
		})
	}
}
