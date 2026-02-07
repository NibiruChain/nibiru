// Copyright (c) 2023-2024 Nibi, Inc.
package evm

import (
	"fmt"
	"math/big"

	"github.com/cometbft/cometbft/crypto/tmhash"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bank "github.com/cosmos/cosmos-sdk/x/bank/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	"github.com/NibiruChain/nibiru/v2/eth"
)

// ZeroGasMeta is the context payload for zero-gas EVM transactions. Stored under CtxKeyZeroGasMeta.
// The three amount fields indicate what has happened: credit step sets CreditedWei; DeductGas sets
// PaidWei; after RefundGas, msg_server sets RefundedWei. Undo logic branches on which amounts are
// non-nil (e.g. RefundedWei == nil means refund has not run yet).
type ZeroGasMeta struct {
	CreditedWei *big.Int     // amount credited up front in ante
	PaidWei     *uint256.Int // amount deducted by DeductGas (full upfront cost), tracked as uint256
	RefundedWei *big.Int     // amount refunded by RefundGas
}

// AmountsToUndoCredit returns the wei amounts to burn during zero-gas undo: feeCollectorBurnWei
// from the fee collector, txSenderBurnWei from the tx sender. Caller must ensure m != nil.
func (m *ZeroGasMeta) AmountsToUndoCredit() (feeCollectorBurnWei, txSenderBurnWei *uint256.Int) {
	zero := uint256.NewInt(0)

	var paid uint256.Int
	if m.PaidWei != nil {
		paid = *m.PaidWei
	} else {
		paid = *zero
	}

	var refunded uint256.Int
	if m.RefundedWei != nil {
		refunded = *uint256.MustFromBig(m.RefundedWei)
	} else {
		refunded = *zero
	}

	var actualGasCost uint256.Int
	if paid.Cmp(&refunded) < 0 {
		actualGasCost = *zero
	} else {
		actualGasCost = *new(uint256.Int).Sub(&paid, &refunded)
	}
	feeCollectorBurnWei = new(uint256.Int).Set(&actualGasCost)

	var credited uint256.Int
	if m.CreditedWei != nil {
		credited = *uint256.MustFromBig(m.CreditedWei)
	} else {
		credited = *zero
	}

	if credited.Cmp(&actualGasCost) < 0 {
		txSenderBurnWei = new(uint256.Int).Set(zero)
	} else {
		txSenderBurnWei = new(uint256.Int).Sub(&credited, &actualGasCost)
	}
	return feeCollectorBurnWei, txSenderBurnWei
}

// FIXME: Explore problems arrising from ERC1155 creating multiple fungible
// tokens that are valid ERC20s with the same address.
// https://github.com/NibiruChain/nibiru/issues/1933
func (fun FunToken) ID() []byte {
	return NewFunTokenID(fun.Erc20Addr.Address, fun.BankDenom)
}

func NewFunTokenID(erc20 gethcommon.Address, bankDenom string) []byte {
	return tmhash.Sum([]byte(erc20.String() + "|" + bankDenom))
}

func funTokenValidationError(err error) error {
	return fmt.Errorf("FunTokenError: %s", err.Error())
}

func (fun FunToken) Validate() error {
	if err := sdk.ValidateDenom(fun.BankDenom); err != nil {
		return funTokenValidationError(err)
	}

	return nil
}

// NewFunToken is a canonical constructor for the [FunToken] type. Using this
// function helps guarantee a consistent string representation from the
// hex-encoded Ethereum address.
func NewFunToken(
	erc20 gethcommon.Address, bankDenom string, isMadeFromCoin bool,
) FunToken {
	return FunToken{
		Erc20Addr: eth.EIP55Addr{
			Address: erc20,
		},
		BankDenom:      bankDenom,
		IsMadeFromCoin: isMadeFromCoin,
	}
}

// ERC20Metadata: Optional metadata fields parsed from an ERC20 contract.
// The [Wrapped Ether contract] is a good example for reference.
//
//	```solidity
//	constract WETH9 {
//	  string public name     = "Wrapped Ether";
//	  string public symbol   = "WETH"
//	  uint8  public decimals = 18;
//	}
//	```
//
// Note that the name and symbol fields may be empty, according to the [ERC20
// specification].
//
// [Wrapped Ether contract]: https://etherscan.io/token/0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2#code
// [ERC20 specification]: https://eips.ethereum.org/EIPS/eip-20
type ERC20Metadata struct {
	Name     string
	Symbol   string
	Decimals uint8
}

// Checks that the necessary ERC20 metadata fields can be parsed from the given
// Bank Coin metadata for a FunToken mapping. ERC20.decimals can only be zero if
// "allowZeroDecimals" is true.
func ValidateFunTokenBankMetadata(
	bc bank.Metadata, allowZeroDecimals bool,
) (out ERC20Metadata, err error) {
	// Bank Coin Denom regex:
	// ```
	// reDnmString = `[a-zA-Z][a-zA-Z0-9/:._-]{2,127}`
	// ```
	// Denominations can be 3 ~ 128 characters long and support letters, followed
	// by either a letter, a number or a separator ('/', ':', '.', '_' or '-').
	err = bc.Validate()
	if err != nil {
		err = fmt.Errorf("invalid token metadata: %w", err)
		return
	}

	// bank.Metadata validation guarantees that both "Base" and "Display" denoms
	// pass "sdk.ValidateDenom" and that the "DenomUnits" slice has exponents in
	// ascending order with at least one element, which must be the base
	// denomination and have exponent 0.
	decimals := uint8(0)
	if len(bc.DenomUnits) > 0 {
		decimalsIdx := len(bc.DenomUnits) - 1
		decimals = uint8(bc.DenomUnits[decimalsIdx].Exponent)
	}

	out = ERC20Metadata{
		Name:     bc.Name,   // safe: guaranteed to not be blank
		Symbol:   bc.Symbol, // safe: guaranteed to not be blank
		Decimals: decimals,
	}
	if out.Decimals == 0 && !allowZeroDecimals {
		err = fmt.Errorf(`got ERC20.decimals = 0, which is considered an error unless "allow_zero_decimals" is true`)
		return
	}
	return out, nil
}

// Gracefully handles "out of gas"
func SafeConsumeGas(ctx sdk.Context, amount uint64, descriptor string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			// Convert panic to error
			ctx.GasMeter().GasRemaining()
			ctx.GasMeter().GasConsumed()
			err = fmt.Errorf("gas consumption failed: gasConsumed=%d, gasRemaining=%d, %v",
				ctx.GasMeter().GasConsumed(),
				ctx.GasMeter().GasRemaining(),
				r,
			)
		}
	}()

	ctx.GasMeter().ConsumeGas(amount, descriptor)
	return nil
}
