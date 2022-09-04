package websocket

import (
	"encoding/json"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

var (
	KeepAlive = true
	Timeout   = 30 * time.Second
)

// NewJSON creates a websocket which as soon as it connects sends the initMsg U.
// Then every message received is deserialized into T and handler is called.
func NewJSON[T any, U any](endpoint string, initMsg U, handler func(T), errHandler func(err error)) (done, stop chan struct{}, err error) {
	initMsgBytes, err := json.Marshal(initMsg)
	if err != nil {
		return nil, nil, err
	}

	done, stop, err = NewRawBytes(
		endpoint, initMsgBytes,
		genericJSONHandler(handler),
		errHandler,
	)

	return
}

// genericJSONHandler creates a function which handles JSON websocket messages
// in case there's a failure in JSON decoding, if the error is an unexpected end
// of the JSON input then no error is emitted since it would mean that
// the connection has been interrupted and the message was received partially.
// This means the websocket error handler would quit immediately afterwards.
// Even in reconnection scenarios the msg would still be lost.
func genericJSONHandler[T any](handler func(T)) func([]byte) {
	return func(rawMsg []byte) {
		x := new(T)
		err := json.Unmarshal(rawMsg, x)
		if err != nil {
			if strings.Contains(err.Error(), "unexpected end of JSON input") {
				return
			} else {
				panic(err)
			}
		}
		handler(*x)
	}
}

func NewRawBytes(endpoint string, initMsg []byte, handler func(msg []byte), errHandler func(err error)) (done, stop chan struct{}, err error) {
	return newWs(endpoint, initMsg, handler, errHandler)
}

func newWs(rpc string, initMsg []byte, handler func(msg []byte), errHandler func(err error)) (done chan struct{}, stop chan struct{}, err error) {
	c, _, err := websocket.DefaultDialer.Dial(rpc, nil)
	if err != nil {
		return
	}

	err = c.WriteMessage(websocket.BinaryMessage, initMsg)
	if err != nil {
		return nil, nil, err
	}

	done = make(chan struct{})
	stop = make(chan struct{})
	silent := int32(0) // keeps track if error is from exit or error is from reading

	// read loop
	go func() {
		defer close(done)
		if KeepAlive {
			wsKeepAlive(c, Timeout)
		}
		for {
			_, msg, err := c.ReadMessage()
			if err != nil {
				silent := atomic.CompareAndSwapInt32(&silent, 1, 1)
				if !silent {
					errHandler(err)
				}
				return
			}
			handler(msg)
		}
	}()

	// exit loop
	go func() {
		select {
		case <-stop:
			atomic.SwapInt32(&silent, 1)
		case <-done:
		}
		_ = c.Close()
	}()

	return
}

func wsKeepAlive(c *websocket.Conn, timeout time.Duration) {
	// TODO(mercilex): make optional
	/*
		ticker := time.NewTicker(timeout)

		lastResponse := time.Now()
		c.SetPongHandler(func(msg string) error {
			lastResponse = time.Now()
			return nil
		})

		go func() {
			defer ticker.Stop()
			for {
				deadline := time.Now().Add(10 * time.Second)
				err := c.WriteControl(websocket.PingMessage, []byte{}, deadline)
				if err != nil {
					return
				}
				<-ticker.C
				if time.Since(lastResponse) > timeout {
					_ = c.Close()
					return
				}
			}
		}()
	*/
	return
}
