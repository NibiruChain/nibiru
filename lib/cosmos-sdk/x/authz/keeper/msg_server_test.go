package keeper_test

import (
	"time"

	simtestutil "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/testutil/sims"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	authtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/authz"
	banktypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/bank/types"

	"github.com/golang/mock/gomock"
)

func (suite *TestSuite) createAccounts(accs int) []sdk.AccAddress {
	addrs := simtestutil.CreateIncrementalAccounts(2)
	suite.accountKeeper.EXPECT().GetAccount(gomock.Any(), suite.addrs[0]).Return(authtypes.NewBaseAccountWithAddress(suite.addrs[0])).AnyTimes()
	suite.accountKeeper.EXPECT().GetAccount(gomock.Any(), suite.addrs[1]).Return(authtypes.NewBaseAccountWithAddress(suite.addrs[1])).AnyTimes()
	return addrs
}

func (suite *TestSuite) TestGrant() {
	grantee, granter := suite.addrs[0], suite.addrs[1]
	expiration := suite.ctx.BlockTime().Add(time.Hour)
	msg, err := authz.NewMsgGrant(
		granter,
		grantee,
		banktypes.NewSendAuthorization(sdk.NewCoins(sdk.NewInt64Coin("stake", 10)), nil),
		&expiration,
	)
	suite.Require().NoError(err)

	_, err = suite.msgSrvr.Grant(suite.ctx, msg)
	suite.Require().ErrorIs(err, authz.ErrAuthzDisabled)

	authorizations, err := suite.authzKeeper.GetAuthorizations(suite.ctx, grantee, granter)
	suite.Require().NoError(err)
	suite.Require().Empty(authorizations)
}

func (suite *TestSuite) TestRevoke() {
	grantee, granter := suite.addrs[0], suite.addrs[1]
	suite.createSendAuthorization(grantee, granter)

	msg := authz.NewMsgRevoke(granter, grantee, bankSendAuthMsgType)
	_, err := suite.msgSrvr.Revoke(suite.ctx, &msg)
	suite.Require().ErrorIs(err, authz.ErrAuthzDisabled)

	authorization, _ := suite.authzKeeper.GetAuthorization(suite.ctx, grantee, granter, bankSendAuthMsgType)
	suite.Require().NotNil(authorization)
}

func (suite *TestSuite) TestExec() {
	grantee, granter := suite.addrs[0], suite.addrs[1]
	coins := sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(10)))

	msg := &banktypes.MsgSend{
		FromAddress: granter.String(),
		ToAddress:   grantee.String(),
		Amount:      coins,
	}

	req := authz.NewMsgExec(grantee, []sdk.Msg{msg})
	_, err := suite.msgSrvr.Exec(suite.ctx, &req)
	suite.Require().ErrorIs(err, authz.ErrAuthzDisabled)
}
