package common

import (
	"fmt"

	sdkquery "github.com/cosmos/cosmos-sdk/types/query"
)

const DefaultPageItemsLimit uint64 = 50

// ParsePagination: Validates and cleans a PageRequest to make setting values
// less error-prone and use Nibiru-specific defaults.
//  1. This fn is intended to be used with sdkquery.Paginate, which paginates
//     all of the results in a PrefixStore based on the provided PageRequest.
//  2. This fn is panic-safe, so it can be used freely throughout the base app.
//  3. A "page" value of -1 means that a key is given for the prefix store.
//     This means that the PageRequest.Offset will be ignored.
func ParsePagination(
	pageReq *sdkquery.PageRequest,
) (newPageReq *sdkquery.PageRequest, page int, err error) {
	newPageReq = new(sdkquery.PageRequest)
	if pageReq == nil {
		newPageReq = &sdkquery.PageRequest{
			Offset:     0,                     // only offset should be set on nil request
			Limit:      DefaultPageItemsLimit, // using default limit 50 instead of 100.
			CountTotal: false,
			Reverse:    false,
		}
	} else {
		*newPageReq = *pageReq
	}

	// Clean limit
	if newPageReq.Limit <= 0 || newPageReq.Limit > DefaultPageItemsLimit {
		newPageReq.Limit = DefaultPageItemsLimit
	}

	// Clean offset and compute page
	haveKey := newPageReq.Key != nil
	haveOffset := newPageReq.Offset > 0
	switch {
	case haveKey && haveOffset:
		newPageReq = nil
		page = -1
		err = fmt.Errorf("invalid page request, either offset or key is expected, not both.")
	case haveOffset:
		page = (int(newPageReq.Offset) / int(newPageReq.Limit)) + 1
	case haveKey:
		page = -1
	default: // neither key nor offset given
		newPageReq.Offset = 0
		page = (int(newPageReq.Offset) / int(newPageReq.Limit)) + 1
	}
	return newPageReq, page, err
}
