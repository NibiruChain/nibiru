// Copyright (c) 2023-2024 Nibi, Inc.
package keeper

import (
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	bank "github.com/cosmos/cosmos-sdk/x/bank/types"
	gethabi "github.com/ethereum/go-ethereum/accounts/abi"
	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/NibiruChain/nibiru/eth"
	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/evm"
	"github.com/NibiruChain/nibiru/x/evm/embeds"
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
) (info ERC20Metadata, err error) {
	var abi gethabi.ABI = embeds.Contract_ERC20Minter.ABI

	errs := []error{}

	// Load name, symbol, decimals
	name, err := k.LoadERC20Name(ctx, abi, contract)
	errs = append(errs, err)
	symbol, err := k.LoadERC20Symbol(ctx, abi, contract)
	errs = append(errs, err)
	decimals, err := k.LoadERC20Decimals(ctx, abi, contract)
	errs = append(errs, err)

	err = common.CombineErrors(errs...)
	if err != nil {
		err = fmt.Errorf("failed to \"FindERC20Metadata\": %w", err)
		return info, err
	}

	return ERC20Metadata{
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

// CreateFunTokenFromERC20 creates a new FunToken mapping from an existing ERC20 token.
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
func (k *Keeper) CreateFunTokenFromERC20(
	ctx sdk.Context, erc20 eth.HexAddr,
) (funtoken evm.FunToken, err error) {
	erc20Addr := erc20.ToAddr()

	// 1 | ERC20 already registered with FunToken?
	if funtokens := k.FunTokens.Collect(
		ctx, k.FunTokens.Indexes.ERC20Addr.ExactMatch(ctx, erc20Addr),
	); len(funtokens) > 0 {
		return funtoken, fmt.Errorf("Funtoken mapping already created for ERC20 \"%s\"", erc20Addr.Hex())
	}

	// 2 | Get existing ERC20 metadata
	info, err := k.FindERC20Metadata(ctx, erc20Addr)
	if err != nil {
		return
	}
	bankDenom := fmt.Sprintf("erc20/%s", erc20.String())

	// 3 | Coin already registered with FunToken?
	_, isAlreadyCoin := k.bankKeeper.GetDenomMetaData(ctx, bankDenom)
	if isAlreadyCoin {
		return funtoken, fmt.Errorf("Bank coin denom already registered with denom \"%s\"", bankDenom)
	}
	if funtokens := k.FunTokens.Collect(
		ctx, k.FunTokens.Indexes.BankDenom.ExactMatch(ctx, bankDenom),
	); len(funtokens) > 0 {
		return funtoken, fmt.Errorf("Funtoken mapping already created for bank denom \"%s\"", bankDenom)
	}

	// 4 | Set bank coin denom metadata in state
	bankMetadata := bank.Metadata{
		Description: fmt.Sprintf(
			"ERC20 token \"%s\" represented as a bank coin with corresponding FunToken mapping", erc20.String(),
		),
		DenomUnits: []*bank.DenomUnit{
			{
				Denom:    bankDenom,
				Exponent: 0,
			},
		},
		Base:    bankDenom,
		Display: bankDenom,
		Name:    bankDenom,
		Symbol:  info.Symbol,
	}

	err = bankMetadata.Validate()
	if err != nil {
		return
	}
	k.bankKeeper.SetDenomMetaData(ctx, bankMetadata)

	// 5 | Officially create the funtoken mapping
	funtoken = evm.FunToken{
		Erc20Addr:      erc20,
		BankDenom:      bankDenom,
		IsMadeFromCoin: false,
	}

	return funtoken, k.FunTokens.SafeInsert(
		ctx, funtoken.Erc20Addr.ToAddr(),
		funtoken.BankDenom,
		funtoken.IsMadeFromCoin,
	)
}
