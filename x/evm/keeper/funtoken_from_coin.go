package keeper

import (
	"fmt"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bank "github.com/cosmos/cosmos-sdk/x/bank/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
)

func (k *Keeper) createFunTokenFromCoin(
	ctx sdk.Context, bankDenom string,
) (funtoken *evm.FunToken, err error) {
	// 1 | Coin already registered with FunToken?
	if funtokens := k.FunTokens.Collect(ctx, k.FunTokens.Indexes.BankDenom.ExactMatch(ctx, bankDenom)); len(funtokens) > 0 {
		return nil, fmt.Errorf("funtoken mapping already created for bank denom \"%s\"", bankDenom)
	}

	// 2 | Check for denom metadata in bank state
	bankMetadata, isFound := k.bankKeeper.GetDenomMetaData(ctx, bankDenom)
	if !isFound {
		return nil, fmt.Errorf("bank coin denom should have bank metadata for denom \"%s\"", bankDenom)
	}

	// 3 | deploy ERC20 for metadata
	erc20Addr, err := k.deployERC20ForBankCoin(ctx, bankMetadata)
	if err != nil {
		return nil, errors.Wrap(err, "failed to deploy ERC20 for bank coin")
	}

	// 4 | ERC20 already registered with FunToken?
	if funtokens := k.FunTokens.Collect(ctx, k.FunTokens.Indexes.ERC20Addr.ExactMatch(ctx, erc20Addr)); len(funtokens) > 0 {
		return nil, fmt.Errorf("funtoken mapping already created for ERC20 \"%s\"", erc20Addr.Hex())
	}

	// 5 | Officially create the funtoken mapping
	funtoken = &evm.FunToken{
		Erc20Addr: eth.EIP55Addr{
			Address: erc20Addr,
		},
		BankDenom:      bankDenom,
		IsMadeFromCoin: true,
	}

	return funtoken, k.FunTokens.SafeInsert(
		ctx, erc20Addr,
		funtoken.BankDenom,
		funtoken.IsMadeFromCoin,
	)
}

func (k *Keeper) deployERC20ForBankCoin(
	ctx sdk.Context, bankCoin bank.Metadata,
) (erc20Addr gethcommon.Address, err error) {
	erc20Addr = crypto.CreateAddress(evm.EVM_MODULE_ADDRESS, k.GetAccNonce(ctx, evm.EVM_MODULE_ADDRESS))

	// bank.Metadata validation guarantees that both "Base" and "Display" denoms
	// pass "sdk.ValidateDenom" and that the "DenomUnits" slice has exponents in
	// ascending order with at least one element, which must be the base
	// denomination and have exponent 0.
	decimals := uint8(0)
	if len(bankCoin.DenomUnits) > 0 {
		decimalsIdx := len(bankCoin.DenomUnits) - 1
		decimals = uint8(bankCoin.DenomUnits[decimalsIdx].Exponent)
	}

	// pass empty method name to deploy the contract
	packedArgs, err := embeds.SmartContract_ERC20Minter.ABI.Pack("", bankCoin.Name, bankCoin.Symbol, decimals)
	if err != nil {
		return gethcommon.Address{}, errors.Wrap(err, "failed to pack ABI args")
	}
	bytecodeForCall := append(embeds.SmartContract_ERC20Minter.Bytecode, packedArgs...)

	// nil address for contract creation
	_, err = k.CallContractWithInput(
		ctx, evm.EVM_MODULE_ADDRESS, nil, true, bytecodeForCall,
	)
	if err != nil {
		return gethcommon.Address{}, errors.Wrap(err, "failed to deploy ERC20 contract")
	}

	return erc20Addr, nil
}
