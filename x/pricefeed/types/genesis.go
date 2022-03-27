package types

// this line is used by starport scaffolding # genesis/types/import

// DefaultIndex is the default capability global index
const DefaultIndex uint64 = 1

// DefaultGenesis returns the default Capability genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		// this line is used by starport scaffolding # genesis/types/default
		Params:       DefaultParams(),
		PostedPrices: []PostedPrice{},
	}
}

// NewGenesisState creates a new genesis state for the pricefeed module
func NewGenesisState(p Params, pp []PostedPrice) *GenesisState {
	return &GenesisState{
		Params:       p,
		PostedPrices: pp,
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	// this line is used by starport scaffolding # genesis/types/validate
	if err := gs.Params.Validate(); err != nil {
		return err
	}

	return gs.PostedPrices.Validate()
}
