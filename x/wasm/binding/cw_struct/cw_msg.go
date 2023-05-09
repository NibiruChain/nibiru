package cw_struct

import sdk "github.com/cosmos/cosmos-sdk/types"

// BindingMsg corresponds to the 'ExecuteMsg' enum in the CosmWasm binding
// contracts (Rust). It specifies which wasm execute messages can be called with
// Nibiru bindings and specifies the JSON schema that connects app ⇔ Wasm.
//
// See:
// - https://github.com/NibiruChain/cw-nibiru/blob/90df123f8d32d47b5b280ec6ae7dde0f9dbf2787/contracts/bindings-perp/src/msg.rs
type BindingMsg struct {
	// bindings-perp ExecuteMsg enum types
	OpenPosition          *OpenPosition          `json:"open_position,omitempty"`
	ClosePosition         *ClosePosition         `json:"close_position,omitempty"`
	MultiLiquidate        *MultiLiquidate        `json:"multi_liquidate,omitempty"` // TODO
	AddMargin             *AddMargin             `json:"add_margin,omitempty"`
	RemoveMargin          *RemoveMargin          `json:"remove_margin,omitempty"`
	DonateToInsuranceFund *DonateToInsuranceFund `json:"donate_to_insurance_fund,omitempty"` // TODO
	PegShift              *PegShift              `json:"peg_shift,omitempty"`
	DepthShift            *DepthShift            `json:"depth_shift,omitempty"`
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

type PegShift struct {
	Pair    string  `json:"pair"`
	PegMult sdk.Dec `json:"peg_mult"`
}

type DepthShift struct {
	Pair      string  `json:"pair"`
	DepthMult sdk.Dec `json:"depth_mult"`
}

type DonateToInsuranceFund struct {
	Sender   string   `json:"sender"`
	Donation sdk.Coin `json:"donation"`
}
