package jsonrpc_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/app/appconst"
	abci "github.com/NibiruChain/nibiru/v2/bft/abci/types"
	cmtbytes "github.com/NibiruChain/nibiru/v2/bft/libs/bytes"
	cmtlog "github.com/NibiruChain/nibiru/v2/bft/libs/log"
	cmtrpcclient "github.com/NibiruChain/nibiru/v2/bft/rpc/client"
	tmrpctypes "github.com/NibiruChain/nibiru/v2/bft/rpc/core/types"
	cmttypes "github.com/NibiruChain/nibiru/v2/bft/types"
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
	cmtrpcclient.MempoolClient
	txs         cmttypes.Txs
	err         error
	paramsErr   error
	paramsCalls *atomic.Int32
}

// UnconfirmedTxs returns the controlled snapshot or the injected transport error.
func (c *pendingSnapshotClient) UnconfirmedTxs(context.Context, *int) (*tmrpctypes.ResultUnconfirmedTxs, error) {
	if c.err != nil {
		return nil, c.err
	}
	return &tmrpctypes.ResultUnconfirmedTxs{Txs: c.txs, Count: len(c.txs), Total: len(c.txs)}, nil
}

// ABCIQueryWithOptions records unnecessary params queries and can make that
// unrelated upstream service fail. Pending transaction formatting needs only the
// chain ID initialized by NewBackend, not a successful Params query.
func (c *pendingSnapshotClient) ABCIQueryWithOptions(_ context.Context, path string, _ cmtbytes.HexBytes, _ cmtrpcclient.ABCIQueryOptions) (*tmrpctypes.ResultABCIQuery, error) {
	if c.paramsCalls != nil {
		c.paramsCalls.Add(1)
	}
	if c.paramsErr != nil {
		return nil, c.paramsErr
	}
	if path != "/eth.evm.v1.Query/Params" {
		return nil, fmt.Errorf("unexpected query: %s", path)
	}
	value, err := (&evm.QueryParamsResponse{Params: evm.DefaultParams()}).Marshal()
	if err != nil {
		return nil, err
	}
	return &tmrpctypes.ResultABCIQuery{Response: abci.ResponseQuery{Value: value, Height: 1}}, nil
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
// Neither normal operation nor a Params-service outage should cause a redundant
// query or change the response: the backend already knows its chain identity.
func TestPendingTransactionsRPCSnapshot(t *testing.T) {
	enc := app.MakeEncodingConfig()
	sender := evmtest.NewEthPrivAcc()
	recipient := gethcommon.HexToAddress("0x1234")
	chainID := big.NewInt(appconst.ETH_CHAIN_ID_DEFAULT)
	encode := func(msg sdk.Msg) cmttypes.Tx {
		builder := enc.TxConfig.NewTxBuilder()
		require.NoError(t, builder.SetMsgs(msg))
		bz, err := enc.TxConfig.TxEncoder()(builder.GetTx())
		require.NoError(t, err)
		return bz
	}

	bankTx := encode(banktypes.NewMsgSend(sender.NibiruAddr, sender.NibiruAddr, sdk.NewCoins(sdk.NewInt64Coin("unibi", 1))))
	wantHashes := make([]gethcommon.Hash, 0, 2)
	txs := cmttypes.Txs{bankTx}
	for nonce := uint64(0); nonce < 2; nonce++ {
		msg := evm.NewTx(&evm.EvmTxArgs{ChainID: chainID, Nonce: nonce, To: &recipient, GasLimit: 21_000, GasPrice: big.NewInt(1), Amount: big.NewInt(123)})
		msg.From = sender.EthAddr.Hex()
		require.NoError(t, msg.Sign(gethcore.LatestSignerForChainID(chainID), sender.KeyringSigner))
		txs = append(txs, encode(msg))
		wantHashes = append(wantHashes, msg.AsTransaction().Hash())
	}

	for _, tc := range []struct {
		name      string
		paramsErr error
	}{
		{name: "params available"},
		{name: "params unavailable", paramsErr: errors.New("params service unavailable")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			paramsCalls := new(atomic.Int32)
			snapshot := &pendingSnapshotClient{txs: txs, paramsErr: tc.paramsErr, paramsCalls: paramsCalls}
			rpcClient := pendingRPCFixture(t, snapshot, enc)
			var pending []map[string]json.RawMessage
			require.NoError(t, rpcClient.Call(&pending, "eth_getPendingTransactions"))
			require.Len(t, pending, 2)
			gotHashes := make([]gethcommon.Hash, 0, len(pending))
			for _, tx := range pending {
				var hash gethcommon.Hash
				var from gethcommon.Address
				var gotChainID hexutil.Big
				require.NoError(t, json.Unmarshal(tx["hash"], &hash))
				require.NoError(t, json.Unmarshal(tx["from"], &from))
				require.NoError(t, json.Unmarshal(tx["chainId"], &gotChainID))
				require.Equal(t, chainID, gotChainID.ToInt())
				gotHashes = append(gotHashes, hash)
				require.Equal(t, sender.EthAddr, from, "unmanaged sender must remain visible")
				for _, field := range []string{"blockHash", "blockNumber", "transactionIndex"} {
					require.JSONEq(t, "null", string(tx[field]), field)
				}
			}
			require.ElementsMatch(t, wantHashes, gotHashes, "pending RPC must preserve transaction identity without promising an order")
			require.Zero(t, paramsCalls.Load(), "formatting pending messages must not query chain params")
		})
	}
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
		{name: "malformed Cosmos transaction", snapshot: pendingSnapshotClient{txs: cmttypes.Txs{{0xff}}}, wantError: true},
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
