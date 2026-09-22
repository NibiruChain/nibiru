package impl

import (
	"encoding/hex"
	"errors"
	"math/big"
	"strings"
	"testing"

	cmttypes "github.com/cometbft/cometbft/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/eth"
)

const (
	realCometTxHash  = "624CAA0738E95CE04C22D4D012FF67AD86521FB93B2F64CE9967A8FE31EAB3A1"
	realCometTxHash2 = "C073755909DB4D003DC4FC2524229138E6F97E3E3A6827712BE4AD1F09768664"
	realEVMHash      = "0x66ef47cf9bda9bfb9c5e465a44e9a825bf1d8f013ed9b4abaf6960c02e9d85fc"
	realEVMHash2     = "0x7c20ff9dcd9d22f8e93e1a8f3a875c2a2582a2035ab42e48b3008bd0fe6ca4e2"
)

func TestClassifyTxHash(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantType  TxHash
		canonical string
	}{
		{"real CometBFT hash", realCometTxHash, CometBFTTxHash{}, realCometTxHash},
		{"lowercase CometBFT hash", strings.ToLower(realCometTxHash), CometBFTTxHash{}, realCometTxHash},
		{"another real CometBFT hash", realCometTxHash2, CometBFTTxHash{}, realCometTxHash2},
		{"real EVM hash", realEVMHash, EVMTxHash{}, realEVMHash},
		{"another real EVM hash", realEVMHash2, EVMTxHash{}, realEVMHash2},
		{"uppercase EVM prefix and body", "0X" + strings.ToUpper(strings.TrimPrefix(realEVMHash, "0x")), EVMTxHash{}, realEVMHash},
		{"unprefixed EVM-shaped input follows CometBFT grammar", strings.TrimPrefix(realEVMHash, "0x"), CometBFTTxHash{}, strings.ToUpper(strings.TrimPrefix(realEVMHash, "0x"))},
		{"zero CometBFT hash", strings.Repeat("0", txHashHexLength), CometBFTTxHash{}, strings.Repeat("0", txHashHexLength)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ClassifyTxHash(tt.input)
			require.NoError(t, err)
			require.IsType(t, tt.wantType, got)
			require.Equal(t, tt.canonical, got.Canonical())

			expectedBytes, err := hex.DecodeString(strings.TrimPrefix(tt.canonical, "0x"))
			require.NoError(t, err)
			switch hash := got.(type) {
			case CometBFTTxHash:
				require.Equal(t, expectedBytes, hash.Bytes())
			case EVMTxHash:
				require.Equal(t, expectedBytes, hash.Hash().Bytes())
			}
		})
	}
}

func TestClassifyTxHashUsesCometBFTHashFormatting(t *testing.T) {
	tx := cmttypes.Tx([]byte("a representative CometBFT transaction"))
	input := eth.TmTxHashToString(tx.Hash())

	got, err := ClassifyTxHash(input)
	require.NoError(t, err)
	cometHash, ok := got.(CometBFTTxHash)
	require.True(t, ok)
	require.Equal(t, tx.Hash(), cometHash.Bytes())
	require.Equal(t, input, cometHash.Canonical())
}

func TestClassifyTxHashUsesGethTransactionHash(t *testing.T) {
	to := gethcommon.HexToAddress("0x1111111111111111111111111111111111111111")
	tx := gethcore.NewTx(&gethcore.LegacyTx{Nonce: 7, GasPrice: big.NewInt(1), Gas: 21_000, To: &to, Value: big.NewInt(42)})
	got, err := ClassifyTxHash(tx.Hash().Hex())
	require.NoError(t, err)
	evmHash, ok := got.(EVMTxHash)
	require.True(t, ok)
	require.Equal(t, tx.Hash(), evmHash.Hash())
}

func TestClassifyTxHashInvalid(t *testing.T) {
	tests := []string{
		"", " " + realCometTxHash, realCometTxHash + "\n", realCometTxHash[:txHashHexLength-1], realCometTxHash + "0",
		realCometTxHash[:10] + "g" + realCometTxHash[11:], realEVMHash[:10] + "g" + realEVMHash[11:], "0x", "0y" + realCometTxHash,
	}
	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			got, err := ClassifyTxHash(input)
			require.Error(t, err)
			require.Nil(t, got)
			require.True(t, errors.Is(err, ErrInvalidTxHash))
			require.Contains(t, err.Error(), txHashFormat)
		})
	}
}

func TestTxHashValuesAreImmutable(t *testing.T) {
	parsed, err := ClassifyTxHash(realCometTxHash)
	require.NoError(t, err)
	cometHash := parsed.(CometBFTTxHash)
	bytes := cometHash.Bytes()
	bytes[0] = 0
	require.Equal(t, realCometTxHash, cometHash.Canonical())

	parsed, err = ClassifyTxHash(realEVMHash)
	require.NoError(t, err)
	evmHash := parsed.(EVMTxHash)
	gethHash := evmHash.Hash()
	gethHash[0] = 0
	require.Equal(t, realEVMHash, evmHash.Canonical())
}
