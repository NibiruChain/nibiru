package signing

import (
	cryptotypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/crypto/types"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/tx"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/tx/signing"
)

// SigVerifiableTx defines a transaction interface for all signature verification
// handlers.
type SigVerifiableTx interface {
	sdk.Tx
	GetSigners() []sdk.AccAddress
	GetPubKeys() ([]cryptotypes.PubKey, error) // If signer already has pubkey in context, this list will have nil in its place
	GetSignaturesV2() ([]signing.SignatureV2, error)
}

// Tx defines a transaction interface that supports all standard message, signature
// fee, memo, tips, and auxiliary interfaces.
type Tx interface {
	SigVerifiableTx

	sdk.TxWithMemo
	sdk.FeeTx
	tx.TipTx
	sdk.TxWithTimeoutHeight
}
