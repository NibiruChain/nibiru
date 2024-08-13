package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/v2/x/inflation/keeper"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testapp"

	inflationtypes "github.com/NibiruChain/nibiru/v2/x/inflation/types"
)

type QueryServerSuite struct {
	suite.Suite

	nibiruApp *app.NibiruApp
	ctx       sdk.Context
}

func (s *QueryServerSuite) SetupSuite() {
	nibiruApp, ctx := testapp.NewNibiruTestAppAndContext()
	s.nibiruApp = nibiruApp
	s.ctx = ctx
}

func TestSuite_QueryServerSuite_RunAll(t *testing.T) {
	suite.Run(t, new(QueryServerSuite))
}

func (s *QueryServerSuite) TestQueryPeriod() {
	nibiruApp, ctx := s.nibiruApp, s.ctx

	resp, err := nibiruApp.InflationKeeper.Period(
		sdk.WrapSDKContext(ctx), &inflationtypes.QueryPeriodRequest{},
	)

	s.NoError(err)
	s.Assert().Equal(uint64(0), resp.Period)

	nibiruApp.InflationKeeper.CurrentPeriod.Next(ctx)

	resp, err = nibiruApp.InflationKeeper.Period(
		sdk.WrapSDKContext(ctx), &inflationtypes.QueryPeriodRequest{},
	)
	s.NoError(err)
	s.Assert().Equal(uint64(1), resp.Period)
}

func (s *QueryServerSuite) TestQuerySkippedEpochs() {
	nibiruApp, ctx := s.nibiruApp, s.ctx
	resp, err := nibiruApp.InflationKeeper.SkippedEpochs(
		sdk.WrapSDKContext(ctx), &inflationtypes.QuerySkippedEpochsRequest{},
	)

	s.Require().NoError(err)
	s.Assert().Equal(uint64(0), resp.SkippedEpochs)

	nibiruApp.InflationKeeper.NumSkippedEpochs.Next(ctx)

	resp, err = nibiruApp.InflationKeeper.SkippedEpochs(
		sdk.WrapSDKContext(ctx), &inflationtypes.QuerySkippedEpochsRequest{},
	)
	s.NoError(err)
	s.Assert().Equal(uint64(1), resp.SkippedEpochs)
}

func (s *QueryServerSuite) TestQueryEpochMintProvision() {
	nibiruApp, ctx := s.nibiruApp, s.ctx
	resp, err := nibiruApp.InflationKeeper.EpochMintProvision(
		sdk.WrapSDKContext(ctx), &inflationtypes.QueryEpochMintProvisionRequest{},
	)
	s.NoError(err)
	s.NotNil(resp)
}

func (s *QueryServerSuite) TestQueryInflationRate() {
	nibiruApp, ctx := s.nibiruApp, s.ctx
	resp, err := nibiruApp.InflationKeeper.InflationRate(
		sdk.WrapSDKContext(ctx), &inflationtypes.QueryInflationRateRequest{},
	)
	s.NoError(err)
	s.NotNil(resp)
}

func (s *QueryServerSuite) TestQueryCirculatingSupply() {
	nibiruApp, ctx := s.nibiruApp, s.ctx
	resp, err := nibiruApp.InflationKeeper.CirculatingSupply(
		sdk.WrapSDKContext(ctx), &inflationtypes.QueryCirculatingSupplyRequest{},
	)
	s.NoError(err)
	s.NotNil(resp)
}

func (s *QueryServerSuite) TestQueryParams() {
	nibiruApp, ctx := s.nibiruApp, s.ctx
	resp, err := nibiruApp.InflationKeeper.Params.Get(ctx)
	s.NoError(err)
	s.NotNil(resp)

	queryServer := keeper.NewQuerier(nibiruApp.InflationKeeper)

	resp2, err := queryServer.Params(sdk.WrapSDKContext(ctx), &inflationtypes.QueryParamsRequest{})
	s.NoError(err)
	s.NotNil(resp2)
}
