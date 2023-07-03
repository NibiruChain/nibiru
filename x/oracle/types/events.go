package types

// Oracle module event types
const (
	EventTypeExchangeRateUpdate = "exchange_rate_update"
	EventTypePrevote            = "prevote"
	EventTypeVote               = "vote"
	EventTypeAggregatePrevote   = "aggregate_prevote"
	EventTypeAggregateVote      = "aggregate_vote"

	AttributeKeyPair          = "pair"
	AttributeKeyVoter         = "voter"
	AttributeKeyExchangeRate  = "exchange_rate"
	AttributeKeyExchangeRates = "exchange_rates"
	AttributeKeyOperator      = "operator"
	AttributeKeyFeeder        = "feeder"

	AttributeValueCategory = ModuleName
)
