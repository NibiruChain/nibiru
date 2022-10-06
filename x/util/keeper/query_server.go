package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/util/types"
)

type queryServer struct {
	k utiltypes.BankKeeper
}

func NewQueryServer(k utiltypes.BankKeeper) utiltypes.QueryServer {
	return &queryServer{k: k}
}

func (q queryServer) ModuleAccounts(
	ctx context.Context,
	_ *utiltypes.QueryModuleAccountsRequest,
) (*utiltypes.QueryModuleAccountsResponse, error) {
	sdkContext := sdk.UnwrapSDKContext(ctx)

	var moduleAccountsWithBalances []utiltypes.AccountWithBalance
	for _, acc := range utiltypes.ModuleAccounts {
		balances := q.k.GetAllBalances(sdkContext, acc.Account)
		accWithBalance := utiltypes.AccountWithBalance{
			Name:    acc.Name,
			Address: acc.Account.String(),
			Balance: balances,
		}
		moduleAccountsWithBalances = append(moduleAccountsWithBalances, accWithBalance)
	}

	return &utiltypes.QueryModuleAccountsResponse{}, nil
}
