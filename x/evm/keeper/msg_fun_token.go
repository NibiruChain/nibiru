package keeper

import (
	"context"

	"github.com/NibiruChain/nibiru/precompiles/erc20"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	"github.com/ethereum/go-ethereum/common"

	"github.com/NibiruChain/nibiru/eth/crypto/ethsecp256k1"
	"github.com/NibiruChain/nibiru/x/evm"
)

// CreateFunTokenFromCoin registers existing cosmos coin as an evm erc-20 fungible token contract
func (k *Keeper) CreateFunTokenFromCoin(
	goCtx context.Context, msg *evm.MsgCreateFunTokenFromCoin,
) (*evm.MsgCreateFunTokenFromCoinResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	// Generate contract address. TODO: consider using incremental addresses
	priv, err := ethsecp256k1.GenerateKey()
	if err != nil {
		return nil, err
	}
	newContractAddress := common.BytesToAddress(priv.PubKey().Address().Bytes())
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Save pair in fungible tokens mapping
	err = k.FunTokens.SafeInsert(ctx, newContractAddress, msg.Denom, true)
	if err != nil {
		return nil, err
	}

	// Create and register new ERC-20 precompile
	erc20Precompile, err := erc20.NewPrecompile(
		evm.NewFunToken(newContractAddress, msg.Denom, true),
		k.bankKeeper.(bankkeeper.Keeper),
	)
	if err != nil {
		return nil, err
	}
	err = k.AddEVMExtensions(ctx, erc20Precompile)
	if err != nil {
		return nil, err
	}

	_ = ctx.EventManager().EmitTypedEvent(&evm.EventCreateFunTokenFromCoin{
		Creator:         msg.Sender,
		Denom:           msg.Denom,
		ContractAddress: newContractAddress.String(),
	})

	return &evm.MsgCreateFunTokenFromCoinResponse{NewContractAddress: newContractAddress.String()}, nil
}
