package eip712_test

import (
	"encoding/hex"
	"strings"
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/eth/encoding"
	"github.com/NibiruChain/nibiru/eth/ethereum/eip712"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/stretchr/testify/require"
)

// Testing Constants
var (
	chainID = "cataclysm" + "-1"
	ctx     = client.Context{}.WithTxConfig(
		encoding.MakeConfig(app.ModuleBasics).TxConfig,
	)
)
var feePayerAddress = "nibi17xpfvakm2amg962yls6f84z3kell8c5ljcjw34"

type TestCaseStruct struct {
	txBuilder              client.TxBuilder
	expectedFeePayer       string
	expectedGas            uint64
	expectedFee            math.Int
	expectedMemo           string
	expectedMsg            string
	expectedSignatureBytes []byte
}

func TestBlankTxBuilder(t *testing.T) {
	txBuilder := ctx.TxConfig.NewTxBuilder()

	err := eip712.PreprocessLedgerTx(
		chainID,
		keyring.TypeLedger,
		txBuilder,
	)

	require.Error(t, err)
}

func TestNonLedgerTxBuilder(t *testing.T) {
	txBuilder := ctx.TxConfig.NewTxBuilder()

	err := eip712.PreprocessLedgerTx(
		chainID,
		keyring.TypeLocal,
		txBuilder,
	)

	require.NoError(t, err)
}

func TestInvalidChainId(t *testing.T) {
	txBuilder := ctx.TxConfig.NewTxBuilder()

	err := eip712.PreprocessLedgerTx(
		"invalid-chain-id",
		keyring.TypeLedger,
		txBuilder,
	)

	require.Error(t, err)
}

func createBasicTestCase(t *testing.T) TestCaseStruct {
	t.Helper()
	txBuilder := ctx.TxConfig.NewTxBuilder()

	feePayer, err := sdk.AccAddressFromBech32(feePayerAddress)
	require.NoError(t, err)

	txBuilder.SetFeePayer(feePayer)

	// Create signature unrelated to payload for testing
	signatureHex := strings.Repeat("01", 65)
	signatureBytes, err := hex.DecodeString(signatureHex)
	require.NoError(t, err)

	_, privKey := testutil.PrivKeyEth()
	sigsV2 := signing.SignatureV2{
		PubKey: privKey.PubKey(), // Use unrelated public key for testing
		Data: &signing.SingleSignatureData{
			SignMode:  signing.SignMode_SIGN_MODE_DIRECT,
			Signature: signatureBytes,
		},
		Sequence: 0,
	}

	err = txBuilder.SetSignatures(sigsV2)
	require.NoError(t, err)

	return TestCaseStruct{
		txBuilder:              txBuilder,
		expectedFeePayer:       feePayer.String(),
		expectedGas:            0,
		expectedFee:            math.NewInt(0),
		expectedMemo:           "",
		expectedMsg:            "",
		expectedSignatureBytes: signatureBytes,
	}
}

func createPopulatedTestCase(t *testing.T) TestCaseStruct {
	t.Helper()
	basicTestCase := createBasicTestCase(t)
	txBuilder := basicTestCase.txBuilder

	gasLimit := uint64(200000)
	memo := ""
	denom := baseDenom
	feeAmount := math.NewInt(2000)

	txBuilder.SetFeeAmount(sdk.NewCoins(
		sdk.NewCoin(
			denom,
			feeAmount,
		)))

	txBuilder.SetGasLimit(gasLimit)
	txBuilder.SetMemo(memo)

	msgSend := banktypes.MsgSend{
		FromAddress: feePayerAddress,
		ToAddress:   "nibi12luku6uxehhak02py4rcz65zu0swh7wjun6msa",
		Amount: sdk.NewCoins(
			sdk.NewCoin(
				baseDenom,
				math.NewInt(10000000),
			),
		),
	}

	err := txBuilder.SetMsgs(&msgSend)
	require.NoError(t, err)

	return TestCaseStruct{
		txBuilder:              txBuilder,
		expectedFeePayer:       basicTestCase.expectedFeePayer,
		expectedGas:            gasLimit,
		expectedFee:            feeAmount,
		expectedMemo:           memo,
		expectedMsg:            msgSend.String(),
		expectedSignatureBytes: basicTestCase.expectedSignatureBytes,
	}
}
