package backend

import (
	"bufio"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	dbm "github.com/cometbft/cometbft-db"

	tmrpctypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/server"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/eth"
	"github.com/NibiruChain/nibiru/eth/crypto/hd"
	"github.com/NibiruChain/nibiru/eth/encoding"
	"github.com/NibiruChain/nibiru/eth/indexer"
	"github.com/NibiruChain/nibiru/eth/rpc"
	"github.com/NibiruChain/nibiru/eth/rpc/backend/mocks"
	"github.com/NibiruChain/nibiru/x/evm"
	evmtest "github.com/NibiruChain/nibiru/x/evm/evmtest"
)

type BackendSuite struct {
	suite.Suite

	backend *Backend
	from    common.Address
	acc     sdk.AccAddress
	signer  keyring.Signer
}

func TestBackendSuite(t *testing.T) {
	suite.Run(t, new(BackendSuite))
}

const ChainID = eth.EIP155ChainID_Testnet + "-1"

// SetupTest is executed before every BackendTestSuite test
func (s *BackendSuite) SetupTest() {
	ctx := server.NewDefaultContext()
	ctx.Viper.Set("telemetry.global-labels", []interface{}{})

	baseDir := s.T().TempDir()
	nodeDirName := "node"
	clientDir := filepath.Join(baseDir, nodeDirName, "nibirucli")
	keyRing, err := s.generateTestKeyring(clientDir)
	if err != nil {
		panic(err)
	}

	// Create Account with set sequence
	s.acc = sdk.AccAddress(evmtest.NewEthAddr().Bytes())
	accounts := map[string]client.TestAccount{}
	accounts[s.acc.String()] = client.TestAccount{
		Address: s.acc,
		Num:     uint64(1),
		Seq:     uint64(1),
	}

	from, priv := evmtest.PrivKeyEth()
	s.from = from
	s.signer = evmtest.NewSigner(priv)
	s.Require().NoError(err)

	encCfg := encoding.MakeConfig(app.ModuleBasics)
	evm.RegisterInterfaces(encCfg.InterfaceRegistry)
	eth.RegisterInterfaces(encCfg.InterfaceRegistry)
	clientCtx := client.Context{}.WithChainID(ChainID).
		WithHeight(1).
		WithTxConfig(encCfg.TxConfig).
		WithKeyringDir(clientDir).
		WithKeyring(keyRing).
		WithAccountRetriever(client.TestAccountRetriever{Accounts: accounts})

	allowUnprotectedTxs := false
	idxer := indexer.NewKVIndexer(dbm.NewMemDB(), ctx.Logger, clientCtx)

	s.backend = NewBackend(ctx, ctx.Logger, clientCtx, allowUnprotectedTxs, idxer)
	s.backend.cfg.JSONRPC.GasCap = 0
	s.backend.cfg.JSONRPC.EVMTimeout = 0
	s.backend.queryClient.QueryClient = mocks.NewEVMQueryClient(s.T())
	s.backend.clientCtx.Client = mocks.NewClient(s.T())
	s.backend.ctx = rpc.NewContextWithHeight(1)

	s.backend.clientCtx.Codec = encCfg.Codec
}

// buildEthereumTx returns an example legacy Ethereum transaction
func (s *BackendSuite) buildEthereumTx() (*evm.MsgEthereumTx, []byte) {
	ethTxParams := evm.EvmTxArgs{
		ChainID:  s.backend.chainID,
		Nonce:    uint64(0),
		To:       &common.Address{},
		Amount:   big.NewInt(0),
		GasLimit: 100000,
		GasPrice: big.NewInt(1),
	}
	msgEthereumTx := evm.NewTx(&ethTxParams)

	// A valid msg should have empty `From`
	msgEthereumTx.From = s.from.Hex()

	txBuilder := s.backend.clientCtx.TxConfig.NewTxBuilder()
	err := txBuilder.SetMsgs(msgEthereumTx)
	s.Require().NoError(err)

	bz, err := s.backend.clientCtx.TxConfig.TxEncoder()(txBuilder.GetTx())
	s.Require().NoError(err)
	return msgEthereumTx, bz
}

// buildFormattedBlock returns a formatted block for testing
func (s *BackendSuite) buildFormattedBlock(
	blockRes *tmrpctypes.ResultBlockResults,
	resBlock *tmrpctypes.ResultBlock,
	fullTx bool,
	tx *evm.MsgEthereumTx,
	validator sdk.AccAddress,
	baseFee *big.Int,
) map[string]interface{} {
	header := resBlock.Block.Header
	gasLimit := int64(^uint32(0)) // for `MaxGas = -1` (DefaultConsensusParams)
	gasUsed := new(big.Int).SetUint64(uint64(blockRes.TxsResults[0].GasUsed))

	root := common.Hash{}.Bytes()
	receipt := gethcore.NewReceipt(root, false, gasUsed.Uint64())
	bloom := gethcore.CreateBloom(gethcore.Receipts{receipt})

	ethRPCTxs := []interface{}{}
	if tx != nil {
		if fullTx {
			rpcTx, err := rpc.NewRPCTxFromEthTx(
				tx.AsTransaction(),
				common.BytesToHash(header.Hash()),
				uint64(header.Height),
				uint64(0),
				baseFee,
				s.backend.chainID,
			)
			s.Require().NoError(err)
			ethRPCTxs = []interface{}{rpcTx}
		} else {
			ethRPCTxs = []interface{}{common.HexToHash(tx.Hash)}
		}
	}

	return rpc.FormatBlock(
		header,
		resBlock.Block.Size(),
		gasLimit,
		gasUsed,
		ethRPCTxs,
		bloom,
		common.BytesToAddress(validator.Bytes()),
		baseFee,
	)
}

func (s *BackendSuite) generateTestKeyring(clientDir string) (keyring.Keyring, error) {
	buf := bufio.NewReader(os.Stdin)
	encCfg := encoding.MakeConfig(app.ModuleBasics)
	return keyring.New(
		sdk.KeyringServiceName(), // appName
		keyring.BackendTest,      // backend
		clientDir,                // rootDir
		buf,                      // userInput
		encCfg.Codec,             // codec
		[]keyring.Option{hd.EthSecp256k1Option()}...,
	)
}

func (s *BackendSuite) signAndEncodeEthTx(msgEthereumTx *evm.MsgEthereumTx) []byte {
	from, priv := evmtest.PrivKeyEth()
	signer := evmtest.NewSigner(priv)

	queryClient := s.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
	RegisterParamsWithoutHeader(queryClient, 1)

	ethSigner := gethcore.LatestSigner(s.backend.ChainConfig())
	msgEthereumTx.From = from.String()
	err := msgEthereumTx.Sign(ethSigner, signer)
	s.Require().NoError(err)

	tx, err := msgEthereumTx.BuildTx(s.backend.clientCtx.TxConfig.NewTxBuilder(), eth.EthBaseDenom)
	s.Require().NoError(err)

	txEncoder := s.backend.clientCtx.TxConfig.TxEncoder()
	txBz, err := txEncoder(tx)
	s.Require().NoError(err)

	return txBz
}
