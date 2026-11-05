package gosdk

import (
	cmtrpcclient "github.com/NibiruChain/nibiru/v2/bft/rpc/client"
	cmtrpchttp "github.com/NibiruChain/nibiru/v2/bft/rpc/client/http"
)

var _ cmtrpcclient.Client = (*cmtrpchttp.HTTP)(nil)

// NewRPCClient: A remote Comet-BFT RPC client. An error is returned on
// invalid remote. The function panics when remote is nil.
//
// Args:
//   - rpcEndpt: endpoint in the form <protocol>://<host>:<port>
//   - websocket: websocket path (which always seems to be "/websocket")
func NewRPCClient(rpcEndpt string, websocket string) (*cmtrpchttp.HTTP, error) {
	return cmtrpchttp.New(rpcEndpt, websocket)
}
