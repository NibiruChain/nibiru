// Copyright (c) 2023-2024 Nibi, Inc.
package types

import (
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/x/common"
)

type TestSuite struct {
	suite.Suite
}

// TestNilQueries: Checks that all expected sad paths for nil msgs error
func (s *TestSuite) TestNilQueries() {
	for _, testCase := range []func() error{
		func() error {
			var req *QueryEthAccountRequest = nil
			return req.Validate()
		},
		func() error {
			var req *QueryNibiruAccountRequest = nil
			return req.Validate()
		},
		func() error {
			var req *QueryValidatorAccountRequest = nil
			_, err := req.Validate()
			return err
		},
		func() error {
			var req *QueryBalanceRequest = nil
			return req.Validate()
		},
		func() error {
			var req *QueryStorageRequest = nil
			return req.Validate()
		},
		func() error {
			var req *QueryCodeRequest = nil
			return req.Validate()
		},
		func() error {
			var req *EthCallRequest = nil
			return req.Validate()
		},
		func() error {
			var req *QueryTraceTxRequest = nil
			return req.Validate()
		},
		func() error {
			var req *QueryTraceBlockRequest = nil
			return req.Validate()
		},
	} {
		err := testCase()
		s.Require().ErrorContains(err, common.ErrNilGrpcMsg.Error())
	}
}
