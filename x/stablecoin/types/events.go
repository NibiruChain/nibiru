package types

// Event types for x/stablecoin
const (
	EventTypeMintStable = "stable_minted"
	EventTypeBurnStable = "stable_burned"
	EventTypeMintNIBI   = "nibi_minted"
	EventTypeBurnNIBI   = "nibi_burned"

	AttributeFrom = "from"
	AttributeTo   = "to"
	AttributeAmount
)
