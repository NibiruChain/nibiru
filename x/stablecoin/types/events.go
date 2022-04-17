package types

// Event types for x/stablecoin
const (
	EventTypeMintStable = "stable_minted"
	EventTypeBurnStable = "stable_burned"
	EventTypeMintGov    = "nibi_minted"
	EventTypeBurnGov    = "nibi_burned"

	AttributeFrom = "from"
	AttributeTo   = "to"
	AttributeAmount
)
