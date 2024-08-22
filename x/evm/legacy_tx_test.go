package evm_test

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"

	"github.com/NibiruChain/nibiru/v2/x/evm"
)

func (suite *TxDataTestSuite) TestNewLegacyTx() {
	testCases := []struct {
		name string
		tx   *gethcore.Transaction
	}{
		{
			"non-empty Transaction",
			gethcore.NewTx(&gethcore.AccessListTx{
				Nonce:      1,
				Data:       []byte("data"),
				Gas:        100,
				Value:      big.NewInt(1),
				AccessList: gethcore.AccessList{},
				To:         &suite.addr,
				V:          big.NewInt(1),
				R:          big.NewInt(1),
				S:          big.NewInt(1),
			}),
		},
	}

	for _, tc := range testCases {
		tx, err := evm.NewLegacyTx(tc.tx)
		suite.Require().NoError(err)

		suite.Require().NotEmpty(tc.tx)
		suite.Require().Equal(uint8(0), tx.TxType())
	}
}

func (suite *TxDataTestSuite) TestLegacyTxTxType() {
	tx := evm.LegacyTx{}
	actual := tx.TxType()

	suite.Require().Equal(uint8(0), actual)
}

func (suite *TxDataTestSuite) TestLegacyTxCopy() {
	tx := &evm.LegacyTx{}
	txData := tx.Copy()

	suite.Require().Equal(&evm.LegacyTx{}, txData)
	// TODO: Test for different pointers
}

func (suite *TxDataTestSuite) TestLegacyTxGetChainID() {
	tx := evm.LegacyTx{}
	actual := tx.GetChainID()

	suite.Require().Nil(actual)
}

func (suite *TxDataTestSuite) TestLegacyTxGetAccessList() {
	tx := evm.LegacyTx{}
	actual := tx.GetAccessList()

	suite.Require().Nil(actual)
}

func (suite *TxDataTestSuite) TestLegacyTxGetData() {
	testCases := []struct {
		name string
		tx   evm.LegacyTx
	}{
		{
			"non-empty transaction",
			evm.LegacyTx{
				Data: nil,
			},
		},
	}

	for _, tc := range testCases {
		actual := tc.tx.GetData()

		suite.Require().Equal(tc.tx.Data, actual, tc.name)
	}
}

func (suite *TxDataTestSuite) TestLegacyTxGetGas() {
	testCases := []struct {
		name string
		tx   evm.LegacyTx
		exp  uint64
	}{
		{
			"non-empty gas",
			evm.LegacyTx{
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

func (suite *TxDataTestSuite) TestLegacyTxGetGasPrice() {
	testCases := []struct {
		name string
		tx   evm.LegacyTx
		exp  *big.Int
	}{
		{
			"empty gasPrice",
			evm.LegacyTx{
				GasPrice: nil,
			},
			nil,
		},
		{
			"non-empty gasPrice",
			evm.LegacyTx{
				GasPrice: &suite.sdkInt,
			},
			(&suite.sdkInt).BigInt(),
		},
	}

	for _, tc := range testCases {
		actual := tc.tx.GetGasFeeCapWei()

		suite.Require().Equal(tc.exp, actual, tc.name)
	}
}

func (suite *TxDataTestSuite) TestLegacyTxGetGasTipCap() {
	testCases := []struct {
		name string
		tx   evm.LegacyTx
		exp  *big.Int
	}{
		{
			"non-empty gasPrice",
			evm.LegacyTx{
				GasPrice: &suite.sdkInt,
			},
			(&suite.sdkInt).BigInt(),
		},
	}

	for _, tc := range testCases {
		actual := tc.tx.GetGasTipCapWei()

		suite.Require().Equal(tc.exp, actual, tc.name)
	}
}

func (suite *TxDataTestSuite) TestLegacyTxGetGasFeeCap() {
	testCases := []struct {
		name string
		tx   evm.LegacyTx
		exp  *big.Int
	}{
		{
			"non-empty gasPrice",
			evm.LegacyTx{
				GasPrice: &suite.sdkInt,
			},
			(&suite.sdkInt).BigInt(),
		},
	}

	for _, tc := range testCases {
		actual := tc.tx.GetGasFeeCapWei()

		suite.Require().Equal(tc.exp, actual, tc.name)
	}
}

func (suite *TxDataTestSuite) TestLegacyTxGetValue() {
	testCases := []struct {
		name string
		tx   evm.LegacyTx
		exp  *big.Int
	}{
		{
			"empty amount",
			evm.LegacyTx{
				Amount: nil,
			},
			nil,
		},
		{
			"non-empty amount",
			evm.LegacyTx{
				Amount: &suite.sdkInt,
			},
			(&suite.sdkInt).BigInt(),
		},
	}

	for _, tc := range testCases {
		actual := tc.tx.GetValueWei()

		suite.Require().Equal(tc.exp, actual, tc.name)
	}
}

func (suite *TxDataTestSuite) TestLegacyTxGetNonce() {
	testCases := []struct {
		name string
		tx   evm.LegacyTx
		exp  uint64
	}{
		{
			"none-empty nonce",
			evm.LegacyTx{
				Nonce: suite.uint64,
			},
			suite.uint64,
		},
	}
	for _, tc := range testCases {
		actual := tc.tx.GetNonce()

		suite.Require().Equal(tc.exp, actual)
	}
}

func (suite *TxDataTestSuite) TestLegacyTxGetTo() {
	testCases := []struct {
		name string
		tx   evm.LegacyTx
		exp  *common.Address
	}{
		{
			"empty address",
			evm.LegacyTx{
				To: "",
			},
			nil,
		},
		{
			"non-empty address",
			evm.LegacyTx{
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

func (suite *TxDataTestSuite) TestLegacyTxAsEthereumData() {
	tx := &evm.LegacyTx{}
	txData := tx.AsEthereumData()

	suite.Require().Equal(&gethcore.LegacyTx{}, txData)
}

func (suite *TxDataTestSuite) TestLegacyTxSetSignatureValues() {
	testCases := []struct {
		name string
		v    *big.Int
		r    *big.Int
		s    *big.Int
	}{
		{
			"non-empty values",
			suite.bigInt,
			suite.bigInt,
			suite.bigInt,
		},
	}
	for _, tc := range testCases {
		tx := &evm.LegacyTx{}
		tx.SetSignatureValues(nil, tc.v, tc.r, tc.s)

		v, r, s := tx.GetRawSignatureValues()

		suite.Require().Equal(tc.v, v, tc.name)
		suite.Require().Equal(tc.r, r, tc.name)
		suite.Require().Equal(tc.s, s, tc.name)
	}
}

func (suite *TxDataTestSuite) TestLegacyTxValidate() {
	testCases := []struct {
		name     string
		tx       evm.LegacyTx
		expError bool
	}{
		{
			"empty",
			evm.LegacyTx{},
			true,
		},
		{
			"gas price is nil",
			evm.LegacyTx{
				GasPrice: nil,
			},
			true,
		},
		{
			"gas price is negative",
			evm.LegacyTx{
				GasPrice: &suite.sdkMinusOneInt,
			},
			true,
		},
		{
			"amount is negative",
			evm.LegacyTx{
				GasPrice: &suite.sdkInt,
				Amount:   &suite.sdkMinusOneInt,
			},
			true,
		},
		{
			"to address is invalid",
			evm.LegacyTx{
				GasPrice: &suite.sdkInt,
				Amount:   &suite.sdkInt,
				To:       suite.invalidAddr,
			},
			true,
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

func (suite *TxDataTestSuite) TestLegacyTxEffectiveGasPrice() {
	testCases := []struct {
		name    string
		tx      evm.LegacyTx
		baseFee *big.Int
		exp     *big.Int
	}{
		{
			"non-empty legacy tx",
			evm.LegacyTx{
				GasPrice: &suite.sdkInt,
			},
			(&suite.sdkInt).BigInt(),
			(&suite.sdkInt).BigInt(),
		},
	}

	for _, tc := range testCases {
		actual := tc.tx.EffectiveGasPriceWei(tc.baseFee)

		suite.Require().Equal(tc.exp, actual, tc.name)
	}
}

func (suite *TxDataTestSuite) TestLegacyTxEffectiveFee() {
	testCases := []struct {
		name    string
		tx      evm.LegacyTx
		baseFee *big.Int
		exp     *big.Int
	}{
		{
			"non-empty legacy tx",
			evm.LegacyTx{
				GasPrice: &suite.sdkInt,
				GasLimit: uint64(1),
			},
			(&suite.sdkInt).BigInt(),
			(&suite.sdkInt).BigInt(),
		},
	}

	for _, tc := range testCases {
		actual := tc.tx.EffectiveFeeWei(tc.baseFee)

		suite.Require().Equal(tc.exp, actual, tc.name)
	}
}

func (suite *TxDataTestSuite) TestLegacyTxEffectiveCost() {
	testCases := []struct {
		name    string
		tx      evm.LegacyTx
		baseFee *big.Int
		exp     *big.Int
	}{
		{
			"non-empty legacy tx",
			evm.LegacyTx{
				GasPrice: &suite.sdkInt,
				GasLimit: uint64(1),
				Amount:   &suite.sdkZeroInt,
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

func (suite *TxDataTestSuite) TestLegacyTxFeeCost() {
	tx := &evm.LegacyTx{}

	suite.Require().Panics(func() { tx.Fee() }, "should panic")
	suite.Require().Panics(func() { tx.Cost() }, "should panic")
}
