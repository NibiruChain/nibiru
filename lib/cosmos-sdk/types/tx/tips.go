package tx

import (
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
)

// TipTx defines the interface to be implemented by Txs that handle Tips.
type TipTx interface {
	sdk.FeeTx
	GetTip() *Tip
}
