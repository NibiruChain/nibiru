package types

import (
	fmt "fmt"
	time "time"
)

func (s ReserveSnapshot) Validate() error {
	err := s.Amm.Pair.Validate()
	if err != nil {
		return err
	}

	if (s.Amm.BaseReserve.String() == "<nil>") || (s.Amm.QuoteReserve.String() == "<nil>") {
		// prevents panics from usage of 'new(Dec)' or 'sdk.Dec{}'
		return fmt.Errorf("nil dec value in snapshot. snapshot: %v", s.String())
	}

	if s.Amm.BaseReserve.IsNegative() {
		return fmt.Errorf("base asset reserve from snapshot cannot be negative: %d", s.Amm.BaseReserve)
	}

	if s.Amm.QuoteReserve.IsNegative() {
		return fmt.Errorf("quote asset reserve from snapshot cannot be negative: %d", s.Amm.QuoteReserve)
	}

	if s.Amm.PriceMultiplier.IsNegative() {
		return fmt.Errorf("peg multiplier from snapshot cannot be negative: %d", s.Amm.PriceMultiplier)
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
