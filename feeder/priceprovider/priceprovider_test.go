package priceprovider_test

import (
	mock_priceprovider "github.com/NibiruChain/nibiru/feeder/mocks/priceprovider"
	"github.com/NibiruChain/nibiru/feeder/priceprovider"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestAggregatePriceProvider(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		pp1 := mock_priceprovider.NewMockPriceProvider(gomock.NewController(t))
		pp2 := mock_priceprovider.NewMockPriceProvider(gomock.NewController(t))
		pp1.EXPECT().Close().Times(1)
		pp2.EXPECT().Close().Times(1)

		valid := priceprovider.PriceResponse{
			Symbol:         "btcusd",
			Price:          100_000.1,
			Valid:          true,
			Source:         "valid-mock",
			LastUpdateTime: time.Now(),
		}

		invalid := priceprovider.PriceResponse{
			Symbol: "btcusd",
			Valid:  false,
			Source: "invalid-mock",
		}
		pp1.EXPECT().GetPrice(gomock.Eq("btcusd")).Return(valid).MaxTimes(1)
		pp2.EXPECT().GetPrice(gomock.Eq("btcusd")).Return(invalid).MaxTimes(1)

		epp := priceprovider.NewAggregatePriceProvider([]priceprovider.PriceProvider{pp1, pp2})
		defer epp.Close()

		got := epp.GetPrice("btcusd")
		require.Equal(t, valid, got)

	})
}

func TestExchangeToChainSymbolPriceProvider(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		pp := mock_priceprovider.NewMockPriceProvider(gomock.NewController(t))

		priceProviderPriceResponse := priceprovider.PriceResponse{
			Symbol:         "tBTCUSDT",
			Price:          100_000.1,
			Valid:          true,
			Source:         "mock",
			LastUpdateTime: time.Now(),
		}

		pp.EXPECT().GetPrice(gomock.Eq("tBTCUSDT")).Return(priceProviderPriceResponse)
		pp.EXPECT().Close().Times(1)

		epp := priceprovider.NewExchangeToChainSymbolPriceProvider(pp, map[string]string{"ubtc:unusd": "tBTCUSDT"})
		defer epp.Close()

		exchangeToChainSymbol := epp.GetPrice("ubtc:unusd")

		require.Equal(t, priceProviderPriceResponse.Price, exchangeToChainSymbol.Price)
		require.Equal(t, priceProviderPriceResponse.Valid, exchangeToChainSymbol.Valid)
		require.Equal(t, priceProviderPriceResponse.Source, exchangeToChainSymbol.Source)
		require.Equal(t, priceProviderPriceResponse.LastUpdateTime, exchangeToChainSymbol.LastUpdateTime)
		require.Equal(t, "ubtc:unusd", exchangeToChainSymbol.Symbol)
	})

	t.Run("missing mapping", func(t *testing.T) {
		pp := mock_priceprovider.NewMockPriceProvider(gomock.NewController(t))
		pp.EXPECT().GetPrice(gomock.Any()).Times(0)
		pp.EXPECT().Close().Times(1)
		epp := priceprovider.NewExchangeToChainSymbolPriceProvider(pp, map[string]string{"ubtc:unusd": "tBTCUSDT"})
		defer epp.Close()
		x := epp.GetPrice("not:exist")
		require.False(t, x.Valid)
		require.Zero(t, x.LastUpdateTime)
		require.Zero(t, x.Price)
	})
}

func TestExpiringPriceProvider(t *testing.T) {
	t.Run("instantiation panic", func(t *testing.T) {
		require.Panics(t, func() {
			priceprovider.NewExpiringPriceProvider(nil, -1)
		})
	})

	t.Run("panic invalid price", func(t *testing.T) {
		pp := mock_priceprovider.NewMockPriceProvider(gomock.NewController(t))
		pp.EXPECT().GetPrice(gomock.Any()).Return(priceprovider.PriceResponse{
			Valid:          true,
			LastUpdateTime: time.Time{},
		})

		require.Panics(t, func() { priceprovider.NewExpiringPriceProvider(pp, 0).GetPrice("") })
	})

	t.Run("not expired price", func(t *testing.T) {
		pp := mock_priceprovider.NewMockPriceProvider(gomock.NewController(t))
		price := priceprovider.PriceResponse{
			Symbol:         "tBTCUSDT",
			Price:          100_000.1,
			Valid:          true,
			Source:         "mock",
			LastUpdateTime: time.Now(),
		}
		pp.EXPECT().GetPrice(gomock.Any()).Return(price)

		epp := priceprovider.NewExpiringPriceProvider(pp, 1*time.Hour)
		require.Equal(t, price, epp.GetPrice(price.Symbol))
	})

	t.Run("expired price", func(t *testing.T) {
		pp := mock_priceprovider.NewMockPriceProvider(gomock.NewController(t))
		price := priceprovider.PriceResponse{
			Symbol:         "tBTCUSDT",
			Price:          100_000.1,
			Valid:          true,
			Source:         "mock",
			LastUpdateTime: time.Now().Add(-1 * 30 * time.Second),
		}
		pp.EXPECT().GetPrice(gomock.Any()).Return(price)
		pp.EXPECT().Close().Times(1)

		epp := priceprovider.NewExpiringPriceProvider(pp, 10*time.Second)
		defer epp.Close()
		gotPriceResponse := epp.GetPrice(price.Symbol)
		require.False(t, gotPriceResponse.Valid)
		require.Equal(t, price.Symbol, gotPriceResponse.Symbol)
		require.Equal(t, price.Source, gotPriceResponse.Source)
		require.Equal(t, price.LastUpdateTime, gotPriceResponse.LastUpdateTime)
	})
}
