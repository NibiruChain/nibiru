package testutil

import (
	"fmt"
	"time"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktestutil "github.com/cosmos/cosmos-sdk/x/bank/client/testutil"
	tmrand "github.com/tendermint/tendermint/libs/rand"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common"
	testutilcli "github.com/NibiruChain/nibiru/x/testutil/cli"
)

func DefaultFeeString(denom string) string {
	feeCoins := sdk.NewCoins(sdk.NewCoin(denom, sdk.NewInt(10)))
	return fmt.Sprintf("--%s=%s", flags.FlagFees, feeCoins.String())
}

// DefaultConfig returns a default configuration suitable for nearly all
// testing requirements.
func DefaultConfig() testutilcli.Config {
	encCfg := app.MakeTestEncodingConfig()

	return testutilcli.Config{
		Codec:             encCfg.Marshaler,
		TxConfig:          encCfg.TxConfig,
		LegacyAmino:       encCfg.Amino,
		InterfaceRegistry: encCfg.InterfaceRegistry,
		AccountRetriever:  authtypes.AccountRetriever{},
		AppConstructor: func(val testutilcli.Validator) servertypes.Application {
			return NewTestApp(true)
		},
		GenesisState:  app.ModuleBasics.DefaultGenesis(encCfg.Marshaler),
		TimeoutCommit: time.Second / 2,
		ChainID:       "chain-" + tmrand.NewRand().Str(6),
		NumValidators: 1,
		BondDenom:     common.GovDenom,
		MinGasPrices:  fmt.Sprintf("0.000006%s", common.GovDenom),
		AccountTokens: sdk.TokensFromConsensusPower(1000, sdk.DefaultPowerReduction),
		StakingTokens: sdk.TokensFromConsensusPower(500, sdk.DefaultPowerReduction),
		BondedTokens:  sdk.TokensFromConsensusPower(100, sdk.DefaultPowerReduction),
		StartingTokens: sdk.NewCoins(
			sdk.NewCoin(common.StableDenom, sdk.TokensFromConsensusPower(100, sdk.DefaultPowerReduction)),
			sdk.NewCoin(common.GovDenom, sdk.TokensFromConsensusPower(1000, sdk.DefaultPowerReduction)),
			sdk.NewCoin(common.CollDenom, sdk.TokensFromConsensusPower(100, sdk.DefaultPowerReduction)),
		),
		PruningStrategy: storetypes.PruningOptionNothing,
		CleanupDir:      true,
		SigningAlgo:     string(hd.Secp256k1Type),
		KeyringOptions:  []keyring.Option{},
	}
}

// FillWalletFromValidator fills the wallet with some coins that come from the validator.
// Used for cli tests.
func FillWalletFromValidator(
	addr sdk.AccAddress, balance sdk.Coins, val *testutilcli.Validator, feesDenom string,
) (sdk.AccAddress, error) {
	_, err := banktestutil.MsgSendExec(
		val.ClientCtx,
		val.Address,
		addr,
		balance,
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastBlock),
		DefaultFeeString(feesDenom),
	)
	if err != nil {
		return nil, err
	}

	return addr, nil
}
