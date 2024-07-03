// Copyright (c) 2023-2024 Nibi, Inc.
package keeper

import (
	"context"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	"github.com/NibiruChain/nibiru/x/evm"
	"github.com/NibiruChain/nibiru/x/evm/embeds"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// SendFunTokenToErc20 Sends a coin with a valid "FunToken" mapping to the
// given recipient address ("to_eth_addr") in the corresponding ERC20
// representation.
func (k *Keeper) SendFunTokenToErc20(
	goCtx context.Context, msg *evm.MsgSendFunTokenToErc20,
) (resp *evm.MsgSendFunTokenToErc20Response, err error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	sender := sdk.MustAccAddressFromBech32(msg.Sender)
	recipient := msg.ToEthAddr.ToAddr()
	bankDenom := msg.BankCoin.Denom
	amount := msg.BankCoin.Amount

	funtokens := k.FunTokens.Collect(ctx, k.FunTokens.Indexes.BankDenom.ExactMatch(ctx, bankDenom))
	if len(funtokens) == 0 {
		return nil, fmt.Errorf("Funtoken for bank denom \"%s\" does not exist", bankDenom)
	}
	erc20ContractAddr := funtokens[0].Erc20Addr.ToAddr()

	// Step 1: Send coins to the evm module account
	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, sender, evm.ModuleName, sdk.Coins{msg.BankCoin})
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to send coins to module account")
	}

	// Step 2: evm call to erc20 minter: mint tokens for a recipient
	evmResp, err := k.CallContract(
		ctx,
		embeds.EmbeddedContractERC20Minter.ABI,
		evm.ModuleAddressEVM(),
		&erc20ContractAddr,
		true,
		"mint",
		recipient,
		amount,
	)
	if err != nil {
		return nil, err
	}
	if evmResp.Failed() {
		return nil,
			fmt.Errorf("failed to mint ERC-20 tokens of contract %s", erc20ContractAddr.String())
	}
	return &evm.MsgSendFunTokenToErc20Response{}, nil
}
