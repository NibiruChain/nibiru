package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"sync"
	"syscall"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/store/pruning/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/testutil"
	net "github.com/cosmos/cosmos-sdk/testutil/network"
	"github.com/cosmos/cosmos-sdk/testutil/sims"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	"golang.org/x/sync/errgroup"

	"cosmossdk.io/math"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdktestutil "github.com/cosmos/cosmos-sdk/testutil"

	tmrand "github.com/cometbft/cometbft/libs/rand"
	"github.com/cometbft/cometbft/node"
	tmclient "github.com/cometbft/cometbft/rpc/client"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/server"
	serverapi "github.com/cosmos/cosmos-sdk/server/api"
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"google.golang.org/grpc"

	"github.com/NibiruChain/nibiru/x/common/denoms"

	"github.com/NibiruChain/nibiru/app"
)

// package-wide network lock to only allow one test network at a time
var lock = new(sync.Mutex)

// AppConstructor defines a function which accepts a network configuration and
// creates an ABCI Application to provide to Tendermint.
type AppConstructor = func(val Validator) servertypes.Application

type (
	// Network defines a in-process testing network. It is primarily intended
	// for client and integration testing. The Network struct can spawn any
	// number of validators, each with its own RPC and API clients.
	//
	// ### Constraints
	//
	// 1. Only the first validator will have a functional RPC and API
	//    server/client.
	// 2. Due to constraints in Tendermint's JSON-RPC implementation, only one
	//    test network can run at a time. For this reason, it's essential to
	//    invoke `Network.Cleanup` after testing to allow other tests to create
	//    networks.
	Network struct {
		BaseDir    string
		Config     Config
		Validators []*Validator
		Logger     Logger
	}

	// Validator defines an in-process Tendermint validator node. Through this
	// object, a client can make RPC and API calls and interact with any client
	// command or handler.
	Validator struct {
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

		// ValAddress - validator operator (valoper) address
		ValAddress sdk.ValAddress

		// RPCClient wraps most important rpc calls a client would make to
		// listen for events, test if it also implements events.EventSwitch.
		//
		// RPCClient implementations in "github.com/cometbft/cometbft/rpc" v0.37.2:
		// - rcp.HTTP
		// - rpc.Local
		RPCClient tmclient.Client

		tmNode *node.Node

		// API exposes the app's REST and gRPC interfaces, allowing clients to
		// read from state and broadcast txs. The API server connects to the
		// underlying ABCI application.
		api            *serverapi.Server
		grpc           *grpc.Server
		grpcWeb        *http.Server
		secretMnemonic string
		errGroup       *errgroup.Group
		cancelFn       context.CancelFunc
	}
)

// NewAppConstructor returns a new simapp AppConstructor
func NewAppConstructor(encodingCfg app.EncodingConfig, chainID string) AppConstructor {
	return func(val Validator) servertypes.Application {
		return app.NewNibiruApp(
			val.Ctx.Logger,
			dbm.NewMemDB(),
			nil,
			true,
			encodingCfg,
			sims.EmptyAppOptions{},
			baseapp.SetPruning(types.NewPruningOptionsFromString(val.AppConfig.Pruning)),
			baseapp.SetMinGasPrices(val.AppConfig.MinGasPrices),
			baseapp.SetChainID(chainID),
		)
	}
}

// BuildNetworkConfig returns a configuration for a local in-testing network
func BuildNetworkConfig(appGenesis app.GenesisState) Config {
	encCfg := app.MakeEncodingConfig()

	chainID := "chain-" + tmrand.NewRand().Str(6)
	return Config{
		Codec:             encCfg.Codec,
		TxConfig:          encCfg.TxConfig,
		LegacyAmino:       encCfg.Amino,
		InterfaceRegistry: encCfg.InterfaceRegistry,
		AccountRetriever:  authtypes.AccountRetriever{},
		AppConstructor:    NewAppConstructor(encCfg, chainID),
		GenesisState:      appGenesis,
		TimeoutCommit:     time.Second / 2,
		ChainID:           chainID,
		NumValidators:     1,
		BondDenom:         denoms.NIBI,
		MinGasPrices:      fmt.Sprintf("0.000006%s", denoms.NIBI),
		AccountTokens:     sdk.TokensFromConsensusPower(1000, sdk.DefaultPowerReduction),
		StakingTokens:     sdk.TokensFromConsensusPower(500, sdk.DefaultPowerReduction),
		BondedTokens:      sdk.TokensFromConsensusPower(100, sdk.DefaultPowerReduction),
		StartingTokens: sdk.NewCoins(
			sdk.NewCoin(denoms.NUSD, sdk.TokensFromConsensusPower(1e12, sdk.DefaultPowerReduction)),
			sdk.NewCoin(denoms.NIBI, sdk.TokensFromConsensusPower(1e12, sdk.DefaultPowerReduction)),
			sdk.NewCoin(denoms.USDC, sdk.TokensFromConsensusPower(1e12, sdk.DefaultPowerReduction)),
		),
		PruningStrategy: types.PruningOptionNothing,
		CleanupDir:      true,
		SigningAlgo:     string(hd.Secp256k1Type),
		KeyringOptions:  []keyring.Option{},
	}
}

// New creates a new Network for integration tests.
func New(logger Logger, baseDir string, cfg Config) (*Network, error) {
	// only one caller/test can create and use a network at a time
	logger.Log("acquiring test network lock")
	lock.Lock()

	network := &Network{
		Logger:     logger,
		BaseDir:    baseDir,
		Validators: make([]*Validator, cfg.NumValidators),
		Config:     cfg,
	}

	logger.Log("preparing test network...")

	monikers := make([]string, cfg.NumValidators)
	nodeIDs := make([]string, cfg.NumValidators)
	valPubKeys := make([]cryptotypes.PubKey, cfg.NumValidators)

	var (
		genAccounts []authtypes.GenesisAccount
		genBalances []banktypes.Balance
		genFiles    []string
	)

	buf := bufio.NewReader(os.Stdin)

	// generate private keys, node IDs, and initial transactions
	for i := 0; i < cfg.NumValidators; i++ {
		appCfg := serverconfig.DefaultConfig()
		appCfg.Pruning = cfg.PruningStrategy
		appCfg.MinGasPrices = cfg.MinGasPrices
		appCfg.API.Enable = true
		appCfg.API.Swagger = false
		appCfg.Telemetry.Enabled = false

		ctx := server.NewDefaultContext()
		tmCfg := ctx.Config
		tmCfg.Consensus.TimeoutCommit = cfg.TimeoutCommit

		// Only allow the first validator to expose an RPC, API and gRPC
		// server/client due to Tendermint in-process constraints.
		apiAddr := ""
		tmCfg.RPC.ListenAddress = ""
		appCfg.GRPC.Enable = false
		appCfg.GRPCWeb.Enable = false
		apiListenAddr := ""
		if i == 0 {
			if cfg.APIAddress != "" {
				apiListenAddr = cfg.APIAddress
			} else {
				var err error
				apiListenAddr, _, _, err = net.FreeTCPAddr()
				if err != nil {
					return nil, err
				}
			}

			appCfg.API.Address = apiListenAddr
			apiURL, err := url.Parse(apiListenAddr)
			if err != nil {
				return nil, err
			}
			apiAddr = fmt.Sprintf("http://%s:%s", apiURL.Hostname(), apiURL.Port())

			if cfg.RPCAddress != "" {
				tmCfg.RPC.ListenAddress = cfg.RPCAddress
			} else {
				rpcAddr, _, _, err := net.FreeTCPAddr()
				if err != nil {
					return nil, err
				}
				tmCfg.RPC.ListenAddress = rpcAddr
			}

			if cfg.GRPCAddress != "" {
				appCfg.GRPC.Address = cfg.GRPCAddress
			} else {
				_, grpcPort, _, err := net.FreeTCPAddr()
				if err != nil {
					return nil, err
				}
				appCfg.GRPC.Address = fmt.Sprintf("0.0.0.0:%s", grpcPort)
			}
			appCfg.GRPC.Enable = true

			// GRPCWeb now uses the same address than
			appCfg.GRPCWeb.Enable = true
		}

		loggerNoOp := log.NewNopLogger()
		if cfg.EnableTMLogging {
			loggerNoOp = log.NewLogger(os.Stdout)
		}

		ctx.Logger = loggerNoOp

		nodeDirName := fmt.Sprintf("node%d", i)
		nodeDir := filepath.Join(network.BaseDir, nodeDirName, "simd")
		clientDir := filepath.Join(network.BaseDir, nodeDirName, "simcli")
		gentxsDir := filepath.Join(network.BaseDir, "gentxs")

		err := os.MkdirAll(filepath.Join(nodeDir, "config"), 0o755)
		if err != nil {
			return nil, err
		}

		err = os.MkdirAll(clientDir, 0o755)
		if err != nil {
			return nil, err
		}

		tmCfg.SetRoot(nodeDir)
		tmCfg.Moniker = nodeDirName
		monikers[i] = nodeDirName

		proxyAddr, _, _, err := net.FreeTCPAddr()
		if err != nil {
			return nil, err
		}
		tmCfg.ProxyApp = proxyAddr

		p2pAddr, _, _, err := net.FreeTCPAddr()
		if err != nil {
			return nil, err
		}

		tmCfg.P2P.ListenAddress = p2pAddr
		tmCfg.P2P.AddrBookStrict = false
		tmCfg.P2P.AllowDuplicateIP = true

		nodeID, pubKey, err := genutil.InitializeNodeValidatorFiles(tmCfg)
		if err != nil {
			return nil, err
		}

		nodeIDs[i] = nodeID
		valPubKeys[i] = pubKey

		kb, err := keyring.New(sdk.KeyringServiceName(), keyring.BackendTest, clientDir, buf, cfg.Codec, cfg.KeyringOptions...)
		if err != nil {
			return nil, err
		}

		keyringAlgos, _ := kb.SupportedAlgorithms()
		algo, err := keyring.NewSigningAlgoFromString(cfg.SigningAlgo, keyringAlgos)
		if err != nil {
			return nil, err
		}

		var mnemonic string
		if i < len(cfg.Mnemonics) {
			mnemonic = cfg.Mnemonics[i]
		}

		addr, secret, err := sdktestutil.GenerateSaveCoinKey(kb, nodeDirName, mnemonic, true, algo)
		if err != nil {
			return nil, err
		}

		info := map[string]string{"secret": secret}
		infoBz, err := json.Marshal(info)
		if err != nil {
			return nil, err
		}

		// save private key seed words
		err = writeFile(fmt.Sprintf("%v.json", "key_seed"), clientDir, infoBz)
		if err != nil {
			return nil, err
		}

		balances := sdk.NewCoins(
			sdk.NewCoin(fmt.Sprintf("%stoken", nodeDirName), cfg.AccountTokens),
			sdk.NewCoin(cfg.BondDenom, cfg.StakingTokens),
		)

		balances = balances.Add(cfg.StartingTokens...)

		genFiles = append(genFiles, tmCfg.GenesisFile())
		genBalances = append(genBalances, banktypes.Balance{Address: addr.String(), Coins: balances.Sort()})
		genAccounts = append(genAccounts, authtypes.NewBaseAccount(addr, nil, 0, 0))

		commission, err := math.LegacyNewDecFromStr("0.05")
		if err != nil {
			return nil, err
		}

		interfaceRegistry := testutil.CodecOptions{}.NewInterfaceRegistry()
		cdc := codec.NewProtoCodec(interfaceRegistry)
		txConfig := authtx.NewTxConfig(cdc, authtx.DefaultSignModes)

		valAddrCodec := txConfig.SigningContext().ValidatorAddressCodec()
		valStr, err := valAddrCodec.BytesToString(sdk.ValAddress(addr))
		if err != nil {
			return nil, err
		}

		createValMsg, err := stakingtypes.NewMsgCreateValidator(
			valStr,
			valPubKeys[i],
			sdk.NewCoin(cfg.BondDenom, cfg.BondedTokens),
			stakingtypes.NewDescription(nodeDirName, "", "", "", ""),
			stakingtypes.NewCommissionRates(commission, math.LegacyOneDec(), math.LegacyOneDec()),
			math.OneInt(),
		)
		if err != nil {
			return nil, err
		}

		p2pURL, err := url.Parse(p2pAddr)
		if err != nil {
			return nil, err
		}

		memo := fmt.Sprintf("%s@%s:%s", nodeIDs[i], p2pURL.Hostname(), p2pURL.Port())
		fee := sdk.NewCoins(sdk.NewCoin(fmt.Sprintf("%stoken", nodeDirName), math.ZeroInt()))
		txBuilder := cfg.TxConfig.NewTxBuilder()
		err = txBuilder.SetMsgs(createValMsg)
		if err != nil {
			return nil, err
		}
		txBuilder.SetFeeAmount(fee)    // Arbitrary fee
		txBuilder.SetGasLimit(1000000) // Need at least 100386
		txBuilder.SetMemo(memo)

		txFactory := tx.Factory{}
		txFactory = txFactory.
			WithChainID(cfg.ChainID).
			WithMemo(memo).
			WithKeybase(kb).
			WithTxConfig(cfg.TxConfig)

		err = tx.Sign(nil, txFactory, nodeDirName, txBuilder, true)
		if err != nil {
			return nil, err
		}

		txBz, err := cfg.TxConfig.TxJSONEncoder()(txBuilder.GetTx())
		if err != nil {
			return nil, err
		}
		err = writeFile(fmt.Sprintf("%v.json", nodeDirName), gentxsDir, txBz)
		if err != nil {
			return nil, err
		}

		serverconfig.WriteConfigFile(filepath.Join(nodeDir, "config", "app.toml"), appCfg)

		clientCtx := client.Context{}.
			WithKeyringDir(clientDir).
			WithKeyring(kb).
			WithHomeDir(tmCfg.RootDir).
			WithChainID(cfg.ChainID).
			WithInterfaceRegistry(cfg.InterfaceRegistry).
			WithCodec(cfg.Codec).
			WithLegacyAmino(cfg.LegacyAmino).
			WithTxConfig(cfg.TxConfig).
			WithAccountRetriever(cfg.AccountRetriever)

		network.Validators[i] = &Validator{
			AppConfig:      appCfg,
			ClientCtx:      clientCtx,
			Ctx:            ctx,
			Dir:            filepath.Join(network.BaseDir, nodeDirName),
			NodeID:         nodeID,
			PubKey:         pubKey,
			Moniker:        nodeDirName,
			RPCAddress:     tmCfg.RPC.ListenAddress,
			P2PAddress:     tmCfg.P2P.ListenAddress,
			APIAddress:     apiAddr,
			Address:        addr,
			ValAddress:     sdk.ValAddress(addr),
			secretMnemonic: secret,
		}
	}

	err := initGenFiles(cfg, genAccounts, genBalances, genFiles)
	if err != nil {
		return nil, err
	}
	err = collectGenFiles(cfg, network.Validators, network.BaseDir)
	if err != nil {
		return nil, err
	}

	logger.Log("starting test network...")
	for idx, v := range network.Validators {
		err := startInProcess(cfg, v)
		if err != nil {
			return nil, err
		}
		logger.Log("started validator", idx)
	}

	height, err := network.LatestHeight()
	if err != nil {
		return nil, err
	}

	logger.Log("started test network at height:", height)

	// Ensure we cleanup incase any test was abruptly halted (e.g. SIGINT) as
	// any defer in a test would not be called.
	trapSignal(network.Cleanup)

	return network, err
}

// LatestHeight returns the latest height of the network or an error if the
// query fails or no validators exist.
func (n *Network) LatestHeight() (int64, error) {
	if len(n.Validators) == 0 {
		return 0, errors.New("no validators available")
	}

	status, err := n.Validators[0].RPCClient.Status(context.Background())
	if err != nil {
		return 0, err
	}

	return status.SyncInfo.LatestBlockHeight, nil
}

// WaitForHeight performs a blocking check where it waits for a block to be
// committed after a given block. If that height is not reached within a timeout,
// an error is returned. Regardless, the latest height queried is returned.
func (n *Network) WaitForHeight(h int64) (int64, error) {
	return n.WaitForHeightWithTimeout(h, 40*time.Second)
}

// WaitForHeightWithTimeout is the same as WaitForHeight except the caller can
// provide a custom timeout.
func (n *Network) WaitForHeightWithTimeout(h int64, t time.Duration) (int64, error) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	timeout := time.NewTimer(t)
	defer timeout.Stop()

	if len(n.Validators) == 0 {
		return 0, errors.New("no validators available")
	}

	var latestHeight int64
	val := n.Validators[0]

	for {
		select {
		case <-timeout.C:
			return latestHeight, errors.New("timeout exceeded waiting for block")
		case <-ticker.C:
			status, err := val.RPCClient.Status(context.Background())
			if err == nil && status != nil {
				latestHeight = status.SyncInfo.LatestBlockHeight
				if latestHeight >= h {
					return latestHeight, nil
				}
			}
		}
	}
}

// WaitForNextBlock waits for the next block to be committed, returning an error
// upon failure.
func (n *Network) WaitForNextBlock() error {
	lastBlock, err := n.LatestHeight()
	if err != nil {
		return err
	}

	_, err = n.WaitForHeight(lastBlock + 1)
	if err != nil {
		return err
	}

	return err
}

// WaitForDuration waits for at least the duration provided in blockchain time.
func (n *Network) WaitForDuration(duration time.Duration) error {
	if len(n.Validators) == 0 {
		return fmt.Errorf("no validators")
	}
	val := n.Validators[0]
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	lastBlock, err := val.RPCClient.Block(ctx, nil)
	if err != nil {
		return err
	}

	waitAtLeastUntil := lastBlock.Block.Time.Add(duration)

	for {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		block, err := val.RPCClient.Block(ctx, nil)
		if err != nil {
			return err
		}
		if block.Block.Time.After(waitAtLeastUntil) {
			return nil
		}
	}
}

// Cleanup removes the root testing (temporary) directory and stops both the
// Tendermint and API services. It allows other callers to create and start
// test networks. This method must be called when a test is finished, typically
// in a defer.
func (n *Network) Cleanup() {
	defer func() {
		lock.Unlock()
		n.Logger.Log("released test network lock")
	}()

	n.Logger.Log("cleaning up test network...")

	for _, v := range n.Validators {
		if v.tmNode != nil && v.tmNode.IsRunning() {
			_ = v.tmNode.Stop()
		}

		if v.api != nil {
			_ = v.api.Close()
		}

		if v.grpc != nil {
			v.grpc.Stop()
			if v.grpcWeb != nil {
				_ = v.grpcWeb.Close()
			}
		}
	}

	// Give a brief pause for things to finish closing in other processes.
	// Hopefully this helps with the address-in-use errors. 100ms chosen
	// randomly.
	time.Sleep(100 * time.Millisecond)

	if n.Config.CleanupDir {
		_ = os.RemoveAll(n.BaseDir)
	}

	n.Logger.Log("finished cleaning up test network")
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

func (n *Network) keyBaseAndInfoForAddr(addr sdk.AccAddress) (keyring.Keyring, *keyring.Record, error) {
	for _, v := range n.Validators {
		info, err := v.ClientCtx.Keyring.KeyByAddress(addr)
		if err == nil {
			return v.ClientCtx.Keyring, info, nil
		}
	}

	return nil, nil, fmt.Errorf("address not found in any of the known validators keyrings: %s", addr.String())
}

// trapSignal traps SIGINT and SIGTERM and calls os.Exit once a signal is received.
func trapSignal(cleanupFunc func()) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs

		if cleanupFunc != nil {
			cleanupFunc()
		}
		exitCode := 128

		switch sig {
		case syscall.SIGINT:
			exitCode += int(syscall.SIGINT)
		case syscall.SIGTERM:
			exitCode += int(syscall.SIGTERM)
		}

		os.Exit(exitCode)
	}()
}
