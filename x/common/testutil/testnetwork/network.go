package testnetwork

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

	srvconfig "github.com/cosmos/cosmos-sdk/server/config"
	"github.com/ethereum/go-ethereum/common"

	serverconfig "github.com/NibiruChain/nibiru/app/server/config"

	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/store/pruning/types"
	"github.com/cosmos/cosmos-sdk/testutil/sims"

	"cosmossdk.io/math"
	dbm "github.com/cometbft/cometbft-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdktestutil "github.com/cosmos/cosmos-sdk/testutil"

	tmrand "github.com/cometbft/cometbft/libs/rand"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/server"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/NibiruChain/nibiru/x/common/denoms"

	"github.com/NibiruChain/nibiru/app"
)

// package-wide network lock to only allow one test network at a time
var lock = new(sync.Mutex)

// AppConstructor defines a function which accepts a network configuration and
// creates an ABCI Application to provide to Tendermint.
type AppConstructor = func(val Validator) servertypes.Application

// Network defines an in-process testing network. It is primarily intended
// for client and integration testing. The Network struct can spawn any
// number of validators, each with its own RPC and API clients.
//
// ### Constraints
//
//  1. Only the first validator will have a functional RPC and API
//     server/client.
//  2. Due to constraints in Tendermint's JSON-RPC implementation, only one
//     test network can run at a time. For this reason, it's essential to
//     invoke `Network.Cleanup` after testing to allow other tests to create
//     networks.
//
// Each of the "Validators" has a "Logger", each being a shared reference to the
// `Network.Logger`. This helps simplify debugging.
type Network struct {
	BaseDir    string
	Config     Config
	Validators []*Validator
	Logger     Logger
}

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

/*
New creates a new Network for integration tests.

Example:

	import (
		"suite"
		"github.com/NibiruChain/nibiru/app"
		"github.com/NibiruChain/nibiru/x/common/testutil/genesis"
		"github.com/NibiruChain/nibiru/x/common/testutil/testnetwork"
	)

	var s *suite.Suite // For some test suite...
	encodingConfig := app.MakeEncodingConfig()
	genesisState := genesis.NewTestGenesisState(encodingConfig)
	cfg = testnetwork.BuildNetworkConfig(genesisState)
	network, err := testnetwork.New(s.T(), s.T().TempDir(), cfg)
	s.Require().NoError(err)
*/
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
	for valIdx := 0; valIdx < cfg.NumValidators; valIdx++ {
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
		if valIdx == 0 {
			if cfg.APIAddress != "" {
				apiListenAddr = cfg.APIAddress
			} else {
				var err error
				apiListenAddr, _, err = server.FreeTCPAddr()
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
				rpcAddr, _, err := server.FreeTCPAddr()
				if err != nil {
					return nil, err
				}
				tmCfg.RPC.ListenAddress = rpcAddr
			}

			if cfg.GRPCAddress != "" {
				appCfg.GRPC.Address = cfg.GRPCAddress
			} else {
				_, grpcPort, err := server.FreeTCPAddr()
				if err != nil {
					return nil, err
				}
				appCfg.GRPC.Address = fmt.Sprintf("0.0.0.0:%s", grpcPort)
			}
			appCfg.GRPC.Enable = true

			_, grpcWebPort, err := server.FreeTCPAddr()
			if err != nil {
				return nil, err
			}
			appCfg.GRPCWeb.Address = fmt.Sprintf("0.0.0.0:%s", grpcWebPort)
			appCfg.GRPCWeb.Enable = true

			if cfg.JSONRPCAddress != "" {
				appCfg.JSONRPC.Address = cfg.JSONRPCAddress
			} else {
				_, jsonRPCPort, err := server.FreeTCPAddr()
				if err != nil {
					return nil, err
				}
				appCfg.JSONRPC.Address = fmt.Sprintf("0.0.0.0:%s", jsonRPCPort)
			}
			appCfg.JSONRPC.Enable = true
			appCfg.JSONRPC.API = serverconfig.GetAPINamespaces()
		}

		loggerNoOp := log.NewNopLogger()
		if cfg.EnableTMLogging {
			loggerNoOp = log.NewTMLogger(log.NewSyncWriter(os.Stdout))
		}

		ctx.Logger = loggerNoOp

		nodeDirName := fmt.Sprintf("node%d", valIdx)
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
		monikers[valIdx] = nodeDirName

		proxyAddr, _, err := server.FreeTCPAddr()
		if err != nil {
			return nil, err
		}
		tmCfg.ProxyApp = proxyAddr

		p2pAddr, _, err := server.FreeTCPAddr()
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

		nodeIDs[valIdx] = nodeID
		valPubKeys[valIdx] = pubKey

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
		if valIdx < len(cfg.Mnemonics) {
			mnemonic = cfg.Mnemonics[valIdx]
		}

		addr, secret, err := sdktestutil.GenerateSaveCoinKey(kb, nodeDirName, mnemonic, true, algo)
		ethAddr := common.BytesToAddress(addr.Bytes())

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

		createValMsg, err := stakingtypes.NewMsgCreateValidator(
			sdk.ValAddress(addr),
			valPubKeys[valIdx],
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

		memo := fmt.Sprintf("%s@%s:%s", nodeIDs[valIdx], p2pURL.Hostname(), p2pURL.Port())
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

		err = tx.Sign(txFactory, nodeDirName, txBuilder, true)
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

		srvconfig.WriteConfigFile(filepath.Join(nodeDir, "config", "app.toml"), appCfg)

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

		network.Validators[valIdx] = &Validator{
			AppConfig:      appCfg,
			ClientCtx:      clientCtx,
			Ctx:            ctx,
			Dir:            filepath.Join(network.BaseDir, nodeDirName),
			Logger:         logger,
			NodeID:         nodeID,
			PubKey:         pubKey,
			Moniker:        nodeDirName,
			RPCAddress:     tmCfg.RPC.ListenAddress,
			P2PAddress:     tmCfg.P2P.ListenAddress,
			APIAddress:     apiAddr,
			Address:        addr,
			EthAddress:     ethAddr,
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
	server.TrapSignal(network.Cleanup)

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
func (n *Network) WaitForNextBlockVerbose() (int64, error) {
	lastBlock, err := n.LatestHeight()
	if err != nil {
		return -1, err
	}

	newBlock := lastBlock + 1
	_, err = n.WaitForHeight(newBlock)
	if err != nil {
		return lastBlock, err
	}

	return newBlock, err
}

func (n *Network) WaitForNextBlock() error {
	_, err := n.WaitForNextBlockVerbose()
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

	// We use a wait group here to ensure that all services are stopped before
	// cleaning up.
	var waitGroup sync.WaitGroup

	for _, v := range n.Validators {
		waitGroup.Add(1)

		go func(v *Validator) {
			defer waitGroup.Done()
			stopValidatorNode(v)
		}(v)
	}

	waitGroup.Wait()

	// TODO: Is there a cleaner way to do this with a synchronous check?
	// https://github.com/NibiruChain/nibiru/issues/1955

	// Give a brief pause for things to finish closing in other processes.
	// Hopefully this helps with the address-in-use errors.
	// Timeout of 100ms chosen randomly.
	// Timeout of 250ms chosen because 100ms was not enough. | 2024-07-02
	maxRetries := 5
	stopped := false
	for i := 0; i < maxRetries; i++ {
		if ValidatorsStopped(n.Validators) {
			stopped = true
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if !stopped {
		panic("cleanup did not succeed within the max retry count")
	}

	if n.Config.CleanupDir {
		_ = os.RemoveAll(n.BaseDir)
	}

	n.Logger.Log("finished cleaning up test network")
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
