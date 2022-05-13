package types

const (
	EventSnapshotSaved    = "reserve_snapshot_saved"
	AttributeBlockHeight  = "block_height"
	AttributeQuoteReserve = "quote_reserve"
	AttributeBaseReserve  = "base_reserve"

	EventSwapQuoteForBase     = "swap_input"
	EventSwapBaseForQuote     = "swap_output"
	AttributeQuoteAssetAmount = "quote_asset_amount"
	AttributeBaseAssetAmount  = "base_asset_amount"
)
