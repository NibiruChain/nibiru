package precompile_test

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/evm/precompile"
)

func TestP256Precompile_SuccessAndFailure(t *testing.T) {
	deps := evmtest.NewTestDeps()

	priv := staticP256Key()
	msgHash := sha256.Sum256([]byte("nibiru-p256-precompile"))

	r, s, err := ecdsa.Sign(rand.Reader, priv, msgHash[:])
	require.NoError(t, err)

	validInput := encodeP256Input(msgHash[:], r, s, priv.X, priv.Y)

	evmObj, _ := deps.NewEVM()
	resp, err := deps.EvmKeeper.CallContract(
		evmObj,
		deps.Sender.EthAddr,
		&precompile.PrecompileAddr_P256,
		validInput,
		50_000,
		evm.COMMIT_READONLY, /*commit*/
		nil,
	)
	require.NoError(t, err)
	require.Empty(t, resp.VmError)
	require.Len(t, resp.Ret, 32)
	require.Equal(t, byte(1), resp.Ret[len(resp.Ret)-1])

	// Flip a byte in the hash to force verification failure.
	badHash := msgHash
	badHash[0] ^= 0x01
	invalidInput := encodeP256Input(badHash[:], r, s, priv.X, priv.Y)

	evmObj, _ = deps.NewEVM()
	resp, err = deps.EvmKeeper.CallContract(
		evmObj,
		deps.Sender.EthAddr,
		&precompile.PrecompileAddr_P256,
		invalidInput,
		50_000,
		evm.COMMIT_READONLY, /*commit*/
		nil,
	)
	require.NoError(t, err)
	require.Len(t, resp.Ret, 32)
	require.Equal(t, byte(0), resp.Ret[len(resp.Ret)-1])
}

func TestP256Precompile_InvalidLength(t *testing.T) {
	deps := evmtest.NewTestDeps()
	evmObj, _ := deps.NewEVM()

	_, err := deps.EvmKeeper.CallContract(
		evmObj,
		deps.Sender.EthAddr,
		&precompile.PrecompileAddr_P256,
		[]byte{0x01, 0x02, 0x03},
		50_000,
		evm.COMMIT_READONLY, /*commit*/
		nil,
	)
	require.ErrorContains(t, err, "invalid input length")
}

func encodeP256Input(
	hash []byte,
	r, s, qx, qy *big.Int,
) []byte {
	if len(hash) != 32 {
		panic("hash must be 32 bytes")
	}

	var buf bytes.Buffer
	buf.Write(hash)
	for _, v := range []*big.Int{r, s, qx, qy} {
		buf.Write(leftPad(v.Bytes(), 32))
	}
	return buf.Bytes()
}

func leftPad(bz []byte, size int) []byte {
	if len(bz) > size {
		panic("value does not fit in fixed buffer")
	}
	if len(bz) == size {
		return bz
	}
	padded := make([]byte, size)
	copy(padded[size-len(bz):], bz)
	return padded
}

// staticP256Key builds a deterministic private key for repeatable test vectors.
func staticP256Key() *ecdsa.PrivateKey {
	priv := new(ecdsa.PrivateKey)
	priv.Curve = elliptic.P256()
	priv.D = big.NewInt(1) // simple, deterministic scalar
	priv.X, priv.Y = priv.ScalarBaseMult(priv.D.Bytes())
	return priv
}
