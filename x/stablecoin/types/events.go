package types

// x/stablecoin events
const (
	EventTypeMintStable = "mint_stable"
	EventTypeBurnStable = "burn_stable"
	EventTypeMintMtrx   = "mint_mtrx"
	EventTypeBurnMtrx   = "burn_mtrx"

	AttributeFromAddr    = "from"
	AttributeToAddr      = "to"
	AttributeTokenAmount = "amount"
	AttributeTokenDenom  = "denom"
)
