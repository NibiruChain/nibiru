package types

import (
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
)

func NewReserveSnapshot(
	pair asset.Pair,
	baseReserve, quoteReserve sdk.Dec,
	pegMultiplier sdk.Dec,
	blockTime time.Time,
) ReserveSnapshot {
	return ReserveSnapshot{
		Pair:          pair,
		BaseReserve:   baseReserve,
		QuoteReserve:  quoteReserve,
		PegMultiplier: pegMultiplier,
		TimestampMs:   blockTime.UnixMilli(),
	}
}

func (s ReserveSnapshot) Validate() error {
	err := s.Pair.Validate()
	if err != nil {
		return err
	}

	if (s.BaseReserve.String() == "<nil>") || (s.QuoteReserve.String() == "<nil>") {
		// prevents panics from usage of 'new(Dec)' or 'sdk.Dec{}'
		return fmt.Errorf("nil dec value in snapshot. snapshot: %v", s.String())
	}

	if s.BaseReserve.IsNegative() {
		return fmt.Errorf("base asset reserve from snapshot cannot be negative: %d", s.BaseReserve)
	}

	if s.QuoteReserve.IsNegative() {
		return fmt.Errorf("quote asset reserve from snapshot cannot be negative: %d", s.QuoteReserve)
	}

	if s.PegMultiplier.IsNegative() {
		return fmt.Errorf("peg multiplier from snapshot cannot be negative: %d", s.PegMultiplier)
	}

	// -62135596800000 in Unix milliseconds is equivalent to "0001-01-01 00:00:00 +0000 UTC".
	// This is the earliest possible value for a ctx.blockTime().UnixMilli()
	const earliestMs int64 = -62135596800000
	if s.TimestampMs < earliestMs {
		snapshotTime := time.Unix(s.TimestampMs/1e3, s.TimestampMs%1e3).UTC()
		earliestTime := time.Unix(earliestMs/1e3, earliestMs%1e3).UTC()
		return fmt.Errorf("snapshot time (%s, milli=%v) should not be before "+
			"earliest possible UTC time (%s, milli=%v): ",
			snapshotTime, s.TimestampMs, earliestTime, earliestMs)
	}

	return nil
}

// GetUpperMarkPriceFluctuationLimit returns the maximum limit price based on the fluctuationLimitRatio
func (s ReserveSnapshot) GetUpperMarkPriceFluctuationLimit(fluctuationLimitRatio sdk.Dec) sdk.Dec {
	return s.getMarkPrice().Mul(sdk.OneDec().Add(fluctuationLimitRatio))
}

// GetLowerMarkPriceFluctuationLimit returns the minimum limit price based on the fluctuationLimitRatio
func (s ReserveSnapshot) GetLowerMarkPriceFluctuationLimit(fluctuationLimitRatio sdk.Dec) sdk.Dec {
	return s.getMarkPrice().Mul(sdk.OneDec().Sub(fluctuationLimitRatio))
}

// getMarkPrice returns the price of the mark price at the moment of the snapshot.
// It is the equivalent of getMarkPrice from Market
func (s ReserveSnapshot) getMarkPrice() sdk.Dec {
	if s.BaseReserve.IsNil() || s.BaseReserve.IsZero() ||
		s.QuoteReserve.IsNil() || s.QuoteReserve.IsZero() {
		return sdk.ZeroDec()
	}

	return s.QuoteReserve.Quo(s.BaseReserve).Mul(s.PegMultiplier)
}
