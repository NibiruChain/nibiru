package feeder

import (
	"fmt"
	"github.com/NibiruChain/nibiru/feeder/pkg/oracle"
	"github.com/NibiruChain/nibiru/feeder/pkg/priceprovider"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"gopkg.in/yaml.v2"
	"os"
)

func init() {
	// app.SetPrefixes(app.AccountAddressPrefix)
}

const (
	BinanceExchangeName = "binance"
)

const (
	MemCacheName  = "mem"
	FileCacheName = "file"
)

const (
	RawConfigEnv = `FEEDER_CONFIG`
)

type rawConfig struct {
	GRPCEndpoint                string                       `yaml:"grpc_endpoint"`
	TendermintWebsocketEndpoint string                       `yaml:"tendermint_websocket_endpoint"`
	Validator                   string                       `yaml:"validator"`
	Feeder                      string                       `yaml:"feeder"`
	Cache                       string                       `yaml:"cache"`
	PrivateKeyHex               string                       `yaml:"key_ring"`
	ChainToExchangeSymbols      map[string]map[string]string `yaml:"chain_to_exchange_symbols"`
}

func (r rawConfig) toConfig() (*Config, error) {
	if r.GRPCEndpoint == "" {
		return nil, fmt.Errorf("no GRPC endpoint")
	}

	if r.TendermintWebsocketEndpoint == "" {
		return nil, fmt.Errorf("no tendermint endpoint")
	}

	valAddr, err := sdk.ValAddressFromBech32(r.Validator)
	if err != nil {
		return nil, err
	}

	feeder, err := sdk.AccAddressFromBech32(r.Feeder)
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
	default:
		return nil, fmt.Errorf("unknown prevotes cache type: %s", r.Cache)
	}

	return &Config{
		GRPCEndpoint:                r.GRPCEndpoint,
		TendermintWebsocketEndpoint: r.TendermintWebsocketEndpoint,
		Validator:                   valAddr,
		Feeder:                      feeder,
		Cache:                       cache,
		KeyRing:                     kr,
		ChainToExchangeSymbols:      r.ChainToExchangeSymbols,
	}, nil
}

func getRawConfig() (*rawConfig, error) {
	confYaml, ok := os.LookupEnv(RawConfigEnv)
	if !ok {
		return nil, fmt.Errorf("yaml config not found in env variable: %s", RawConfigEnv)
	}

	conf := new(rawConfig)
	err := yaml.Unmarshal([]byte(confYaml), conf)
	if err != nil {
		return nil, err
	}

	return conf, nil
}

func GetConfig() Config {
	raw, err := getRawConfig()
	if err != nil {
		panic(err)
	}
	conf, err := raw.toConfig()
	if err != nil {
		panic(err)
	}

	return *conf
}

type Config struct {
	GRPCEndpoint                string
	TendermintWebsocketEndpoint string
	Validator                   sdk.ValAddress
	Feeder                      sdk.AccAddress
	Cache                       oracle.PrevotesCache
	KeyRing                     keyring.Keyring
	ChainToExchangeSymbols      map[string]map[string]string
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
		default:
			return nil, fmt.Errorf("unsupported exchange: %s", exchange)
		}
	}

	return priceprovider.NewAggregatePriceProvider(pps), nil
}
