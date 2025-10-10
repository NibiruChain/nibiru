package evmstate

import (
	bankkeeper "github.com/NibiruChain/nibiru/v2/x/bank/keeper"
)

var (
	_ bankkeeper.NibiruExtKeeper = (*NibiruBankKeeper)(nil)
	_ bankkeeper.Keeper          = (*NibiruBankKeeper)(nil)
)

type NibiruBankKeeper struct {
	bankkeeper.BaseKeeper
}
