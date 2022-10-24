package feeder

import (
	"io"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	mock_feeder "github.com/NibiruChain/nibiru/feeder/mocks/feeder/types"
	"github.com/NibiruChain/nibiru/feeder/types"
	"github.com/NibiruChain/nibiru/x/common"
)

func TestRun(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Run("events stream params timeout", func(t *testing.T) {
		ps := mock_feeder.NewMockPricePoster(ctrl)
		pp := mock_feeder.NewMockPriceProvider(ctrl)

		es := mock_feeder.NewMockEventsStream(ctrl)

		es.EXPECT().ParamsUpdate().
			Return(make(chan types.Params))

		require.Panics(t, func() {
			Run(es, ps, pp, zerolog.New(io.Discard))
		})
	})
}

func TestParamsUpdate(t *testing.T) {
	tf := initFeeder(t)
	defer tf.close()
	p := types.Params{
		Pairs:            []common.AssetPair{common.Pair_NIBI_NUSD},
		VotePeriodBlocks: 50,
	}

	tf.paramsUpdate <- p
	time.Sleep(10 * time.Millisecond)
	require.Equal(t, tf.f.params, p)
}

func TestVotingPeriod(t *testing.T) {
	tf := initFeeder(t)
	defer tf.close()

	validPrice := types.Price{
		Pair:   common.Pair_BTC_NUSD,
		Price:  100_000.8,
		Source: "mock-source",
		Valid:  true,
	}

	invalidPrice := types.Price{
		Pair:   common.Pair_ETH_NUSD,
		Price:  7000.11,
		Source: "mock-source",
		Valid:  false,
	}

	abstainPrice := invalidPrice
	abstainPrice.Price = 0.0

	tf.mockPriceProvider.EXPECT().GetPrice(common.Pair_BTC_NUSD).Return(validPrice)
	tf.mockPriceProvider.EXPECT().GetPrice(common.Pair_ETH_NUSD).Return(invalidPrice)
	tf.mockPricePoster.EXPECT().SendPrices(gomock.Any(), []types.Price{validPrice, abstainPrice})
	// trigger voting period.
	tf.newVotingPeriod <- types.VotingPeriod{Height: 100}
	time.Sleep(10 * time.Millisecond)
}

type testFeeder struct {
	f                 *Feeder
	mockPriceProvider *mock_feeder.MockPriceProvider
	mockEventsStream  *mock_feeder.MockEventsStream
	mockPricePoster   *mock_feeder.MockPricePoster
	newVotingPeriod   chan types.VotingPeriod
	paramsUpdate      chan types.Params
	close             func()
}

func initFeeder(t *testing.T) testFeeder {
	ctrl := gomock.NewController(t)
	ps := mock_feeder.NewMockPricePoster(ctrl)
	pp := mock_feeder.NewMockPriceProvider(ctrl)
	es := mock_feeder.NewMockEventsStream(ctrl)
	params := make(chan types.Params, 1)
	es.EXPECT().ParamsUpdate().AnyTimes().Return(params)
	nvp := make(chan types.VotingPeriod, 1)
	es.EXPECT().VotingPeriodStarted().AnyTimes().Return(nvp)

	params <- types.Params{Pairs: []common.AssetPair{common.Pair_BTC_NUSD, common.Pair_ETH_NUSD}}
	f := Run(es, ps, pp, zerolog.New(io.Discard))
	es.EXPECT().Close()
	pp.EXPECT().Close()
	ps.EXPECT().Close()

	return testFeeder{
		f:                 f,
		mockPriceProvider: pp,
		mockEventsStream:  es,
		mockPricePoster:   ps,
		newVotingPeriod:   nvp,
		paramsUpdate:      params,
		close: func() {
			f.Close()
		},
	}
}
