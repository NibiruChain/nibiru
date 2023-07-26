package cli

import (
	"encoding/json"
	"time"

	sdkmath "cosmossdk.io/math"

	tmconfig "github.com/cometbft/cometbft/config"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Config defines the necessary configuration used to bootstrap and start an
// in-process local testing network.
type Config struct {
	Codec             codec.Codec
	LegacyAmino       *codec.LegacyAmino // TODO: Remove!
	InterfaceRegistry codectypes.InterfaceRegistry

	TxConfig         client.TxConfig
	AccountRetriever client.AccountRetriever
	AppConstructor   AppConstructor             // the ABCI application constructor
	GenesisState     map[string]json.RawMessage // custom genesis state to provide
	TimeoutCommit    time.Duration              // the consensus commitment timeout
	ChainID          string                     // the network chain-id
	NumValidators    int                        // the total number of validators to create and bond
	Mnemonics        []string                   // custom user-provided validator operator mnemonics
	BondDenom        string                     // the staking bond denomination
	MinGasPrices     string                     // the minimum gas prices each validator will accept
	AccountTokens    sdkmath.Int                // the amount of unique validator tokens (e.g. 1000node0)
	StakingTokens    sdkmath.Int                // the amount of tokens each validator has available to stake
	BondedTokens     sdkmath.Int                // the amount of tokens each validator stakes
	StartingTokens   sdk.Coins                  // Additional tokens to be added to the starting block to validators
	PruningStrategy  string                     // the pruning strategy each validator will have
	EnableTMLogging  bool                       // enable Tendermint logging to STDOUT
	CleanupDir       bool                       // remove base temporary directory during cleanup
	SigningAlgo      string                     // signing algorithm for keys
	KeyringOptions   []keyring.Option           // keyring configuration options
	RPCAddress       string                     // RPC listen address (including port)
	APIAddress       string                     // REST API listen address (including port)
	GRPCAddress      string                     // GRPC server listen address (including port)
	PrintMnemonic    bool                       // print the mnemonic of first validator as log output for testing
}

func (cfg *Config) AbsorbServerConfig(srvCfg *serverconfig.Config) {
	cfg.GRPCAddress = srvCfg.GRPC.Address
	cfg.APIAddress = srvCfg.API.Address
}

func (cfg *Config) AbsorbTmConfig(tmCfg *tmconfig.Config) {
	cfg.RPCAddress = tmCfg.RPC.ListenAddress
}

// AbsorbListenAddresses ensures that the listening addresses for an active
// node are set on the network config.
func (cfg *Config) AbsorbListenAddresses(val *Validator) {
	cfg.AbsorbServerConfig(val.AppConfig)
	cfg.AbsorbTmConfig(val.Ctx.Config)
}
