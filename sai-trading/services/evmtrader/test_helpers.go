//go:build e2e || test
// +build e2e test

package evmtrader

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"google.golang.org/grpc"
)

func (t *EVMTrader) Client() *ethclient.Client {
	return t.client
}

func (t *EVMTrader) AccountAddr() common.Address {
	return t.accountAddr
}

func (t *EVMTrader) Addrs() ContractAddresses {
	return t.addrs
}

func (t *EVMTrader) GRPCConn() *grpc.ClientConn {
	return t.grpcConn
}

func (t *EVMTrader) QueryERC20Balance(ctx context.Context, erc20ABI abi.ABI, token common.Address, account common.Address) (*big.Int, error) {
	return t.queryERC20Balance(ctx, erc20ABI, token, account)
}

func GetERC20ABI() abi.ABI {
	return getERC20ABI()
}
