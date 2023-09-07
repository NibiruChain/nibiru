package types

import (
	"encoding/json"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	epochstypes "github.com/NibiruChain/nibiru/x/epochs/types"

	"github.com/cosmos/cosmos-sdk/codec"
)

// DefaultGenesis returns the default Capability genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Markets:          []Market{},
		Amms:             []AMM{},
		Positions:        []Position{},
		ReserveSnapshots: []ReserveSnapshot{},
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	for _, m := range gs.Markets {
		if err := m.Validate(); err != nil {
			return err
		}
	}

	for _, m := range gs.Amms {
		if err := m.Validate(); err != nil {
			return err
		}
	}

	for _, pos := range gs.Positions {
		if err := pos.Validate(); err != nil {
			return err
		}
	}

	return nil
}

func DefaultMarket(pair asset.Pair) Market {
	return Market{
		Pair:                            pair,
		Enabled:                         true,
		Version:                         1,
		LatestCumulativePremiumFraction: sdk.ZeroDec(),
		ExchangeFeeRatio:                sdk.MustNewDecFromStr("0.0010"),
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
}

func GetGenesisStateFromAppState(cdc codec.JSONCodec, appState map[string]json.RawMessage) *GenesisState {
	var genesisState GenesisState

	if appState[ModuleName] != nil {
		cdc.MustUnmarshalJSON(appState[ModuleName], &genesisState)
	}

	return &genesisState
}
