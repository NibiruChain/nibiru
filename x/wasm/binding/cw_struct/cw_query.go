package cw_struct

import sdk "github.com/cosmos/cosmos-sdk/types"

// BindingQuery corresponds to the BindingQuery enum in CosmWasm binding
// contracts (Rust). It specifies which queries can be called into the
// Nibiru bindings, and describes their JSON schema for connecting app ⇔ Wasm.
type BindingQuery struct {
	// TODO
	Reserves        *Reserves
	AllMarkets      *AllMarkets
	BasePrice       *BasePrice
	Positions       *Positions
	Position        *Position
	PremiumFraction *PremiumFraction
	Metrics         *Metrics
	ModuleAccounts  *ModuleAccounts
	PerpParams      *PerpParams
}

type Reserves struct {
	Pair string `json:"pair"`
}

type AllMarkets struct {
}

type BasePrice struct {
	Pair       string  `json:"pair"`
	IsLong     bool    `json:"is_long"`
	BaseAmount sdk.Int `json:"base_amount"`
}

type Positions struct {
	Trader string `json:"trader"`
}

type Position struct {
	Trader string `json:"trader"`
	Pair   string `json:"pair"`
}

type PremiumFraction struct {
	Pair string `json:"pair"`
}

type Metrics struct {
	Pair string `json:"pair"`
}

type ModuleAccounts struct {
}

type PerpParams struct {
}
