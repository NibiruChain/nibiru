package keeper_test

import (
	"github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/core/02-client/types"
)

func (suite *KeeperTestSuite) TestParams() {
	expParams := types.DefaultParams()

	params := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetParams(suite.chainA.GetContext())
	suite.Require().Equal(expParams, params)

	expParams.AllowedClients = []string{}
	suite.chainA.App.GetIBCKeeper().ClientKeeper.SetParams(suite.chainA.GetContext(), expParams)
	_ = suite.chainA.App.GetIBCKeeper().ClientKeeper.GetParams(suite.chainA.GetContext())
	suite.Require().Empty(expParams.AllowedClients)
}
