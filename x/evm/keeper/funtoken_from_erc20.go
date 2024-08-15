// Copyright (c) 2023-2024 Nibi, Inc.
package keeper

import (
	"fmt"
	"math/big"

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
	info, err := k.FindERC20Metadata(ctx, erc20)
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
	bankMetadata := bank.Metadata{
		Description: fmt.Sprintf(
			"ERC20 token \"%s\" represented as a bank coin with a corresponding FunToken mapping", erc20.String(),
		),
		DenomUnits: []*bank.DenomUnit{
			{
				Denom:    bankDenom,
				Exponent: 0, // TODO(k-yang): determine which exponent to use
			},
		},
		Base:    bankDenom,
		Display: bankDenom,
		Name:    bankDenom,
		Symbol:  info.Symbol,
	}

	err = bankMetadata.Validate()
	if err != nil {
		return funtoken, fmt.Errorf("failed to validate bank metadata: %w", err)
	}
	k.bankKeeper.SetDenomMetaData(ctx, bankMetadata)

	// 5 | Officially create the funtoken mapping
	funtoken = &evm.FunToken{
		Erc20Addr:      eth.NewHexAddr(erc20),
		BankDenom:      bankDenom,
		IsMadeFromCoin: false,
	}

	return funtoken, k.FunTokens.SafeInsert(
		ctx, funtoken.Erc20Addr.Addr(),
		funtoken.BankDenom,
		funtoken.IsMadeFromCoin,
	)
}
