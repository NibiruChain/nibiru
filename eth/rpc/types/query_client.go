// Copyright (c) 2023-2024 Nibi, Inc.
package types

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/types/tx"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/proto/tendermint/crypto"

	"github.com/cosmos/cosmos-sdk/client"

	evmtypes "github.com/NibiruChain/nibiru/x/evm/types"
)

// QueryClient defines a gRPC Client used for:
//   - TM transaction simulation
//   - EVM module queries
type QueryClient struct {
	tx.ServiceClient
	evmtypes.QueryClient
}

// NewQueryClient creates a new gRPC query client
func NewQueryClient(clientCtx client.Context) *QueryClient {
	return &QueryClient{
		ServiceClient: tx.NewServiceClient(clientCtx),
		QueryClient:   evmtypes.NewQueryClient(clientCtx),
	}
}

// GetProof performs an ABCI query with the given key and returns a merkle proof. The desired
// tendermint height to perform the query should be set in the client context. The query will be
// performed at one below this height (at the IAVL version) in order to obtain the correct merkle
// proof. Proof queries at height less than or equal to 2 are not supported.
// Issue: https://github.com/cosmos/cosmos-sdk/issues/6567
func (QueryClient) GetProof(
	clientCtx client.Context, storeKey string, key []byte,
) ([]byte, *crypto.ProofOps, error) {
	height := clientCtx.Height
	// ABCI queries at height less than or equal to 2 are not supported.
	// Base app does not support queries for height less than or equal to 1, and
	// the base app uses 0 indexing.
	//
	// Ethereum uses 1 indexing for the intial block height, therefore <= 2 means
	// <= (Eth) height 3.
	if height <= 2 {
		return nil, nil, fmt.Errorf(
			"proof queries at ABCI block height <= 2 are not supported")
	}

	abciReq := abci.RequestQuery{
		Path:   fmt.Sprintf("store/%s/key", storeKey),
		Data:   key,
		Height: height,
		Prove:  true,
	}

	abciRes, err := clientCtx.QueryABCI(abciReq)
	if err != nil {
		return nil, nil, err
	}

	return abciRes.Value, abciRes.ProofOps, nil
}
