package gosdk_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	sdktypestx "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/tx"
	"github.com/NibiruChain/nibiru/v2/x/gosdk"
)

func TestBroadcasterGrpc(t *testing.T) {
	for _, broadcast := range []struct {
		name string
		mode sdktypestx.BroadcastMode
		call func(gosdk.BroadcasterGrpc, []byte) (*sdk.TxResponse, error)
	}{
		{
			name: "BroadcastTx",
			mode: sdktypestx.BroadcastMode_BROADCAST_MODE_BLOCK,
			call: func(b gosdk.BroadcasterGrpc, txBytes []byte) (*sdk.TxResponse, error) {
				return b.BroadcastTx(txBytes, sdktypestx.BroadcastMode_BROADCAST_MODE_BLOCK)
			},
		},
		{
			name: "BroadcastTxSync",
			mode: sdktypestx.BroadcastMode_BROADCAST_MODE_SYNC,
			call: gosdk.BroadcasterGrpc.BroadcastTxSync,
		},
		{
			name: "BroadcastTxAsync",
			mode: sdktypestx.BroadcastMode_BROADCAST_MODE_ASYNC,
			call: gosdk.BroadcasterGrpc.BroadcastTxAsync,
		},
	} {
		t.Run(broadcast.name, func(t *testing.T) {
			for _, tc := range []struct {
				name     string
				response *sdk.TxResponse
				err      error
			}{
				{
					name: "grpc error",
					err:  status.Error(codes.Unavailable, "broadcast unavailable"),
				},
				{
					name:     "success",
					response: &sdk.TxResponse{TxHash: "ABC123"},
				},
				{
					name:     "transaction rejected",
					response: &sdk.TxResponse{Code: 5, RawLog: "insufficient funds"},
				},
			} {
				t.Run(tc.name, func(t *testing.T) {
					txBytes := []byte("signed transaction")
					// Intercept the RPC so the generated client runs without a node.
					conn, err := grpc.NewClient("passthrough:///broadcast-test",
						grpc.WithTransportCredentials(insecure.NewCredentials()),
						grpc.WithUnaryInterceptor(func(
							_ context.Context, method string, req, reply interface{},
							_ *grpc.ClientConn, _ grpc.UnaryInvoker, _ ...grpc.CallOption,
						) error {
							require.Equal(t, "/cosmos.tx.v1beta1.Service/BroadcastTx", method)
							require.Equal(t, &sdktypestx.BroadcastTxRequest{
								TxBytes: txBytes,
								Mode:    broadcast.mode,
							}, req)
							if tc.err != nil {
								return tc.err
							}
							reply.(*sdktypestx.BroadcastTxResponse).TxResponse = tc.response
							return nil
						}),
					)
					require.NoError(t, err)
					t.Cleanup(func() { require.NoError(t, conn.Close()) })

					response, err := broadcast.call(gosdk.BroadcasterGrpc{GRPC: conn}, txBytes)
					require.ErrorIs(t, err, tc.err)
					require.Same(t, tc.response, response)
				})
			}
		})
	}
}
