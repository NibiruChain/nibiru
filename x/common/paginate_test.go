package common_test

import (
	"testing"

	sdkquery "github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/v2/x/common"
)

type paginateSuite struct {
	suite.Suite
}

func TestPaginateTestSuite(t *testing.T) {
	suite.Run(t, new(paginateSuite))
}

func (s *paginateSuite) TestParsePagination() {
	for _, tc := range []struct {
		name       string
		pageReq    *sdkquery.PageRequest
		wantErr    bool
		wantPage   int
		wantOffset int
	}{
		{
			name:       "blank (default): no key, no offset",
			pageReq:    &sdkquery.PageRequest{},
			wantErr:    false,
			wantPage:   1,
			wantOffset: 0,
		},

		{
			name:       "nil (default)",
			pageReq:    nil,
			wantErr:    false,
			wantPage:   1,
			wantOffset: 0,
		},

		{
			name: "custom: has key, no offset",
			pageReq: &sdkquery.PageRequest{
				Key:   []byte("haskey"),
				Limit: 25,
			},
			wantErr:    false,
			wantPage:   -1,
			wantOffset: 0,
		},

		{
			name: "custom: no key, has offset",
			pageReq: &sdkquery.PageRequest{
				Offset: 256,
				Limit:  12,
			},
			wantErr:    false,
			wantPage:   22, // floor(256 / 12) + 1 =  22
			wantOffset: 256,
		},

		{
			name: "custom: has key, has offset",
			pageReq: &sdkquery.PageRequest{
				Key:    []byte("haskey-and-offset"),
				Offset: 99,
			},
			wantErr:    true,
			wantPage:   -1,
			wantOffset: 99,
		},
	} {
		s.T().Run(tc.name, func(t *testing.T) {
			gotPageReq, gotPage, gotErr := common.ParsePagination(tc.pageReq)

			s.EqualValues(tc.wantPage, gotPage)

			if tc.wantErr {
				s.Error(gotErr)
				return
			}
			s.NoError(gotErr)

			s.EqualValues(tc.wantOffset, int(gotPageReq.Offset))

			// ----------------------------
			// Checks on fields of tc.pageReq
			// ----------------------------
			if tc.pageReq == nil {
				return
			}

			var wantLimit uint64
			if tc.pageReq.Limit > 0 && tc.pageReq.Limit < 50 {
				wantLimit = tc.pageReq.Limit
			} else {
				wantLimit = common.DefaultPageItemsLimit
			}
			s.EqualValues(wantLimit, gotPageReq.Limit)
		})
	}
}
