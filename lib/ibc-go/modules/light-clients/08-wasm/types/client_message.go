package types

import (
	sdkioerrors "cosmossdk.io/errors"

	"github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/core/exported"
)

var _ exported.ClientMessage = &ClientMessage{}

// ClientType is a Wasm light client.
func (ClientMessage) ClientType() string {
	return Wasm
}

// ValidateBasic defines a basic validation for the wasm client message.
func (c ClientMessage) ValidateBasic() error {
	if len(c.Data) == 0 {
		return sdkioerrors.Wrap(ErrInvalidData, "data cannot be empty")
	}

	return nil
}
