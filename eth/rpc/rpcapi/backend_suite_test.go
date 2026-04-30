package rpcapi_test

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"sync"
	"testing"

	"github.com/cosmos/cosmos-sdk/client/flags"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/suite"

	bankcli "github.com/NibiruChain/nibiru/v2/x/bank/client/cli"

	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"

	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"

	"github.com/NibiruChain/nibiru/v2/eth/rpc"
	"github.com/NibiruChain/nibiru/v2/x/evm"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/eth/rpc/rpcapi"

	"github.com/NibiruChain/nibiru/v2/x/nutil"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil/localnet"
)

func Test(t *testing.T) {
	suite.Run(t, new(Suite))
	suite.Run(t, new(BackendSuite))
	suite.Run(t, new(NodeSuite))
}

type BackendSuite struct {
	testutil.LogRoutingSuite

	cli localnet.CLI

	evmSenderPrivateKey *ecdsa.PrivateKey
	evmSenderEthAddr    gethcommon.Address
	evmSenderNibiAddr   sdk.AccAddress
	backend             *rpcapi.Backend
	ethChainID          *big.Int
	SuccessfulTxs       map[string]SuccessfulTx
	accInfo             accInfoFixture
}

// testMutex is used to synchronize the tests which are broadcasting transactions concurrently
var testMutex sync.Mutex

var (
	recipient    = evmtest.NewEthPrivAcc().EthAddr
	amountToSend = evm.NativeToWei(big.NewInt(1))
)

type accInfoFixture struct {
	Recipient                     gethcommon.Address
	UnusedAddress                 gethcommon.Address
	RecipientBalanceBeforeBlock   rpc.BlockNumber
	ExpectedERC20InitialSupplyWei *big.Int
}

func (s *BackendSuite) SetupSuite() {
	s.T().Log("------------- SetupSuite: BEGIN ------------- ")
	s.LogRoutingSuite.SetupSuite()
	if err := nutil.EnsureLocalBlockchain(); err != nil {
		s.T().Skipf("localnet unavailable: %v", err)
	}

	localnetCLI, err := localnet.NewCLI()
	s.Require().NoError(err)
	s.cli = localnetCLI

	s.backend = localnetCLI.EthRpcBackend
	s.ethChainID = appconst.GetEthChainID(localnet.ChainID)
	s.SuccessfulTxs = make(map[string]SuccessfulTx)
	s.Require().NoError(s.cli.WaitForNextBlock())

	testAccPrivateKey, _ := crypto.GenerateKey()
	s.evmSenderPrivateKey = testAccPrivateKey
	s.evmSenderEthAddr = crypto.PubkeyToAddress(testAccPrivateKey.PublicKey)
	recipient = evmtest.NewEthPrivAcc().EthAddr
	s.T().Logf("SetupSuite: Funding `s.evmSenderEthAddr`: %s", s.evmSenderEthAddr.Hex())
	s.evmSenderNibiAddr = eth.EthAddrToNibiruAddr(s.evmSenderEthAddr)
	funds := sdk.NewCoins(sdk.NewInt64Coin(eth.EthBaseDenom, 100_000_000))

	txResp, err := s.cli.ExecTxCmd(
		bankcli.NewTxCmd(),
		[]string{"send", localnet.KeyName, s.evmSenderNibiAddr.String(), funds.String()},
	)
	s.Require().NoError(err)
	s.Require().NotNil(txResp.TxHash)
	s.Require().NoError(s.cli.WaitForNextBlock())
	s.T().Logf(
		"s.evmSenderEthAddr: %s, funds: %s, localnet validator: %s",
		s.evmSenderEthAddr, funds, s.cli.FromAddr,
	)

	recipientBalanceBeforeHeight, err := s.cli.LatestHeight()
	s.Require().NoError(err)
	s.accInfo = accInfoFixture{
		Recipient:                     recipient,
		UnusedAddress:                 evmtest.NewEthPrivAcc().EthAddr,
		RecipientBalanceBeforeBlock:   rpc.NewBlockNumber(big.NewInt(recipientBalanceBeforeHeight)),
		ExpectedERC20InitialSupplyWei: erc20InitialSupplyWei(),
	}

	transferTxHash := s.SendNibiViaEthTransfer(recipient, amountToSend, true /*waitForNextBlock*/)
	s.T().Logf("SetupSuite: Send Transfer TX and use the results in the tests\ntransfer tx hash: %s", transferTxHash.Hex())
	{
		blockNumber, blockHash, txReceipt, err := WaitForReceipt(s, transferTxHash)
		s.NotNil(blockNumber, "expect block number")
		s.NotNil(blockHash, "expect block hash")
		s.Require().NotNil(txReceipt)
		s.Require().NoError(err)
		s.Require().Equal(transferTxHash, txReceipt.TxHash)
		blockNumberRpc := rpc.NewBlockNumber(blockNumber)
		s.SuccessfulTxs["transfer"] = SuccessfulTx{
			BlockNumber:    blockNumber,
			BlockHash:      blockHash,
			Receipt:        txReceipt,
			BlockNumberRpc: &blockNumberRpc,
		}
	}

	s.T().Log("SetupSuite: Deploy test erc20 contract")
	deployContractTxHash, _ := s.DeployTestContract(true)
	{
		blockNumber, blockHash, txReceipt, err := WaitForReceipt(s, deployContractTxHash)
		s.NotNil(blockNumber, "expect block number")
		s.NotNil(blockHash, "expect block hash")
		s.Require().NotNil(txReceipt)
		s.Require().NoError(err)
		blockNumberRpc := rpc.NewBlockNumber(blockNumber)
		s.SuccessfulTxs["deployContract"] = SuccessfulTx{
			BlockNumber:    blockNumber,
			BlockNumberRpc: &blockNumberRpc,
			BlockHash:      blockHash,
			Receipt:        txReceipt,
		}
	}

	for txName, tx := range s.SuccessfulTxs {
		s.T().Logf(
			"SuccessfulTx(%s){ BlockNumber: %s, BlockHash: %s, TxHash: %s }",
			txName, tx.BlockNumber, tx.BlockHash.Hex(), tx.Receipt.TxHash.Hex(),
		)
	}
	s.T().Log("------------- SetupSuite: END   ------------- ")
}

func (s *BackendSuite) TearDownSuite() {
	s.Require().NoError(s.cli.Close())
}

func (s *BackendSuite) SuccessfulTxTransfer() SuccessfulTx {
	return s.SuccessfulTxs["transfer"]
}

func (s *BackendSuite) SuccessfulTxDeployContract() SuccessfulTx {
	return s.SuccessfulTxs["deployContract"]
}

func erc20InitialSupplyWei() *big.Int {
	initialSupply := big.NewInt(1_000_000)
	decimals := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	return new(big.Int).Mul(initialSupply, decimals)
}

type SuccessfulTx struct {
	BlockNumber    *big.Int
	BlockHash      *gethcommon.Hash
	Receipt        *rpcapi.TransactionReceipt
	BlockNumberRpc *rpc.BlockNumber
}

// SendNibiViaEthTransfer sends nibi using the eth rpc backend
func (s *BackendSuite) SendNibiViaEthTransfer(
	to gethcommon.Address,
	amount *big.Int,
	waitForNextBlock bool,
) gethcommon.Hash {
	nonce := s.getCurrentNonce(s.evmSenderEthAddr)
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
	nonce := s.getCurrentNonce(s.evmSenderEthAddr)
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
	contractAddr := crypto.CreateAddress(s.evmSenderEthAddr, nonce)
	return txHash, contractAddr
}

// SendTransaction signs and sends raw ethereum transaction
func SendTransaction(s *BackendSuite, tx *gethcore.LegacyTx, waitForNextBlock bool) gethcommon.Hash {
	signer := gethcore.LatestSignerForChainID(s.ethChainID)
	signedTx, err := gethcore.SignNewTx(s.evmSenderPrivateKey, signer, tx)
	s.Require().NoError(err)
	txBz, err := signedTx.MarshalBinary()
	s.Require().NoError(err)
	txHash, err := s.cli.EvmRpc.Eth.SendRawTransaction(txBz)
	s.Require().NoError(err)
	if waitForNextBlock {
		s.Require().NoError(s.cli.WaitForNextBlock())
	}
	return txHash
}

// WaitForReceipt checks for a receipt, waiting up to two blocks if needed.
// Broadcasted localnet txs should be visible within that small block window.
func WaitForReceipt(
	s *BackendSuite,
	txHash gethcommon.Hash,
) (
	blockNumber *big.Int,
	blockHash *gethcommon.Hash,
	receipt *rpcapi.TransactionReceipt,
	err error,
) {
	for attempt := 0; attempt < 3; attempt++ {
		receipt, err = s.cli.EvmRpc.Eth.GetTransactionReceipt(txHash)
		if err == nil && receipt != nil {
			blockNumber = receipt.BlockNumber
			blockHash = &receipt.BlockHash
			return
		}
		if attempt < 2 {
			s.Require().NoError(s.cli.WaitForNextBlock())
		}
	}
	if err != nil {
		return
	}
	err = fmt.Errorf("receipt not found after waiting two blocks for tx: %s", txHash.Hex())
	return
}

// getUnibiBalance returns the balance of an address in unibi
func (s *BackendSuite) getUnibiBalance(address gethcommon.Address) *big.Int {
	latestBlock := rpc.EthLatestBlockNumber
	latestBlockOrHash := rpc.BlockNumberOrHash{BlockNumber: &latestBlock}
	balance, err := s.cli.EvmRpc.Eth.GetBalance(address, latestBlockOrHash)
	s.Require().NoError(err)
	return evm.WeiToNative(balance.ToInt())
}

// getCurrentNonce returns the current nonce of the funded account
func (s *BackendSuite) getCurrentNonce(address gethcommon.Address) uint64 {
	pendingBlock := rpc.EthPendingBlockNumber
	nonce, err := s.cli.EvmRpc.Eth.GetTransactionCount(address, rpc.BlockNumberOrHash{
		BlockNumber: &pendingBlock,
	})
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
func (s *BackendSuite) buildContractCreationTx(nonce uint64, gasLimit uint64) *gethcore.Transaction {
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
	signedTx, err := gethcore.SignNewTx(s.evmSenderPrivateKey, signer, creationTx)
	s.Require().NoError(err)

	return signedTx
}

// buildContractCallTx builds a contract call transaction
func (s *BackendSuite) buildContractCallTx(
	contractAddr gethcommon.Address,
	nonce uint64,
	gasLimit uint64,
) *gethcore.Transaction {
	// recipient := crypto.CreateAddress(s.evmSenderEthAddr, 29381)
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
	signedTx, err := gethcore.SignNewTx(s.evmSenderPrivateKey, signer, transferTx)
	s.Require().NoError(err)

	return signedTx
}
