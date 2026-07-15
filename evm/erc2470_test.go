package evm_test

// Copyright (c) 2026 Nibi, Inc.

import (
	"encoding/binary"
	"math/big"
	"testing"

	gethcommon "github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/evm"
	"github.com/NibiruChain/nibiru/v2/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testapp"
)

func TestERC2470Create2(t *testing.T) {
	deps := evmtest.NewTestDeps()
	require.NoError(t, testapp.FundAccount(
		deps.App.BankKeeper,
		deps.Ctx(),
		deps.Sender.NibiruAddr,
		sdk.NewCoins(sdk.NewCoin(evm.EVMBankDenom, sdk.NewInt(1_000_000_000))),
	))
	require.NoError(t, testapp.FundFeeCollector(
		deps.App.BankKeeper, deps.Ctx(), sdk.NewInt(1_000_000_000),
	))

	factory := gethcommon.HexToAddress(evm.ERC2470Address)
	require.Equal(t, 308, len(deps.NewStateDB().GetCode(factory)))

	// Creation code that installs runtime code returning 0x2a from any call.
	initCode := commonHex("600a600c600039600a6000f3602a60005260206000f3")
	var salt [32]byte
	binary.BigEndian.PutUint64(salt[24:], 2470)
	predicted := crypto.CreateAddress2(factory, salt, crypto.Keccak256(initCode))

	// ERC-2470's deploy(bytes,bytes32) selector and ABI arguments.
	selector := crypto.Keccak256([]byte("deploy(bytes,bytes32)"))[:4]
	input := append(selector, abiBytes32Bytes(initCode, salt)...)
	nonce := deps.EvmKeeper.GetAccNonce(deps.Ctx(), deps.Sender.EthAddr)
	tx, err := evmtest.NewEthTxMsgFromTxData(
		&deps, gethcore.LegacyTxType, input, nonce, &factory, big.NewInt(0), 500_000, nil,
	)
	require.NoError(t, err)
	resp, err := deps.App.EvmKeeper.EthereumTx(sdk.WrapSDKContext(deps.Ctx()), tx)
	require.NoError(t, err)
	require.Empty(t, resp.VmError)
	deps.Commit()

	require.Equal(t, predicted, gethcommon.BytesToAddress(resp.Ret))
	require.Equal(t, initCode[len(initCode)-10:], deps.NewStateDB().GetCode(predicted))
}

func commonHex(s string) []byte { return gethcommon.Hex2Bytes(s) }

func abiBytes32Bytes(data []byte, salt [32]byte) []byte {
	result := make([]byte, 96+((len(data)+31)/32)*32)
	copy(result[0:32], big.NewInt(64).FillBytes(make([]byte, 32)))
	copy(result[32:64], salt[:])
	new(big.Int).SetUint64(uint64(len(data))).FillBytes(result[64:96])
	copy(result[96:], data)
	return result
}
