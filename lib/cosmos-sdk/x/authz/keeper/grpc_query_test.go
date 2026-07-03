package keeper_test

import (
	gocontext "context"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

func (suite *TestSuite) TestGRPCQueriesReturnEmptyWhenAuthzDisabled() {
	queryClient, addrs := suite.queryClient, suite.addrs
	suite.createSendAuthorization(addrs[0], addrs[1])
	suite.createSendAuthorization(addrs[0], addrs[2])

	grants, err := queryClient.Grants(gocontext.Background(), &authz.QueryGrantsRequest{
		Granter:    addrs[1].String(),
		Grantee:    addrs[0].String(),
		MsgTypeUrl: banktypes.SendAuthorization{}.MsgTypeURL(),
	})
	suite.Require().NoError(err)
	suite.Require().Empty(grants.Grants)

	granterGrants, err := queryClient.GranterGrants(gocontext.Background(), &authz.QueryGranterGrantsRequest{
		Granter: addrs[1].String(),
	})
	suite.Require().NoError(err)
	suite.Require().Empty(granterGrants.Grants)

	granteeGrants, err := queryClient.GranteeGrants(gocontext.Background(), &authz.QueryGranteeGrantsRequest{
		Grantee: addrs[0].String(),
	})
	suite.Require().NoError(err)
	suite.Require().Empty(granteeGrants.Grants)
}

func (suite *TestSuite) TestGRPCQueriesRejectNilRequests() {
	_, err := suite.authzKeeper.Grants(suite.ctx.Context(), nil)
	suite.Require().Error(err)

	_, err = suite.authzKeeper.GranterGrants(suite.ctx.Context(), nil)
	suite.Require().Error(err)

	_, err = suite.authzKeeper.GranteeGrants(suite.ctx.Context(), nil)
	suite.Require().Error(err)
}

func (suite *TestSuite) createSendAuthorization(grantee, granter sdk.AccAddress) authz.Authorization {
	exp := suite.ctx.BlockHeader().Time.Add(time.Hour)
	newCoins := sdk.NewCoins(sdk.NewInt64Coin("steak", 100))
	authorization := &banktypes.SendAuthorization{SpendLimit: newCoins}
	err := suite.authzKeeper.SaveGrant(suite.ctx, grantee, granter, authorization, &exp)
	suite.Require().NoError(err)
	return authorization
}
