package precompile

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"fmt"
	"math/big"

	"github.com/NibiruChain/nibiru/v2/app/keepers"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

var _ vm.PrecompiledContract = (*precompileP256)(nil)

// Precompile address for RIP-7212 style P-256 verification.
//
// Input: 160 bytes -> hash (32) | r (32) | s (32) | qx (32) | qy (32)
// Output: 32 bytes where the final byte is 1 on success and 0 on failure.
var PrecompileAddr_P256 = gethcommon.HexToAddress("0x0000000000000000000000000000000000000100")

const (
	p256InputLen  = 32 * 5
	p256OutputLen = 32
	// RIP-7212 suggests ~3400 gas per verification; tune as needed.
	p256VerifyGas = 3400
)

type precompileP256 struct{}

func PrecompileP256(_ keepers.PublicKeepers) NibiruCustomPrecompile {
	return precompileP256{}
}

func (p precompileP256) Address() gethcommon.Address {
	return PrecompileAddr_P256
}

func (p precompileP256) RequiredGas(_ []byte) uint64 {
	return p256VerifyGas
}

func (p precompileP256) Run(
	_ *vm.EVM,
	_ gethcommon.Address,
	contract *vm.Contract,
	_ bool,
	_ bool,
) (bz []byte, err error) {
	defer func() {
		err = ErrPrecompileRun(err, p)
	}()

	if len(contract.Input) != p256InputLen {
		return nil, fmt.Errorf("invalid input length: want %d, got %d", p256InputLen, len(contract.Input))
	}

	hash := contract.Input[:32]
	r := new(big.Int).SetBytes(contract.Input[32:64])
	s := new(big.Int).SetBytes(contract.Input[64:96])
	qx := new(big.Int).SetBytes(contract.Input[96:128])
	qy := new(big.Int).SetBytes(contract.Input[128:])

	curve := elliptic.P256()
	pub := ecdsa.PublicKey{Curve: curve, X: qx, Y: qy}

	valid := curve.IsOnCurve(qx, qy) && ecdsa.Verify(&pub, hash, r, s)
	return p.encodeResult(valid), nil
}

func (p precompileP256) encodeResult(valid bool) []byte {
	out := make([]byte, p256OutputLen)
	if valid {
		out[p256OutputLen-1] = 1
	}
	return out
}
