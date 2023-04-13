package cw_struct

import sdk "github.com/cosmos/cosmos-sdk/types"

type BindingMsg struct {
	OpenPosition          *OpenPosition          `json:"open_position,omitempty"`
	ClosePosition         *ClosePosition         `json:"close_position,omitempty"`
	MultiLiquidate        *MultiLiquidate        `json:"multi_liquidate,omitempty"`
	AddMargin             *AddMargin             `json:"add_margin,omitempty"`
	RemoveMargin          *RemoveMargin          `json:"remove_margin,omitempty"`
	DonateToInsuranceFund *DonateToInsuranceFund `json:"donate_to_insurance_fund,omitempty"`
}

type OpenPosition struct {
	Sender          string  `json:"sender"`
	Pair            string  `json:"pair"`
	IsLong          bool    `json:"is_long"`
	QuoteAmount     sdk.Int `json:"quote_amount"`
	Leverage        sdk.Dec `json:"leverage"`
	BaseAmountLimit sdk.Int `json:"base_amount_limit"`
}

type ClosePosition struct {
	Sender string `json:"sender"`
	Pair   string `json:"pair"`
}

type MultiLiquidate struct {
	Sender       string            `json:"sender"`
	Liquidations []LiquidationArgs `json:"liquidations"`
}

type LiquidationArgs struct {
	Pair   string `json:"pair"`
	Trader string `json:"trader"`
}

type AddMargin struct {
	Sender string   `json:"sender"`
	Pair   string   `json:"pair"`
	Margin sdk.Coin `json:"margin"`
}

type RemoveMargin struct {
	Sender string   `json:"sender"`
	Pair   string   `json:"pair"`
	Margin sdk.Coin `json:"margin"`
}

type DonateToInsuranceFund struct {
	Sender   string   `json:"sender"`
	Donation sdk.Coin `json:"donation"`
}
