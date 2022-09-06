package config

import (
	"fmt"
	"github.com/NibiruChain/nibiru/feeder"
	"github.com/NibiruChain/nibiru/feeder/oracle"
	"github.com/NibiruChain/nibiru/feeder/priceprovider"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v2"
	"os"
)

const (
	BinanceExchangeName  = "binance"
	BitfinexExchangeName = "bitfinex"
)

type CacheType string

const (
	MemCacheName  CacheType = "mem"
	FileCacheName CacheType = "file"
)

const (
	RawConfigEnv = `FEEDER_CONFIG`
)

func Get() *Config {
	// TODO
	s, ok := os.LookupEnv(RawConfigEnv)
	if !ok {
		panic("no conf")
	}

	x := new(Config)
	err := yaml.Unmarshal([]byte(s), x)
	if err != nil {
		panic(err)
	}

	return x
}

// Config defines the raw configurations.
type Config struct {
	// ChainID is the chain's ID.
	ChainID string `yaml:"chain_id"`
	// GRPCEndpoint is the GRPC endpoint of the node.
	GRPCEndpoint string `yaml:"grpc_endpoint"`
	// TendermintWebsocketEndpoint is the tendermint websocket endpoint (ex: wss://rpc.something.io/websocket)
	TendermintWebsocketEndpoint string `yaml:"tendermint_websocket_endpoint"`
	// Validator is the validator address as string.
	Validator string `yaml:"validator"`
	// Feeder is the feeder address as string.
	Feeder string `yaml:"feeder"`
	// Cache is the cache type
	Cache CacheType `yaml:"cache"`
	// PrivateKeyHex is the hex encoding of the private key of the feeder.
	PrivateKeyHex string `yaml:"key_ring"`
	// ChainToExchangeSymbols is a map of exchange names to a map of
	ChainToExchangeSymbols map[string]map[string]string `yaml:"chain_to_exchange_symbols"`
	// Tracing defines the tracing level
	Tracing zerolog.Level `yaml:"tracing"`
}

// DialFeeder generates a new feeder.Feeder instance.
func (r Config) DialFeeder() (*feeder.Feeder, error) {
	if r.ChainID == "" {
		return nil, fmt.Errorf("missing chain ID")
	}
	if r.GRPCEndpoint == "" {
		return nil, fmt.Errorf("no GRPC endpoint")
	}

	if r.TendermintWebsocketEndpoint == "" {
		return nil, fmt.Errorf("no tendermint endpoint")
	}

	valAddr, err := types.ValAddressFromBech32(r.Validator)
	if err != nil {
		return nil, err
	}

	feederAddr, err := types.AccAddressFromBech32(r.Feeder)
	if err != nil {
		return nil, err
	}

	if r.PrivateKeyHex == "" {
		return nil, fmt.Errorf("no private key provided")
	}

	kr := oracle.NewPrivKeyKeyring(r.PrivateKeyHex)
	if _, _, err := kr.Sign("", []byte("test message to ensure all works")); err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}

	if len(r.ChainToExchangeSymbols) == 0 {
		return nil, fmt.Errorf("no chain to exchange symbols provided")
	}

	var cache oracle.PrevotesCache
	switch r.Cache {
	case MemCacheName:
		cache = &oracle.MemPrevoteCache{}
		log.Warn().Msg("using MemPrevoteCache means in case the oracle feeder is closed and reactivated then the last prevote will not be forwarded and will lead to slash")
	default:
		return nil, fmt.Errorf("unknown prevotes cache type: %s", r.Cache)
	}

	// prepare events client
	es, err := oracle.NewStream(r.TendermintWebsocketEndpoint, r.GRPCEndpoint)
	if err != nil {
		return nil, err
	}

	// prepare tx client
	tx, err := oracle.NewTxClient(r.GRPCEndpoint, valAddr, feederAddr, cache, kr, r.ChainID)
	if err != nil {
		return nil, err
	}

	// prepare pricefeeder
	pp, err := PriceProviderFromChainToExchangeSymbols(r.ChainToExchangeSymbols)
	if err != nil {
		return nil, err
	}

	return feeder.Dial(tx, es, pp)
}

func PriceProviderFromChainToExchangeSymbols(symbols map[string]map[string]string) (priceprovider.PriceProvider, error) {
	pps := make([]priceprovider.PriceProvider, 0, len(symbols))
	for exchange, chainToExchangeSymbols := range symbols {
		switch exchange {
		case BinanceExchangeName:
			pp, err := priceprovider.DialBinance()
			if err != nil {
				return nil, fmt.Errorf("unable to dial %s: %w", exchange, err)
			}
			pp = priceprovider.NewExchangeToChainSymbolPriceProvider(pp, chainToExchangeSymbols)
			pps = append(pps, pp)

		case BitfinexExchangeName:
			symbols := make([]string, 0, len(chainToExchangeSymbols))
			for _, s := range chainToExchangeSymbols {
				symbols = append(symbols, s)
			}
			pp, err := priceprovider.DialBitfinex(symbols)
			if err != nil {
				return nil, fmt.Errorf("unable to dial %s: %w", exchange, err)
			}
			pp = priceprovider.NewExchangeToChainSymbolPriceProvider(pp, chainToExchangeSymbols)
			pps = append(pps, pp)
		default:
			return nil, fmt.Errorf("unsupported exchange: %s", exchange)
		}
	}

	return priceprovider.NewAggregatePriceProvider(pps), nil
}
