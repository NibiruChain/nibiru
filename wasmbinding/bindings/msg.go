package bindings

import sdk "github.com/cosmos/cosmos-sdk/types"

type NibiruMsg struct {
	OpenPosition  *OpenPosition  `json:"open_position,omitempty"`
	ClosePosition *ClosePosition `json:"close_position,omitempty"`
}

type OpenPosition struct {
	Pair                 string  `json:"pair"`
	Side                 int     `json:"side"`
	QuoteAssetAmount     sdk.Int `json:"quote_asset_amount"`
	Leverage             sdk.Dec `json:"leverage"`
	BaseAssetAmountLimit sdk.Int `json:"base_asset_amount_limit"`
}

type ClosePosition struct {
	Pair string `json:"pair"`
}
