// Copyright (c) 2023-2024 Nibi, Inc.
package evmante

import (
	sdkioerrors "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/NibiruChain/nibiru/v2/x/common/asset"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/statedb"
	oracleKeeper "github.com/NibiruChain/nibiru/v2/x/oracle/keeper"
	txfeeskeeper "github.com/NibiruChain/nibiru/v2/x/txfees/keeper"
	txfeestypes "github.com/NibiruChain/nibiru/v2/x/txfees/types"
)

const feeTokenUsedKey string = "feeTokenUsed"

// AnteDecVerifyEthAcc validates an account balance checks
type AnteDecVerifyEthAcc struct {
	evmKeeper     *EVMKeeper
	txFeesKeeper  txfeeskeeper.Keeper
	accountKeeper evm.AccountKeeper
	oracleKeeper  oracleKeeper.Keeper
}

// NewAnteDecVerifyEthAcc creates a new EthAccountVerificationDecorator
func NewAnteDecVerifyEthAcc(k *EVMKeeper, ak evm.AccountKeeper, txf txfeeskeeper.Keeper, ok oracleKeeper.Keeper) AnteDecVerifyEthAcc {
	return AnteDecVerifyEthAcc{
		evmKeeper:     k,
		accountKeeper: ak,
		txFeesKeeper:  txf,
		oracleKeeper:  ok,
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

		canCover := false
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

		// check whether the sender has enough balance to pay for the transaction cost in alternative token
		feeTokens := anteDec.txFeesKeeper.GetFeeTokens(ctx)
		for _, feeToken := range feeTokens {
			var ratio sdkmath.LegacyDec

			out, err := anteDec.evmKeeper.GetErc20Balance(ctx, fromAddr, gethcommon.HexToAddress(feeToken.Address))
			if err != nil {
				return ctx, err
			}

			switch feeToken.TokenType {
			case txfeestypes.FeeTokenType_FEE_TOKEN_TYPE_CONVERTIBLE:
				ratio = sdkmath.LegacyOneDec()
			case txfeestypes.FeeTokenType_FEE_TOKEN_TYPE_SWAPPABLE:
				price, err := anteDec.oracleKeeper.GetExchangeRateTwap(ctx, asset.Pair(feeToken.Pair))
				if err != nil {
					return ctx, sdkioerrors.Wrapf(
						err, "failed to get exchange rate for pair %s", feeToken.Pair,
					)
				}
				basePrice, err := anteDec.oracleKeeper.GetExchangeRateTwap(ctx, asset.Pair("unibi:uusd"))
				if err != nil {
					return ctx, sdkioerrors.Wrapf(err, "failed to get TWAP for unibi:uusd")
				}
				if price.IsZero() {
					return ctx, sdkioerrors.Wrapf(sdkerrors.ErrInvalidRequest, "price for %s is zero", feeToken.Pair)
				}
				ratio = basePrice.Quo(price)
			}

			tokenBalance := sdkmath.LegacyNewDecFromBigInt(out)
			txCost := sdkmath.LegacyNewDecFromBigInt(txData.Cost())
			if tokenBalance.GTE(txCost.Mul(ratio)) {
				canCover = true
				ctx = ctx.WithValue(feeTokenUsedKey, feeToken.Address).WithValue("ratio", ratio)
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
