package testutil

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/NibiruChain/nibiru/x/common"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/testutil/network"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	storetypes "github.com/cosmos/cosmos-sdk/store/types"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

func DefaultFeeString(cfg network.Config) string {
	feeCoins := sdk.NewCoins(sdk.NewCoin(cfg.BondDenom, sdk.NewInt(10)))
	return fmt.Sprintf("--%s=%s", flags.FlagFees, feeCoins.String())
}

const (
	base   = 10
	bitlen = 64
)

func ParseUint64SliceFromString(s string, separator string) ([]uint64, error) {
	var parsedInts []uint64
	for _, s := range strings.Split(s, separator) {
		s = strings.TrimSpace(s)

		parsed, err := strconv.ParseUint(s, base, bitlen)
		if err != nil {
			return []uint64{}, err
		}
		parsedInts = append(parsedInts, parsed)
	}
	return parsedInts, nil
}

func ParseSdkIntFromString(s string, separator string) ([]sdk.Int, error) {
	var parsedInts []sdk.Int
	for _, weightStr := range strings.Split(s, separator) {
		weightStr = strings.TrimSpace(weightStr)

		parsed, err := strconv.ParseUint(weightStr, base, bitlen)
		if err != nil {
			return parsedInts, err
		}
		parsedInts = append(parsedInts, sdk.NewIntFromUint64(parsed))
	}
	return parsedInts, nil
}

// DefaultConfig returns a default configuration suitable for nearly all
// testing requirements.
func DefaultConfig() network.Config {
	encCfg := app.MakeTestEncodingConfig()

	return network.Config{
		Codec:             encCfg.Marshaler,
		TxConfig:          encCfg.TxConfig,
		LegacyAmino:       encCfg.Amino,
		InterfaceRegistry: encCfg.InterfaceRegistry,
		AccountRetriever:  authtypes.AccountRetriever{},
		AppConstructor:    NewAppConstructor(),
		GenesisState:      app.ModuleBasics.DefaultGenesis(encCfg.Marshaler),
		TimeoutCommit:     1 * time.Second / 2,
		ChainID:           "nibiru-code-test",
		NumValidators:     1,
		BondDenom:         sdk.DefaultBondDenom,
		MinGasPrices:      fmt.Sprintf("0.000006%s", sdk.DefaultBondDenom),
		AccountTokens:     sdk.TokensFromConsensusPower(1000, sdk.DefaultPowerReduction),
		StakingTokens:     sdk.TokensFromConsensusPower(500, sdk.DefaultPowerReduction),
		BondedTokens:      sdk.TokensFromConsensusPower(100, sdk.DefaultPowerReduction),
		StartingTokens: sdk.NewCoins(
			sdk.NewCoin(common.StableDenom, sdk.TokensFromConsensusPower(100, sdk.DefaultPowerReduction)),
			sdk.NewCoin(common.GovDenom, sdk.TokensFromConsensusPower(1000, sdk.DefaultPowerReduction)),
			sdk.NewCoin(common.CollDenom, sdk.TokensFromConsensusPower(100, sdk.DefaultPowerReduction)),
			sdk.NewCoin("stake", sdk.TokensFromConsensusPower(1000, sdk.DefaultPowerReduction)),
		),
		PruningStrategy: storetypes.PruningOptionNothing,
		CleanupDir:      true,
		SigningAlgo:     string(hd.Secp256k1Type),
		KeyringOptions:  []keyring.Option{},
	}
}

func NewAppConstructor() network.AppConstructor {
	return func(val network.Validator) servertypes.Application {
		return New(true)
	}
}
