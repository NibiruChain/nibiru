package keeper

import (
	"bytes"

	"github.com/MatrixDao/matrix/x/lockup/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// combineKeys combine bytes array into a single bytes.
func combineKeys(keys ...[]byte) []byte {
	return bytes.Join(keys, types.KeyIndexSeparator)
}

// lockStoreKey returns action store key from ID.
func lockStoreKey(lockId uint64) []byte {
	return combineKeys(types.KeyPrefixPeriodLock, sdk.Uint64ToBigEndian(lockId))
}
