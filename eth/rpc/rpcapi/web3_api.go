// Copyright (c) 2023-2024 Nibi, Inc.
package rpcapi

import (
	appconst "github.com/NibiruChain/nibiru/v2/app/appconst"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

// APIWeb3 is the web3_ prefixed set of APIs in the Web3 JSON-RPC spec.
type APIWeb3 struct{}

// NewImplWeb3API creates an instance of the Web3 API.
func NewImplWeb3API() *APIWeb3 {
	return &APIWeb3{}
}

// ClientVersion returns the client version in the Web3 user agent format.
func (a *APIWeb3) ClientVersion() string {
	return appconst.RuntimeVersion()
}

// Sha3 returns the keccak-256 hash of the passed-in input.
func (a *APIWeb3) Sha3(input string) hexutil.Bytes {
	return crypto.Keccak256(hexutil.Bytes(input))
}
