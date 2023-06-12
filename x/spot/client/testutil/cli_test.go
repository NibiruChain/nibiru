package testutil

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	testutilcli "github.com/NibiruChain/nibiru/x/common/testutil/cli"
	genesis "github.com/NibiruChain/nibiru/x/common/testutil/genesis"
)

func TestIntegrationTestSuite(t *testing.T) {
	coinsFromGenesis := []string{
		denoms.NIBI,
		denoms.NUSD,
		denoms.USDC,
		"coin-1",
		"coin-2",
		"coin-3",
		"coin-4",
		"coin-5",
	}

	app.SetPrefixes(app.AccountAddressPrefix)
	genesisState := genesis.NewTestGenesisState(app.MakeEncodingConfig())

	genesisState = WhitelistGenesisAssets(
		genesisState,
		coinsFromGenesis,
	)

	homeDir := t.TempDir()
	cfg := testutilcli.BuildNetworkConfig(genesisState)
	cfg.StartingTokens = sdk.NewCoins(
		sdk.NewInt64Coin(denoms.NIBI, 2e12), // for pool creation fee and more for tx fees
	)

	for _, coin := range coinsFromGenesis {
		cfg.StartingTokens = cfg.StartingTokens.Add(sdk.NewInt64Coin(coin, 40000))
	}

	suite.Run(t, NewIntegrationTestSuite(homeDir, cfg))
}
