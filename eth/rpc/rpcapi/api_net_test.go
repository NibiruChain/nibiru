package rpcapi_test

import (
	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil/testnetwork"
)

func (s *NodeSuite) TestNetNamespace() {
	api := s.netAPI
	s.Require().True(api.Listening())
	s.EqualValues(
		appconst.GetEthChainID(testnetwork.LocalnetChainID).String(), api.Version())
	s.Equal(0, api.PeerCount())
}
