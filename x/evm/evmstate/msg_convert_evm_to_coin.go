// Copyright (c) 2023-2024 Nibi, Inc.
package evmstate

import (
	"fmt"
	"math/big"

	sdkioerrors "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/tracing"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/holiman/uint256"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
)

// convertEvmToCoinForCoinOriginated is part of the
// "eth.evm.v1.MsgConvertEvmToCoin" tx. This function handles conversion of ERC20
// tokens that were originally bank coins back into coin form. The EVM module
// owns the ERC20 contract and will burn the tokens
func (k Keeper) convertEvmToCoinForCoinOriginated(
	sdb *SDB,
	sender evm.Addrs,
	toAddress sdk.AccAddress,
	erc20Addr gethcommon.Address,
	amount *uint256.Int,
	bankDenom string,
) error {
	bankCoins := sdk.NewCoins(sdk.NewCoin(
		bankDenom, sdkmath.NewIntFromBigInt(amount.ToBig()),
	))

	// 1 | Burn the ERC20 tokens from the sender's account
	contractInput, err := embeds.SmartContract_ERC20MinterWithMetadataUpdates.ABI.Pack(
		"burnFromAuthority",
		sender.Eth /*from: address where we burn the token balance from*/, amount.ToBig(),
	)
	if err != nil {
		return err
	}

	unusedBigInt := big.NewInt(0)
	evmMsg := core.Message{
		To:               &erc20Addr,
		From:             evm.EVM_MODULE_ADDRESS,
		Nonce:            k.GetAccNonce(sdb.Ctx(), evm.EVM_MODULE_ADDRESS),
		Value:            unusedBigInt,
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

	evmObj := k.NewEVM(sdb.Ctx(), evmMsg, k.GetEVMConfig(sdb.Ctx()), nil /*tracer*/, sdb)
	evmResp, err := k.CallContract(
		evmObj,
		evm.EVM_MODULE_ADDRESS,
		&erc20Addr,
		contractInput,
		evm.Erc20GasLimitExecute,
		evm.COMMIT_ETH_TX, /*commit*/
		nil,
	)
	if err != nil {
		return err
	}
	if evmResp.Failed() {
		return fmt.Errorf("failed to burn ERC20 tokens: %s", evmResp.VmError)
	}

	// 2 | Send Bank Coins from the EVM module to the recipient
	err = k.Bank.SendCoinsFromModuleToAccount(sdb.Ctx(), evm.ModuleName, toAddress, bankCoins)
	if err != nil {
		return sdkioerrors.Wrap(err, "failed to send coins from module to account")
	}

	// Emit event
	_ = sdb.Ctx().EventManager().EmitTypedEvent(&evm.EventConvertEvmToCoin{
		Sender:               sender.Bech32.String(),
		Erc20ContractAddress: erc20Addr.Hex(),
		ToAddress:            toAddress.String(),
		BankCoin:             bankCoins[0],
		SenderEthAddr:        sender.Eth.Hex(),
		EvmLogs:              evm.LogsToLogLite(evmResp.Logs),
	})

	return nil
}

// convertEvmToCoinForERC20Originated handles conversion of ERC20 tokens that
// were originally ERC20. The EVM module doesn't own the ERC20 contract, so it
// transfers tokens to itself and mints bank coins
func (k Keeper) convertEvmToCoinForERC20Originated(
	sdb *SDB,
	sender evm.Addrs,
	toAddress sdk.AccAddress,
	erc20Addr gethcommon.Address,
	amount *uint256.Int,
	bankDenom string,
) error {
	// 1 | Transfer ERC20 tokens from sender to EVM module
	contractInput, err := embeds.SmartContract_ERC20MinterWithMetadataUpdates.ABI.Pack("transfer", evm.EVM_MODULE_ADDRESS, amount.ToBig())
	if err != nil {
		return err
	}

	var evmObj *vm.EVM
	{
		unusedBigInt := big.NewInt(0)
		evmMsg := core.Message{
			From:             sender.Eth,
			To:               &evm.EVM_MODULE_ADDRESS,
			Nonce:            k.GetAccNonce(sdb.Ctx(), sender.Eth),
			Value:            unusedBigInt,
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
		evmObj = k.NewEVM(sdb.Ctx(), evmMsg, k.GetEVMConfig(sdb.Ctx()), nil /*tracer*/, sdb)
	}

	balIncrease, evmResp, err := k.ERC20().Transfer(
		erc20Addr,              /*erc20Contract gethcommon.Address*/
		sender.Eth,             /*sender*/
		evm.EVM_MODULE_ADDRESS, /*recipient*/
		amount.ToBig(),         /*amount*/
		sdb.Ctx(),
		evmObj,
	)
	if err != nil {
		return sdkioerrors.Wrap(err, "failed to transfer ERC20 tokens")
	}
	if evmResp.Failed() {
		return fmt.Errorf("failed to transfer ERC20 tokens: %s", evmResp.VmError)
	}

	bankCoin := sdk.NewCoin(bankDenom, sdkmath.NewIntFromBigInt(balIncrease))

	// 2 | Mint Bank Coins to the recipient
	err = k.Bank.MintCoins(sdb.Ctx(), evm.ModuleName, sdk.NewCoins(bankCoin))
	if err != nil {
		return sdkioerrors.Wrap(err, "failed to mint coins")
	}

	// 3 | Send the minted coins to the recipient
	err = k.Bank.SendCoinsFromModuleToAccount(sdb.Ctx(), evm.ModuleName, toAddress, sdk.NewCoins(bankCoin))
	if err != nil {
		return sdkioerrors.Wrap(err, "failed to send coins to recipient")
	}

	// Emit event
	_ = sdb.Ctx().EventManager().EmitTypedEvent(&evm.EventConvertEvmToCoin{
		Sender:               sender.Bech32.String(),
		Erc20ContractAddress: erc20Addr.Hex(),
		ToAddress:            toAddress.String(),
		BankCoin:             bankCoin,
		SenderEthAddr:        sender.Eth.Hex(),
		EvmLogs:              evm.LogsToLogLite(evmResp.Logs),
	})

	return nil
}

// NOTE: This function is unsafe and assumes all arguments are valid. It should
// never be called directly.
// It can only be called from the logic:
//   - (1) Inside of "eth.evm.v1.MsgConvertEvmToCoin"
//   - (2) Or inside the of FunToken precompile "sendToBank"
func (k Keeper) convertEvmToCoinForWNIBI(
	sdb *SDB,
	erc20 eth.EIP55Addr,
	sender evm.Addrs,
	toAddrBech32 sdk.AccAddress,
	amount *uint256.Int,
) (withdrawWei *uint256.Int, err error) {
	// Amount validation occured at the beginning of [Keeper.ConvertEvmToCoin]
	withdrawWei = amount

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

	unusedBigInt := big.NewInt(0)
	evmMsg := core.Message{
		To:               &erc20.Address,
		From:             sender.Eth,
		Nonce:            k.GetAccNonce(sdb.Ctx(), sender.Eth),
		Value:            unusedBigInt,
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
	evmObj := k.NewEVM(sdb.Ctx(), evmMsg, k.GetEVMConfig(sdb.Ctx()), nil /*tracer*/, sdb)

	if sdb.GetCodeSize(erc20.Address) == 0 {
		err = fmt.Errorf("ConvertEvmToCoin: %s: canonical WNIBI %s", evm.ErrCanonicalWnibi, erc20.Hex())
		return
	}

	wnibiBalBefore, err := k.ERC20().BalanceOf(erc20.Address, sender.Eth, sdb.Ctx(), evmObj)
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

	evmResp, err := k.CallContract(
		evmObj,
		sender.Eth,               /* fromAcc */
		&erc20.Address,           /* contract */
		contractInput,            /* contractInput */
		evm.Erc20GasLimitExecute, /* gasLimit */
		evm.COMMIT_ETH_TX,        /*commit*/
		nil,                      /* weiValue */
	)
	if err != nil {
		return withdrawWei, fmt.Errorf("failed to convert WNIBI to NIBI via WNIBI.withdraw: %w", err)
	} else if evmResp.Failed() {
		err = fmt.Errorf("failed to convert WNIBI to NIBI via WNIBI.withdraw: VmError: %s", evmResp.VmError)
		return withdrawWei, err
	}

	wnibiBalAfter, err := k.ERC20().BalanceOf(erc20.Address, sender.Eth, sdb.Ctx(), evmObj)
	if err != nil {
		err = fmt.Errorf("ConvertEvmToCoin: failed to query ERC20 balance: %w", err)
		return
	}
	if new(big.Int).Sub(wnibiBalBefore, wnibiBalAfter).Cmp(withdrawWei.ToBig()) != 0 {
		err = fmt.Errorf("WNIBI withdraw failed: { withdraw amount: %s, balBefore: %s, balAfter: %s }", withdrawWei, wnibiBalBefore, wnibiBalAfter)
		return
	}

	// Transfer NIBI to the recipient
	if bal := sdb.GetBalance(sender.Eth); bal.Cmp(withdrawWei) < 0 {
		// This error should be impossible, assuming that the WNIBI contract
		// because the amount subtracted in WNIBI is the amount the sender gains
		// in NIBI. We include this check to keep the function defensive.
		err = fmt.Errorf("sender has insufficient funds in NIBI { balance: %s, transfer amount: %s }", bal, withdrawWei)
		return
	}
	sdb.SubBalance(sender.Eth, withdrawWei, tracing.BalanceChangeTransfer)
	sdb.AddBalance(eth.NibiruAddrToEthAddr(toAddrBech32), withdrawWei, tracing.BalanceChangeTransfer)

	_ = sdb.Ctx().EventManager().EmitTypedEvent(&evm.EventConvertEvmToCoin{
		Sender:               sender.Bech32.String(),
		Erc20ContractAddress: erc20.Hex(),
		ToAddress:            toAddrBech32.String(),
		BankCoin: sdk.NewCoin(appconst.DENOM_UNIBI, sdkmath.NewIntFromBigInt(
			evm.WeiToNative(withdrawWei.ToBig()),
		)),
		SenderEthAddr: sender.Eth.Hex(),
		EvmLogs:       evm.LogsToLogLite(evmResp.Logs),
	})

	return withdrawWei, nil
}
