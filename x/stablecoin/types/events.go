package types

// Event types for x/stablecoin
const (
	EventTypeMintStable = "stable_minted"
	EventTypeBurnStable = "stable_burned"
	EventTypeMintMtrx   = "mtrx_minted"
	EventTypeBurnMtrx   = "mtrx_burned"

	AttributeFrom = "from"
	AttributeTo   = "to"
	AttributeAmount
)
