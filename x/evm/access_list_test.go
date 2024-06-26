// Copyright (c) 2023-2024 Nibi, Inc.
package evm_test

import (
	"github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"

	"github.com/NibiruChain/nibiru/x/evm"
)

func (suite *TxDataTestSuite) TestTestNewAccessList() {
	testCases := []struct {
		name          string
		ethAccessList *gethcore.AccessList
		expAl         evm.AccessList
	}{
		{
			"ethAccessList is nil",
			nil,
			nil,
		},
		{
			"non-empty ethAccessList",
			&gethcore.AccessList{{Address: suite.addr, StorageKeys: []common.Hash{{0}}}},
			evm.AccessList{{Address: suite.hexAddr, StorageKeys: []string{common.Hash{}.Hex()}}},
		},
	}
	for _, tc := range testCases {
		al := evm.NewAccessList(tc.ethAccessList)

		suite.Require().Equal(tc.expAl, al)
	}
}

func (suite *TxDataTestSuite) TestAccessListToEthAccessList() {
	ethAccessList := gethcore.AccessList{{Address: suite.addr, StorageKeys: []common.Hash{{0}}}}
	al := evm.NewAccessList(&ethAccessList)
	actual := al.ToEthAccessList()

	suite.Require().Equal(&ethAccessList, actual)
}
