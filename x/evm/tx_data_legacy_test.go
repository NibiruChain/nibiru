package evm_test

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"

	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
)

func (suite *Suite) TestNewLegacyTx() {
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

func (suite *Suite) TestLegacyTxTxType() {
	tx := evm.LegacyTx{}
	actual := tx.TxType()

	suite.Require().Equal(uint8(0), actual)
}

func (suite *Suite) TestLegacyTxCopy() {
	tx := &evm.LegacyTx{}
	txData := tx.Copy()

	suite.Require().Equal(&evm.LegacyTx{}, txData)
	// TODO: Test for different pointers
}

func (suite *Suite) TestLegacyTxGetChainID() {
	tx := evm.LegacyTx{}
	actual := tx.GetChainID()

	suite.Require().Nil(actual)
}

func (suite *Suite) TestLegacyTxGetAccessList() {
	tx := evm.LegacyTx{}
	actual := tx.GetAccessList()

	suite.Require().Nil(actual)
}

func (suite *Suite) TestLegacyTxGetData() {
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

func (suite *Suite) TestLegacyTxGetGas() {
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

func (suite *Suite) TestLegacyTxGetGasPrice() {
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

func (suite *Suite) TestLegacyTxGetGasTipCap() {
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

func (suite *Suite) TestLegacyTxGetGasFeeCap() {
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

func (suite *Suite) TestLegacyTxGetValue() {
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

func (suite *Suite) TestLegacyTxGetNonce() {
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

func (suite *Suite) TestLegacyTxGetTo() {
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

func (suite *Suite) TestLegacyTxAsEthereumData() {
	tx := &evm.LegacyTx{}
	txData := tx.AsEthereumData()

	suite.Require().Equal(&gethcore.LegacyTx{}, txData)
}

func (suite *Suite) TestLegacyTxSetSignatureValues() {
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

func (suite *Suite) TestLegacyTxValidate() {
	testCases := []struct {
		name     string
		tx       func(tx *evm.LegacyTx) *evm.LegacyTx
		expError bool
	}{
		{
			name:     "empty",
			tx:       func(_ *evm.LegacyTx) *evm.LegacyTx { return new(evm.LegacyTx) },
			expError: true,
		},
		{
			name: "gas price is nil",
			tx: func(tx *evm.LegacyTx) *evm.LegacyTx {
				tx.GasPrice = nil
				return tx
			},
			expError: true,
		},
		{
			name: "gas price is negative",
			tx: func(tx *evm.LegacyTx) *evm.LegacyTx {
				tx.GasPrice = &suite.sdkMinusOneInt
				return tx
			},
			expError: true,
		},
		{
			name: "amount is negative",
			tx: func(tx *evm.LegacyTx) *evm.LegacyTx {
				tx.Amount = &suite.sdkMinusOneInt
				return tx
			},
			expError: true,
		},
		{
			name: "to address is invalid",
			tx: func(tx *evm.LegacyTx) *evm.LegacyTx {
				tx.To = suite.invalidAddr
				return tx
			},
			expError: true,
		},
	}

	for _, tc := range testCases {
		got := tc.tx(evmtest.ValidLegacyTx())
		err := got.Validate()

		if tc.expError {
			suite.Require().Error(err, tc.name)
			continue
		}

		suite.Require().NoError(err, tc.name)
	}
}

func (suite *Suite) TestLegacyTxEffectiveGasPrice() {
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
		actual := tc.tx.EffectiveGasPriceWeiPerGas(tc.baseFee)

		suite.Require().Equal(tc.exp, actual, tc.name)
	}
}

func (suite *Suite) TestLegacyTxEffectiveFee() {
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

func (suite *Suite) TestLegacyTxEffectiveCost() {
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

func (suite *Suite) TestLegacyTxFeeCost() {
	tx := &evm.LegacyTx{}

	suite.Require().Panics(func() { tx.Fee() }, "should panic")
	suite.Require().Panics(func() { tx.Cost() }, "should panic")
}
