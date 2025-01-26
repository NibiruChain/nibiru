package testnetwork

import (
	"encoding/json"
	"fmt"
	"time"

	sdkmath "cosmossdk.io/math"
	tmconfig "github.com/cometbft/cometbft/config"
	tmrand "github.com/cometbft/cometbft/libs/rand"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/store/pruning/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/NibiruChain/nibiru/v2/app"
	serverconfig "github.com/NibiruChain/nibiru/v2/app/server/config"
	"github.com/NibiruChain/nibiru/v2/x/common/denoms"
)

// Config: Defines the parameters needed to start a local test [Network].
type Config struct {
	Codec             codec.Codec
	LegacyAmino       *codec.LegacyAmino // TODO: Remove!
	InterfaceRegistry codectypes.InterfaceRegistry

	TxConfig         client.TxConfig
	AccountRetriever client.AccountRetriever

	AppConstructor  AppConstructor             // the ABCI application constructor
	GenesisState    map[string]json.RawMessage // custom genesis state to provide
	TimeoutCommit   time.Duration              // TimeoutCommit: the consensus commitment timeout.
	ChainID         string                     // the network chain-id
	NumValidators   int                        // the total number of validators to create and bond
	Mnemonics       []string                   // custom user-provided validator operator mnemonics
	BondDenom       string                     // the staking bond denomination
	MinGasPrices    string                     // the minimum gas prices each validator will accept
	AccountTokens   sdkmath.Int                // the amount of unique validator tokens (e.g. 1000node0)
	StakingTokens   sdkmath.Int                // the amount of tokens each validator has available to stake
	BondedTokens    sdkmath.Int                // the amount of tokens each validator stakes
	StartingTokens  sdk.Coins                  // Additional tokens to be added to the starting block to validators
	PruningStrategy string                     // the pruning strategy each validator will have
	EnableTMLogging bool                       // enable Tendermint logging to STDOUT
	CleanupDir      bool                       // remove base temporary directory during cleanup
	SigningAlgo     string                     // signing algorithm for keys
	KeyringOptions  []keyring.Option           // keyring configuration options
	RPCAddress      string                     // RPC listen address (including port)
	APIAddress      string                     // REST API listen address (including port)
	GRPCAddress     string                     // GRPC server listen address (including port)
	JSONRPCAddress  string                     // JSON-RPC listen address (including port)
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

// BuildNetworkConfig returns a configuration for a local in-testing network
func BuildNetworkConfig(appGenesis app.GenesisState) *Config {
	encCfg := app.MakeEncodingConfig()

	chainID := "chain-" + tmrand.NewRand().Str(6)
	return &Config{
		AccountRetriever:  authtypes.AccountRetriever{},
		AccountTokens:     sdk.TokensFromConsensusPower(1000, sdk.DefaultPowerReduction),
		AppConstructor:    NewAppConstructor(encCfg, chainID),
		BondDenom:         denoms.NIBI,
		BondedTokens:      sdk.TokensFromConsensusPower(100, sdk.DefaultPowerReduction),
		ChainID:           chainID,
		CleanupDir:        true,
		Codec:             encCfg.Codec,
		EnableTMLogging:   false, // super noisy
		GenesisState:      appGenesis,
		InterfaceRegistry: encCfg.InterfaceRegistry,
		KeyringOptions:    []keyring.Option{},
		LegacyAmino:       encCfg.Amino,
		MinGasPrices:      fmt.Sprintf("0.000006%s", denoms.NIBI),
		NumValidators:     1,
		PruningStrategy:   types.PruningOptionNothing,
		SigningAlgo:       string(hd.Secp256k1Type),
		StakingTokens:     sdk.TokensFromConsensusPower(500, sdk.DefaultPowerReduction),
		StartingTokens: sdk.NewCoins(
			sdk.NewCoin(denoms.NUSD, sdk.TokensFromConsensusPower(1e12, sdk.DefaultPowerReduction)),
			sdk.NewCoin(denoms.NIBI, sdk.TokensFromConsensusPower(1e12, sdk.DefaultPowerReduction)),
			sdk.NewCoin(denoms.USDC, sdk.TokensFromConsensusPower(1e12, sdk.DefaultPowerReduction)),
		),
		TimeoutCommit: time.Second / 2,
		TxConfig:      encCfg.TxConfig,
	}
}
