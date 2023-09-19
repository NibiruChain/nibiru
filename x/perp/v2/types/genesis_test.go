package types_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	epochstypes "github.com/NibiruChain/nibiru/x/epochs/types"
	"github.com/NibiruChain/nibiru/x/perp/v2/types"
)

func TestDefaultGenesis(t *testing.T) {
	// Initialization
	genesis := types.DefaultGenesis()

	// Test the conditions
	require.NotNil(t, genesis)
	require.Empty(t, genesis.Markets)
	require.Empty(t, genesis.Amms)
	require.Empty(t, genesis.Positions)
	require.Empty(t, genesis.ReserveSnapshots)
}

func TestGenesisValidate(t *testing.T) {
	pair := asset.MustNewPair("ubtc:unusd")
	validMarket := types.DefaultMarket(pair)
	validAmms := types.AMM{
		BaseReserve:     sdk.OneDec(),
		QuoteReserve:    sdk.OneDec(),
		PriceMultiplier: sdk.OneDec(),
		SqrtDepth:       sdk.OneDec(),
	}
	validPositions := types.Position{
		TraderAddress:                   "cosmos1zaavvzxez0elundtn32qnk9lkm8kmcszzsv80v",
		Pair:                            pair,
		Size_:                           sdk.OneDec(),
		Margin:                          sdk.OneDec(),
		OpenNotional:                    sdk.OneDec(),
		LatestCumulativePremiumFraction: sdk.OneDec(),
		LastUpdatedBlockNumber:          0,
	}
	invalidMarket := &types.Market{
		Pair:                            pair,
		Enabled:                         true,
		LatestCumulativePremiumFraction: sdk.ZeroDec(),
		ExchangeFeeRatio:                sdk.MustNewDecFromStr("-0.0010"),
		EcosystemFundFeeRatio:           sdk.MustNewDecFromStr("0.0010"),
		LiquidationFeeRatio:             sdk.MustNewDecFromStr("0.0500"),
		PartialLiquidationRatio:         sdk.MustNewDecFromStr("0.5000"),
		FundingRateEpochId:              epochstypes.ThirtyMinuteEpochID,
		MaxFundingRate:                  sdk.NewDec(1),
		TwapLookbackWindow:              time.Minute * 30,
		PrepaidBadDebt:                  sdk.NewCoin(denoms.USDC, sdk.ZeroInt()),
		MaintenanceMarginRatio:          sdk.MustNewDecFromStr("0.0625"),
		MaxLeverage:                     sdk.NewDec(10),
	}
	invalidAmms := types.AMM{
		BaseReserve:     sdk.ZeroDec(),
		QuoteReserve:    sdk.OneDec(),
		PriceMultiplier: sdk.OneDec(),
		SqrtDepth:       sdk.OneDec(),
	}
	invalidPositions := types.Position{
		TraderAddress:                   "cosmos1zaavvzxez0elundtn32qnk9lkm8kmcszzsv80v",
		Pair:                            pair,
		Size_:                           sdk.ZeroDec(),
		Margin:                          sdk.OneDec(),
		OpenNotional:                    sdk.OneDec(),
		LatestCumulativePremiumFraction: sdk.OneDec(),
		LastUpdatedBlockNumber:          0,
	}

	tests := []struct {
		name         string
		setupGenesis func() *types.GenesisState
		shouldFail   bool
	}{
		{
			name: "valid genesis",
			setupGenesis: func() *types.GenesisState {
				genesis := types.GenesisState{
					Markets:          []types.Market{validMarket},
					Amms:             []types.AMM{validAmms},
					Positions:        []types.Position{validPositions},
					ReserveSnapshots: []types.ReserveSnapshot{},
				}

				return &genesis
			},
			shouldFail: false,
		},
		{
			name: "invalid market",
			setupGenesis: func() *types.GenesisState {
				genesis := types.GenesisState{
					Markets:          []types.Market{*invalidMarket},
					Amms:             []types.AMM{validAmms},
					Positions:        []types.Position{validPositions},
					ReserveSnapshots: []types.ReserveSnapshot{},
				}

				return &genesis
			},
			shouldFail: true,
		},
		{
			name: "invalid amm",
			setupGenesis: func() *types.GenesisState {
				genesis := types.GenesisState{
					Markets:          []types.Market{*invalidMarket},
					Amms:             []types.AMM{invalidAmms},
					Positions:        []types.Position{validPositions},
					ReserveSnapshots: []types.ReserveSnapshot{},
				}

				return &genesis
			},
			shouldFail: true,
		},
		{
			name: "invalid position",
			setupGenesis: func() *types.GenesisState {
				genesis := types.GenesisState{
					Markets:          []types.Market{*invalidMarket},
					Amms:             []types.AMM{validAmms},
					Positions:        []types.Position{invalidPositions},
					ReserveSnapshots: []types.ReserveSnapshot{},
				}

				return &genesis
			},
			shouldFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.setupGenesis().Validate()
			if tt.shouldFail {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestDefaultMarket(t *testing.T) {
	// Initialize asset pair
	pair := asset.NewPair("ubtc", "unusd")

	// Initialize the market with default values
	market := types.DefaultMarket(pair)

	// Test the conditions
	require.NotNil(t, market)
	// Continue testing individual attributes of market as required
}

func TestGetGenesisStateFromAppState(t *testing.T) {
	// Initialize codec and AppState
	var cdc codec.JSONCodec
	appState := make(map[string]json.RawMessage)

	// Insert valid ModuleName data into AppState
	// insert valid ModuleName data

	// Get the GenesisState
	genesis := types.GetGenesisStateFromAppState(cdc, appState)

	// Test the conditions
	require.NotNil(t, genesis)
}
