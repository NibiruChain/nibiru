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
	// Check if WNIBI is well-defined.
	// If it is, convert NIBI (Bank Coin) into WNIBI (ERC20)
	evmParams := k.GetParams(ctx)

	// 1 | Sender deposits NIBI and receives WNIBI
	stateDB := k.Bank.StateDB
	if stateDB == nil {
		stateDB = k.NewStateDB(ctx, k.TxConfig(ctx, gethcommon.Hash{}))
	}
	defer func() {
		k.Bank.StateDB = nil
	}()

	// isTx: value to use for commit in any EVM calls
	isTx := true

	senderEthAddr := eth.NibiruAddrToEthAddr(senderBech32)
	erc20 := evmParams.CanonicalWnibi
	if stateDB.GetCodeSize(erc20.Address) == 0 {
		err = fmt.Errorf("ConvertEvmToCoin: the canonical WNIBI address in state is a not a smart contract: canonical WNIBI %s ", erc20.Hex())
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

	// TODO: UD-DEBUG: deploy WNIBI in test and make sure this works.
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

	fmt.Println("TODO: UD-DEBUG: ")
	fmt.Printf("wnibiBalBefore: %v\n", wnibiBalBefore)
	fmt.Printf("wnibiBalAfter: %v\n", wnibiBalAfter)
	fmt.Printf("senderEthAddr: %v\n", senderEthAddr)
	fmt.Printf("evmResp: %v\n", evmResp)
	fmt.Printf("erc20.Hex(): %v\n", erc20.Hex())
	fmt.Printf("depositWei: %s\n", depositWei)

	if new(big.Int).Sub(wnibiBalBefore, wnibiBalAfter).Cmp(depositWei) != 0 {
		err = fmt.Errorf("WNIBI deposit failed: deposit amount %s, balBefore %s, balAfter %s", depositWei, wnibiBalBefore, wnibiBalAfter)
		return
	}

	panic("WOOHOO!")

	// 2 | Sender tranfers WNIBI to intended recipient

	// evmParams.CanonicalWnibi.He

}
