// Copyright (c) 2023-2024 Nibi, Inc.
package keeper

import (
	"fmt"
	"math/big"

	sdkioerrors "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
	"github.com/NibiruChain/nibiru/v2/x/evm/statedb"
)

// convertEvmToCoinForCoinOriginated is part of the
// "eth.evm.v1.MsgConvertEvmToCoin" tx. This function handles conversion of ERC20
// tokens that were originally bank coins back into coin form. The EVM module
// owns the ERC20 contract and will burn the tokens
func (k Keeper) convertEvmToCoinForCoinOriginated(
	ctx sdk.Context,
	sender evm.Addrs,
	toAddress sdk.AccAddress,
	erc20Addr gethcommon.Address,
	amount *big.Int,
	bankDenom string,
	stateDB *statedb.StateDB,
) (*evm.MsgConvertEvmToCoinResponse, error) {
	bankCoins := sdk.NewCoins(sdk.NewCoin(bankDenom, sdkmath.NewIntFromBigInt(amount)))

	// 1 | Burn the ERC20 tokens from the sender's account
	contractInput, err := embeds.SmartContract_ERC20MinterWithMetadataUpdates.ABI.Pack(
		"burnFromAuthority",
		sender.Eth /*from: address where we burn the token balance from*/, amount,
	)
	if err != nil {
		return nil, err
	}

	unusedBigInt := big.NewInt(0)
	evmMsg := core.Message{
		To:               &erc20Addr,
		From:             evm.EVM_MODULE_ADDRESS,
		Nonce:            k.GetAccNonce(ctx, evm.EVM_MODULE_ADDRESS),
		Value:            unusedBigInt,
		GasLimit:         Erc20GasLimitExecute,
		GasPrice:         unusedBigInt,
		GasFeeCap:        unusedBigInt,
		GasTipCap:        unusedBigInt,
		Data:             contractInput,
		AccessList:       gethcore.AccessList{},
		BlobGasFeeCap:    &big.Int{},
		BlobHashes:       []gethcommon.Hash{},
		SkipNonceChecks:  true,
		SkipFromEOACheck: true,
	}

	evmObj := k.NewEVM(ctx, evmMsg, k.GetEVMConfig(ctx), nil /*tracer*/, stateDB)
	evmResp, err := k.CallContractWithInput(
		ctx,
		evmObj,
		evm.EVM_MODULE_ADDRESS,
		&erc20Addr,
		true, /*commit*/
		contractInput,
		Erc20GasLimitExecute,
		nil,
	)
	if err != nil {
		return nil, err
	}

	if evmResp.Failed() {
		return nil, fmt.Errorf("failed to burn ERC20 tokens: %s", evmResp.VmError)
	}

	// 2 | Send Bank Coins from the EVM module to the recipient
	err = k.Bank.SendCoinsFromModuleToAccount(ctx, evm.ModuleName, toAddress, bankCoins)
	if err != nil {
		return nil, sdkioerrors.Wrap(err, "failed to send coins from module to account")
	}

	// Commit the stateDB
	if err = stateDB.Commit(); err != nil {
		return nil, sdkioerrors.Wrap(err, "failed to commit stateDB")
	}

	// Emit event
	_ = ctx.EventManager().EmitTypedEvent(&evm.EventConvertEvmToCoin{
		Sender:               sender.Bech32.String(),
		Erc20ContractAddress: erc20Addr.Hex(),
		ToAddress:            toAddress.String(),
		BankCoin:             bankCoins[0],
		SenderEthAddr:        sender.Eth.Hex(),
	})

	// Emit tx logs of Burn event
	err = ctx.EventManager().EmitTypedEvent(&evm.EventTxLog{Logs: evmResp.Logs})
	if err == nil {
		k.updateBlockBloom(ctx, evmResp, uint64(k.EvmState.BlockTxIndex.GetOr(ctx, 0)))
	}

	return &evm.MsgConvertEvmToCoinResponse{}, nil
}

// convertEvmToCoinForERC20Originated handles conversion of ERC20 tokens that
// were originally ERC20. The EVM module doesn't own the ERC20 contract, so it
// transfers tokens to itself and mints bank coins
func (k Keeper) convertEvmToCoinForERC20Originated(
	ctx sdk.Context,
	sender evm.Addrs,
	toAddress sdk.AccAddress,
	erc20Addr gethcommon.Address,
	amount *big.Int,
	bankDenom string,
	stateDB *statedb.StateDB,
) (*evm.MsgConvertEvmToCoinResponse, error) {
	// 1 | Transfer ERC20 tokens from sender to EVM module
	contractInput, err := embeds.SmartContract_ERC20MinterWithMetadataUpdates.ABI.Pack("transfer", evm.EVM_MODULE_ADDRESS, amount)
	if err != nil {
		return nil, err
	}

	var evmObj *vm.EVM
	{
		unusedBigInt := big.NewInt(0)
		evmMsg := core.Message{
			From:             sender.Eth,
			To:               &evm.EVM_MODULE_ADDRESS,
			Nonce:            k.GetAccNonce(ctx, sender.Eth),
			Value:            unusedBigInt,
			GasLimit:         Erc20GasLimitExecute,
			GasPrice:         unusedBigInt,
			GasFeeCap:        unusedBigInt,
			GasTipCap:        unusedBigInt,
			Data:             contractInput,
			AccessList:       gethcore.AccessList{},
			BlobGasFeeCap:    &big.Int{},
			BlobHashes:       []gethcommon.Hash{},
			SkipNonceChecks:  true,
			SkipFromEOACheck: true,
		}
		evmObj = k.NewEVM(ctx, evmMsg, k.GetEVMConfig(ctx), nil /*tracer*/, stateDB)
	}

	balIncrease, evmResp, err := k.ERC20().Transfer(
		erc20Addr,              /*erc20Contract gethcommon.Address*/
		sender.Eth,             /*sender*/
		evm.EVM_MODULE_ADDRESS, /*recipient*/
		amount,                 /*amount*/
		ctx,
		evmObj,
	)
	if err != nil {
		return nil, sdkioerrors.Wrap(err, "failed to transfer ERC20 tokens")
	}
	if evmResp.Failed() {
		return nil, fmt.Errorf("failed to transfer ERC20 tokens: %s", evmResp.VmError)
	}

	bankCoin := sdk.NewCoin(bankDenom, sdkmath.NewIntFromBigInt(balIncrease))

	// 2 | Mint Bank Coins to the recipient
	err = k.Bank.MintCoins(ctx, evm.ModuleName, sdk.NewCoins(bankCoin))
	if err != nil {
		return nil, sdkioerrors.Wrap(err, "failed to mint coins")
	}

	// 3 | Send the minted coins to the recipient
	err = k.Bank.SendCoinsFromModuleToAccount(ctx, evm.ModuleName, toAddress, sdk.NewCoins(bankCoin))
	if err != nil {
		return nil, sdkioerrors.Wrap(err, "failed to send coins to recipient")
	}

	// Commit the stateDB
	if err = stateDB.Commit(); err != nil {
		return nil, sdkioerrors.Wrap(err, "failed to commit stateDB")
	}

	// Emit event
	_ = ctx.EventManager().EmitTypedEvent(&evm.EventConvertEvmToCoin{
		Sender:               sender.Bech32.String(),
		Erc20ContractAddress: erc20Addr.Hex(),
		ToAddress:            toAddress.String(),
		BankCoin:             bankCoin,
		SenderEthAddr:        sender.Eth.Hex(),
	})

	// Emit tx logs of Transfer event
	err = ctx.EventManager().EmitTypedEvent(&evm.EventTxLog{Logs: evmResp.Logs})
	if err == nil {
		k.updateBlockBloom(ctx, evmResp, uint64(k.EvmState.BlockTxIndex.GetOr(ctx, 0)))
	}

	return &evm.MsgConvertEvmToCoinResponse{}, nil
}

func (k Keeper) convertEvmToCoinForWNIBI(
	ctx sdk.Context,
	stateDB *statedb.StateDB,
	erc20 eth.EIP55Addr,
	sender evm.Addrs,
	toAddrBech32 sdk.AccAddress,
	amount sdkmath.Int,
) (resp *evm.MsgConvertEvmToCoinResponse, err error) {
	// isTx: value to use for commit in any EVM calls
	isTx := true

	withdrawWei, err := ParseWeiAsMultipleOfMicronibi(amount.BigInt())
	if err != nil {
		return nil, sdkioerrors.Wrapf(err, "ConvertEvmToCoin: invalid wei amount %s", amount)
	}

	// Unwrap from the sender "WNIBI.withdraw"
	//
	//	```solidity
	//	function withdraw(
	//	    uint amount
	//	) external;
	//	```
	contractInput, err := embeds.SmartContract_WNIBI.ABI.Pack(
		"withdraw",
		withdrawWei.ToBig(),
	)
	if err != nil {
		err = fmt.Errorf("ABI packing error in WNIBI.withdraw: %w", err)
		return
	}

	var evmObj *vm.EVM
	{
		unusedBigInt := big.NewInt(0)
		evmMsg := core.Message{
			To:               &erc20.Address,
			From:             sender.Eth,
			Nonce:            k.GetAccNonce(ctx, sender.Eth),
			Value:            unusedBigInt,
			GasLimit:         Erc20GasLimitExecute,
			GasPrice:         unusedBigInt,
			GasFeeCap:        unusedBigInt,
			GasTipCap:        unusedBigInt,
			Data:             contractInput,
			AccessList:       gethcore.AccessList{},
			BlobGasFeeCap:    &big.Int{},
			BlobHashes:       []gethcommon.Hash{},
			SkipNonceChecks:  false,
			SkipFromEOACheck: false,
		}
		evmObj = k.NewEVM(ctx, evmMsg, k.GetEVMConfig(ctx), nil /*tracer*/, stateDB)
	}

	if stateDB.GetCodeSize(erc20.Address) == 0 {
		err = fmt.Errorf("ConvertEvmToCoin: %s: canonical WNIBI %s", evm.ErrCanonicalWnibi, erc20.Hex())
		return
	}

	wnibiBalBefore, err := k.ERC20().BalanceOf(erc20.Address, sender.Eth, ctx, evmObj)
	if err != nil {
		err = fmt.Errorf("ConvertEvmToCoin: failed to query ERC20 balance: %w", err)
		return
	}
	if wnibiBalBefore.Cmp(withdrawWei.ToBig()) < 0 {
		err = fmt.Errorf(
			"ConvertEvmToCoin: insufficient funds to convert WNIBI into NIBI: WNIBI balance %s, withdrawamount %s", wnibiBalBefore, withdrawWei,
		)
		return
	}

	evmResp, err := k.CallContractWithInput(
		ctx,
		evmObj,
		sender.Eth,     /* fromAcc */
		&erc20.Address, /* contract */
		isTx,           /* commit */
		contractInput,
		Erc20GasLimitExecute,
		nil,
	)
	if err != nil {
		return resp, fmt.Errorf("failed to convert WNIBI to NIBI via WNIBI.withdraw: %w", err)
	} else if evmResp.Failed() {
		err = fmt.Errorf("failed to convert WNIBI to NIBI via WNIBI.withdraw: VmError: %s", evmResp.VmError)
		return resp, err
	}

	wnibiBalAfter, err := k.ERC20().BalanceOf(erc20.Address, sender.Eth, ctx, evmObj)
	if err != nil {
		err = fmt.Errorf("ConvertEvmToCoin: failed to query ERC20 balance: %w", err)
		return
	}
	if new(big.Int).Sub(wnibiBalBefore, wnibiBalAfter).Cmp(withdrawWei.ToBig()) != 0 {
		err = fmt.Errorf("WNIBI withdraw failed: withdraw amount %s, balBefore %s, balAfter %s", withdrawWei, wnibiBalBefore, wnibiBalAfter)
		return
	}

	// Commit the stateDB
	if err = stateDB.Commit(); err != nil {
		return nil, sdkioerrors.Wrap(err, "failed to commit stateDB")
	}

	withdrawnMicronibi := sdk.NewCoin(appconst.BondDenom, sdkmath.NewIntFromBigInt(
		evm.WeiToNative(withdrawWei.ToBig()),
	))
	if err := k.Bank.SendCoins(ctx, sender.Bech32, toAddrBech32,
		sdk.NewCoins(withdrawnMicronibi),
	); err != nil {
		return resp, err
	}

	_ = ctx.EventManager().EmitTypedEvent(&evm.EventConvertEvmToCoin{
		Sender:               sender.Bech32.String(),
		Erc20ContractAddress: erc20.Hex(),
		ToAddress:            toAddrBech32.String(),
		BankCoin:             withdrawnMicronibi,
		SenderEthAddr:        sender.Eth.Hex(),
	})

	return &evm.MsgConvertEvmToCoinResponse{}, nil
}
