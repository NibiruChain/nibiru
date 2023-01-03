package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type OracleKeeper interface {
	GetExchangeRate(ctx sdk.Context, pair string) (sdk.Dec, error)
	SetPrice(ctx sdk.Context, pair string, price sdk.Dec)
}
