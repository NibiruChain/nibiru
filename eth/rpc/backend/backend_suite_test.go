package backend_test

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/client/flags"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/v2/x/common/testutil"

	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"

	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"

	"github.com/NibiruChain/nibiru/v2/eth/rpc"
	"github.com/NibiruChain/nibiru/v2/x/evm"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/eth/rpc/backend"

	"github.com/NibiruChain/nibiru/v2/x/common/testutil/genesis"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testnetwork"
)

// testMutex is used to synchronize the tests which are broadcasting transactions concurrently
var testMutex sync.Mutex

var (
	recipient    = evmtest.NewEthPrivAcc().EthAddr
	amountToSend = evm.NativeToWei(big.NewInt(1))
)

var (
	transferTxBlockNumber rpc.BlockNumber
	transferTxBlockHash   gethcommon.Hash
	transferTxHash        gethcommon.Hash
)

var (
	testContractAddress       gethcommon.Address
	deployContractBlockNumber rpc.BlockNumber
)

type BackendSuite struct {
	suite.Suite
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
	testutil.RetrySuiteRunIfDbClosed(t, func() {
		suite.Run(t, new(BackendSuite))
	}, 2)
}

func (s *BackendSuite) SetupSuite() {
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
	s.Require().NoError(err)
	s.Require().NotNil(txResp.TxHash)
	s.NoError(s.network.WaitForNextBlock())

	// Send Transfer TX and use the results in the tests
	s.Require().NoError(err)
	transferTxHash = s.SendNibiViaEthTransfer(recipient, amountToSend, true /*waitForNextBlock*/)
	blockNumber, blockHash, _ := WaitForReceipt(s, transferTxHash)
	s.Require().NotNil(blockNumber)
	s.Require().NotNil(blockHash)
	transferTxBlockNumber = rpc.NewBlockNumber(blockNumber)
	transferTxBlockHash = *blockHash

	// Deploy test erc20 contract
	deployContractTxHash, contractAddress := s.DeployTestContract(true)
	testContractAddress = contractAddress
	blockNumber, blockHash, _ = WaitForReceipt(s, deployContractTxHash)
	s.Require().NotNil(blockNumber)
	s.Require().NotNil(blockHash)
	deployContractBlockNumber = rpc.NewBlockNumber(blockNumber)
}

// SendNibiViaEthTransfer sends nibi using the eth rpc backend
func (s *BackendSuite) SendNibiViaEthTransfer(
	to gethcommon.Address,
	amount *big.Int,
	waitForNextBlock bool,
) gethcommon.Hash {
	nonce := s.getCurrentNonce(s.fundedAccEthAddr)
	return SendTransaction(
		s,
		&gethcore.LegacyTx{
			To:       &to,
			Nonce:    uint64(nonce),
			Value:    amount,
			Gas:      params.TxGas,
			GasPrice: big.NewInt(1),
		},
		waitForNextBlock,
	)
}

// DeployTestContract deploys test erc20 contract
func (s *BackendSuite) DeployTestContract(waitForNextBlock bool) (gethcommon.Hash, gethcommon.Address) {
	packedArgs, err := embeds.SmartContract_TestERC20.ABI.Pack("")
	s.Require().NoError(err)
	bytecodeForCall := append(embeds.SmartContract_TestERC20.Bytecode, packedArgs...)
	nonce := s.getCurrentNonce(s.fundedAccEthAddr)
	s.Require().NoError(err)

	txHash := SendTransaction(
		s,
		&gethcore.LegacyTx{
			Nonce:    uint64(nonce),
			Data:     bytecodeForCall,
			Gas:      1_500_000,
			GasPrice: big.NewInt(1),
		},
		waitForNextBlock,
	)
	contractAddr := crypto.CreateAddress(s.fundedAccEthAddr, nonce)
	return txHash, contractAddr
}

// SendTransaction signs and sends raw ethereum transaction
func SendTransaction(s *BackendSuite, tx *gethcore.LegacyTx, waitForNextBlock bool) gethcommon.Hash {
	signer := gethcore.LatestSignerForChainID(s.ethChainID)
	signedTx, err := gethcore.SignNewTx(s.fundedAccPrivateKey, signer, tx)
	s.Require().NoError(err)
	txBz, err := signedTx.MarshalBinary()
	s.Require().NoError(err)
	txHash, err := s.backend.SendRawTransaction(txBz)
	s.Require().NoError(err)
	if waitForNextBlock {
		s.Require().NoError(s.network.WaitForNextBlock())
	}
	return txHash
}

// WaitForReceipt waits for a transaction to be included in a block, returns block number, block hash and receipt
func WaitForReceipt(s *BackendSuite, txHash gethcommon.Hash) (*big.Int, *gethcommon.Hash, *backend.TransactionReceipt) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	for {
		receipt, err := s.backend.GetTransactionReceipt(txHash)
		if err != nil {
			return nil, nil, nil
		}
		if receipt != nil {
			return receipt.BlockNumber, &receipt.BlockHash, receipt
		}
		select {
		case <-ctx.Done():
			fmt.Println("Timeout reached, transaction not included in a block yet.")
			return nil, nil, nil
		default:
			time.Sleep(1 * time.Second)
		}
	}
}

// getUnibiBalance returns the balance of an address in unibi
func (s *BackendSuite) getUnibiBalance(address gethcommon.Address) *big.Int {
	latestBlock := rpc.EthLatestBlockNumber
	latestBlockOrHash := rpc.BlockNumberOrHash{BlockNumber: &latestBlock}
	balance, err := s.backend.GetBalance(address, latestBlockOrHash)
	s.Require().NoError(err)
	return evm.WeiToNative(balance.ToInt())
}

// getCurrentNonce returns the current nonce of the funded account
func (s *BackendSuite) getCurrentNonce(address gethcommon.Address) uint64 {
	nonce, err := s.backend.GetTransactionCount(address, rpc.EthPendingBlockNumber)
	s.Require().NoError(err)

	return uint64(*nonce)
}

// broadcastSDKTx broadcasts the given SDK transaction and returns the response
func (s *BackendSuite) broadcastSDKTx(sdkTx sdk.Tx) *sdk.TxResponse {
	txBytes, err := s.backend.ClientCtx().TxConfig.TxEncoder()(sdkTx)
	s.Require().NoError(err)

	syncCtx := s.backend.ClientCtx().WithBroadcastMode(flags.BroadcastSync)
	rsp, err := syncCtx.BroadcastTx(txBytes)
	s.Require().NoError(err)
	return rsp
}

// buildContractCreationTx builds a contract creation transaction
func (s *BackendSuite) buildContractCreationTx(nonce uint64, gasLimit uint64) gethcore.Transaction {
	packedArgs, err := embeds.SmartContract_TestERC20.ABI.Pack("")
	s.Require().NoError(err)
	bytecodeForCall := append(embeds.SmartContract_TestERC20.Bytecode, packedArgs...)

	creationTx := &gethcore.LegacyTx{
		Nonce:    nonce,
		Data:     bytecodeForCall,
		Gas:      gasLimit,
		GasPrice: big.NewInt(1),
	}

	signer := gethcore.LatestSignerForChainID(s.ethChainID)
	signedTx, err := gethcore.SignNewTx(s.fundedAccPrivateKey, signer, creationTx)
	s.Require().NoError(err)

	return *signedTx
}

// buildContractCallTx builds a contract call transaction
func (s *BackendSuite) buildContractCallTx(
	contractAddr gethcommon.Address,
	nonce uint64,
	gasLimit uint64,
) gethcore.Transaction {
	//recipient := crypto.CreateAddress(s.fundedAccEthAddr, 29381)
	transferAmount := big.NewInt(100)

	packedArgs, err := embeds.SmartContract_TestERC20.ABI.Pack(
		"transfer",
		recipient,
		transferAmount,
	)
	s.Require().NoError(err)

	transferTx := &gethcore.LegacyTx{
		Nonce:    nonce,
		Data:     packedArgs,
		Gas:      gasLimit,
		GasPrice: big.NewInt(1),
		To:       &contractAddr,
	}

	signer := gethcore.LatestSignerForChainID(s.ethChainID)
	signedTx, err := gethcore.SignNewTx(s.fundedAccPrivateKey, signer, transferTx)
	s.Require().NoError(err)

	return *signedTx
}
