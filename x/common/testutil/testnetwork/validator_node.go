package testnetwork

import (
	"context"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/suite"

	serverconfig "github.com/NibiruChain/nibiru/v2/app/server/config"
	"github.com/NibiruChain/nibiru/v2/eth"
	ethrpc "github.com/NibiruChain/nibiru/v2/eth/rpc"
	"github.com/NibiruChain/nibiru/v2/eth/rpc/backend"
	"github.com/NibiruChain/nibiru/v2/eth/rpc/rpcapi"

	"github.com/cometbft/cometbft/node"
	tmclient "github.com/cometbft/cometbft/rpc/client"
	"github.com/cosmos/cosmos-sdk/client"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/server"
	serverapi "github.com/cosmos/cosmos-sdk/server/api"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc"

	geth "github.com/ethereum/go-ethereum"
	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
)

// Validator defines an in-process Tendermint validator node. Through this
// object, a client can make RPC and API calls and interact with any client
// command or handler.
type Validator struct {
	AppConfig *serverconfig.Config
	ClientCtx client.Context
	Ctx       *server.Context
	// Dir is the root directory of the validator node data and config. Passed to the Tendermint config.
	Dir string

	// NodeID is a unique ID for the validator generated when the
	// 'cli.Network' is started.
	NodeID string
	PubKey cryptotypes.PubKey

	// Moniker is a human-readable name that identifies a validator. A
	// moniker is optional and may be empty.
	Moniker string

	// APIAddress is the endpoint that the validator API server binds to.
	// Only the first validator of a 'cli.Network' exposes the full API.
	APIAddress string

	// RPCAddress is the endpoint that the RPC server binds to. Only the
	// first validator of a 'cli.Network' exposes the full API.
	RPCAddress string

	// P2PAddress is the endpoint that the RPC server binds to. The P2P
	// server handles Tendermint peer-to-peer (P2P) networking and is
	// critical for blockchain replication and consensus. It allows nodes
	// to gossip blocks, transactions, and consensus messages. Only the
	// first validator of a 'cli.Network' exposes the full API.
	P2PAddress string

	// Address - account address
	Address sdk.AccAddress

	// EthAddress - Ethereum address
	EthAddress common.Address

	// ValAddress - validator operator (valoper) address
	ValAddress sdk.ValAddress

	// RPCClient wraps most important rpc calls a client would make to
	// listen for events, test if it also implements events.EventSwitch.
	//
	// RPCClient implementations in "github.com/cometbft/cometbft/rpc" v0.37.2:
	// - rpc.HTTP
	// - rpc.Local
	RPCClient tmclient.Client

	JSONRPCClient     *ethclient.Client
	EthRpcQueryClient *ethrpc.QueryClient
	EthRpcBackend     *backend.Backend
	EthTxIndexer      eth.EVMTxIndexer

	EthRPC_ETH  *rpcapi.EthAPI
	EthRpc_WEB3 *rpcapi.APIWeb3
	EthRpc_NET  *rpcapi.NetAPI

	Logger Logger

	tmNode *node.Node

	// API exposes the app's REST and gRPC interfaces, allowing clients to
	// read from state and broadcast txs. The API server connects to the
	// underlying ABCI application.
	api            *serverapi.Server
	grpc           *grpc.Server
	grpcWeb        *http.Server
	secretMnemonic string
	jsonrpc        *http.Server
	jsonrpcDone    chan struct{}
}

// stopValidatorNode shuts down all services associated with a validator node.
//
// It gracefully stops the Tendermint node, API, gRPC, gRPC-Web, and JSON-RPC
// services. This function is designed to be run concurrently for multiple
// validators during network cleanup.
//
// The function uses graceful shutdown methods where available to allow ongoing
// operations to complete before terminating. This approach helps prevent
// resource leaks and ensures a clean shutdown of all components.
//
// Parameters:
//   - v: Pointer to the Validator struct containing service references.
//
// Note: Errors during shutdown are currently ignored to ensure all services
// attempt to stop, even if one fails. Consider adding error logging for
// debugging in production environments.
func stopValidatorNode(v *Validator) {
	if v.tmNode != nil && v.tmNode.IsRunning() {
		if err := v.tmNode.Stop(); err != nil {
			v.Logger.Logf("Error stopping Validator.tmNode: %w", err)
		}
		v.tmNode.Wait() // Wait for the service to fully stop
	}

	if v.api != nil {
		// Close the API server.
		// Any blocked "Accept" operations will be unblocked and return errors.
		err := v.api.Close()
		if err != nil {
			v.Logger.Logf("❌ Error closing the API server: %w", err)
		}
	}

	if v.grpc != nil {
		// GracefulStop stops the gRPC server gracefully. It stops the server from
		// accepting new connections and RPCs and blocks until all the pending RPCs are
		// finished.
		v.grpc.GracefulStop()
	}

	if v.grpcWeb != nil {
		err := v.grpcWeb.Close()
		if err != nil {
			v.Logger.Logf("❌ Error closing the gRPC web server: %w", err)
		}
	}

	if v.jsonrpc != nil {
		// Note that this is a graceful shutdown replacement for:
		// _ = v.jsonrpc.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := v.jsonrpc.Shutdown(ctx); err != nil {
			// Log the error or handle it as appropriate for your application
			v.Logger.Logf("❌ Error shutting down JSON-RPC server: %w", err)
		} else {
			v.Logger.Log("✅ Successfully shut down JSON-RPC server")
			v.jsonrpc = nil
		}
	}

	if v.tmNode != nil {
		v.tmNode.Wait()
	}
}

func ValidatorsStopped(vals []*Validator) (stopped bool) {
	for _, v := range vals {
		if !v.IsStopped() {
			return false
		}
	}
	return true
}

// IsStopped returns true if the validator node is stopped
func (v *Validator) IsStopped() bool {
	switch {
	case v == nil:
		return true
	case v.tmNode == nil:
		return true
	case v.tmNode.IsRunning():
		return false
	}
	return true
}

func (val Validator) SecretMnemonic() string {
	return val.secretMnemonic
}

func (val Validator) SecretMnemonicSlice() []string {
	return strings.Fields(val.secretMnemonic)
}

// LogMnemonic logs a secret to the network's logger for debugging and manual
// testing
func LogMnemonic(l Logger, secret string) {
	lines := []string{
		"THIS MNEMONIC IS FOR TESTING PURPOSES ONLY",
		"DO NOT USE IN PRODUCTION",
		"",
		strings.Join(strings.Fields(secret)[0:8], " "),
		strings.Join(strings.Fields(secret)[8:16], " "),
		strings.Join(strings.Fields(secret)[16:24], " "),
	}

	lineLengths := make([]int, len(lines))
	for i, line := range lines {
		lineLengths[i] = len(line)
	}

	maxLineLength := 0
	for _, lineLen := range lineLengths {
		if lineLen > maxLineLength {
			maxLineLength = lineLen
		}
	}

	l.Log("\n")
	l.Log(strings.Repeat("+", maxLineLength+8))
	for _, line := range lines {
		l.Logf("++  %s  ++\n", centerText(line, maxLineLength))
	}
	l.Log(strings.Repeat("+", maxLineLength+8))
	l.Log("\n")
}

// centerText: Centers text across a fixed width, filling either side with
// whitespace buffers
func centerText(text string, width int) string {
	textLen := len(text)
	leftBuffer := strings.Repeat(" ", (width-textLen)/2)
	rightBuffer := strings.Repeat(" ", (width-textLen)/2+(width-textLen)%2)

	return fmt.Sprintf("%s%s%s", leftBuffer, text, rightBuffer)
}

func (val *Validator) AssertERC20Balance(
	contract gethcommon.Address,
	accAddr gethcommon.Address,
	expectedBalance *big.Int,
	s *suite.Suite,
) {
	input, err := embeds.SmartContract_ERC20Minter.ABI.Pack("balanceOf", accAddr)
	s.NoError(err)
	msg := geth.CallMsg{
		From: accAddr,
		To:   &contract,
		Data: input,
	}
	recipientBalanceBeforeBytes, err := val.JSONRPCClient.CallContract(context.Background(), msg, nil)
	s.NoError(err)
	balance := new(big.Int).SetBytes(recipientBalanceBeforeBytes)
	s.Equal(expectedBalance.String(), balance.String())
}
