// Copyright (c) 2023-2024 Nibi, Inc.
package keeper

import (
	"fmt"
	"math/big"

	sdkioerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
)

func (k *Keeper) convertCoinToEvmForWNIBI(
	ctx sdk.Context,
	msg *evm.MsgConvertCoinToEvm,
	senderBech32 sdk.AccAddress,
) (resp *evm.MsgConvertCoinToEvmResponse, err error) {
	balMicronibi := k.Bank.GetBalance(ctx, senderBech32, msg.BankCoin.Denom)
	if balMicronibi.IsLT(msg.BankCoin) {
		err = fmt.Errorf(
			"ConvertCoinToEvm: insufficient funds to convert NIBI into WNIBI, balance %s, msg.BankCoin %s", balMicronibi, msg.BankCoin,
		)
		return
	}

	var (
		// Check if WNIBI is well-defined (non-zero bytecode size).
		// If it is, convert NIBI -> WNIBI (Bank Coin -> ERC20)
		evmParams = k.GetParams(ctx)

		senderEthAddr = eth.NibiruAddrToEthAddr(senderBech32)

		// ERC20 contract taken to be WNIBI.sol
		erc20 = evmParams.CanonicalWnibi
	)

	// -------------------------------------------------------------------------
	// STEP 1: Sender deposits NIBI and receives WNIBI
	// -------------------------------------------------------------------------

	stateDB := k.Bank.StateDB
	if stateDB == nil {
		stateDB = k.NewStateDB(ctx, k.TxConfig(ctx, gethcommon.Hash{}))
	}
	defer func() {
		k.Bank.StateDB = nil
	}()

	if stateDB.GetCodeSize(erc20.Address) == 0 {
		err = fmt.Errorf("ConvertCoinToEvm: %s: canonical WNIBI %s", evm.ErrCanonicalWnibi, erc20.Hex())
		return
	}

	depositWei := evm.NativeToWei(
		msg.BankCoin.Amount.BigInt(),
	)

	contractInput, err := embeds.SmartContract_WNIBI.ABI.Pack(
		"deposit",
	)
	if err != nil {
		err = fmt.Errorf("ABI packing error in WNIBI.deposit: %w", err)
		return resp, err
	}

	var evmObj *vm.EVM
	{
		unusedBigInt := big.NewInt(0)
		evmMsg := core.Message{
			To:               &erc20.Address,
			From:             senderEthAddr,
			Nonce:            k.GetAccNonce(ctx, senderEthAddr),
			Value:            depositWei,
			GasLimit:         evm.Erc20GasLimitExecute,
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

	wnibiBalBefore, err := k.ERC20().BalanceOf(erc20.Address, senderEthAddr, ctx, evmObj)
	if err != nil {
		err = fmt.Errorf("ConvertCoinToEvm: failed to query ERC20 balance: %w", err)
		return
	}

	evmResp, err := k.CallContract(
		ctx,
		evmObj,
		senderEthAddr,  /* fromAcc */
		&erc20.Address, /* contract */
		contractInput,
		evm.Erc20GasLimitExecute,
		evm.COMMIT_ETH_TX, /*commit*/
		depositWei,
	)
	if err != nil {
		return resp, fmt.Errorf("failed to convert NIBI to WNIBI via WNIBI.deposit: %w", err)
	} else if evmResp.Failed() {
		err = fmt.Errorf("failed to convert NIBI to WNIBI via WNIBI.deposit: VmError: %s", evmResp.VmError)
		return resp, err
	}

	wnibiBalAfter, err := k.ERC20().BalanceOf(erc20.Address, senderEthAddr, ctx, evmObj)
	if err != nil {
		err = fmt.Errorf("ConvertCoinToEvm: failed to query ERC20 balance: %w", err)
		return
	}

	if new(big.Int).Sub(wnibiBalAfter, wnibiBalBefore).Cmp(depositWei) != 0 {
		err = fmt.Errorf("WNIBI deposit failed: deposit amount %s, balBefore %s, balAfter %s", depositWei, wnibiBalBefore, wnibiBalAfter)
		return
	}

	// -------------------------------------------------------------------------
	// STEP 2: Sender transfers WNIBI to intended recipient
	// -------------------------------------------------------------------------
	balIncrease, evmResp, err := k.ERC20().Transfer(
		erc20.Address,         /*erc20Contract*/
		senderEthAddr,         /*sender*/
		msg.ToEthAddr.Address, /*recipient*/
		depositWei,            /*amount*/
		ctx,                   /*ctx*/
		evmObj,                /*evmObj*/
	)
	if err != nil {
		return resp, fmt.Errorf("failed to convert WNIBI to NIBI: %w", err)
	} else if evmResp.Failed() {
		err = fmt.Errorf("failed to convert WNIBI to NIBI: VmError: %s", evmResp.VmError)
		return resp, err
	} else if balIncrease.Cmp(depositWei) != 0 {
		err = fmt.Errorf(
			"ConvertCoinToEvm: transfer of WNIBI succeeded but did not deliver the correct number of tokens: transfer amount %s, balance increase %s, senderHex %s, recipient %s",
			depositWei,
			balIncrease,
			senderEthAddr,
			msg.ToEthAddr.Hex(),
		)
		return
	}

	// Commit the stateDB to the BankKeeperExtension because we don't go through
	// ApplyEvmMsg at all in this tx.
	if err := stateDB.Commit(); err != nil {
		return nil, fmt.Errorf("%s: %w", evm.ErrStateDBCommit, err)
	}

	_ = ctx.EventManager().EmitTypedEvent(&evm.EventConvertCoinToEvm{
		Sender:               senderBech32.String(),
		Erc20ContractAddress: erc20.Hex(),
		ToEthAddr:            msg.ToEthAddr.Hex(),
		BankCoin:             msg.BankCoin,
	})

	return &evm.MsgConvertCoinToEvmResponse{}, nil
}

// Converts Bank Coins for FunToken mapping that was born from a coin
// (IsMadeFromCoin=true) into the ERC20 tokens. EVM module owns the ERC-20
// contract and can mint the ERC-20 tokens.
func (k Keeper) convertCoinToEvmBornCoin(
	ctx sdk.Context,
	sender sdk.AccAddress,
	recipient gethcommon.Address,
	coin sdk.Coin,
	funTokenMapping evm.FunToken,
) (*evm.MsgConvertCoinToEvmResponse, error) {
	// 1 | Send Bank Coins to the EVM module
	err := k.Bank.SendCoinsFromAccountToModule(ctx, sender, evm.ModuleName, sdk.NewCoins(coin))
	if err != nil {
		return nil, sdkioerrors.Wrap(err, "failed to send coins to module account")
	}

	// 2 | Mint ERC20 tokens to the recipient
	erc20Addr := funTokenMapping.Erc20Addr.Address
	contractInput, err := embeds.SmartContract_ERC20MinterWithMetadataUpdates.ABI.Pack("mint", recipient, coin.Amount.BigInt())
	if err != nil {
		return nil, err
	}
	unusedBigInt := big.NewInt(0)
	evmMsg := core.Message{
		To:               &erc20Addr,
		From:             evm.EVM_MODULE_ADDRESS,
		Nonce:            k.GetAccNonce(ctx, evm.EVM_MODULE_ADDRESS),
		Value:            unusedBigInt, // amount
		GasLimit:         evm.Erc20GasLimitExecute,
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
	txConfig := k.TxConfig(ctx, gethcommon.Hash{})
	stateDB := k.Bank.StateDB
	if stateDB == nil {
		stateDB = k.NewStateDB(ctx, txConfig)
	}
	defer func() {
		k.Bank.StateDB = nil
	}()

	evmObj := k.NewEVM(ctx, evmMsg, k.GetEVMConfig(ctx), nil /*tracer*/, stateDB)
	evmResp, err := k.CallContract(
		ctx,
		evmObj,
		evm.EVM_MODULE_ADDRESS,
		&erc20Addr,
		contractInput,
		evm.Erc20GasLimitExecute,
		evm.COMMIT_ETH_TX, /*commit*/
		nil,
	)
	if err != nil {
		return nil, err
	}

	if evmResp.Failed() {
		return nil,
			fmt.Errorf("failed to mint erc-20 tokens of contract %s", erc20Addr.String())
	}

	if err = stateDB.Commit(); err != nil {
		return nil, sdkioerrors.Wrap(err, evm.ErrStateDBCommit)
	}

	_ = ctx.EventManager().EmitTypedEvent(&evm.EventConvertCoinToEvm{
		Sender:               sender.String(),
		Erc20ContractAddress: erc20Addr.String(),
		ToEthAddr:            recipient.String(),
		BankCoin:             coin,
	})

	// Emit tx logs of Mint event
	err = ctx.EventManager().EmitTypedEvent(&evm.EventTxLog{Logs: evmResp.Logs})
	if err == nil {
		k.updateBlockBloom(ctx, evmResp, uint64(k.EvmState.BlockTxIndex.GetOr(ctx, 0)))
	}

	return &evm.MsgConvertCoinToEvmResponse{}, nil
}

// Converts a coin that was originally an ERC20 token, and that was converted to
// a bank coin, back to an ERC20 token. EVM module does not own the ERC-20
// contract and cannot mint the ERC-20 tokens. EVM module has escrowed tokens in
// the first conversion from ERC-20 to bank coin.
func (k Keeper) convertCoinToEvmBornERC20(
	ctx sdk.Context,
	sender sdk.AccAddress,
	recipient gethcommon.Address,
	coin sdk.Coin,
	funTokenMapping evm.FunToken,
) (*evm.MsgConvertCoinToEvmResponse, error) {
	// needs to run first to populate the StateDB on the BankKeeperExtension
	stateDB := k.Bank.StateDB
	if stateDB == nil {
		stateDB = k.NewStateDB(ctx, k.TxConfig(ctx, gethcommon.Hash{}))
	}
	defer func() {
		k.Bank.StateDB = nil
	}()

	erc20Addr := funTokenMapping.Erc20Addr.Address
	// 1 | Caller transfers Bank Coins to be converted to ERC20 tokens.
	if err := k.Bank.SendCoinsFromAccountToModule(
		ctx,
		sender,
		evm.ModuleName,
		sdk.NewCoins(coin),
	); err != nil {
		return nil, sdkioerrors.Wrap(err, "error sending Bank Coins to the EVM")
	}

	// 3 | In the FunToken ERC20 ΓåÆ BC conversion process that preceded this
	// TxMsg, the Bank Coins were minted. Consequently, to preserve an invariant
	// on the sum of the FunToken's bank and ERC20 supply, we burn the coins here
	// in the BC ΓåÆ ERC20 conversion.
	if err := k.Bank.BurnCoins(ctx, evm.ModuleName, sdk.NewCoins(coin)); err != nil {
		return nil, sdkioerrors.Wrap(err, "failed to burn coins")
	}

	// 2 | EVM sends ERC20 tokens to the "to" account.
	// This should never fail due to the EVM account lacking ERc20 fund because
	// the account must have sent the EVM module ERC20 tokens in the mapping
	// in order to create the coins originally.
	//
	// Said another way, if an asset is created as an ERC20 and some amount is
	// converted to its Bank Coin representation, a balance of the ERC20 is left
	// inside the EVM module account in order to convert the coins back to
	// ERC20s.
	contractInput, err := embeds.SmartContract_ERC20MinterWithMetadataUpdates.ABI.Pack("transfer", recipient, coin.Amount.BigInt())
	if err != nil {
		return nil, err
	}
	unusedBigInt := big.NewInt(0)
	evmMsg := core.Message{
		To:               &erc20Addr,
		From:             evm.EVM_MODULE_ADDRESS,
		Nonce:            k.GetAccNonce(ctx, evm.EVM_MODULE_ADDRESS),
		Value:            unusedBigInt, // amount
		GasLimit:         evm.Erc20GasLimitExecute,
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
	_, evmResp, err := k.ERC20().Transfer(
		erc20Addr,
		evm.EVM_MODULE_ADDRESS,
		recipient,
		coin.Amount.BigInt(),
		ctx,
		evmObj,
	)
	if err != nil {
		return nil, sdkioerrors.Wrap(err, "failed to transfer ERC-20 tokens")
	}

	// Commit the stateDB to the BankKeeperExtension because we don't go through
	// ApplyEvmMsg at all in this tx.
	if err := stateDB.Commit(); err != nil {
		return nil, sdkioerrors.Wrap(err, evm.ErrStateDBCommit)
	}

	// Emit event with the actual amount received
	_ = ctx.EventManager().EmitTypedEvent(&evm.EventConvertCoinToEvm{
		Sender:               sender.String(),
		Erc20ContractAddress: funTokenMapping.Erc20Addr.String(),
		ToEthAddr:            recipient.String(),
		BankCoin:             coin,
	})

	// Emit tx logs of Transfer event
	err = ctx.EventManager().EmitTypedEvent(&evm.EventTxLog{Logs: evmResp.Logs})
	if err == nil {
		k.updateBlockBloom(ctx, evmResp, uint64(k.EvmState.BlockTxIndex.GetOr(ctx, 0)))
	}

	return &evm.MsgConvertCoinToEvmResponse{}, nil
}
