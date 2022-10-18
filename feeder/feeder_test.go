package feeder

import (
	"io"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/golang/mock/gomock"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	mock_feeder "github.com/NibiruChain/nibiru/feeder/mocks/feeder"
	"github.com/NibiruChain/nibiru/feeder/types"
	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/testutil"
)

func valAddr() sdk.ValAddress {
	return sdk.ValAddress(testutil.AccAddress())
}

func TestRun(t *testing.T) {
	ctrl := gomock.NewController(t)

	t.Run("events stream val set timeout", func(t *testing.T) {
		ps := mock_feeder.NewMockPricePoster(ctrl)
		pp := mock_feeder.NewMockPriceProvider(ctrl)

		es := mock_feeder.NewMockEventsStream(ctrl)
		es.EXPECT().ValidatorSetChanged().
			Return(make(chan types.ValidatorSetChanges))

		require.Panics(t, func() {
			Run(es, ps, pp, zerolog.New(io.Discard))
		})
	})

	t.Run("events stream invalid val set - has no in validators", func(t *testing.T) {
		ps := mock_feeder.NewMockPricePoster(ctrl)
		pp := mock_feeder.NewMockPriceProvider(ctrl)

		es := mock_feeder.NewMockEventsStream(ctrl)
		// insert no vals in
		vsc := make(chan types.ValidatorSetChanges, 1)
		vsc <- types.ValidatorSetChanges{}

		es.EXPECT().ValidatorSetChanged().
			Return(vsc)

		require.Panics(t, func() {
			Run(es, ps, pp, zerolog.New(io.Discard))
		})
	})

	t.Run("events stream invalid val set - has no in validators", func(t *testing.T) {
		ps := mock_feeder.NewMockPricePoster(ctrl)
		pp := mock_feeder.NewMockPriceProvider(ctrl)

		es := mock_feeder.NewMockEventsStream(ctrl)
		// insert 1 val in, 1 val out
		vsc := make(chan types.ValidatorSetChanges, 1)
		vsc <- types.ValidatorSetChanges{
			In:  []sdk.ValAddress{valAddr()},
			Out: []sdk.ValAddress{valAddr()},
		}

		es.EXPECT().ValidatorSetChanged().
			Return(vsc)

		require.Panics(t, func() {
			Run(es, ps, pp, zerolog.New(io.Discard))
		})
	})

	t.Run("events stream params timeout", func(t *testing.T) {
		ps := mock_feeder.NewMockPricePoster(ctrl)
		pp := mock_feeder.NewMockPriceProvider(ctrl)

		es := mock_feeder.NewMockEventsStream(ctrl)
		// insert 1 val addr
		vsc := make(chan types.ValidatorSetChanges, 1)
		vsc <- types.ValidatorSetChanges{
			In: []sdk.ValAddress{valAddr()},
		}
		es.EXPECT().ValidatorSetChanged().
			Return(vsc)

		es.EXPECT().ParamsUpdate().
			Return(make(chan types.Params))

		require.Panics(t, func() {
			Run(es, ps, pp, zerolog.New(io.Discard))
		})
	})
}

func TestValidatorSetChanges(t *testing.T) {
	tf := initFeeder(t)
	defer tf.close()
	// update valset
	expected := types.ValidatorSetChanges{
		In: []sdk.ValAddress{valAddr()},
	}
	tf.validatorSetChanges <- expected
	time.Sleep(10 * time.Millisecond)

	require.Len(t, tf.f.validatorSet, len(expected.In)+1)
	require.True(t, tf.f.validatorSet.Has(expected.In[0]))

	// remove val
	tf.validatorSetChanges <- types.ValidatorSetChanges{
		In:  nil,
		Out: expected.In,
	}
	time.Sleep(10 * time.Millisecond)
	require.Len(t, tf.f.validatorSet, 1)
	require.False(t, tf.f.validatorSet.Has(expected.In[0]))
}

func TestParamsUpdate(t *testing.T) {
	tf := initFeeder(t)
	defer tf.close()
	p := types.Params{
		Symbols:          []common.AssetPair{common.Pair_NIBI_NUSD},
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
		Symbol: common.Pair_BTC_NUSD,
		Price:  100_000.8,
		Source: "mock-source",
		Valid:  true,
	}

	invalidPrice := types.Price{
		Symbol: common.Pair_ETH_NUSD,
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
	f                   *Feeder
	mockPriceProvider   *mock_feeder.MockPriceProvider
	mockEventsStream    *mock_feeder.MockEventsStream
	mockPricePoster     *mock_feeder.MockPricePoster
	validatorSetChanges chan types.ValidatorSetChanges
	newVotingPeriod     chan types.VotingPeriod
	paramsUpdate        chan types.Params
	close               func()
}

func initFeeder(t *testing.T) testFeeder {
	ctrl := gomock.NewController(t)
	ps := mock_feeder.NewMockPricePoster(ctrl)
	pp := mock_feeder.NewMockPriceProvider(ctrl)
	es := mock_feeder.NewMockEventsStream(ctrl)
	vsc := make(chan types.ValidatorSetChanges, 1)
	es.EXPECT().ValidatorSetChanged().AnyTimes().Return(vsc)
	params := make(chan types.Params, 1)
	es.EXPECT().ParamsUpdate().AnyTimes().Return(params)
	nvp := make(chan types.VotingPeriod, 1)
	es.EXPECT().VotingPeriodStarted().AnyTimes().Return(nvp)

	params <- types.Params{Symbols: []common.AssetPair{common.Pair_BTC_NUSD, common.Pair_ETH_NUSD}}
	initialValSet := types.ValidatorSetChanges{
		In: []sdk.ValAddress{valAddr()},
	}
	vsc <- initialValSet
	f := Run(es, ps, pp, zerolog.New(io.Discard))
	es.EXPECT().Close()
	pp.EXPECT().Close()
	ps.EXPECT().Close()

	return testFeeder{
		f:                   f,
		mockPriceProvider:   pp,
		mockEventsStream:    es,
		mockPricePoster:     ps,
		validatorSetChanges: vsc,
		newVotingPeriod:     nvp,
		paramsUpdate:        params,
		close: func() {
			f.Close()
		},
	}
}
