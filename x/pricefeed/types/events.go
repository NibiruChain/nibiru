package types

// Pricefeed module event types
const (
	EventTypePairPriceUpdated   = "market_price_updated"
	EventTypeOracleUpdatedPrice = "oracle_updated_price"
	EventTypeNoValidPrices      = "no_valid_prices"

	AttributeValueCategory = ModuleName
	AttributePairID        = "pair_id"
	AttributePairPrice     = "market_price"
	AttributeOracle        = "oracle"
	AttributeExpiry        = "expiry"
)
