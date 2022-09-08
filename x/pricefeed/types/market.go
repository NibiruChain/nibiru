package types

import (
	"errors"
	"fmt"
	"strings"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
)

// Markets defines the markets
type Markets []Market

func NewMarket(pair common.AssetPair, oracles []sdk.AccAddress, active bool) Market {
	if err := pair.Validate(); err != nil {
		panic(err)
	}
	var strOracles []string
	for _, oracle := range oracles {
		strOracles = append(strOracles, oracle.String())
	}
	return Market{
		PairID:  pair.String(),
		Oracles: strOracles,
		Active:  active,
	}
}

// NewCurrentPrice returns an instance of CurrentPrice
// Args:
//
//	token0 (string):
//	token1 (string):
//	price (sdk.Dec): Price in units of token1 / token0
//
// Returns:
//
//	CurrentPrice: Price for the asset pair.
func NewCurrentPrice(token0 string, token1 string, price sdk.Dec) CurrentPrice {
	assetPair := common.AssetPair{Token0: token0, Token1: token1}
	return CurrentPrice{PairID: assetPair.String(), Price: price}
}

// CurrentPrices is a slice of CurrentPrice
type CurrentPrices []CurrentPrice

// CurrentPriceResponses is a slice of CurrentPriceResponse
type CurrentPriceResponses []CurrentPriceResponse

// NewPostedPrice returns a new PostedPrice
func NewPostedPrice(pair common.AssetPair, oracle sdk.AccAddress, price sdk.Dec, expiry time.Time,
) PostedPrice {
	return PostedPrice{
		PairID: pair.String(),
		Oracle: oracle.String(),
		Price:  price,
		Expiry: expiry,
	}
}

// Validate performs a basic check of a PostedPrice params.
func (pp PostedPrice) Validate() error {
	if strings.TrimSpace(pp.PairID) == "" {
		return errors.New("market id cannot be blank")
	}
	if _, err := common.NewAssetPair(pp.PairID); err != nil {
		return err
	}
	if len(pp.Oracle) == 0 {
		return errors.New("oracle address cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(pp.Oracle); err != nil {
		return err
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
		if !(pp.Oracle == "") && seenPrices[pp.PairID+pp.Oracle] {
			return fmt.Errorf("duplicated posted price for market id %s and oracle address %s", pp.PairID, pp.Oracle)
		}

		if err := pp.Validate(); err != nil {
			return err
		}
		seenPrices[pp.PairID+pp.Oracle] = true
	}

	return nil
}

// PostedPriceResponses is a slice of PostedPriceResponse
type PostedPriceResponses []PostedPriceResponse

// SortDecs provides the interface needed to sort sdk.Dec slices
type SortDecs []sdk.Dec

func (a SortDecs) Len() int           { return len(a) }
func (a SortDecs) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a SortDecs) Less(i, j int) bool { return a[i].LT(a[j]) }
