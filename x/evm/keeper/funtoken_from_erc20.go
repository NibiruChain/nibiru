// Copyright (c) 2023-2024 Nibi, Inc.
package keeper

import (
	"fmt"
	"math/big"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bank "github.com/cosmos/cosmos-sdk/x/bank/types"
	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
)

// FindERC20Metadata retrieves the metadata of an ERC20 token.
//
// Parameters:
//   - ctx: The SDK context for the transaction.
//   - contract: The Ethereum address of the ERC20 contract.
//
// Returns:
//   - info: ERC20Metadata containing name, symbol, and decimals.
//   - err: An error if metadata retrieval fails.
func (k Keeper) FindERC20Metadata(
	ctx sdk.Context,
	contract gethcommon.Address,
) (info *ERC20Metadata, err error) {
	// Load name, symbol, decimals
	name, err := k.LoadERC20Name(ctx, embeds.SmartContract_ERC20Minter.ABI, contract)
	if err != nil {
		return nil, err
	}

	symbol, err := k.LoadERC20Symbol(ctx, embeds.SmartContract_ERC20Minter.ABI, contract)
	if err != nil {
		return nil, err
	}

	decimals, err := k.LoadERC20Decimals(ctx, embeds.SmartContract_ERC20Minter.ABI, contract)
	if err != nil {
		return nil, err
	}

	return &ERC20Metadata{
		Name:     name,
		Symbol:   symbol,
		Decimals: decimals,
	}, nil
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

type (
	ERC20String struct{ Value string }
	// ERC20Uint8: Unpacking type for "uint8" from Solidity. This is only used in
	// the "ERC20.decimals" function.
	ERC20Uint8 struct{ Value uint8 }
	ERC20Bool  struct{ Value bool }
	// ERC20BigInt: Unpacking type for "uint256" from Solidity.
	ERC20BigInt struct{ Value *big.Int }
)

// createFunTokenFromERC20 creates a new FunToken mapping from an existing ERC20 token.
//
// This function performs the following steps:
//  1. Checks if the ERC20 token is already registered as a FunToken.
//  2. Retrieves the metadata of the existing ERC20 token.
//  3. Verifies that the corresponding bank coin denom is not already registered.
//  4. Sets the bank coin denom metadata in the state.
//  5. Creates and inserts the new FunToken mapping.
//
// Parameters:
//   - ctx: The SDK context for the transaction.
//   - erc20: The Ethereum address of the ERC20 token in HexAddr format.
//
// Returns:
//   - funtoken: The created FunToken mapping.
//   - err: An error if any step fails, nil otherwise.
//
// Possible errors:
//   - If the ERC20 token is already registered as a FunToken.
//   - If the ERC20 metadata cannot be retrieved.
//   - If the bank coin denom is already registered.
//   - If the bank metadata validation fails.
//   - If the FunToken insertion fails.
func (k *Keeper) createFunTokenFromERC20(
	ctx sdk.Context, erc20 gethcommon.Address,
) (funtoken *evm.FunToken, err error) {
	// 1 | ERC20 already registered with FunToken?
	if funtokens := k.FunTokens.Collect(ctx, k.FunTokens.Indexes.ERC20Addr.ExactMatch(ctx, erc20)); len(funtokens) > 0 {
		return funtoken, fmt.Errorf("funtoken mapping already created for ERC20 \"%s\"", erc20)
	}

	// 2 | Get existing ERC20 metadata
	erc20Info, err := k.FindERC20Metadata(ctx, erc20)
	if err != nil {
		return funtoken, err
	}

	bankDenom := fmt.Sprintf("erc20/%s", erc20.String())

	// 3 | Coin already registered with FunToken?
	_, isFound := k.bankKeeper.GetDenomMetaData(ctx, bankDenom)
	if isFound {
		return funtoken, fmt.Errorf("bank coin denom already registered with denom \"%s\"", bankDenom)
	}
	if funtokens := k.FunTokens.Collect(ctx, k.FunTokens.Indexes.BankDenom.ExactMatch(ctx, bankDenom)); len(funtokens) > 0 {
		return funtoken, fmt.Errorf("funtoken mapping already created for bank denom \"%s\"", bankDenom)
	}

	// 4 | Set bank coin denom metadata in state
	bankMetadata := erc20Info.ToBankMetadata(bankDenom, erc20)

	err = bankMetadata.Validate()
	if err != nil {
		return funtoken, fmt.Errorf("failed to validate bank metadata: %w", err)
	}
	k.bankKeeper.SetDenomMetaData(ctx, bankMetadata)

	// 5 | Officially create the funtoken mapping
	funtoken = &evm.FunToken{
		Erc20Addr: eth.EIP55Addr{
			Address: erc20,
		},
		BankDenom:      bankDenom,
		IsMadeFromCoin: false,
	}

	return funtoken, k.FunTokens.SafeInsert(
		ctx, erc20, bankDenom, false,
	)
}

// ToBankMetadata produces the "bank.Metadata" corresponding to a FunToken
// mapping created from an ERC20 token.
//
// The first argument of DenomUnits is required and the official base unit
// onchain, meaning the denom must be equivalent to bank.Metadata.Base.
//
// Decimals for an ERC20 are synonymous to "bank.DenomUnit.Exponent" in what
// they mean for external clients like wallets.
func (erc20Info ERC20Metadata) ToBankMetadata(
	bankDenom string, erc20 gethcommon.Address,
) bank.Metadata {
	var symbol string
	if erc20Info.Symbol != "" {
		symbol = erc20Info.Symbol
	} else {
		symbol = bankDenom
	}

	var name string
	if erc20Info.Name != "" {
		name = erc20Info.Name
	} else {
		name = bankDenom
	}

	denomUnits := []*bank.DenomUnit{
		{
			Denom:    bankDenom,
			Exponent: 0,
		},
	}
	display := symbol
	if erc20Info.Decimals > 0 {
		denomUnits = append(denomUnits, &bank.DenomUnit{
			Denom:    display,
			Exponent: uint32(erc20Info.Decimals),
		})
	}
	return bank.Metadata{
		Description: fmt.Sprintf(
			"ERC20 token \"%s\" represented as a Bank Coin with a corresponding FunToken mapping", erc20.String(),
		),
		DenomUnits: denomUnits,
		Base:       bankDenom,
		Display:    display,
		Name:       name,
		Symbol:     symbol,
	}
}

// Converts a coin that was originally an ERC20 token, and that was converted to a bank coin, back to an ERC20 token.
// EVM module does not own the ERC-20 contract and cannot mint the ERC-20 tokens.
// EVM module has escrowed tokens in the first conversion from ERC-20 to bank coin.
func (k Keeper) convertCoinNativeERC20(
	ctx sdk.Context,
	sender sdk.AccAddress,
	recipient gethcommon.Address,
	coin sdk.Coin,
	funTokenMapping evm.FunToken,
) (*evm.MsgConvertCoinToEvmResponse, error) {
	erc20Addr := funTokenMapping.Erc20Addr.Address

	recipientBalanceBefore, err := k.ERC20().BalanceOf(erc20Addr, recipient, ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrieve balance")
	}
	if recipientBalanceBefore == nil {
		return nil, fmt.Errorf("failed to retrieve balance, balance is nil")
	}

	// Escrow Coins on module account
	if err := k.bankKeeper.SendCoinsFromAccountToModule(
		ctx,
		sender,
		evm.ModuleName,
		sdk.NewCoins(coin),
	); err != nil {
		return nil, errors.Wrap(err, "failed to escrow coins")
	}

	// verify that the EVM module account has enough escrowed ERC-20 to transfer
	// should never fail, because the coins were minted from the escrowed tokens, but check just in case
	evmModuleBalance, err := k.ERC20().BalanceOf(
		erc20Addr,
		evm.EVM_MODULE_ADDRESS,
		ctx,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrieve balance")
	}
	if evmModuleBalance == nil {
		return nil, fmt.Errorf("failed to retrieve balance, balance is nil")
	}
	if evmModuleBalance.Cmp(coin.Amount.BigInt()) < 0 {
		return nil, fmt.Errorf("insufficient balance in EVM module account")
	}

	// unescrow ERC-20 tokens from EVM module address
	res, err := k.ERC20().Transfer(
		erc20Addr,
		evm.EVM_MODULE_ADDRESS,
		recipient,
		coin.Amount.BigInt(),
		ctx,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to transfer ERC20 tokens")
	}
	if !res {
		return nil, fmt.Errorf("failed to transfer ERC20 tokens")
	}

	// Check expected Receiver balance after transfer execution
	recipientBalanceAfter, err := k.ERC20().BalanceOf(erc20Addr, recipient, ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrieve balance")
	}
	if recipientBalanceAfter == nil {
		return nil, fmt.Errorf("failed to retrieve balance, balance is nil")
	}

	expectedFinalBalance := big.NewInt(0).Add(recipientBalanceBefore, coin.Amount.BigInt())
	if r := recipientBalanceAfter.Cmp(expectedFinalBalance); r != 0 {
		return nil, fmt.Errorf("expected balance after transfer to be %s, got %s", expectedFinalBalance, recipientBalanceAfter)
	}

	// Burn escrowed Coins
	err = k.bankKeeper.BurnCoins(ctx, evm.ModuleName, sdk.NewCoins(coin))
	if err != nil {
		return nil, errors.Wrap(err, "failed to burn coins")
	}

	_ = ctx.EventManager().EmitTypedEvent(&evm.EventConvertCoinToEvm{
		Sender:               sender.String(),
		Erc20ContractAddress: funTokenMapping.Erc20Addr.String(),
		ToEthAddr:            recipient.String(),
		BankCoin:             coin,
	})

	return &evm.MsgConvertCoinToEvmResponse{}, nil
}
