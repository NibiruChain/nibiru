// Copyright (c) 2023-2024 Nibi, Inc.
package keeper

import (
	"fmt"
	"math/big"

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
			"ConvertEvmToCoin: insufficient funds to send WNIBI, balance %s, msg.BankCoin %s", balMicronibi, msg.BankCoin,
		)
		return
	}

	var (
		// Check if WNIBI is well-defined (non-zero bytecode size).
		// If it is, convert NIBI -> WNIBI (Bank Coin -> ERC20)
		evmParams = k.GetParams(ctx)

		// isTx: value to use for commit in any EVM calls
		isTx = true

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
		err = fmt.Errorf("ABI packing error in WNIBI.despoit: %w", err)
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

	wnibiBalBefore, err := k.ERC20().BalanceOf(erc20.Address, senderEthAddr, ctx, evmObj)
	if err != nil {
		err = fmt.Errorf("ConvertEvmToCoin: failed to query ERC20 balance: %w", err)
		return
	}

	evmResp, err := k.CallContractWithInput(
		ctx,
		evmObj,
		senderEthAddr,  /* fromAcc */
		&erc20.Address, /* contract */
		isTx,           /* commit */
		contractInput,
		Erc20GasLimitExecute,
		depositWei,
	)
	if err != nil {
		return resp, fmt.Errorf("failed to convert WNIBI to NIBI: %w", err)
	} else if evmResp.Failed() {
		err = fmt.Errorf("failed to convert WNIBI to NIBI: VmError: %s", evmResp.VmError)
		return resp, err
	}

	wnibiBalAfter, err := k.ERC20().BalanceOf(erc20.Address, senderEthAddr, ctx, evmObj)
	if err != nil {
		err = fmt.Errorf("ConvertEvmToCoin: failed to query ERC20 balance: %w", err)
		return
	}

	if new(big.Int).Sub(wnibiBalAfter, wnibiBalBefore).Cmp(depositWei) != 0 {
		err = fmt.Errorf("WNIBI deposit failed: deposit amount %s, balBefore %s, balAfter %s", depositWei, wnibiBalBefore, wnibiBalAfter)
		return
	}

	// -------------------------------------------------------------------------
	// STEP 2: Sender tranfers WNIBI to intended recipient
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
			"ConvertCoinToEvm: transfer of WNIBI succeeded but did not deliver the correct number of tokens: transfer amount %s, balance increase %s, senderHex %s, receipient %s",
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
		return nil, fmt.Errorf("failed to commit stateDB: %w", err)
	}

	_ = ctx.EventManager().EmitTypedEvent(&evm.EventConvertCoinToEvm{
		Sender:               senderBech32.String(),
		Erc20ContractAddress: erc20.Hex(),
		ToEthAddr:            msg.ToEthAddr.Hex(),
		BankCoin:             msg.BankCoin,
	})

	return &evm.MsgConvertCoinToEvmResponse{}, nil
}
