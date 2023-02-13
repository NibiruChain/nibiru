package bindings

type NibiruQuery struct {
	Position  *Position  `json:"position,omitempty"`
	Positions *Positions `json:"positions,omitempty"`
}

type Position struct {
	Trader string `json:"trader"`
	Pair   string `json:"pair"`
}

type Positions struct {
	Trader string `json:"trader"`
}
