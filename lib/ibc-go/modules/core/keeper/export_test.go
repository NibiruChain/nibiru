package keeper

import sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"

// ConvertToErrorEvents is a wrapper around convertToErrorEvents
// to allow the function to be directly called in tests.
func ConvertToErrorEvents(events sdk.Events) sdk.Events {
	return convertToErrorEvents(events)
}
