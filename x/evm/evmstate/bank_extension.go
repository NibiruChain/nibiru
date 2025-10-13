package evmstate

import (
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
)

var (
	_ bankkeeper.NibiruExtKeeper = (*NibiruBankKeeper)(nil)
	_ bankkeeper.Keeper          = (*NibiruBankKeeper)(nil)
)

type NibiruBankKeeper struct {
	bankkeeper.BaseKeeper
}
