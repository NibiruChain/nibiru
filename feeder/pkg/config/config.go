package config

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// HexString is a string alias for developer comprehension.
type HexString = string

func MustGet() Config {
	c, err := Get()
	if err != nil {
		panic(err)
	}
	return c
}

func Get() (Config, error) {
	panic("impl")
}

type Config struct {
	PrivateKey                  HexString
	IsDelegated                 bool
	ValidatorAddress            sdk.AccAddress
	ChainSymbolToExchangeSymbol ChainExchangeSymbolMap
	RPCEndpoint                 string
	GRPCEndpoint                string
}

type ChainExchangeSymbolMap map[string]map[string]string

func (c ChainExchangeSymbolMap) Get(symbol string, exchange string) (string, error) {
	exchangeMap, ok := c[symbol]
	if !ok {
		return "", fmt.Errorf("cannot find exchange '%s' in exchange list", exchange)
	}

	exchangeSymbol, ok := exchangeMap[exchange]
	if !ok {
		return "", fmt.Errorf("cannot find symbol '%s' in exchange '%s'", symbol, exchangeSymbol)
	}

	return exchangeSymbol, nil
}

func (c ChainExchangeSymbolMap) ExchangesList() []string {
	l := make([]string, 0, len(c))
	for exchange := range c {
		l = append(l, exchange)
	}
	return l
}
