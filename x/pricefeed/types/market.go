package types

import (
	"errors"
	"fmt"
	"strings"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// NewPair returns a new Pair
func NewPair(id, base, quote string, oracles []sdk.AccAddress, active bool) Pair {
	return Pair{
		PairID:     id,
		BaseAsset:  base,
		QuoteAsset: quote,
		Oracles:    oracles,
		Active:     active,
	}
}

// Validate performs a basic validation of the market params
func (m Pair) Validate() error {
	if strings.TrimSpace(m.PairID) == "" {
		return errors.New("market id cannot be blank")
	}
	if err := sdk.ValidateDenom(m.BaseAsset); err != nil {
		return fmt.Errorf("invalid base asset: %w", err)
	}
	if err := sdk.ValidateDenom(m.QuoteAsset); err != nil {
		return fmt.Errorf("invalid quote asset: %w", err)
	}
	seenOracles := make(map[string]bool)
	for i, oracle := range m.Oracles {
		if len(oracle) == 0 {
			return fmt.Errorf("oracle %d is empty", i)
		}
		if seenOracles[oracle.String()] {
			return fmt.Errorf("duplicated oracle %s", oracle)
		}
		seenOracles[oracle.String()] = true
	}
	return nil
}

// ToPairResponse returns a new PairResponse from a Pair
func (m Pair) ToPairResponse() PairResponse {
	return NewPairResponse(m.PairID, m.BaseAsset, m.QuoteAsset, m.Oracles, m.Active)
}

// Pairs is a slice of Pair
type Pairs []Pair

// Validate checks if all the markets are valid and there are no duplicated
// entries.
func (ms Pairs) Validate() error {
	seenPairs := make(map[string]bool)
	for _, m := range ms {
		if seenPairs[m.PairID] {
			return fmt.Errorf("duplicated market %s", m.PairID)
		}
		if err := m.Validate(); err != nil {
			return err
		}
		seenPairs[m.PairID] = true
	}
	return nil
}

// NewPairResponse returns a new PairResponse
func NewPairResponse(id, base, quote string, oracles []sdk.AccAddress, active bool) PairResponse {
	var strOracles []string
	for _, oracle := range oracles {
		strOracles = append(strOracles, oracle.String())
	}

	return PairResponse{
		PairID:     id,
		BaseAsset:  base,
		QuoteAsset: quote,
		Oracles:    strOracles,
		Active:     active,
	}
}

// PairResponses is a slice of PairResponse
type PairResponses []PairResponse

// NewCurrentPrice returns an instance of CurrentPrice
func NewCurrentPrice(pairID string, price sdk.Dec) CurrentPrice {
	return CurrentPrice{PairID: pairID, Price: price}
}

// CurrentPrices is a slice of CurrentPrice
type CurrentPrices []CurrentPrice

// NewCurrentPriceResponse returns an instance of CurrentPriceResponse
func NewCurrentPriceResponse(pairID string, price sdk.Dec) CurrentPriceResponse {
	return CurrentPriceResponse{PairID: pairID, Price: price}
}

// CurrentPriceResponses is a slice of CurrentPriceResponse
type CurrentPriceResponses []CurrentPriceResponse

// NewPostedPrice returns a new PostedPrice
func NewPostedPrice(pairID string, oracle sdk.AccAddress, price sdk.Dec, expiry time.Time) PostedPrice {
	return PostedPrice{
		PairID:        pairID,
		OracleAddress: oracle,
		Price:         price,
		Expiry:        expiry,
	}
}

// Validate performs a basic check of a PostedPrice params.
func (pp PostedPrice) Validate() error {
	if strings.TrimSpace(pp.PairID) == "" {
		return errors.New("market id cannot be blank")
	}
	if len(pp.OracleAddress) == 0 {
		return errors.New("oracle address cannot be empty")
	}
	if pp.Price.IsNegative() {
		return fmt.Errorf("posted price cannot be negative %s", pp.Price)
	}
	if pp.Expiry.Unix() <= 0 {
		return errors.New("expiry time cannot be zero")
	}
	return nil
}

// PostedPrices is a slice of PostedPrice
type PostedPrices []PostedPrice

// Validate checks if all the posted prices are valid and there are no
// duplicated entries.
func (pps PostedPrices) Validate() error {
	seenPrices := make(map[string]bool)
	for _, pp := range pps {
		if !pp.OracleAddress.Empty() && seenPrices[pp.PairID+pp.OracleAddress.String()] {
			return fmt.Errorf("duplicated posted price for marked id %s and oracle address %s", pp.PairID, pp.OracleAddress)
		}

		if err := pp.Validate(); err != nil {
			return err
		}
		seenPrices[pp.PairID+pp.OracleAddress.String()] = true
	}

	return nil
}

// NewPostedPrice returns a new PostedPrice
func NewPostedPriceResponse(pairID string, oracle sdk.AccAddress, price sdk.Dec, expiry time.Time) PostedPriceResponse {
	return PostedPriceResponse{
		PairID:        pairID,
		OracleAddress: oracle.String(),
		Price:         price,
		Expiry:        expiry,
	}
}

// PostedPriceResponses is a slice of PostedPriceResponse
type PostedPriceResponses []PostedPriceResponse

// SortDecs provides the interface needed to sort sdk.Dec slices
type SortDecs []sdk.Dec

func (a SortDecs) Len() int           { return len(a) }
func (a SortDecs) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a SortDecs) Less(i, j int) bool { return a[i].LT(a[j]) }
