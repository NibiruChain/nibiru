package evm_test

import (
	"math/big"
	"testing"

	sdkmath "cosmossdk.io/math"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethcore "github.com/ethereum/go-ethereum/core/types"

	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"

	"github.com/stretchr/testify/suite"
)

type TxDataTestSuite struct {
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

func (suite *TxDataTestSuite) SetupTest() {
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
	suite.Run(t, new(TxDataTestSuite))
}

func (suite *TxDataTestSuite) TestNewDynamicFeeTx() {
	testCases := []struct {
		name     string
		expError bool
		tx       *gethcore.Transaction
	}{
		{
			"non-empty tx",
			false,
			gethcore.NewTx(&gethcore.DynamicFeeTx{
				Nonce:      1,
				Data:       []byte("data"),
				Gas:        100,
				Value:      big.NewInt(1),
				AccessList: gethcore.AccessList{},
				To:         &suite.addr,
				V:          suite.bigInt,
				R:          suite.bigInt,
				S:          suite.bigInt,
			}),
		},
		{
			"value out of bounds tx",
			true,
			gethcore.NewTx(&gethcore.DynamicFeeTx{
				Nonce:      1,
				Data:       []byte("data"),
				Gas:        100,
				Value:      suite.overflowBigInt,
				AccessList: gethcore.AccessList{},
				To:         &suite.addr,
				V:          suite.bigInt,
				R:          suite.bigInt,
				S:          suite.bigInt,
			}),
		},
		{
			"gas fee cap out of bounds tx",
			true,
			gethcore.NewTx(&gethcore.DynamicFeeTx{
				Nonce:      1,
				Data:       []byte("data"),
				Gas:        100,
				GasFeeCap:  suite.overflowBigInt,
				Value:      big.NewInt(1),
				AccessList: gethcore.AccessList{},
				To:         &suite.addr,
				V:          suite.bigInt,
				R:          suite.bigInt,
				S:          suite.bigInt,
			}),
		},
		{
			"gas tip cap out of bounds tx",
			true,
			gethcore.NewTx(&gethcore.DynamicFeeTx{
				Nonce:      1,
				Data:       []byte("data"),
				Gas:        100,
				GasTipCap:  suite.overflowBigInt,
				Value:      big.NewInt(1),
				AccessList: gethcore.AccessList{},
				To:         &suite.addr,
				V:          suite.bigInt,
				R:          suite.bigInt,
				S:          suite.bigInt,
			}),
		},
	}
	for _, tc := range testCases {
		tx, err := evm.NewDynamicFeeTx(tc.tx)

		if tc.expError {
			suite.Require().Error(err)
		} else {
			suite.Require().NoError(err)
			suite.Require().NotEmpty(tx)
			suite.Require().Equal(uint8(2), tx.TxType())
		}
	}
}

func (suite *TxDataTestSuite) TestDynamicFeeTxAsEthereumData() {
	feeConfig := &gethcore.DynamicFeeTx{
		Nonce:      1,
		Data:       []byte("data"),
		Gas:        100,
		Value:      big.NewInt(1),
		AccessList: gethcore.AccessList{},
		To:         &suite.addr,
		V:          suite.bigInt,
		R:          suite.bigInt,
		S:          suite.bigInt,
	}

	tx := gethcore.NewTx(feeConfig)

	dynamicFeeTx, err := evm.NewDynamicFeeTx(tx)
	suite.Require().NoError(err)

	res := dynamicFeeTx.AsEthereumData()
	resTx := gethcore.NewTx(res)

	suite.Require().Equal(feeConfig.Nonce, resTx.Nonce())
	suite.Require().Equal(feeConfig.Data, resTx.Data())
	suite.Require().Equal(feeConfig.Gas, resTx.Gas())
	suite.Require().Equal(feeConfig.Value, resTx.Value())
	suite.Require().Equal(feeConfig.AccessList, resTx.AccessList())
	suite.Require().Equal(feeConfig.To, resTx.To())
}

func (suite *TxDataTestSuite) TestDynamicFeeTxCopy() {
	tx := &evm.DynamicFeeTx{}
	txCopy := tx.Copy()

	suite.Require().Equal(&evm.DynamicFeeTx{}, txCopy)
	// TODO: Test for different pointers
}

func (suite *TxDataTestSuite) TestDynamicFeeTxGetChainID() {
	testCases := []struct {
		name string
		tx   evm.DynamicFeeTx
		exp  *big.Int
	}{
		{
			"empty chainID",
			evm.DynamicFeeTx{
				ChainID: nil,
			},
			nil,
		},
		{
			"non-empty chainID",
			evm.DynamicFeeTx{
				ChainID: &suite.sdkInt,
			},
			(&suite.sdkInt).BigInt(),
		},
	}

	for _, tc := range testCases {
		actual := tc.tx.GetChainID()

		suite.Require().Equal(tc.exp, actual, tc.name)
	}
}

func (suite *TxDataTestSuite) TestDynamicFeeTxGetAccessList() {
	testCases := []struct {
		name string
		tx   evm.DynamicFeeTx
		exp  gethcore.AccessList
	}{
		{
			"empty accesses",
			evm.DynamicFeeTx{
				Accesses: nil,
			},
			nil,
		},
		{
			"nil",
			evm.DynamicFeeTx{
				Accesses: evm.NewAccessList(nil),
			},
			nil,
		},
		{
			"non-empty accesses",
			evm.DynamicFeeTx{
				Accesses: evm.AccessList{
					{
						Address:     suite.hexAddr,
						StorageKeys: []string{},
					},
				},
			},
			gethcore.AccessList{
				{
					Address:     suite.addr,
					StorageKeys: []common.Hash{},
				},
			},
		},
	}

	for _, tc := range testCases {
		actual := tc.tx.GetAccessList()

		suite.Require().Equal(tc.exp, actual, tc.name)
	}
}

func (suite *TxDataTestSuite) TestDynamicFeeTxGetData() {
	testCases := []struct {
		name string
		tx   evm.DynamicFeeTx
	}{
		{
			"non-empty transaction",
			evm.DynamicFeeTx{
				Data: nil,
			},
		},
	}

	for _, tc := range testCases {
		actual := tc.tx.GetData()

		suite.Require().Equal(tc.tx.Data, actual, tc.name)
	}
}

func (suite *TxDataTestSuite) TestDynamicFeeTxGetGas() {
	testCases := []struct {
		name string
		tx   evm.DynamicFeeTx
		exp  uint64
	}{
		{
			"non-empty gas",
			evm.DynamicFeeTx{
				GasLimit: suite.uint64,
			},
			suite.uint64,
		},
	}

	for _, tc := range testCases {
		actual := tc.tx.GetGas()

		suite.Require().Equal(tc.exp, actual, tc.name)
	}
}

func (suite *TxDataTestSuite) TestDynamicFeeTxGetGasPrice() {
	testCases := []struct {
		name string
		tx   evm.DynamicFeeTx
		exp  *big.Int
	}{
		{
			"non-empty gasFeeCap",
			evm.DynamicFeeTx{
				GasFeeCap: &suite.sdkInt,
			},
			(&suite.sdkInt).BigInt(),
		},
	}

	for _, tc := range testCases {
		actual := tc.tx.GetGasPrice()

		suite.Require().Equal(tc.exp, actual, tc.name)
	}
}

func (suite *TxDataTestSuite) TestDynamicFeeTxGetGasTipCap() {
	testCases := []struct {
		name string
		tx   evm.DynamicFeeTx
		exp  *big.Int
	}{
		{
			"empty gasTipCap",
			evm.DynamicFeeTx{
				GasTipCap: nil,
			},
			nil,
		},
		{
			"non-empty gasTipCap",
			evm.DynamicFeeTx{
				GasTipCap: &suite.sdkInt,
			},
			(&suite.sdkInt).BigInt(),
		},
	}

	for _, tc := range testCases {
		actual := tc.tx.GetGasTipCap()

		suite.Require().Equal(tc.exp, actual, tc.name)
	}
}

func (suite *TxDataTestSuite) TestDynamicFeeTxGetGasFeeCap() {
	testCases := []struct {
		name string
		tx   evm.DynamicFeeTx
		exp  *big.Int
	}{
		{
			"empty gasFeeCap",
			evm.DynamicFeeTx{
				GasFeeCap: nil,
			},
			nil,
		},
		{
			"non-empty gasFeeCap",
			evm.DynamicFeeTx{
				GasFeeCap: &suite.sdkInt,
			},
			(&suite.sdkInt).BigInt(),
		},
	}

	for _, tc := range testCases {
		actual := tc.tx.GetGasFeeCap

		suite.Require().Equal(tc.exp, actual, tc.name)
	}
}

func (suite *TxDataTestSuite) TestDynamicFeeTxGetValue() {
	testCases := []struct {
		name string
		tx   evm.DynamicFeeTx
		exp  *big.Int
	}{
		{
			"empty amount",
			evm.DynamicFeeTx{
				Amount: nil,
			},
			nil,
		},
		{
			"non-empty amount",
			evm.DynamicFeeTx{
				Amount: &suite.sdkInt,
			},
			(&suite.sdkInt).BigInt(),
		},
	}

	for _, tc := range testCases {
		actual := tc.tx.GetValue()

		suite.Require().Equal(tc.exp, actual, tc.name)
	}
}

func (suite *TxDataTestSuite) TestDynamicFeeTxGetNonce() {
	testCases := []struct {
		name string
		tx   evm.DynamicFeeTx
		exp  uint64
	}{
		{
			"non-empty nonce",
			evm.DynamicFeeTx{
				Nonce: suite.uint64,
			},
			suite.uint64,
		},
	}

	for _, tc := range testCases {
		actual := tc.tx.GetNonce()

		suite.Require().Equal(tc.exp, actual, tc.name)
	}
}

func (suite *TxDataTestSuite) TestDynamicFeeTxGetTo() {
	testCases := []struct {
		name string
		tx   evm.DynamicFeeTx
		exp  *common.Address
	}{
		{
			"empty suite.address",
			evm.DynamicFeeTx{
				To: "",
			},
			nil,
		},
		{
			"non-empty suite.address",
			evm.DynamicFeeTx{
				To: suite.hexAddr,
			},
			&suite.addr,
		},
	}

	for _, tc := range testCases {
		actual := tc.tx.GetTo()

		suite.Require().Equal(tc.exp, actual, tc.name)
	}
}

func (suite *TxDataTestSuite) TestDynamicFeeTxSetSignatureValues() {
	testCases := []struct {
		name    string
		chainID *big.Int
		r       *big.Int
		v       *big.Int
		s       *big.Int
	}{
		{
			"empty values",
			nil,
			nil,
			nil,
			nil,
		},
		{
			"non-empty values",
			suite.bigInt,
			suite.bigInt,
			suite.bigInt,
			suite.bigInt,
		},
	}

	for _, tc := range testCases {
		tx := &evm.DynamicFeeTx{}
		tx.SetSignatureValues(tc.chainID, tc.v, tc.r, tc.s)

		v, r, s := tx.GetRawSignatureValues()
		chainID := tx.GetChainID()

		suite.Require().Equal(tc.v, v, tc.name)
		suite.Require().Equal(tc.r, r, tc.name)
		suite.Require().Equal(tc.s, s, tc.name)
		suite.Require().Equal(tc.chainID, chainID, tc.name)
	}
}

func (suite *TxDataTestSuite) TestDynamicFeeTxValidate() {
	testCases := []struct {
		name     string
		tx       evm.DynamicFeeTx
		expError bool
	}{
		{
			"empty",
			evm.DynamicFeeTx{},
			true,
		},
		{
			"gas tip cap is nil",
			evm.DynamicFeeTx{
				GasTipCap: nil,
			},
			true,
		},
		{
			"gas fee cap is nil",
			evm.DynamicFeeTx{
				GasTipCap: &suite.sdkZeroInt,
			},
			true,
		},
		{
			"gas tip cap is negative",
			evm.DynamicFeeTx{
				GasTipCap: &suite.sdkMinusOneInt,
				GasFeeCap: &suite.sdkZeroInt,
			},
			true,
		},
		{
			"gas tip cap is negative",
			evm.DynamicFeeTx{
				GasTipCap: &suite.sdkZeroInt,
				GasFeeCap: &suite.sdkMinusOneInt,
			},
			true,
		},
		{
			"gas fee cap < gas tip cap",
			evm.DynamicFeeTx{
				GasTipCap: &suite.sdkInt,
				GasFeeCap: &suite.sdkZeroInt,
			},
			true,
		},
		{
			"amount is negative",
			evm.DynamicFeeTx{
				GasTipCap: &suite.sdkInt,
				GasFeeCap: &suite.sdkInt,
				Amount:    &suite.sdkMinusOneInt,
			},
			true,
		},
		{
			"to suite.address is invalid",
			evm.DynamicFeeTx{
				GasTipCap: &suite.sdkInt,
				GasFeeCap: &suite.sdkInt,
				Amount:    &suite.sdkInt,
				To:        suite.invalidAddr,
			},
			true,
		},
		{
			"chain ID not present on AccessList txs",
			evm.DynamicFeeTx{
				GasTipCap: &suite.sdkInt,
				GasFeeCap: &suite.sdkInt,
				Amount:    &suite.sdkInt,
				To:        suite.hexAddr,
				ChainID:   nil,
			},
			true,
		},
		{
			"no errors",
			evm.DynamicFeeTx{
				GasTipCap: &suite.sdkInt,
				GasFeeCap: &suite.sdkInt,
				Amount:    &suite.sdkInt,
				To:        suite.hexAddr,
				ChainID:   &suite.sdkInt,
			},
			false,
		},
	}

	for _, tc := range testCases {
		err := tc.tx.Validate()

		if tc.expError {
			suite.Require().Error(err, tc.name)
			continue
		}

		suite.Require().NoError(err, tc.name)
	}
}

func (suite *TxDataTestSuite) TestDynamicFeeTxEffectiveGasPrice() {
	testCases := []struct {
		name    string
		tx      evm.DynamicFeeTx
		baseFee *big.Int
		exp     *big.Int
	}{
		{
			"non-empty dynamic fee tx",
			evm.DynamicFeeTx{
				GasTipCap: &suite.sdkInt,
				GasFeeCap: &suite.sdkInt,
			},
			(&suite.sdkInt).BigInt(),
			(&suite.sdkInt).BigInt(),
		},
	}

	for _, tc := range testCases {
		actual := tc.tx.EffectiveGasPrice(tc.baseFee)

		suite.Require().Equal(tc.exp, actual, tc.name)
	}
}

func (suite *TxDataTestSuite) TestDynamicFeeTxEffectiveFee() {
	testCases := []struct {
		name    string
		tx      evm.DynamicFeeTx
		baseFee *big.Int
		exp     *big.Int
	}{
		{
			"non-empty dynamic fee tx",
			evm.DynamicFeeTx{
				GasTipCap: &suite.sdkInt,
				GasFeeCap: &suite.sdkInt,
				GasLimit:  uint64(1),
			},
			(&suite.sdkInt).BigInt(),
			(&suite.sdkInt).BigInt(),
		},
	}

	for _, tc := range testCases {
		actual := tc.tx.EffectiveFee(tc.baseFee)

		suite.Require().Equal(tc.exp, actual, tc.name)
	}
}

func (suite *TxDataTestSuite) TestDynamicFeeTxEffectiveCost() {
	testCases := []struct {
		name    string
		tx      evm.DynamicFeeTx
		baseFee *big.Int
		exp     *big.Int
	}{
		{
			"non-empty dynamic fee tx",
			evm.DynamicFeeTx{
				GasTipCap: &suite.sdkInt,
				GasFeeCap: &suite.sdkInt,
				GasLimit:  uint64(1),
				Amount:    &suite.sdkZeroInt,
			},
			(&suite.sdkInt).BigInt(),
			(&suite.sdkInt).BigInt(),
		},
	}

	for _, tc := range testCases {
		actual := tc.tx.EffectiveCost(tc.baseFee)

		suite.Require().Equal(tc.exp, actual, tc.name)
	}
}

func (suite *TxDataTestSuite) TestDynamicFeeTxFeeCost() {
	tx := &evm.DynamicFeeTx{}
	suite.Require().Panics(func() { tx.Fee() }, "should panic")
	suite.Require().Panics(func() { tx.Cost() }, "should panic")
}
