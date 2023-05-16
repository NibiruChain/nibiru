package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type PositionResp struct {
	Position *Position
	// The amount of quote assets exchanged.
	ExchangedNotionalValue sdk.Dec
	// The amount of base assets exchanged.
	ExchangedPositionSize sdk.Dec
	// The amount of bad debt accrued during this position change.
	// Measured in absolute value of quote units.
	// If greater than zero, then the position change event will likely fail.
	BadDebt sdk.Dec
	// The funding payment applied on this position change.
	FundingPayment sdk.Dec
	// The amount of PnL realized on this position changed, measured in quote
	// units.
	RealizedPnl sdk.Dec
	// The unrealized PnL in the position after the position change.
	UnrealizedPnlAfter sdk.Dec
	// The amount of margin the trader has to give to the vault.
	// A negative value means the vault pays the trader.
	MarginToVault sdk.Dec
	// The position's notional value after the position change, measured in quote
	// units.
	PositionNotional sdk.Dec
}

type LiquidateResp struct {
	// Amount of bad debt created by the liquidation event
	BadDebt sdk.Int
	// Fee paid to the liquidator
	FeeToLiquidator sdk.Int
	// Fee paid to the Perp EF fund
	FeeToPerpEcosystemFund sdk.Int
	// Address of the liquidator
	Liquidator string
	// Position response from the close or open reverse position
	PositionResp *PositionResp
}
