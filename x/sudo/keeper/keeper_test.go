package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/sudo"
)

func TestSudoKeeper(t *testing.T) {
	suite.Run(t, new(Suite))
}

type Suite struct{ testutil.LogRoutingSuite }

func (s *Suite) TestCheckPermissions() {
	var mockContractAddrStrs []string
	var mockContractAddrs []sdk.AccAddress
	for _, addrStr := range []string{"addraaa", "addrccc"} {
		mockAddr := sdk.AccAddress(addrStr)
		mockContractAddrs = append(mockContractAddrs, mockAddr)
		mockContractAddrStrs = append(mockContractAddrStrs, mockAddr.String())
	}

	nibiru, ctx := testapp.NewNibiruTestAppAndContext()
	nibiru.SudoKeeper.Sudoers.Set(ctx, sudo.Sudoers{
		Root:      "mockroot",
		Contracts: mockContractAddrStrs,
	})

	err := nibiru.SudoKeeper.CheckPermissions(sdk.AccAddress("addrbbb"), ctx)
	s.Require().Error(err)
	for _, mockAddr := range mockContractAddrs {
		err := nibiru.SudoKeeper.CheckPermissions(mockAddr, ctx)
		s.Require().NoError(err)
	}
}
