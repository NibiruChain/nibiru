package evm_test

import (
	"fmt"
	"math"
	"math/big"
	"reflect"
	"strings"
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/eth/crypto/ethsecp256k1"
	"github.com/NibiruChain/nibiru/v2/eth/encoding"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
)

type MsgsSuite struct {
	suite.Suite

	signer        keyring.Signer
	from          common.Address
	to            common.Address
	chainID       *big.Int
	hundredBigInt *big.Int

	clientCtx client.Context
}

func TestMsgsSuite(t *testing.T) {
	suite.Run(t, new(MsgsSuite))
}

func (s *MsgsSuite) SetupTest() {
	ethAcc := evmtest.NewEthPrivAcc()
	from, privFrom := ethAcc.EthAddr, ethAcc.PrivKey

	s.signer = evmtest.NewSigner(privFrom)
	s.from = from
	s.to = evmtest.NewEthPrivAcc().EthAddr
	s.chainID = big.NewInt(1)
	s.hundredBigInt = big.NewInt(100)

	encodingConfig := encoding.MakeConfig(app.ModuleBasics)
	s.clientCtx = client.Context{}.WithTxConfig(encodingConfig.TxConfig)
}

func (s *MsgsSuite) TestMsgEthereumTx_Constructor() {
	evmTx := &evm.EvmTxArgs{
		Nonce:    0,
		To:       &s.to,
		GasLimit: 100000,
		Input:    []byte("test"),
	}
	msg := evm.NewTx(evmTx)

	// suite.Require().Equal(msg.Data.To, suite.to.Hex())
	s.Require().Equal(msg.Route(), evm.RouterKey)
	s.Require().Equal(msg.Type(), evm.TypeMsgEthereumTx)
	// suite.Require().NotNil(msg.To)
	s.Require().Equal(msg.GetMsgs(), []sdk.Msg{msg})
	s.Require().Panics(func() { msg.GetSigners() })
	s.Require().Panics(func() { msg.GetSignBytes() })

	evmTx2 := &evm.EvmTxArgs{
		Nonce:    0,
		GasLimit: 100000,
		Input:    []byte("test"),
	}
	msg = evm.NewTx(evmTx2)
	s.Require().NotNil(msg)
}

func (s *MsgsSuite) TestMsgEthereumTx_BuildTx() {
	evmTx := &evm.EvmTxArgs{
		Nonce:     0,
		To:        &s.to,
		GasLimit:  100000,
		GasPrice:  big.NewInt(1),
		GasFeeCap: big.NewInt(1),
		GasTipCap: big.NewInt(0),
		Input:     []byte("test"),
	}
	testCases := []struct {
		name     string
		msg      *evm.MsgEthereumTx
		expError bool
	}{
		{
			"build tx - pass",
			evm.NewTx(evmTx),
			false,
		},
		{
			"build tx - fail: nil data",
			evm.NewTx(evmTx),
			true,
		},
	}

	for _, tc := range testCases {
		if strings.Contains(tc.name, "nil data") {
			tc.msg.Data = nil
		}

		tx, err := tc.msg.BuildTx(s.clientCtx.TxConfig.NewTxBuilder(), evm.DefaultEVMDenom)
		if tc.expError {
			s.Require().Error(err)
		} else {
			s.Require().NoError(err)

			s.Require().Empty(tx.GetMemo())
			s.Require().Empty(tx.GetTimeoutHeight())
			s.Require().Equal(uint64(100000), tx.GetGas())
			s.Require().Equal(sdk.NewCoins(sdk.NewCoin(evm.DefaultEVMDenom, sdkmath.NewInt(100000))), tx.GetFee())
		}
	}
}

func invalidAddr() string { return "0x0000" }

func (s *MsgsSuite) TestMsgEthereumTx_ValidateBasic() {
	var (
		hundredInt   = big.NewInt(100)
		validChainID = big.NewInt(9000)
		zeroInt      = big.NewInt(0)
		minusOneInt  = big.NewInt(-1)
		//nolint:all
		exp_2_255 = new(big.Int).Exp(big.NewInt(2), big.NewInt(255), nil)
	)
	testCases := []struct {
		msg        string
		to         string
		amount     *big.Int
		gasLimit   uint64
		gasPrice   *big.Int
		gasFeeCap  *big.Int
		gasTipCap  *big.Int
		from       string
		accessList *gethcore.AccessList
		chainID    *big.Int
		expectPass bool
		errMsg     string
	}{
		{
			msg:        "pass with recipient - Legacy Tx",
			to:         s.to.Hex(),
			from:       s.from.Hex(),
			amount:     hundredInt,
			gasLimit:   1000,
			gasPrice:   hundredInt,
			gasFeeCap:  nil,
			gasTipCap:  nil,
			chainID:    validChainID,
			expectPass: true,
		},
		{
			msg:        "pass with recipient - AccessList Tx",
			to:         s.to.Hex(),
			amount:     hundredInt,
			gasLimit:   1000,
			gasPrice:   zeroInt,
			gasFeeCap:  nil,
			gasTipCap:  nil,
			accessList: &gethcore.AccessList{},
			chainID:    validChainID,
			expectPass: true,
		},
		{
			msg:        "pass with recipient - DynamicFee Tx",
			to:         s.to.Hex(),
			amount:     hundredInt,
			gasLimit:   1000,
			gasPrice:   zeroInt,
			gasFeeCap:  hundredInt,
			gasTipCap:  zeroInt,
			accessList: &gethcore.AccessList{},
			chainID:    validChainID,
			expectPass: true,
		},
		{
			msg:        "pass contract - Legacy Tx",
			to:         "",
			from:       s.from.Hex(),
			amount:     hundredInt,
			gasLimit:   1000,
			gasPrice:   hundredInt,
			gasFeeCap:  nil,
			gasTipCap:  nil,
			chainID:    validChainID,
			expectPass: true,
		},
		{
			msg:        "maxInt64 gas limit overflow",
			to:         s.to.Hex(),
			from:       s.from.Hex(),
			amount:     hundredInt,
			gasLimit:   math.MaxInt64 + 1,
			gasPrice:   hundredInt,
			gasFeeCap:  nil,
			gasTipCap:  nil,
			chainID:    validChainID,
			expectPass: false,
			errMsg:     "gas limit must be less than math.MaxInt64",
		},
		{
			msg:        "nil amount - Legacy Tx",
			to:         s.to.Hex(),
			from:       s.from.Hex(),
			amount:     nil,
			gasLimit:   1000,
			gasPrice:   hundredInt,
			gasFeeCap:  nil,
			gasTipCap:  nil,
			chainID:    validChainID,
			expectPass: true,
		},
		{
			msg:        "negative amount - Legacy Tx",
			to:         s.to.Hex(),
			from:       s.from.Hex(),
			amount:     minusOneInt,
			gasLimit:   1000,
			gasPrice:   hundredInt,
			gasFeeCap:  nil,
			gasTipCap:  nil,
			chainID:    validChainID,
			expectPass: false,
			errMsg:     "amount cannot be negative",
		},
		{
			msg:        "zero gas limit - Legacy Tx",
			to:         s.to.Hex(),
			from:       s.from.Hex(),
			amount:     hundredInt,
			gasLimit:   0,
			gasPrice:   hundredInt,
			gasFeeCap:  nil,
			gasTipCap:  nil,
			chainID:    validChainID,
			expectPass: false,
			errMsg:     "gas limit must not be zero",
		},
		{
			msg:        "nil gas price - Legacy Tx",
			to:         s.to.Hex(),
			amount:     hundredInt,
			gasLimit:   1000,
			gasPrice:   nil,
			gasFeeCap:  nil,
			gasTipCap:  nil,
			chainID:    validChainID,
			expectPass: false,
			errMsg:     "gas price cannot be nil",
		},
		{
			msg:        "negative gas price - Legacy Tx",
			to:         s.to.Hex(),
			from:       s.from.Hex(),
			amount:     hundredInt,
			gasLimit:   1000,
			gasPrice:   minusOneInt,
			gasFeeCap:  nil,
			gasTipCap:  nil,
			chainID:    validChainID,
			expectPass: false,
			errMsg:     "gas price cannot be negative",
		},
		{
			msg:        "zero gas price - Legacy Tx",
			to:         s.to.Hex(),
			from:       s.from.Hex(),
			amount:     hundredInt,
			gasLimit:   1000,
			gasPrice:   zeroInt,
			gasFeeCap:  nil,
			gasTipCap:  nil,
			chainID:    validChainID,
			expectPass: true,
		},
		{
			msg:        "invalid from address - Legacy Tx",
			to:         s.to.Hex(),
			from:       invalidAddr(),
			amount:     hundredInt,
			gasLimit:   1000,
			gasPrice:   zeroInt,
			gasFeeCap:  nil,
			gasTipCap:  nil,
			chainID:    validChainID,
			expectPass: false,
			errMsg:     "invalid from address",
		},
		{
			msg:        "out of bound gas fee - Legacy Tx",
			to:         s.to.Hex(),
			from:       s.from.Hex(),
			amount:     hundredInt,
			gasLimit:   1000,
			gasPrice:   exp_2_255,
			gasFeeCap:  nil,
			gasTipCap:  nil,
			chainID:    validChainID,
			expectPass: false,
			errMsg:     "out of bound",
		},
		{
			msg:        "nil amount - AccessListTx",
			to:         s.to.Hex(),
			amount:     nil,
			gasLimit:   1000,
			gasPrice:   hundredInt,
			gasFeeCap:  nil,
			gasTipCap:  nil,
			accessList: &gethcore.AccessList{},
			chainID:    validChainID,
			expectPass: true,
		},
		{
			msg:        "negative amount - AccessListTx",
			to:         s.to.Hex(),
			amount:     minusOneInt,
			gasLimit:   1000,
			gasPrice:   hundredInt,
			gasFeeCap:  nil,
			gasTipCap:  nil,
			accessList: &gethcore.AccessList{},
			chainID:    validChainID,
			expectPass: false,
			errMsg:     "amount cannot be negative",
		},
		{
			msg:        "zero gas limit - AccessListTx",
			to:         s.to.Hex(),
			amount:     hundredInt,
			gasLimit:   0,
			gasPrice:   zeroInt,
			gasFeeCap:  nil,
			gasTipCap:  nil,
			accessList: &gethcore.AccessList{},
			chainID:    validChainID,
			expectPass: false,
			errMsg:     "gas limit must not be zero",
		},
		{
			msg:        "nil gas price - AccessListTx",
			to:         s.to.Hex(),
			amount:     hundredInt,
			gasLimit:   1000,
			gasPrice:   nil,
			gasFeeCap:  nil,
			gasTipCap:  nil,
			accessList: &gethcore.AccessList{},
			chainID:    validChainID,
			expectPass: false,
			errMsg:     "cannot be nil: invalid gas price",
		},
		{
			msg:        "negative gas price - AccessListTx",
			to:         s.to.Hex(),
			amount:     hundredInt,
			gasLimit:   1000,
			gasPrice:   minusOneInt,
			gasFeeCap:  nil,
			gasTipCap:  nil,
			accessList: &gethcore.AccessList{},
			chainID:    validChainID,
			expectPass: false,
			errMsg:     "gas price cannot be negative",
		},
		{
			msg:        "zero gas price - AccessListTx",
			to:         s.to.Hex(),
			amount:     hundredInt,
			gasLimit:   1000,
			gasPrice:   zeroInt,
			gasFeeCap:  nil,
			gasTipCap:  nil,
			accessList: &gethcore.AccessList{},
			chainID:    validChainID,
			expectPass: true,
		},
		{
			msg:        "invalid from address - AccessListTx",
			to:         s.to.Hex(),
			amount:     hundredInt,
			gasLimit:   1000,
			gasPrice:   zeroInt,
			gasFeeCap:  nil,
			gasTipCap:  nil,
			from:       invalidAddr(),
			accessList: &gethcore.AccessList{},
			chainID:    validChainID,
			expectPass: false,
			errMsg:     "invalid from address",
		},
		{
			msg:        "chain ID not set on AccessListTx",
			to:         s.to.Hex(),
			amount:     hundredInt,
			gasLimit:   1000,
			gasPrice:   zeroInt,
			gasFeeCap:  nil,
			gasTipCap:  nil,
			accessList: &gethcore.AccessList{},
			chainID:    nil,
			expectPass: false,
			errMsg:     "chain ID must be present on AccessList txs",
		},
		{
			msg:        "nil tx.Data - AccessList Tx",
			to:         s.to.Hex(),
			amount:     hundredInt,
			gasLimit:   1000,
			gasPrice:   zeroInt,
			gasFeeCap:  nil,
			gasTipCap:  nil,
			accessList: &gethcore.AccessList{},
			expectPass: false,
			errMsg:     "failed to unpack tx data",
		},
		{
			msg:        "happy, valid chain ID",
			to:         s.to.Hex(),
			amount:     hundredInt,
			gasLimit:   1000,
			gasPrice:   zeroInt,
			gasFeeCap:  nil,
			gasTipCap:  nil,
			accessList: &gethcore.AccessList{},
			chainID:    hundredInt,
			expectPass: true,
			errMsg:     "",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.msg, func() {
			to := common.HexToAddress(tc.to)
			evmTx := &evm.EvmTxArgs{
				ChainID:   tc.chainID,
				Nonce:     1,
				To:        &to,
				Amount:    tc.amount,
				GasLimit:  tc.gasLimit,
				GasPrice:  tc.gasPrice,
				GasFeeCap: tc.gasFeeCap,
				Accesses:  tc.accessList,
			}
			tx := evm.NewTx(evmTx)
			tx.From = tc.from

			// apply nil assignment here to test ValidateBasic function instead of NewTx
			if strings.Contains(tc.msg, "nil tx.Data") {
				tx.Data = nil
			}

			// for legacy_Tx need to sign tx because the chainID is derived
			// from signature
			if tc.accessList == nil && tc.from == s.from.Hex() {
				ethSigner := gethcore.LatestSignerForChainID(tc.chainID)
				err := tx.Sign(ethSigner, s.signer)
				s.Require().NoError(err)
			}

			err := tx.ValidateBasic()

			if tc.expectPass {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
				s.Require().Contains(err.Error(), tc.errMsg)
			}
		})
	}
}

func (s *MsgsSuite) TestMsgEthereumTx_ValidateBasicAdvanced() {
	hundredInt := big.NewInt(100)
	evmTx := &evm.EvmTxArgs{
		ChainID:   hundredInt,
		Nonce:     1,
		Amount:    big.NewInt(10),
		GasLimit:  100000,
		GasPrice:  big.NewInt(150),
		GasFeeCap: big.NewInt(200),
	}

	testCases := []struct {
		msg        string
		msgBuilder func() *evm.MsgEthereumTx
		expectPass bool
	}{
		{
			"fails - invalid tx hash",
			func() *evm.MsgEthereumTx {
				msg := evm.NewTx(evmTx)
				msg.Hash = "0x00"
				return msg
			},
			false,
		},
		{
			"fails - invalid size",
			func() *evm.MsgEthereumTx {
				msg := evm.NewTx(evmTx)
				msg.Size_ = 1
				return msg
			},
			false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.msg, func() {
			err := tc.msgBuilder().ValidateBasic()
			if tc.expectPass {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *MsgsSuite) TestMsgEthereumTx_Sign() {
	testCases := []struct {
		msg        string
		txParams   *evm.EvmTxArgs
		ethSigner  gethcore.Signer
		malleate   func(tx *evm.MsgEthereumTx)
		expectPass bool
	}{
		{
			"pass - EIP2930 signer",
			&evm.EvmTxArgs{
				ChainID:  s.chainID,
				Nonce:    0,
				To:       &s.to,
				GasLimit: 100000,
				Input:    []byte("test"),
				Accesses: &gethcore.AccessList{},
			},
			gethcore.NewEIP2930Signer(s.chainID),
			func(tx *evm.MsgEthereumTx) { tx.From = s.from.Hex() },
			true,
		},
		{
			"pass - EIP155 signer",
			&evm.EvmTxArgs{
				ChainID:  s.chainID,
				Nonce:    0,
				To:       &s.to,
				GasLimit: 100000,
				Input:    []byte("test"),
			},
			gethcore.NewEIP155Signer(s.chainID),
			func(tx *evm.MsgEthereumTx) { tx.From = s.from.Hex() },
			true,
		},
		{
			"pass - Homestead signer",
			&evm.EvmTxArgs{
				ChainID:  s.chainID,
				Nonce:    0,
				To:       &s.to,
				GasLimit: 100000,
				Input:    []byte("test"),
			},
			gethcore.HomesteadSigner{},
			func(tx *evm.MsgEthereumTx) { tx.From = s.from.Hex() },
			true,
		},
		{
			"pass - Frontier signer",
			&evm.EvmTxArgs{
				ChainID:  s.chainID,
				Nonce:    0,
				To:       &s.to,
				GasLimit: 100000,
				Input:    []byte("test"),
			},
			gethcore.FrontierSigner{},
			func(tx *evm.MsgEthereumTx) { tx.From = s.from.Hex() },
			true,
		},
		{
			"no from address ",
			&evm.EvmTxArgs{
				ChainID:  s.chainID,
				Nonce:    0,
				To:       &s.to,
				GasLimit: 100000,
				Input:    []byte("test"),
				Accesses: &gethcore.AccessList{},
			},
			gethcore.NewEIP2930Signer(s.chainID),
			func(tx *evm.MsgEthereumTx) { tx.From = "" },
			false,
		},
		{
			"from address ≠ signer address",
			&evm.EvmTxArgs{
				ChainID:  s.chainID,
				Nonce:    0,
				To:       &s.to,
				GasLimit: 100000,
				Input:    []byte("test"),
				Accesses: &gethcore.AccessList{},
			},
			gethcore.NewEIP2930Signer(s.chainID),
			func(tx *evm.MsgEthereumTx) { tx.From = s.to.Hex() },
			false,
		},
	}

	for i, tc := range testCases {
		tx := evm.NewTx(tc.txParams)
		tc.malleate(tx)
		err := tx.Sign(tc.ethSigner, s.signer)
		if tc.expectPass {
			s.Require().NoError(err, "valid test %d failed: %s", i, tc.msg)

			sender, err := tx.GetSender(s.chainID)
			s.Require().NoError(err, tc.msg)
			s.Require().Equal(tx.From, sender.Hex(), tc.msg)
		} else {
			s.Require().Error(err, "invalid test %d passed: %s", i, tc.msg)
		}
	}
}

func (s *MsgsSuite) TestMsgEthereumTx_Getters() {
	evmTx := &evm.EvmTxArgs{
		ChainID:  s.chainID,
		Nonce:    0,
		To:       &s.to,
		GasLimit: 50,
		GasPrice: s.hundredBigInt,
		Accesses: &gethcore.AccessList{},
	}
	testCases := []struct {
		name      string
		ethSigner gethcore.Signer
		exp       *big.Int
	}{
		{
			"get fee - pass",

			gethcore.NewEIP2930Signer(s.chainID),
			big.NewInt(5000),
		},
		{
			"get fee - fail: nil data",
			gethcore.NewEIP2930Signer(s.chainID),
			nil,
		},
		{
			"get effective fee - pass",

			gethcore.NewEIP2930Signer(s.chainID),
			big.NewInt(5000),
		},
		{
			"get effective fee - fail: nil data",
			gethcore.NewEIP2930Signer(s.chainID),
			nil,
		},
		{
			"get gas - pass",
			gethcore.NewEIP2930Signer(s.chainID),
			big.NewInt(50),
		},
		{
			"get gas - fail: nil data",
			gethcore.NewEIP2930Signer(s.chainID),
			big.NewInt(0),
		},
	}

	var fee, effFee *big.Int
	for _, tc := range testCases {
		tx := evm.NewTx(evmTx)
		if strings.Contains(tc.name, "nil data") {
			tx.Data = nil
		}
		switch {
		case strings.Contains(tc.name, "get fee"):
			fee = tx.GetFee()
			s.Require().Equal(tc.exp, fee)
		case strings.Contains(tc.name, "get effective fee"):
			effFee = tx.GetEffectiveFee(big.NewInt(0))
			s.Require().Equal(tc.exp, effFee)
		case strings.Contains(tc.name, "get gas"):
			gas := tx.GetGas()
			s.Require().Equal(tc.exp.Uint64(), gas)
		}
	}
}

func (s *MsgsSuite) TestFromEthereumTx() {
	privkey, _ := ethsecp256k1.GenerateKey()
	ethPriv, err := privkey.ToECDSA()
	s.Require().NoError(err)

	// 10^80 is more than 256 bits
	//nolint:all
	exp_10_80 := new(big.Int).Mul(big.NewInt(1), new(big.Int).Exp(big.NewInt(10), big.NewInt(80), nil))

	testCases := []struct {
		msg        string
		expectPass bool
		buildTx    func() *gethcore.Transaction
	}{
		{"success, normal tx", true, func() *gethcore.Transaction {
			tx := gethcore.NewTx(&gethcore.AccessListTx{
				Nonce:    0,
				Data:     nil,
				To:       &s.to,
				Value:    big.NewInt(10),
				GasPrice: big.NewInt(1),
				Gas:      21000,
			})
			tx, err := gethcore.SignTx(tx, gethcore.NewEIP2930Signer(s.chainID), ethPriv)
			s.Require().NoError(err)
			return tx
		}},
		{"success, DynamicFeeTx", true, func() *gethcore.Transaction {
			tx := gethcore.NewTx(&gethcore.DynamicFeeTx{
				Nonce: 0,
				Data:  nil,
				To:    &s.to,
				Value: big.NewInt(10),
				Gas:   21000,
			})
			tx, err := gethcore.SignTx(tx, gethcore.NewLondonSigner(s.chainID), ethPriv)
			s.Require().NoError(err)
			return tx
		}},
		{"fail, value bigger than 256bits - AccessListTx", false, func() *gethcore.Transaction {
			tx := gethcore.NewTx(&gethcore.AccessListTx{
				Nonce:    0,
				Data:     nil,
				To:       &s.to,
				Value:    exp_10_80,
				GasPrice: big.NewInt(1),
				Gas:      21000,
			})
			tx, err := gethcore.SignTx(tx, gethcore.NewEIP2930Signer(s.chainID), ethPriv)
			s.Require().NoError(err)
			return tx
		}},
		{"fail, gas price bigger than 256bits - AccessListTx", false, func() *gethcore.Transaction {
			tx := gethcore.NewTx(&gethcore.AccessListTx{
				Nonce:    0,
				Data:     nil,
				To:       &s.to,
				Value:    big.NewInt(1),
				GasPrice: exp_10_80,
				Gas:      21000,
			})
			tx, err := gethcore.SignTx(tx, gethcore.NewEIP2930Signer(s.chainID), ethPriv)
			s.Require().NoError(err)
			return tx
		}},
		{"fail, value bigger than 256bits - LegacyTx", false, func() *gethcore.Transaction {
			tx := gethcore.NewTx(&gethcore.LegacyTx{
				Nonce:    0,
				Data:     nil,
				To:       &s.to,
				Value:    exp_10_80,
				GasPrice: big.NewInt(1),
				Gas:      21000,
			})
			tx, err := gethcore.SignTx(tx, gethcore.NewEIP2930Signer(s.chainID), ethPriv)
			s.Require().NoError(err)
			return tx
		}},
		{"fail, gas price bigger than 256bits - LegacyTx", false, func() *gethcore.Transaction {
			tx := gethcore.NewTx(&gethcore.LegacyTx{
				Nonce:    0,
				Data:     nil,
				To:       &s.to,
				Value:    big.NewInt(1),
				GasPrice: exp_10_80,
				Gas:      21000,
			})
			tx, err := gethcore.SignTx(tx, gethcore.NewEIP2930Signer(s.chainID), ethPriv)
			s.Require().NoError(err)
			return tx
		}},
	}

	for _, tc := range testCases {
		ethTx := tc.buildTx()
		tx := &evm.MsgEthereumTx{}
		err := tx.FromEthereumTx(ethTx)
		if tc.expectPass {
			s.Require().NoError(err)

			// round-trip test
			s.Require().NoError(assertEqualTxs(tx.AsTransaction(), ethTx))
		} else {
			s.Require().Error(err)
		}
	}
}

// TestTxEncoding tests serializing/de-serializing to/from rlp and JSON.
// adapted from go-ethereum
func (s *MsgsSuite) TestTxEncoding() {
	key, err := crypto.GenerateKey()
	if err != nil {
		s.T().Fatalf("could not generate key: %v", err)
	}
	var (
		signer    = gethcore.NewEIP2930Signer(common.Big1)
		addr      = common.HexToAddress("0x0000000000000000000000000000000000000001")
		recipient = common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d87")
		accesses  = gethcore.AccessList{{Address: addr, StorageKeys: []common.Hash{{0}}}}
	)
	for i := uint64(0); i < 500; i++ {
		var txdata gethcore.TxData
		switch i % 5 {
		case 0:
			// Legacy tx.
			txdata = &gethcore.LegacyTx{
				Nonce:    i,
				To:       &recipient,
				Gas:      1,
				GasPrice: big.NewInt(2),
				Data:     []byte("abcdef"),
			}
		case 1:
			// Legacy tx contract creation.
			txdata = &gethcore.LegacyTx{
				Nonce:    i,
				Gas:      1,
				GasPrice: big.NewInt(2),
				Data:     []byte("abcdef"),
			}
		case 2:
			// Tx with non-zero access list.
			txdata = &gethcore.AccessListTx{
				ChainID:    big.NewInt(1),
				Nonce:      i,
				To:         &recipient,
				Gas:        123457,
				GasPrice:   big.NewInt(10),
				AccessList: accesses,
				Data:       []byte("abcdef"),
			}
		case 3:
			// Tx with empty access list.
			txdata = &gethcore.AccessListTx{
				ChainID:  big.NewInt(1),
				Nonce:    i,
				To:       &recipient,
				Gas:      123457,
				GasPrice: big.NewInt(10),
				Data:     []byte("abcdef"),
			}
		case 4:
			// Contract creation with access list.
			txdata = &gethcore.AccessListTx{
				ChainID:    big.NewInt(1),
				Nonce:      i,
				Gas:        123457,
				GasPrice:   big.NewInt(10),
				AccessList: accesses,
			}
		}
		tx, err := gethcore.SignNewTx(key, signer, txdata)
		if err != nil {
			s.T().Fatalf("could not sign transaction: %v", err)
		}
		// RLP
		parsedTx, err := encodeDecodeBinary(tx)
		if err != nil {
			s.T().Fatal(err)
		}
		err = assertEqualTxs(parsedTx.AsTransaction(), tx)
		s.Require().NoError(err)
	}
}

func encodeDecodeBinary(tx *gethcore.Transaction) (*evm.MsgEthereumTx, error) {
	data, err := tx.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("rlp encoding failed: %v", err)
	}
	parsedTx := &evm.MsgEthereumTx{}
	if err := parsedTx.UnmarshalBinary(data); err != nil {
		return nil, fmt.Errorf("rlp decoding failed: %v", err)
	}
	return parsedTx, nil
}

func assertEqualTxs(orig *gethcore.Transaction, cpy *gethcore.Transaction) error {
	// compare nonce, price, gaslimit, recipient, amount, payload, V, R, S
	if want, got := orig.Hash(), cpy.Hash(); want != got {
		return fmt.Errorf("parsed tx differs from original tx, want %v, got %v", want, got)
	}
	if want, got := orig.ChainId(), cpy.ChainId(); want.Cmp(got) != 0 {
		return fmt.Errorf("invalid chain id, want %d, got %d", want, got)
	}
	if orig.AccessList() != nil {
		if !reflect.DeepEqual(orig.AccessList(), cpy.AccessList()) {
			return fmt.Errorf("access list wrong")
		}
	}
	return nil
}

func (s *MsgsSuite) TestUnwrapEthererumMsg() {
	_, err := evm.UnwrapEthereumMsg(nil, common.Hash{})
	s.NotNil(err)

	encodingConfig := encoding.MakeConfig(app.ModuleBasics)
	clientCtx := client.Context{}.WithTxConfig(encodingConfig.TxConfig)
	builder, _ := clientCtx.TxConfig.NewTxBuilder().(authtx.ExtensionOptionsTxBuilder)

	tx := builder.GetTx().(sdk.Tx)
	_, err = evm.UnwrapEthereumMsg(&tx, common.Hash{})
	s.NotNil(err)

	evmTxParams := &evm.EvmTxArgs{
		ChainID:  big.NewInt(1),
		Nonce:    0,
		To:       &common.Address{},
		Amount:   big.NewInt(0),
		GasLimit: 0,
		GasPrice: big.NewInt(0),
		Input:    []byte{},
	}

	msg := evm.NewTx(evmTxParams)
	err = builder.SetMsgs(msg)
	s.Nil(err)

	tx = builder.GetTx().(sdk.Tx)
	unwrappedMsg, err := evm.UnwrapEthereumMsg(&tx, msg.AsTransaction().Hash())
	s.Nil(err)
	s.Equal(unwrappedMsg, msg)
}

func (s *MsgsSuite) TestTransactionLogsEncodeDecode() {
	addr := evmtest.NewEthPrivAcc().EthAddr.String()

	txLogs := evm.TransactionLogs{
		Hash: common.BytesToHash([]byte("tx_hash")).String(),
		Logs: []*evm.Log{
			{
				Address:     addr,
				Topics:      []string{common.BytesToHash([]byte("topic")).String()},
				Data:        []byte("data"),
				BlockNumber: 1,
				TxHash:      common.BytesToHash([]byte("tx_hash")).String(),
				TxIndex:     1,
				BlockHash:   common.BytesToHash([]byte("block_hash")).String(),
				Index:       1,
				Removed:     false,
			},
		},
	}

	txLogsEncoded, encodeErr := evm.EncodeTransactionLogs(&txLogs)
	s.Nil(encodeErr)

	txLogsEncodedDecoded, decodeErr := evm.DecodeTransactionLogs(txLogsEncoded)
	s.Nil(decodeErr)
	s.Equal(txLogs, txLogsEncodedDecoded)
}
