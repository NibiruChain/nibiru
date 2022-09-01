package priceprovider

import (
	"context"
	"github.com/bitfinexcom/bitfinex-api-go/v2/websocket"
)

func DialBitfinex(symbols []string) (*Bitfinex, error) {
	panic("impl")
}

type Bitfinex struct {
	symbols []string
	stop    chan struct{}
}

func (c *Bitfinex) connect() error {
	ctx := context.Background()
	ws := websocket.NewWithParams(websocket.NewDefaultParameters())
	_, err := ws.SubscribeTicker(ctx, "")
	if err != nil {
		return err
	}

	for {
		select {
		case <-c.stop:

		}
	}
}
