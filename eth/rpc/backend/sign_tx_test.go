package backend

import (
	"fmt"

	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/crypto"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	goethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"google.golang.org/grpc/metadata"

	"github.com/NibiruChain/nibiru/v2/eth/crypto/ethsecp256k1"
	"github.com/NibiruChain/nibiru/v2/eth/rpc/backend/mocks"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	evmtest "github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
)

func (s *BackendSuite) TestSendTransaction() {
	gasPrice := new(hexutil.Big)
	gas := hexutil.Uint64(1)
	zeroGas := hexutil.Uint64(0)
	toAddr := evmtest.NewEthPrivAcc().EthAddr
	priv, _ := ethsecp256k1.GenerateKey()
	from := common.BytesToAddress(priv.PubKey().Address().Bytes())
	nonce := hexutil.Uint64(1)
	baseFee := math.NewInt(1)
	callArgsDefault := evm.JsonTxArgs{
		From:     &from,
		To:       &toAddr,
		GasPrice: gasPrice,
		Gas:      &gas,
		Nonce:    &nonce,
	}

	hash := common.Hash{}

	testCases := []struct {
		name         string
		registerMock func()
		args         evm.JsonTxArgs
		expHash      common.Hash
		expPass      bool
	}{
		{
			"fail - Can't find account in Keyring",
			func() {},
			evm.JsonTxArgs{},
			hash,
			false,
		},
		{
			"fail - Block error can't set Tx defaults",
			func() {
				var header metadata.MD
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				client := s.backend.clientCtx.Client.(*mocks.Client)
				armor := crypto.EncryptArmorPrivKey(priv, "", "eth_secp256k1")
				err := s.backend.clientCtx.Keyring.ImportPrivKey("test_key", armor, "")
				s.Require().NoError(err)
				RegisterParams(queryClient, &header, 1)
				RegisterBlockError(client, 1)
			},
			callArgsDefault,
			hash,
			false,
		},
		{
			"fail - Cannot validate transaction gas set to 0",
			func() {
				var header metadata.MD
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				client := s.backend.clientCtx.Client.(*mocks.Client)
				armor := crypto.EncryptArmorPrivKey(priv, "", "eth_secp256k1")
				err := s.backend.clientCtx.Keyring.ImportPrivKey("test_key", armor, "")
				s.Require().NoError(err)
				RegisterParams(queryClient, &header, 1)
				_, err = RegisterBlock(client, 1, nil)
				s.Require().NoError(err)
				_, err = RegisterBlockResults(client, 1)
				s.Require().NoError(err)
				RegisterBaseFee(queryClient, baseFee)
				RegisterParamsWithoutHeader(queryClient, 1)
			},
			evm.JsonTxArgs{
				From:     &from,
				To:       &toAddr,
				GasPrice: gasPrice,
				Gas:      &zeroGas,
				Nonce:    &nonce,
			},
			hash,
			false,
		},
		{
			"fail - Cannot broadcast transaction",
			func() {
				client, txBytes := broadcastTx(s, priv, baseFee, callArgsDefault)
				RegisterBroadcastTxError(client, txBytes)
			},
			callArgsDefault,
			common.Hash{},
			false,
		},
		{
			"pass - Return the transaction hash",
			func() {
				client, txBytes := broadcastTx(s, priv, baseFee, callArgsDefault)
				RegisterBroadcastTx(client, txBytes)
			},
			callArgsDefault,
			hash,
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("case %s", tc.name), func() {
			s.SetupTest() // reset test and queries
			tc.registerMock()

			if tc.expPass {
				// Sign the transaction and get the hash
				queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterParamsWithoutHeader(queryClient, 1)
				ethSigner := gethcore.LatestSigner(s.backend.ChainConfig())
				msg := callArgsDefault.ToTransaction()
				err := msg.Sign(ethSigner, s.backend.clientCtx.Keyring)
				s.Require().NoError(err)
				tc.expHash = msg.AsTransaction().Hash()
			}
			responseHash, err := s.backend.SendTransaction(tc.args)
			if tc.expPass {
				s.Require().NoError(err)
				s.Require().Equal(tc.expHash, responseHash)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *BackendSuite) TestSign() {
	ethAcc := evmtest.NewEthPrivAcc()
	from, priv := ethAcc.EthAddr, ethAcc.PrivKey

	testCases := []struct {
		name         string
		registerMock func()
		fromAddr     common.Address
		inputBz      hexutil.Bytes
		expPass      bool
	}{
		{
			"fail - can't find key in Keyring",
			func() {},
			from,
			nil,
			false,
		},
		{
			"pass - sign nil data",
			func() {
				armor := crypto.EncryptArmorPrivKey(priv, "", "eth_secp256k1")
				err := s.backend.clientCtx.Keyring.ImportPrivKey("test_key", armor, "")
				s.Require().NoError(err)
			},
			from,
			nil,
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("case %s", tc.name), func() {
			s.SetupTest() // reset test and queries
			tc.registerMock()

			responseBz, err := s.backend.Sign(tc.fromAddr, tc.inputBz)
			if tc.expPass {
				signature, _, err := s.backend.clientCtx.Keyring.SignByAddress((sdk.AccAddress)(from.Bytes()), tc.inputBz)
				signature[goethcrypto.RecoveryIDOffset] += 27
				s.Require().NoError(err)
				s.Require().Equal((hexutil.Bytes)(signature), responseBz)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *BackendSuite) TestSignTypedData() {
	ethAcc := evmtest.NewEthPrivAcc()
	from, priv := ethAcc.EthAddr, ethAcc.PrivKey
	testCases := []struct {
		name           string
		registerMock   func()
		fromAddr       common.Address
		inputTypedData apitypes.TypedData
		expPass        bool
	}{
		{
			"fail - can't find key in Keyring",
			func() {},
			from,
			apitypes.TypedData{},
			false,
		},
		{
			"fail - empty TypeData",
			func() {
				armor := crypto.EncryptArmorPrivKey(priv, "", "eth_secp256k1")
				err := s.backend.clientCtx.Keyring.ImportPrivKey("test_key", armor, "")
				s.Require().NoError(err)
			},
			from,
			apitypes.TypedData{},
			false,
		},
		// TODO: Generate a TypedData msg
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("case %s", tc.name), func() {
			s.SetupTest() // reset test and queries
			tc.registerMock()

			responseBz, err := s.backend.SignTypedData(tc.fromAddr, tc.inputTypedData)

			if tc.expPass {
				sigHash, _, _ := apitypes.TypedDataAndHash(tc.inputTypedData)
				signature, _, err := s.backend.clientCtx.Keyring.SignByAddress((sdk.AccAddress)(from.Bytes()), sigHash)
				signature[goethcrypto.RecoveryIDOffset] += 27
				s.Require().NoError(err)
				s.Require().Equal((hexutil.Bytes)(signature), responseBz)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func broadcastTx(
	s *BackendSuite,
	priv *ethsecp256k1.PrivKey,
	baseFee math.Int,
	callArgsDefault evm.JsonTxArgs,
) (client *mocks.Client, txBytes []byte) {
	var header metadata.MD
	queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
	client = s.backend.clientCtx.Client.(*mocks.Client)
	armor := crypto.EncryptArmorPrivKey(priv, "", "eth_secp256k1")
	_ = s.backend.clientCtx.Keyring.ImportPrivKey("test_key", armor, "")
	RegisterParams(queryClient, &header, 1)
	_, err := RegisterBlock(client, 1, nil)
	s.Require().NoError(err)
	_, err = RegisterBlockResults(client, 1)
	s.Require().NoError(err)
	RegisterBaseFee(queryClient, baseFee)
	RegisterParamsWithoutHeader(queryClient, 1)
	ethSigner := gethcore.LatestSigner(s.backend.ChainConfig())
	msg := callArgsDefault.ToTransaction()
	err = msg.Sign(ethSigner, s.backend.clientCtx.Keyring)
	s.Require().NoError(err)
	tx, _ := msg.BuildTx(s.backend.clientCtx.TxConfig.NewTxBuilder(), evm.DefaultEVMDenom)
	txEncoder := s.backend.clientCtx.TxConfig.TxEncoder()
	txBytes, _ = txEncoder(tx)
	return client, txBytes
}
