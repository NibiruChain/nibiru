// Copyright (c) 2023-2024 Nibi, Inc.
package keeper

import (
	"fmt"
	"math/big"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bank "github.com/cosmos/cosmos-sdk/x/bank/types"
	gethabi "github.com/ethereum/go-ethereum/accounts/abi"
	gethcommon "github.com/ethereum/go-ethereum/common"
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
) (info *ERC20Metadata, err error) {
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
	ERC20BigInt  struct{ Value *big.Int }
	ERC20Bytes32 struct{ Value [32]byte }
)

// createFunTokenFromERC20 creates a new FunToken mapping from an existing ERC20 token.
//
// This function performs the following steps:
//  1. Checks if the ERC20 token is already registered as a FunToken.
//  2. Retrieves the metadata of the existing ERC20 token.
//  3. Verifies that the ERC20 token include expected functions.
//  4. Verifies that the corresponding bank coin denom is not already registered.
//  5. Sets the bank coin denom metadata in the state.
//  6. Creates and inserts the new FunToken mapping.
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
	evmMsg := gethcore.NewMessage(
		evm.EVM_MODULE_ADDRESS,
		&erc20,
		0,
		big.NewInt(0),
		0,
		big.NewInt(0),
		big.NewInt(0),
		big.NewInt(0),
		[]byte{},
		gethcore.AccessList{},
		false,
	)
	evmObj := k.NewEVM(ctx, evmMsg, k.GetEVMConfig(ctx), evm.NewNoOpTracer(), stateDB)
	erc20Info, err := k.FindERC20Metadata(ctx, evmObj, erc20, nil)
	if err != nil {
		return nil, err
	}

	bankDenom := fmt.Sprintf("erc20/%s", erc20.String())

	// 3 | Verify that the ERC20 token include expected functions
	abi := embeds.SmartContract_ERC20Minter.ABI
	if err := k.checkErc20ImplementsAllRequired(ctx, erc20, abi); err != nil {
		return funtoken, err
	}

	// 4 | Coin already registered with FunToken?
	_, isFound := k.Bank.GetDenomMetaData(ctx, bankDenom)
	if isFound {
		return nil, fmt.Errorf("bank coin denom already registered with denom \"%s\"", bankDenom)
	}
	if funtokens := k.FunTokens.Collect(ctx, k.FunTokens.Indexes.BankDenom.ExactMatch(ctx, bankDenom)); len(funtokens) > 0 {
		return nil, fmt.Errorf("funtoken mapping already created for bank denom \"%s\"", bankDenom)
	}

	// 5 | Set bank coin denom metadata in state
	bankMetadata := erc20Info.ToBankMetadata(bankDenom, erc20)

	err = bankMetadata.Validate()
	if err != nil {
		return nil, fmt.Errorf("failed to validate bank metadata: %w", err)
	}
	k.Bank.SetDenomMetaData(ctx, bankMetadata)

	// 6 | Officially create the funtoken mapping
	funtoken = &evm.FunToken{
		Erc20Addr: eth.EIP55Addr{
			Address: erc20,
		},
		BankDenom:      bankDenom,
		IsMadeFromCoin: false,
	}

	err = stateDB.Commit()
	if err != nil {
		return nil, errors.Wrap(err, "failed to commit stateDB")
	}

	return funtoken, k.FunTokens.SafeInsert(
		ctx, erc20, bankDenom, false,
	)
}

func (k Keeper) checkErc20ImplementsAllRequired(
	ctx sdk.Context, erc20Addr gethcommon.Address, abi *gethabi.ABI,
) error {
	methodNames := []string{
		"balanceOf",
		"transfer",
		"symbol",
		"decimals",
		"name",
		"totalSupply",
	}

	for _, methodName := range methodNames {
		method, ok := abi.Methods[methodName]
		if !ok {
			return fmt.Errorf("method '%s' not found in contract at %s", methodName, erc20Addr.Hex())
		}
		hasMethod, err := k.HasMethodInContract(ctx, erc20Addr, method)
		if err != nil {
			return err
		}
		if !hasMethod {
			return fmt.Errorf("method '%s' not found in contract at %s", methodName, erc20Addr.Hex())
		}
	}
	return nil
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
