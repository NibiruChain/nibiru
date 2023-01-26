package testapp

import (
	"fmt"
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	tmrand "github.com/tendermint/tendermint/libs/rand"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/testutil/cli"
)

// BuildNetworkConfig returns a configuration for a local in-testing network
func BuildNetworkConfig(appGenesis GenesisState) cli.Config {
	encCfg := app.MakeTestEncodingConfig()

	return cli.Config{
		Codec:             encCfg.Marshaler,
		TxConfig:          encCfg.TxConfig,
		LegacyAmino:       encCfg.Amino,
		InterfaceRegistry: encCfg.InterfaceRegistry,
		AccountRetriever:  authtypes.AccountRetriever{},
		AppConstructor: func(val cli.Validator) servertypes.Application {
			return NewTestNibiruAppWithGenesis(appGenesis)
		},
		GenesisState:  appGenesis,
		TimeoutCommit: time.Second / 2,
		ChainID:       "chain-" + tmrand.NewRand().Str(6),
		NumValidators: 1,
		BondDenom:     denoms.NIBI,
		MinGasPrices:  fmt.Sprintf("0.000006%s", denoms.NIBI),
		AccountTokens: sdk.TokensFromConsensusPower(1000, sdk.DefaultPowerReduction),
		StakingTokens: sdk.TokensFromConsensusPower(500, sdk.DefaultPowerReduction),
		BondedTokens:  sdk.TokensFromConsensusPower(100, sdk.DefaultPowerReduction),
		StartingTokens: sdk.NewCoins(
			sdk.NewCoin(denoms.NUSD, sdk.TokensFromConsensusPower(100, sdk.DefaultPowerReduction)),
			sdk.NewCoin(denoms.NIBI, sdk.TokensFromConsensusPower(1000, sdk.DefaultPowerReduction)),
			sdk.NewCoin(denoms.USDC, sdk.TokensFromConsensusPower(100, sdk.DefaultPowerReduction)),
		),
		PruningStrategy: storetypes.PruningOptionNothing,
		CleanupDir:      true,
		SigningAlgo:     string(hd.Secp256k1Type),
		KeyringOptions:  []keyring.Option{},
	}
}
