package keeper

import (
	"fmt"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bank "github.com/cosmos/cosmos-sdk/x/bank/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/NibiruChain/nibiru/eth"
	"github.com/NibiruChain/nibiru/x/evm"
	"github.com/NibiruChain/nibiru/x/evm/embeds"
)

func (k *Keeper) CreateFunTokenFromCoin(
	ctx sdk.Context, bankDenom string,
) (funtoken evm.FunToken, err error) {
	// 1 | Coin already registered with FunToken?
	if funtokens := k.FunTokens.Collect(
		ctx, k.FunTokens.Indexes.BankDenom.ExactMatch(ctx, bankDenom),
	); len(funtokens) > 0 {
		return funtoken, fmt.Errorf("funtoken mapping already created for bank denom \"%s\"", bankDenom)
	}

	// 2 | Check for denom metadata in bank state
	bankCoin, isAlreadyCoin := k.bankKeeper.GetDenomMetaData(ctx, bankDenom)
	if !isAlreadyCoin {
		return funtoken, fmt.Errorf("bank coin denom should have bank metadata for denom \"%s\"", bankDenom)
	}

	// 3 | deploy ERC20 for metadata
	erc20Addr, err := k.DeployERC20ForBankCoin(ctx, bankCoin)
	if err != nil {
		return
	}

	// 4 | ERC20 already registered with FunToken?
	if funtokens := k.FunTokens.Collect(
		ctx, k.FunTokens.Indexes.ERC20Addr.ExactMatch(ctx, erc20Addr),
	); len(funtokens) > 0 {
		return funtoken, fmt.Errorf("funtoken mapping already created for ERC20 \"%s\"", erc20Addr.Hex())
	}

	// 5 | Officially create the funtoken mapping
	funtoken = evm.FunToken{
		Erc20Addr:      eth.NewHexAddr(erc20Addr),
		BankDenom:      bankDenom,
		IsMadeFromCoin: true,
	}

	return funtoken, k.FunTokens.SafeInsert(
		ctx, funtoken.Erc20Addr.ToAddr(),
		funtoken.BankDenom,
		funtoken.IsMadeFromCoin,
	)
}

func (k *Keeper) DeployERC20ForBankCoin(
	ctx sdk.Context, bankCoin bank.Metadata,
) (erc20Addr gethcommon.Address, err error) {
	// bank.Metadata validation guarantees that both "Base" and "Display" denoms
	// pass "sdk.ValidateDenom" and that the "DenomUnits" slice has exponents in
	// ascending order with at least one element, which must be the base
	// denomination and have exponent 0.
	decimals := uint8(0)
	if len(bankCoin.DenomUnits) > 0 {
		decimalsIdx := len(bankCoin.DenomUnits) - 1
		decimals = uint8(bankCoin.DenomUnits[decimalsIdx].Exponent)
	}

	erc20Embed := embeds.SmartContract_ERC20Minter
	callArgs := []any{bankCoin.Name, bankCoin.Symbol, decimals}
	methodName := "" // pass empty method name to deploy the contract
	packedArgs, err := erc20Embed.ABI.Pack(methodName, callArgs...)
	if err != nil {
		err = errors.Wrap(err, "failed to pack ABI args")
		return
	}
	bytecodeForCall := append(erc20Embed.Bytecode, packedArgs...)

	fromEvmAddr := evm.ModuleAddressEVM()
	nonce := k.GetAccNonce(ctx, fromEvmAddr)
	erc20Addr = crypto.CreateAddress(fromEvmAddr, nonce)
	erc20Contract := (*gethcommon.Address)(nil) // nil >> doesn't exist yet
	commit := true

	_, err = k.CallContractWithInput(
		ctx, fromEvmAddr, erc20Contract, commit, bytecodeForCall,
	)
	if err != nil {
		err = errors.Wrap(err, "deploy ERC20 failed")
		return
	}

	return erc20Addr, nil
}
