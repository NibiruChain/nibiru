package types

import (
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
)

func NewReserveSnapshot(
	pair common.AssetPair,
	baseAssetReserve, quoteAssetReserve sdk.Dec,
	blockTime time.Time,
) ReserveSnapshot {
	return ReserveSnapshot{
		Pair:              pair,
		BaseAssetReserve:  baseAssetReserve,
		QuoteAssetReserve: quoteAssetReserve,
		TimestampMs:       blockTime.UnixMilli(),
	}
}

func (s ReserveSnapshot) Validate() error {
	err := s.Pair.Validate()
	if err != nil {
		return err
	}

	if s.BaseAssetReserve.IsNegative() {
		return fmt.Errorf("base asset reserve from snapshot cannot be negative: %d", s.BaseAssetReserve)
	}

	if s.QuoteAssetReserve.IsNegative() {
		return fmt.Errorf("quote asset reserve from snapshot cannot be negative: %d", s.QuoteAssetReserve)
	}

	if s.TimestampMs < 0 {
		return fmt.Errorf("timestamp from snapshot cannot be negative: %d", s.TimestampMs)
	}

	return nil
}
