package evm_test

import (
	"math/big"
	"testing"

	sdkmath "cosmossdk.io/math"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/nutil"
	"github.com/NibiruChain/nibiru/v2/x/sudo"
)

func TestZeroGasClassificationUsesAllowlist(t *testing.T) {
	deps := evmtest.NewTestDeps()
	allowlisted := gethcommon.HexToAddress("0x1111111111111111111111111111111111111111")
	other := gethcommon.HexToAddress("0x2222222222222222222222222222222222222222")
	deps.App.SudoKeeper.ZeroGasActors.Set(deps.Ctx(), sudo.ZeroGasActors{
		AlwaysZeroGasContracts: []string{allowlisted.Hex()},
	})

	tx := newSignedDynamicFeeTx(t, &deps, &allowlisted, big.NewInt(1), big.NewInt(1), big.NewInt(2))
	isZeroGas, txData, err := evm.IsZeroGasMsgEthereumTx(deps.Ctx(), deps.App.SudoKeeper, tx)
	require.NoError(t, err)
	require.True(t, isZeroGas)
	require.True(t, evm.IsZeroGasTxData(deps.Ctx(), deps.App.SudoKeeper, txData))

	zeroValue := (*hexutil.Big)(big.NewInt(0))
	require.True(t, evm.IsZeroGasJsonTxArgs(deps.Ctx(), deps.App.SudoKeeper, evm.JsonTxArgs{
		To:    &allowlisted,
		Value: zeroValue,
	}))

	nonzeroValue := (*hexutil.Big)(big.NewInt(1))
	require.True(t, evm.IsZeroGasJsonTxArgs(deps.Ctx(), deps.App.SudoKeeper, evm.JsonTxArgs{
		To:    &allowlisted,
		Value: nonzeroValue,
	}))
	require.False(t, evm.IsZeroGasJsonTxArgs(deps.Ctx(), deps.App.SudoKeeper, evm.JsonTxArgs{
		To:    &other,
		Value: zeroValue,
	}))
	require.False(t, evm.IsZeroGasJsonTxArgs(deps.Ctx(), deps.App.SudoKeeper, evm.JsonTxArgs{
		Value: zeroValue,
	}))
}

func TestZeroGasSignedClassificationDoesNotNormalizeRawTx(t *testing.T) {
	deps := evmtest.NewTestDeps()
	to := gethcommon.HexToAddress("0x3333333333333333333333333333333333333333")
	deps.App.SudoKeeper.ZeroGasActors.Set(deps.Ctx(), sudo.ZeroGasActors{
		AlwaysZeroGasContracts: []string{to.Hex()},
	})

	tx := newSignedDynamicFeeTx(t, &deps, &to, big.NewInt(0), big.NewInt(1), big.NewInt(2))
	require.ErrorContains(t, tx.ValidateBasic(), "max priority fee per gas higher than max fee per gas")

	isZeroGas, txData, err := evm.IsZeroGasMsgEthereumTx(deps.Ctx(), deps.App.SudoKeeper, tx)
	require.NoError(t, err)
	require.True(t, isZeroGas)
	require.Equal(t, int64(1), txData.GetGasFeeCapWei().Int64())
	require.Equal(t, int64(2), txData.GetGasTipCapWei().Int64())
	require.Equal(t, int64(1), tx.AsTransaction().GasFeeCap().Int64())
	require.Equal(t, int64(2), tx.AsTransaction().GasTipCap().Int64())
}

func TestZeroGasClassificationNilInputs(t *testing.T) {
	deps := evmtest.NewTestDeps()
	isZeroGas, txData, err := evm.IsZeroGasMsgEthereumTx(deps.Ctx(), deps.App.SudoKeeper, nil)
	require.ErrorContains(t, err, "nil MsgEthereumTx")
	require.False(t, isZeroGas)
	require.Nil(t, txData)

	require.False(t, evm.IsZeroGasTxData(deps.Ctx(), deps.App.SudoKeeper, nil))
	require.False(t, evm.IsZeroGasTxData(deps.Ctx(), nil, &evm.DynamicFeeTx{}))
	require.False(t, evm.IsZeroGasJsonTxArgs(deps.Ctx(), nil, evm.JsonTxArgs{}))
}

func TestNormalizeZeroGasMessageZerosOnlyFeePriceFields(t *testing.T) {
	from := gethcommon.HexToAddress("0x4444444444444444444444444444444444444444")
	to := gethcommon.HexToAddress("0x5555555555555555555555555555555555555555")
	msg := core.Message{
		From:      from,
		To:        &to,
		Nonce:     7,
		Value:     big.NewInt(123),
		GasLimit:  456,
		GasPrice:  big.NewInt(1),
		GasFeeCap: big.NewInt(2),
		GasTipCap: big.NewInt(3),
		Data:      []byte{0xab, 0xcd},
	}

	got := evm.NormalizeZeroGasMessage(msg)
	require.Equal(t, 0, got.GasPrice.Sign())
	require.Equal(t, 0, got.GasFeeCap.Sign())
	require.Equal(t, 0, got.GasTipCap.Sign())
	require.Equal(t, from, got.From)
	require.Equal(t, &to, got.To)
	require.Equal(t, uint64(7), got.Nonce)
	require.Equal(t, big.NewInt(123), got.Value)
	require.Equal(t, uint64(456), got.GasLimit)
	require.Equal(t, []byte{0xab, 0xcd}, got.Data)

	require.Equal(t, int64(1), msg.GasPrice.Int64())
	require.Equal(t, int64(2), msg.GasFeeCap.Int64())
	require.Equal(t, int64(3), msg.GasTipCap.Int64())
}

func TestValidateZeroGasTxDataSkipsFeeChecksAndKeepsNonFeeChecks(t *testing.T) {
	deps := evmtest.NewTestDeps()
	to := gethcommon.HexToAddress("0x6666666666666666666666666666666666666666")

	tx := newSignedDynamicFeeTx(t, &deps, &to, big.NewInt(0), big.NewInt(1), big.NewInt(2))
	txData, err := evm.UnpackTxData(tx.Data)
	require.NoError(t, err)
	require.ErrorContains(t, txData.Validate(), "max priority fee per gas higher than max fee per gas")
	require.NoError(t, evm.ValidateZeroGasTxData(txData))

	creation := newSignedDynamicFeeTx(t, &deps, nil, big.NewInt(0), big.NewInt(1), big.NewInt(1))
	creationData, err := evm.UnpackTxData(creation.Data)
	require.NoError(t, err)
	require.ErrorContains(t, evm.ValidateZeroGasTxData(creationData), "zero-gas tx must not be contract creation")

	valueTransfer := newSignedDynamicFeeTx(t, &deps, &to, big.NewInt(1), big.NewInt(1), big.NewInt(1))
	valueTransferData, err := evm.UnpackTxData(valueTransfer.Data)
	require.NoError(t, err)
	require.NoError(t, evm.ValidateZeroGasTxData(valueTransferData))

	zeroValue := newSignedDynamicFeeTx(t, &deps, &to, big.NewInt(0), big.NewInt(1), big.NewInt(1))
	zeroValueData, err := evm.UnpackTxData(zeroValue.Data)
	require.NoError(t, err)
	require.NoError(t, evm.ValidateZeroGasTxData(zeroValueData))

	chainID := sdkmath.NewIntFromBigInt(deps.App.EvmKeeper.EthChainID(deps.Ctx()))
	nilValue := &evm.DynamicFeeTx{
		ChainID:   &chainID,
		Nonce:     0,
		GasLimit:  21_000,
		GasFeeCap: nutil.Ptr(sdkmath.NewInt(1)),
		GasTipCap: nutil.Ptr(sdkmath.NewInt(1)),
		To:        to.Hex(),
	}
	require.NoError(t, evm.ValidateZeroGasTxData(nilValue))

	negativeAmount := sdkmath.NewInt(-1)
	negativeValue := &evm.DynamicFeeTx{
		ChainID:   &chainID,
		Nonce:     0,
		GasLimit:  21_000,
		GasFeeCap: nutil.Ptr(sdkmath.NewInt(1)),
		GasTipCap: nutil.Ptr(sdkmath.NewInt(1)),
		To:        to.Hex(),
		Amount:    &negativeAmount,
	}
	require.ErrorContains(t, evm.ValidateZeroGasTxData(negativeValue), "amount cannot be negative")

	missingChainID := &evm.DynamicFeeTx{
		Nonce:     0,
		GasLimit:  21_000,
		GasFeeCap: nutil.Ptr(sdkmath.NewInt(1)),
		GasTipCap: nutil.Ptr(sdkmath.NewInt(1)),
		To:        to.Hex(),
		Amount:    nutil.Ptr(sdkmath.NewInt(0)),
	}
	require.ErrorContains(t, evm.ValidateZeroGasTxData(missingChainID), "chain ID must be derived from TxData txs")
}

func TestWithZeroGasMetaAndDefaultPriority(t *testing.T) {
	deps := evmtest.NewTestDeps()
	require.False(t, evm.IsZeroGasEthTx(deps.Ctx()))

	ctx := evm.WithZeroGasMeta(deps.Ctx())
	require.True(t, evm.IsZeroGasEthTx(ctx))
	require.NotNil(t, evm.GetZeroGasMeta(ctx))
	require.Equal(t, int64(0), evm.DefaultZeroGasTxPriority)
}

func newSignedDynamicFeeTx(
	t *testing.T,
	deps *evmtest.TestDeps,
	to *gethcommon.Address,
	value *big.Int,
	gasFeeCap *big.Int,
	gasTipCap *big.Int,
) *evm.MsgEthereumTx {
	t.Helper()
	tx, err := evmtest.NewEthTxMsgFromTxData(
		deps,
		gethcore.DynamicFeeTxType,
		nil,
		deps.EvmKeeper.GetAccNonce(deps.Ctx(), deps.Sender.EthAddr),
		to,
		value,
		50_000,
		nil,
	)
	require.NoError(t, err)

	txData, err := evm.UnpackTxData(tx.Data)
	require.NoError(t, err)
	dynamicTx, ok := txData.(*evm.DynamicFeeTx)
	require.True(t, ok)
	dynamicTx.GasFeeCap = nutil.Ptr(sdkmath.NewIntFromBigInt(gasFeeCap))
	dynamicTx.GasTipCap = nutil.Ptr(sdkmath.NewIntFromBigInt(gasTipCap))
	tx.Data, err = evm.PackTxData(dynamicTx)
	require.NoError(t, err)
	require.NoError(t, tx.Sign(deps.GethSigner(), deps.Sender.KeyringSigner))
	return tx
}
