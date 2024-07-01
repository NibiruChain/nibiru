package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
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
	priv, err := ethsecp256k1.GenerateKey()
	if err != nil {
		return nil, err
	}
	newContractAddress := common.BytesToAddress(priv.PubKey().Address().Bytes())
	ctx := sdk.UnwrapSDKContext(goCtx)
	err = k.FunTokens.SafeInsert(ctx, newContractAddress, msg.Denom, true)
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
