package types

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/NibiruChain/nibiru/x/common"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (pair Pair) PairID() string {
	return common.PoolNameFromDenoms([]string{pair.Token0, pair.Token1})
}

// NewPair returns a new Pair
func NewPair(
	token0 string, token1 string, oracles []sdk.AccAddress, active bool,
) Pair {
	return Pair{
		Token0:  token0,
		Token1:  token1,
		Oracles: oracles,
		Active:  active,
	}
}

// name is the name of the pool that corresponds to the two assets on this pair.
func (pair Pair) Name() string {
	assets := common.AssetPair{Token0: pair.Token0, Token1: pair.Token1}
	return assets.Name()
}

func (pair Pair) AsString() string {
	return fmt.Sprintf("%s:%s", pair.Token0, pair.Token1)
}

func (pair Pair) IsProperOrder() bool {
	assets := common.AssetPair{Token0: pair.Token0, Token1: pair.Token1}
	return assets.IsProperOrder()
}

func (pair Pair) Inverse() Pair {
	return Pair{pair.Token1, pair.Token0, pair.Oracles, pair.Active}
}

// Validate performs a basic validation of the market params
func (m Pair) Validate() error {
	if strings.TrimSpace(m.PairID()) == "" {
		return errors.New("market id cannot be blank")
	}
	if err := sdk.ValidateDenom(m.Token1); err != nil {
		return fmt.Errorf("invalid token1 asset: %w", err)
	}
	if err := sdk.ValidateDenom(m.Token0); err != nil {
		return fmt.Errorf("invalid token0 asset: %w", err)
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
	return NewPairResponse(m.Token1, m.Token0, m.Oracles, m.Active)
}

// Pairs is a slice of Pair
type Pairs []Pair

// Validate checks if all the markets are valid and there are no duplicated
// entries.
func (ms Pairs) Validate() error {
	seenPairs := make(map[string]bool)
	for _, m := range ms {
		pairID := common.PoolNameFromDenoms([]string{m.Token0, m.Token1})
		if seenPairs[pairID] {
			return fmt.Errorf("duplicated market %s", pairID)
		}
		if err := m.Validate(); err != nil {
			return err
		}
		seenPairs[pairID] = true
	}
	return nil
}

// NewPairResponse returns a new PairResponse
func NewPairResponse(token1 string, token0 string, oracles []sdk.AccAddress, active bool) PairResponse {
	var strOracles []string
	for _, oracle := range oracles {
		strOracles = append(strOracles, oracle.String())
	}

	pairID := common.PoolNameFromDenoms([]string{token0, token1})
	return PairResponse{
		PairID:  pairID,
		Token1:  token1,
		Token0:  token0,
		Oracles: strOracles,
		Active:  active,
	}
}

// PairResponses is a slice of PairResponse
type PairResponses []PairResponse

/*
NewCurrentPrice returns an instance of CurrentPrice

Args:
  token0 (string):
  token1 (string):
  price (sdk.Dec): Price in units of token1 / token0
Returns:
  (CurrentPrice): Price for the asset pair.
*/
func NewCurrentPrice(token0 string, token1 string, price sdk.Dec) CurrentPrice {
	assetPair := common.AssetPair{Token0: token0, Token1: token1}
	return CurrentPrice{PairID: assetPair.Name(), Price: price}
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
func NewPostedPriceResponse(
	pairID string, oracle sdk.AccAddress, price sdk.Dec, expiry time.Time,
) PostedPriceResponse {
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
