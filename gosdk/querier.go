package gosdk

import (
	"errors"

	wasm "github.com/CosmWasm/wasmd/x/wasm/types"
	"google.golang.org/grpc"

	devgas "github.com/NibiruChain/nibiru/x/devgas/v1/types"
	epochs "github.com/NibiruChain/nibiru/x/epochs/types"
	"github.com/NibiruChain/nibiru/x/evm"
	inflation "github.com/NibiruChain/nibiru/x/inflation/types"
	xoracle "github.com/NibiruChain/nibiru/x/oracle/types"
	tokenfactory "github.com/NibiruChain/nibiru/x/tokenfactory/types"
)

type Querier struct {
	ClientConn *grpc.ClientConn

	// Smart Contracts
	EVM  evm.QueryClient
	Wasm wasm.QueryClient

	// Other Modules
	Devgas       devgas.QueryClient
	Epoch        epochs.QueryClient
	Inflation    inflation.QueryClient
	Oracle       xoracle.QueryClient
	TokenFactory tokenfactory.QueryClient
}

func NewQuerier(
	grpcConn *grpc.ClientConn,
) (Querier, error) {
	if grpcConn == nil {
		return Querier{}, errors.New(
			"cannot create NibiruQueryClient with nil grpc.ClientConn")
	}

	return Querier{
		ClientConn: grpcConn,

		EVM:  evm.NewQueryClient(grpcConn),
		Wasm: wasm.NewQueryClient(grpcConn),

		Devgas:       devgas.NewQueryClient(grpcConn),
		Epoch:        epochs.NewQueryClient(grpcConn),
		Inflation:    inflation.NewQueryClient(grpcConn),
		Oracle:       xoracle.NewQueryClient(grpcConn),
		TokenFactory: tokenfactory.NewQueryClient(grpcConn),
	}, nil
}
