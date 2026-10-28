package types

import (
	sdkioerrors "cosmossdk.io/errors"
)

// ErrUpdateParamsDisabled rejects MsgUpdateParams. The x-consensus contract is
// the only writer of consensus params, so a gov proposal or any other sender
// must execute update_params on the contract instead.
//
// MsgUpdateParams stays registered, rather than removed, so old txs and gov
// proposals that carry it still decode in queries, export and replay. It is a
// registered error, not a plain one, so the reason reaches the tx result
// instead of being redacted to "internal error". Code 1 is reserved by the
// errors package for internal errors, so the first code is 2.
var ErrUpdateParamsDisabled = sdkioerrors.Register(ModuleName, 2,
	"x/consensus MsgUpdateParams is disabled: execute update_params on the "+
		"x-consensus contract")
