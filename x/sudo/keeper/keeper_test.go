package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testapp"
	sudotypes "github.com/NibiruChain/nibiru/v2/x/sudo/types"
)

func TestCheckPermissions(t *testing.T) {
	var mockContractAddrStrs []string
	var mockContractAddrs []sdk.AccAddress
	for _, addrStr := range []string{"addraaa", "addrccc"} {
		mockAddr := sdk.AccAddress(addrStr)
		mockContractAddrs = append(mockContractAddrs, mockAddr)
		mockContractAddrStrs = append(mockContractAddrStrs, mockAddr.String())
	}

	nibiru, ctx := testapp.NewNibiruTestAppAndContext()
	nibiru.SudoKeeper.Sudoers.Set(ctx, sudotypes.Sudoers{
		Root:      "mockroot",
		Contracts: mockContractAddrStrs,
	})

	err := nibiru.SudoKeeper.CheckPermissions(sdk.AccAddress("addrbbb"), ctx)
	require.Error(t, err)
	for _, mockAddr := range mockContractAddrs {
		err := nibiru.SudoKeeper.CheckPermissions(mockAddr, ctx)
		require.NoError(t, err)
	}
}
