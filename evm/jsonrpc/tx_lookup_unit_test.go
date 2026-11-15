package jsonrpc

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	bftclient "github.com/NibiruChain/nibiru/v2/bft/rpc/client"
	coretypes "github.com/NibiruChain/nibiru/v2/bft/rpc/core/types"
	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/client"
)

type lookupTestIndexer struct {
	eth.EVMTxIndexer
	err error
}

func (i lookupTestIndexer) GetByTxHash(common.Hash) (*eth.TxResult, error) {
	return nil, i.err
}

type lookupTestClient struct {
	client.TendermintRPC
	bftclient.MempoolClient
	err error
}

func (c lookupTestClient) UnconfirmedTxs(context.Context, *int) (*coretypes.ResultUnconfirmedTxs, error) {
	if c.err != nil {
		return nil, c.err
	}
	return &coretypes.ResultUnconfirmedTxs{}, nil
}

func TestTransactionLookupMissingAndErrors(t *testing.T) {
	indexErr := errors.New("index unavailable")
	mempoolErr := errors.New("mempool unavailable")
	for _, tc := range []struct {
		name       string
		indexErr   error
		mempoolErr error
		wantErr    error
	}{
		{name: "not indexed or pending", indexErr: fmt.Errorf("wrapped: %w", eth.ErrTxNotFound)},
		{name: "nil index result", indexErr: nil},
		{name: "index error", indexErr: indexErr, wantErr: indexErr},
		{name: "mempool error", indexErr: eth.ErrTxNotFound, mempoolErr: mempoolErr, wantErr: mempoolErr},
	} {
		t.Run(tc.name, func(t *testing.T) {
			backend := Backend{
				ctx:          context.Background(),
				clientCtx:    client.Context{Client: lookupTestClient{err: tc.mempoolErr}},
				evmTxIndexer: lookupTestIndexer{err: tc.indexErr},
			}
			tx, err := backend.GetTransactionByHash(common.Hash{})
			require.Nil(t, tx)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
