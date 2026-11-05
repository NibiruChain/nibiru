package core

import (
	ctypes "github.com/NibiruChain/nibiru/v2/bft/rpc/core/types"
	rpctypes "github.com/NibiruChain/nibiru/v2/bft/rpc/jsonrpc/types"
)

// Health gets node health. Returns empty result (200 OK) on success, no
// response - in case of an error.
// More: https://docs.cometbft.com/v0.37/rpc/#/Info/health
func Health(ctx *rpctypes.Context) (*ctypes.ResultHealth, error) {
	return &ctypes.ResultHealth{}, nil
}
