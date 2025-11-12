package evmtrader

import (
	"context"
	"math/big"

	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
	"github.com/ethereum/go-ethereum/accounts/abi"
	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
)

// getERC20ABI returns the ERC20 ABI.
func getERC20ABI() abi.ABI {
	return *embeds.SmartContract_ERC20MinterWithMetadataUpdates.ABI
}

// getWasmPrecompileABI returns the WASM precompile ABI.
func getWasmPrecompileABI() abi.ABI {
	return *embeds.SmartContract_Wasm.ABI
}

// queryERC20Balance queries the ERC20 balance of an account.
func (t *EVMTrader) queryERC20Balance(ctx context.Context, erc20ABI abi.ABI, token common.Address, account common.Address) (*big.Int, error) {
	data, err := erc20ABI.Pack("balanceOf", account)
	if err != nil {
		return nil, err
	}
	msg := ethereum.CallMsg{
		From: account,
		To:   &token,
		Data: data,
	}
	out, err := t.client.CallContract(ctx, msg, nil)
	if err != nil {
		return big.NewInt(0), nil
	}
	return new(big.Int).SetBytes(out), nil
}

