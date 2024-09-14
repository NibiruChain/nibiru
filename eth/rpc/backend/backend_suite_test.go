package backend_test

import (
	"math/big"
	"testing"

	"crypto/ecdsa"

	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/eth/rpc/backend"

	"github.com/NibiruChain/nibiru/v2/x/common/testutil/genesis"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testnetwork"
)

// --------------------------------------------------------------------
// ------------------------------------------------ Imports using mocks
// --------------------------------------------------------------------
// import (
// 	"bufio"
// 	"math/big"
// 	"os"
// 	"testing"

// 	"crypto/ecdsa"

// 	tmrpctypes "github.com/cometbft/cometbft/rpc/core/types"
// 	"github.com/cosmos/cosmos-sdk/crypto/keyring"
// 	sdk "github.com/cosmos/cosmos-sdk/types"
// 	"github.com/ethereum/go-ethereum/common"
// 	gethcommon "github.com/ethereum/go-ethereum/common"
// 	gethcore "github.com/ethereum/go-ethereum/core/types"
// 	"github.com/ethereum/go-ethereum/crypto"
// 	"github.com/stretchr/testify/suite"

// 	"github.com/NibiruChain/nibiru/v2/app"
// 	"github.com/NibiruChain/nibiru/v2/app/appconst"
// 	"github.com/NibiruChain/nibiru/v2/eth"
// 	"github.com/NibiruChain/nibiru/v2/eth/crypto/hd"
// 	"github.com/NibiruChain/nibiru/v2/eth/encoding"
// 	"github.com/NibiruChain/nibiru/v2/eth/rpc"
// 	"github.com/NibiruChain/nibiru/v2/eth/rpc/backend"

// 	"github.com/NibiruChain/nibiru/v2/x/common/testutil/genesis"
// 	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testapp"
// 	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testnetwork"
// 	"github.com/NibiruChain/nibiru/v2/x/evm"
// 	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
// )

type BackendSuite struct {
	suite.Suite

	// from   common.Address
	// acc    sdk.AccAddress
	// signer keyring.Signer

	cfg                 testnetwork.Config
	network             *testnetwork.Network
	node                *testnetwork.Validator
	fundedAccPrivateKey *ecdsa.PrivateKey
	fundedAccEthAddr    gethcommon.Address
	fundedAccNibiAddr   sdk.AccAddress
	backend             *backend.Backend
	ethChainID          *big.Int
}

func TestBackendSuite(t *testing.T) {
	suite.Run(t, new(BackendSuite))
}

const ChainID = eth.EIP155ChainID_Testnet

func (s *BackendSuite) SetupSuite() {
	// testutil.BeforeIntegrationSuite(s.T())
	testapp.EnsureNibiruPrefix()

	genState := genesis.NewTestGenesisState(app.MakeEncodingConfig())
	homeDir := s.T().TempDir()
	s.cfg = testnetwork.BuildNetworkConfig(genState)
	network, err := testnetwork.New(s.T(), homeDir, s.cfg)
	s.Require().NoError(err)
	s.network = network
	s.node = network.Validators[0]
	s.ethChainID = appconst.GetEthChainID(s.node.ClientCtx.ChainID)
	s.backend = s.node.EthRpcBackend

	testAccPrivateKey, _ := crypto.GenerateKey()
	s.fundedAccPrivateKey = testAccPrivateKey
	s.fundedAccEthAddr = crypto.PubkeyToAddress(testAccPrivateKey.PublicKey)
	s.fundedAccNibiAddr = eth.EthAddrToNibiruAddr(s.fundedAccEthAddr)

	funds := sdk.NewCoins(sdk.NewInt64Coin(eth.EthBaseDenom, 100_000_000))

	txResp, err := testnetwork.FillWalletFromValidator(
		s.fundedAccNibiAddr, funds, s.node, eth.EthBaseDenom,
	)
	s.Require().NoError(err, txResp.TxHash)
	s.NoError(s.network.WaitForNextBlock())
}

// // buildEthereumTx returns an example legacy Ethereum transaction
// func (s *BackendSuite) buildEthereumTx() (*evm.MsgEthereumTx, []byte) {
// 	ethTxParams := evm.EvmTxArgs{
// 		ChainID:  s.ethChainID,
// 		Nonce:    uint64(0),
// 		To:       &common.Address{},
// 		Amount:   big.NewInt(0),
// 		GasLimit: 100000,
// 		GasPrice: big.NewInt(1),
// 	}
// 	msgEthereumTx := evm.NewTx(&ethTxParams)

// 	// A valid msg should have empty `From`
// 	msgEthereumTx.From = s.from.Hex()

// 	txBuilder := s.node.ClientCtx.TxConfig.NewTxBuilder()
// 	err := txBuilder.SetMsgs(msgEthereumTx)
// 	s.Require().NoError(err)

// 	bz, err := s.node.ClientCtx.TxConfig.TxEncoder()(txBuilder.GetTx())
// 	s.Require().NoError(err)
// 	return msgEthereumTx, bz
// }

// // buildFormattedBlock returns a formatted block for testing
// func (s *BackendSuite) buildFormattedBlock(
// 	blockRes *tmrpctypes.ResultBlockResults,
// 	resBlock *tmrpctypes.ResultBlock,
// 	fullTx bool,
// 	tx *evm.MsgEthereumTx,
// 	validator sdk.AccAddress,
// 	baseFee *big.Int,
// ) map[string]interface{} {
// 	header := resBlock.Block.Header
// 	gasLimit := int64(^uint32(0)) // for `MaxGas = -1` (DefaultConsensusParams)
// 	gasUsed := new(big.Int).SetUint64(uint64(blockRes.TxsResults[0].GasUsed))

// 	root := common.Hash{}.Bytes()
// 	receipt := gethcore.NewReceipt(root, false, gasUsed.Uint64())
// 	bloom := gethcore.CreateBloom(gethcore.Receipts{receipt})

// 	ethRPCTxs := []interface{}{}
// 	if tx != nil {
// 		if fullTx {
// 			rpcTx, err := rpc.NewRPCTxFromEthTx(
// 				tx.AsTransaction(),
// 				common.BytesToHash(header.Hash()),
// 				uint64(header.Height),
// 				uint64(0),
// 				baseFee,
// 				s.ethChainID,
// 			)
// 			s.Require().NoError(err)
// 			ethRPCTxs = []interface{}{rpcTx}
// 		} else {
// 			ethRPCTxs = []interface{}{common.HexToHash(tx.Hash)}
// 		}
// 	}

// 	return rpc.FormatBlock(
// 		header,
// 		resBlock.Block.Size(),
// 		gasLimit,
// 		gasUsed,
// 		ethRPCTxs,
// 		bloom,
// 		common.BytesToAddress(validator.Bytes()),
// 		baseFee,
// 	)
// }

// func (s *BackendSuite) generateTestKeyring(clientDir string) (keyring.Keyring, error) {
// 	buf := bufio.NewReader(os.Stdin)
// 	encCfg := encoding.MakeConfig(app.ModuleBasics)
// 	return keyring.New(
// 		sdk.KeyringServiceName(), // appName
// 		keyring.BackendTest,      // backend
// 		clientDir,                // rootDir
// 		buf,                      // userInput
// 		encCfg.Codec,             // codec
// 		[]keyring.Option{hd.EthSecp256k1Option()}...,
// 	)
// }

// func (s *BackendSuite) signAndEncodeEthTx(msgEthereumTx *evm.MsgEthereumTx) []byte {
// 	ethAcc := evmtest.NewEthPrivAcc()
// 	from, priv := ethAcc.EthAddr, ethAcc.PrivKey
// 	signer := evmtest.NewSigner(priv)

// 	ethSigner := gethcore.LatestSigner(s.backend.ChainConfig())
// 	msgEthereumTx.From = from.String()
// 	err := msgEthereumTx.Sign(ethSigner, signer)
// 	s.Require().NoError(err)

// 	tx, err := msgEthereumTx.BuildTx(s.node.ClientCtx.TxConfig.NewTxBuilder(), eth.EthBaseDenom)
// 	s.Require().NoError(err)

// 	txEncoder := s.node.ClientCtx.TxConfig.TxEncoder()
// 	txBz, err := txEncoder(tx)
// 	s.Require().NoError(err)

// 	return txBz
// }
