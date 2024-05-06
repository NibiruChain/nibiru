// Copyright (c) 2023-2024 Nibi, Inc.
package rpcapi

import (
	"context"
	"fmt"

	rpcclient "github.com/cometbft/cometbft/rpc/client"
	"github.com/cosmos/cosmos-sdk/client"

	"github.com/NibiruChain/nibiru/eth"
)

// NetAPI is the eth_ prefixed set of APIs in the Web3 JSON-RPC spec.
type NetAPI struct {
	networkVersion uint64
	// TODO: epic: test(eth-rpc): "github.com/NibiruChain/nibiru/x/common/testutil/cli"
	// Validator.RPCClient should be used to test APIs that depend on the CometBFT
	// RPC client.
	tmClient rpcclient.Client
}

// NewImplNetAPI creates an instance of the public Net Web3 API.
func NewImplNetAPI(clientCtx client.Context) *NetAPI {
	// parse the chainID from a integer string
	chainIDEpoch, err := eth.ParseChainID(clientCtx.ChainID)
	if err != nil {
		panic(err)
	}

	return &NetAPI{
		networkVersion: chainIDEpoch.Uint64(),
		tmClient:       clientCtx.Client.(rpcclient.Client),
	}
}

// Version returns the current ethereum protocol version.
func (s *NetAPI) Version() string {
	return fmt.Sprintf("%d", s.networkVersion)
}

// Listening returns if client is actively listening for network connections.
func (s *NetAPI) Listening() bool {
	ctx := context.Background()
	netInfo, err := s.tmClient.NetInfo(ctx)
	if err != nil {
		return false
	}
	return netInfo.Listening
}

// PeerCount returns the number of peers currently connected to the client.
func (s *NetAPI) PeerCount() int {
	ctx := context.Background()
	netInfo, err := s.tmClient.NetInfo(ctx)
	if err != nil {
		return 0
	}
	return len(netInfo.Peers)
}
