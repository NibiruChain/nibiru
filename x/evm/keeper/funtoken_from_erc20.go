// Copyright (c) 2023-2024 Nibi, Inc.
package keeper

import (
	"fmt"
	"math/big"

	sdkioerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bank "github.com/cosmos/cosmos-sdk/x/bank/types"
	gethabi "github.com/ethereum/go-ethereum/accounts/abi"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
	"github.com/NibiruChain/nibiru/v2/x/evm/statedb"
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
	evmObj *vm.EVM,
	contract gethcommon.Address,
	abi *gethabi.ABI,
) (info *evm.ERC20Metadata, err error) {
	effectiveAbi := embeds.SmartContract_ERC20MinterWithMetadataUpdates.ABI

	if abi != nil {
		effectiveAbi = abi
	}
	// Load name, symbol, decimals
	name, err := k.ERC20().LoadERC20Name(ctx, evmObj, effectiveAbi, contract)
	if err != nil {
		return nil, err
	}

	symbol, err := k.ERC20().LoadERC20Symbol(ctx, evmObj, effectiveAbi, contract)
	if err != nil {
		return nil, err
	}

	decimals, err := k.ERC20().LoadERC20Decimals(ctx, evmObj, effectiveAbi, contract)
	if err != nil {
		return nil, err
	}

	return &evm.ERC20Metadata{
		Name:     name,
		Symbol:   symbol,
		Decimals: decimals,
	}, nil
}

type (
	ERC20String struct{ Value string }
	// ERC20Uint8: Unpacking type for "uint8" from Solidity. This is only used in
	// the "ERC20.decimals" function.
	ERC20Uint8 struct{ Value uint8 }
	ERC20Bool  struct{ Value bool }
	// ERC20BigInt: Unpacking type for "uint256" from Solidity.
	ERC20BigInt  struct{ Value *big.Int }
	ERC20Bytes32 struct{ Value [32]byte }
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
	ctx sdk.Context, erc20 gethcommon.Address, allowZeroDecimals bool,
) (funtoken *evm.FunToken, err error) {
	// 1 | ERC20 already registered with FunToken?
	if funtokens := k.FunTokens.Collect(ctx, k.FunTokens.Indexes.ERC20Addr.ExactMatch(ctx, erc20)); len(funtokens) > 0 {
		return nil, fmt.Errorf("funtoken mapping already created for ERC20 \"%s\"", erc20)
	}

	// 2 | Get existing ERC20 metadata
	// We use dummy values for the tx config and evm config because we aren't in an actual end user transaction, it's just a state query.
	stateDB := k.Bank.StateDB
	if stateDB == nil {
		stateDB = k.NewStateDB(ctx, statedb.NewEmptyTxConfig(gethcommon.BytesToHash(ctx.HeaderHash())))
	}
	defer func() {
		k.Bank.StateDB = nil
	}()
	evmMsg := core.Message{
		To:               &erc20,
		From:             evm.EVM_MODULE_ADDRESS,
		Nonce:            k.GetAccNonce(ctx, evm.EVM_MODULE_ADDRESS),
		Value:            evm.Big0, // amount
		GasLimit:         0,
		GasPrice:         evm.Big0,
		GasFeeCap:        evm.Big0,
		GasTipCap:        evm.Big0,
		Data:             []byte{},
		AccessList:       gethcore.AccessList{},
		SkipNonceChecks:  false,
		SkipFromEOACheck: false,
	}

	evmObj := k.NewEVM(ctx, evmMsg, k.GetEVMConfig(ctx), nil, stateDB)
	erc20Info, err := k.FindERC20Metadata(ctx, evmObj, erc20, nil)
	if err != nil {
		return nil, err
	}

	bankDenom := fmt.Sprintf("erc20/%s", erc20.String())

	// 3 | Coin already registered with FunToken?
	_, isFound := k.Bank.GetDenomMetaData(ctx, bankDenom)
	if isFound {
		return nil, fmt.Errorf("bank coin denom already registered with denom \"%s\"", bankDenom)
	}
	if funtokens := k.FunTokens.Collect(ctx, k.FunTokens.Indexes.BankDenom.ExactMatch(ctx, bankDenom)); len(funtokens) > 0 {
		return nil, fmt.Errorf("funtoken mapping already created for bank denom \"%s\"", bankDenom)
	}

	// 4 | Set bank coin denom metadata in state
	var bankMetadata bank.Metadata
	{
		displayDenom := bankDenom
		denomUnits := []*bank.DenomUnit{
			{
				Denom:    bankDenom,
				Exponent: 0,
			},
		}
		if erc20Info.Decimals > 0 {
			// Bank denom "erc20/{addr}" is ~48 chars. Adding 19 more keeps the
			// length far under the limit of 127 from "sdk.ValidateDenom()".
			displayDenom = fmt.Sprintf("decimals_denom_for-%s", bankDenom)
			denomUnits = append(denomUnits, &bank.DenomUnit{
				Denom:    displayDenom,
				Exponent: uint32(erc20Info.Decimals),
			})
		}
		bankMetadata = bank.Metadata{
			Description: fmt.Sprintf(
				`ERC20 token "%s" represented as a Bank Coin with a corresponding FunToken mapping`, erc20.Hex(),
			),
			DenomUnits: denomUnits,
			Base:       bankDenom,
			Display:    displayDenom,
			Name:       erc20Info.Name,
			Symbol:     erc20Info.Symbol,
		}
	}
	if _, err = evm.ValidateFunTokenBankMetadata(
		bankMetadata,
		allowZeroDecimals,
	); err != nil {
		err = fmt.Errorf(`metadata unsuitable to create FunToken mapping for ERC20 "%s": %w`, erc20.Hex(), err)
		return
	}
	k.Bank.SetDenomMetaData(ctx, bankMetadata)

	// 5 | Officially create the funtoken mapping
	funtoken = &evm.FunToken{
		Erc20Addr: eth.EIP55Addr{
			Address: erc20,
		},
		BankDenom:      bankDenom,
		IsMadeFromCoin: false,
	}

	err = stateDB.Commit()
	if err != nil {
		return nil, sdkioerrors.Wrap(err, evm.ErrStateDBCommit)
	}

	return funtoken, k.FunTokens.SafeInsert(
		ctx, erc20, bankDenom, false,
	)
}
