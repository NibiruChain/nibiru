package evm_test

import (
	"math/big"
	"testing"

	sdkmath "cosmossdk.io/math"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"

	"github.com/stretchr/testify/suite"
)

type Suite struct {
	suite.Suite

	sdkInt         sdkmath.Int
	uint64         uint64
	hexUint64      hexutil.Uint64
	bigInt         *big.Int
	hexBigInt      hexutil.Big
	overflowBigInt *big.Int
	sdkZeroInt     sdkmath.Int
	sdkMinusOneInt sdkmath.Int
	invalidAddr    string
	addr           common.Address
	hexAddr        string
	hexDataBytes   hexutil.Bytes
	hexInputBytes  hexutil.Bytes
}

func (suite *Suite) SetupTest() {
	suite.sdkInt = sdkmath.NewInt(9001)
	suite.uint64 = suite.sdkInt.Uint64()
	suite.hexUint64 = hexutil.Uint64(100)
	suite.bigInt = big.NewInt(1)
	suite.hexBigInt = hexutil.Big(*big.NewInt(1))
	suite.overflowBigInt = big.NewInt(0).Exp(big.NewInt(10), big.NewInt(256), nil)
	suite.sdkZeroInt = sdkmath.ZeroInt()
	suite.sdkMinusOneInt = sdkmath.NewInt(-1)
	suite.invalidAddr = "123456"

	suite.addr = evmtest.NewEthPrivAcc().EthAddr

	suite.hexAddr = suite.addr.Hex()
	suite.hexDataBytes = hexutil.Bytes([]byte("data"))
	suite.hexInputBytes = hexutil.Bytes([]byte("input"))
}

func TestTxDataTestSuite(t *testing.T) {
	suite.Run(t, new(Suite))
}
