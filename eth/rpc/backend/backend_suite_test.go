package backend_test

import (
	"math/big"
	"testing"

	"crypto/ecdsa"

	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/suite"

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

type BackendSuite struct {
	suite.Suite
	cfg                 testnetwork.Config
	network             *testnetwork.Network
	node                *testnetwork.Validator
	fundedAccPrivateKey *ecdsa.PrivateKey
	fundedAccEthAddr    gethcommon.Address
	fundedAccNibiAddr   sdk.AccAddress
	backend             *backend.EVMBackend
	ethChainID          *big.Int
}

func TestBackendSuite(t *testing.T) {
	suite.Run(t, new(BackendSuite))
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
	s.NoError(testnetwork.FillWalletFromValidator(s.fundedAccNibiAddr, funds, s.node, eth.EthBaseDenom))
	s.NoError(s.network.WaitForNextBlock())
}

// SendNibiViaEthTransfer sends nibi using the eth rpc backend
func (s *BackendSuite) SendNibiViaEthTransfer(
	from gethcommon.Address,
	to gethcommon.Address,
	amount *big.Int,
) (rpc.BlockNumber, gethcommon.Hash) {
	block, err := s.backend.BlockNumber()
	s.Require().NoError(err)
	nonce, err := s.backend.GetTransactionCount(from, rpc.BlockNumber(block))
	s.NoError(err)

	signer := gethcore.LatestSignerForChainID(s.ethChainID)
	gasPrice := evm.NativeToWei(big.NewInt(1))
	tx, err := gethcore.SignNewTx(
		s.fundedAccPrivateKey,
		signer,
		&gethcore.LegacyTx{
			Nonce:    uint64(*nonce),
			To:       &to,
			Value:    amount,
			Gas:      params.TxGas,
			GasPrice: gasPrice,
		})
	s.Require().NoError(err)
	txBz, err := tx.MarshalBinary()
	s.Require().NoError(err)
	txHash, err := s.backend.SendRawTransaction(txBz)
	s.Require().NoError(err)
	return rpc.BlockNumber(block), txHash
}
