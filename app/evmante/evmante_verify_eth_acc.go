// Copyright (c) 2023-2024 Nibi, Inc.
package evmante

import (
	"math/big"

	sdkioerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/statedb"
	gastokenante "github.com/NibiruChain/nibiru/v2/x/gastoken/ante"
	gastokenkeeper "github.com/NibiruChain/nibiru/v2/x/gastoken/keeper"
)

const gasTokenUsedKey string = "gasTokenUsed"
const gasTokenAmountKey string = "gasTokenAmount"
const useWNibiKey string = "useWNibi"

// AnteDecVerifyEthAcc validates an account balance checks
type AnteDecVerifyEthAcc struct {
	evmKeeper      *EVMKeeper
	gasTokenKeeper *gastokenkeeper.Keeper
	accountKeeper  evm.AccountKeeper
}

// NewAnteDecVerifyEthAcc creates a new EthAccountVerificationDecorator
func NewAnteDecVerifyEthAcc(k *EVMKeeper, ak evm.AccountKeeper, gtk *gastokenkeeper.Keeper) AnteDecVerifyEthAcc {
	return AnteDecVerifyEthAcc{
		evmKeeper:      k,
		accountKeeper:  ak,
		gasTokenKeeper: gtk,
	}
}

// AnteHandle validates checks that the sender balance is greater than the total
// transaction cost. The account will be set to store if it doesn't exist, i.e.
// cannot be found on store.
//
// This AnteHandler decorator will fail if:
// - any of the msgs is not a MsgEthereumTx
// - from address is empty
// - account balance is lower than the transaction cost
func (anteDec AnteDecVerifyEthAcc) AnteHandle(
	ctx sdk.Context,
	tx sdk.Tx,
	simulate bool,
	next sdk.AnteHandler,
) (newCtx sdk.Context, err error) {
	for i, msg := range tx.GetMsgs() {
		msgEthTx, ok := msg.(*evm.MsgEthereumTx)
		if !ok {
			return ctx, sdkioerrors.Wrapf(sdkerrors.ErrUnknownRequest, "invalid message type %T, expected %T", msg, (*evm.MsgEthereumTx)(nil))
		}
		txData, err := evm.UnpackTxData(msgEthTx.Data)
		if err != nil {
			return ctx, sdkioerrors.Wrapf(err, "failed to unpack tx data any for tx %d", i)
		}
		// sender address should be in the tx cache from the previous AnteHandle call
		from := msgEthTx.GetFrom()
		if from.Empty() {
			return ctx, sdkioerrors.Wrap(sdkerrors.ErrInvalidAddress, "from address cannot be empty")
		}
		// check whether the sender address is EOA
		fromAddr := gethcommon.BytesToAddress(from)
		acct := anteDec.evmKeeper.GetAccount(ctx, fromAddr)
		if acct == nil {
			acc := anteDec.accountKeeper.NewAccountWithAddress(ctx, from)
			anteDec.accountKeeper.SetAccount(ctx, acc)
			acct = statedb.NewEmptyAccount()
		} else if acct.IsContract() {
			return ctx, sdkioerrors.Wrapf(sdkerrors.ErrInvalidType,
				"the sender is not EOA: address %s, codeHash <%s>", fromAddr, acct.CodeHash)
		}
		cost := txData.Cost()
		if cost.Sign() < 0 {
			return ctx, sdkioerrors.Wrapf(
				sdkerrors.ErrInvalidCoins,
				"tx cost (%s) is negative and invalid", cost,
			)
		}
		balanceWei := evm.NativeToWei(acct.BalanceNative.ToBig())
		if balanceWei.Cmp(cost) >= 0 {
			return next(ctx, tx, simulate)
		}

		params, err := anteDec.gasTokenKeeper.GetParams(ctx)
		if err != nil {
			return ctx, sdkioerrors.Wrapf(err, "failed to get gastoken params")
		}
		wnibi := params.WnibiAddress
		if !gethcommon.IsHexAddress(wnibi) {
			return ctx, sdkioerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid WNIBI address in params: %q", wnibi)
		}

		wnibiBal, err := anteDec.evmKeeper.GetErc20Balance(ctx, fromAddr, gethcommon.HexToAddress(wnibi))
		if err != nil {
			return ctx, sdkioerrors.Wrapf(err, "failed to get WNIBI balance for account %s", fromAddr)
		}
		if wnibiBal.Cmp(cost) >= 0 {
			ctx = ctx.WithValue(useWNibiKey, true)
			continue
		}

		canCover := false
		// check whether the sender has enough balance to pay for the transaction cost in alternative token
		feeTokens := anteDec.gasTokenKeeper.GetFeeTokens(ctx)
		for _, feeToken := range feeTokens {
			bal, err := anteDec.evmKeeper.GetErc20Balance(ctx, fromAddr, gethcommon.HexToAddress(feeToken.Erc20Address))
			if err != nil {
				return ctx, err
			}

			amountNeeded, err := gastokenante.GetAmountInFromUniswap(
				ctx,
				anteDec.evmKeeper,
				anteDec.gasTokenKeeper,
				gethcommon.HexToAddress(feeToken.Erc20Address),
				gethcommon.HexToAddress(wnibi),
				big.NewInt(3000),
				cost,
			)
			if err != nil {
				return ctx, sdkioerrors.Wrapf(err, "failed to get amount in from uniswap for token %s", feeToken.Erc20Address)
			}
			if bal.Cmp(amountNeeded) >= 0 {
				canCover = true
				ctx = ctx.WithValue(gasTokenUsedKey, feeToken.Erc20Address)
				ctx = ctx.WithValue(gasTokenAmountKey, amountNeeded)
				break
			}
		}

		if !canCover {
			return ctx, sdkioerrors.Wrapf(
				sdkerrors.ErrInsufficientFunds,
				"sender balance < tx cost (native: %s, required: %s), no ERC20 fallback sufficient",
				balanceWei.String(), cost.String(),
			)
		}
	}
	return next(ctx, tx, simulate)
}
