package cli

import (
	"context"

	abci "github.com/NibiruChain/nibiru/v2/bft/abci/types"
	tmbytes "github.com/NibiruChain/nibiru/v2/bft/libs/bytes"
	cmtrpcclient "github.com/NibiruChain/nibiru/v2/bft/rpc/client"
	rpcclientmock "github.com/NibiruChain/nibiru/v2/bft/rpc/client/mock"
	coretypes "github.com/NibiruChain/nibiru/v2/bft/rpc/core/types"
	cmttypes "github.com/NibiruChain/nibiru/v2/bft/types"

	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/client"
)

var _ client.TendermintRPC = (*MockTendermintRPC)(nil)

type MockTendermintRPC struct {
	rpcclientmock.Client

	responseQuery abci.ResponseQuery
}

// NewMockTendermintRPC returns a mock TendermintRPC implementation.
// It is used for CLI testing.
func NewMockTendermintRPC(respQuery abci.ResponseQuery) MockTendermintRPC {
	return MockTendermintRPC{responseQuery: respQuery}
}

func (MockTendermintRPC) BroadcastTxSync(context.Context, cmttypes.Tx) (*coretypes.ResultBroadcastTx, error) {
	return &coretypes.ResultBroadcastTx{Code: 0}, nil
}

func (m MockTendermintRPC) ABCIQueryWithOptions(
	_ context.Context,
	_ string,
	_ tmbytes.HexBytes,
	_ cmtrpcclient.ABCIQueryOptions,
) (*coretypes.ResultABCIQuery, error) {
	return &coretypes.ResultABCIQuery{Response: m.responseQuery}, nil
}
