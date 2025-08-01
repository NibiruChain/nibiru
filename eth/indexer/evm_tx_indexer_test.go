package indexer_test

import (
	"fmt"
	"math/big"
	"testing"

	dbm "github.com/cometbft/cometbft-db"
	abci "github.com/cometbft/cometbft/abci/types"
	tmlog "github.com/cometbft/cometbft/libs/log"
	cmttypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/eth/crypto/ethsecp256k1"
	"github.com/NibiruChain/nibiru/v2/eth/indexer"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	evmtest "github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
)

func TestEVMTxIndexer(t *testing.T) {
	priv, err := ethsecp256k1.GenerateKey()
	require.NoError(t, err)
	from := common.BytesToAddress(priv.PubKey().Address().Bytes())
	signer := evmtest.NewSigner(priv)
	ethSigner := gethcore.LatestSignerForChainID(nil)

	to := common.BigToAddress(big.NewInt(1))
	ethTxParams := evm.EvmTxArgs{
		Nonce:    0,
		To:       &to,
		Amount:   big.NewInt(1000),
		GasLimit: 21000,
	}
	tx := evm.NewTx(&ethTxParams)
	tx.From = from.Hex()
	require.NoError(t, tx.Sign(ethSigner, signer))
	txHash := tx.AsTransaction().Hash()

	encCfg := app.MakeEncodingConfig()
	eth.RegisterInterfaces(encCfg.InterfaceRegistry)
	evm.RegisterInterfaces(encCfg.InterfaceRegistry)
	clientCtx := client.Context{}.
		WithTxConfig(encCfg.TxConfig).
		WithCodec(encCfg.Codec)

	// build cosmos-sdk wrapper tx
	validEVMTx, err := tx.BuildTx(clientCtx.TxConfig.NewTxBuilder(), eth.EthBaseDenom)
	require.NoError(t, err)
	validEVMTxBz, err := clientCtx.TxConfig.TxEncoder()(validEVMTx)
	require.NoError(t, err)

	// build an invalid wrapper tx
	builder := clientCtx.TxConfig.NewTxBuilder()
	require.NoError(t, builder.SetMsgs(tx))
	invalidTx := builder.GetTx()
	invalidTxBz, err := clientCtx.TxConfig.TxEncoder()(invalidTx)
	require.NoError(t, err)

	testCases := []struct {
		name        string
		block       *cmttypes.Block
		blockResult []*abci.ResponseDeliverTx
		expSuccess  bool
	}{
		{
			"happy, only pending_ethereum_tx presents",
			&cmttypes.Block{
				Header: cmttypes.Header{Height: 1},
				Data:   cmttypes.Data{Txs: []cmttypes.Tx{validEVMTxBz}},
			},
			[]*abci.ResponseDeliverTx{
				{
					Code: 0,
					Events: []abci.Event{
						{
							Type: evm.PendingEthereumTxEvent,
							Attributes: []abci.EventAttribute{
								{Key: evm.PendingEthereumTxEventAttrEthHash, Value: txHash.Hex()},
								{Key: evm.PendingEthereumTxEventAttrIndex, Value: "0"},
							},
						},
					},
				},
			},
			true,
		},
		{
			"happy: code 0, pending_ethereum_tx and typed EventEthereumTx present",
			&cmttypes.Block{Header: cmttypes.Header{Height: 1}, Data: cmttypes.Data{Txs: []cmttypes.Tx{validEVMTxBz}}},
			[]*abci.ResponseDeliverTx{
				{
					Code: 0,
					Events: []abci.Event{
						{
							Type: evm.PendingEthereumTxEvent,
							Attributes: []abci.EventAttribute{
								{Key: evm.PendingEthereumTxEventAttrEthHash, Value: txHash.Hex()},
								{Key: evm.PendingEthereumTxEventAttrIndex, Value: "0"},
							},
						},
						{
							Type: evm.TypeUrlEventEthereumTx,
							Attributes: []abci.EventAttribute{
								{Key: "amount", Value: `"1000"`},
								{Key: "gas_used", Value: `"21000"`},
								{Key: "index", Value: `"0"`},
								{Key: "hash", Value: `"14A84ED06282645EFBF080E0B7ED80D8D8D6A36337668A12B5F229F81CDD3F57"`},
								{Key: "eth_hash", Value: fmt.Sprintf(`"%s"`, txHash.Hex())},
							},
						},
					},
				},
			},
			true,
		},
		{
			"happy: code 11, exceed block gas limit",
			&cmttypes.Block{Header: cmttypes.Header{Height: 1}, Data: cmttypes.Data{Txs: []cmttypes.Tx{validEVMTxBz}}},
			[]*abci.ResponseDeliverTx{
				{
					Code:   11,
					Log:    "out of gas in location: block gas meter; gasWanted: 21000",
					Events: []abci.Event{},
				},
			},
			true,
		},
		{
			"sad: failed eth tx",
			&cmttypes.Block{Header: cmttypes.Header{Height: 1}, Data: cmttypes.Data{Txs: []cmttypes.Tx{validEVMTxBz}}},
			[]*abci.ResponseDeliverTx{
				{
					Code:   15,
					Log:    "nonce mismatch",
					Events: []abci.Event{},
				},
			},
			false,
		},
		{
			"sad: invalid events",
			&cmttypes.Block{Header: cmttypes.Header{Height: 1}, Data: cmttypes.Data{Txs: []cmttypes.Tx{validEVMTxBz}}},
			[]*abci.ResponseDeliverTx{
				{
					Code:   0,
					Events: []abci.Event{},
				},
			},
			false,
		},
		{
			"sad: not eth tx",
			&cmttypes.Block{Header: cmttypes.Header{Height: 1}, Data: cmttypes.Data{Txs: []cmttypes.Tx{invalidTxBz}}},
			[]*abci.ResponseDeliverTx{
				{
					Code:   0,
					Events: []abci.Event{},
				},
			},
			false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db := dbm.NewMemDB()
			idxer := indexer.NewEVMTxIndexer(db, tmlog.NewNopLogger(), clientCtx)

			err = idxer.IndexBlock(tc.block, tc.blockResult)
			require.NoError(t, err)
			if !tc.expSuccess {
				first, err := idxer.FirstIndexedBlock()
				require.NoError(t, err)
				require.Equal(t, int64(-1), first)

				last, err := idxer.LastIndexedBlock()
				require.NoError(t, err)
				require.Equal(t, int64(-1), last)
			} else {
				first, err := idxer.FirstIndexedBlock()
				require.NoError(t, err)
				require.Equal(t, tc.block.Height, first)

				last, err := idxer.LastIndexedBlock()
				require.NoError(t, err)
				require.Equal(t, tc.block.Height, last)

				res1, err := idxer.GetByTxHash(txHash)
				require.NoError(t, err)
				require.NotNil(t, res1)
				res2, err := idxer.GetByBlockAndIndex(1, 0)
				require.NoError(t, err)
				require.Equal(t, res1, res2)
			}
		})
	}
}
