// Copyright (c) 2023-2024 Nibi, Inc.
package rpcapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"strconv"
	"sync"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	pkgerrors "github.com/pkg/errors"

	"github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/filters"
	"github.com/ethereum/go-ethereum/params"
	gethrpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/cometbft/cometbft/libs/log"
	rpcclient "github.com/cometbft/cometbft/rpc/jsonrpc/client"
	cmttypes "github.com/cometbft/cometbft/types"

	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/v2/app/server/config"
	"github.com/NibiruChain/nibiru/v2/eth/rpc"
	"github.com/NibiruChain/nibiru/v2/eth/rpc/pubsub"
	"github.com/NibiruChain/nibiru/v2/x/evm"
)

type WebsocketsServer interface {
	Start()
}

type SubscriptionResponseJSON struct {
	Jsonrpc string  `json:"jsonrpc"`
	Result  any     `json:"result"`
	ID      float64 `json:"id"`
}

type SubscriptionNotification struct {
	Jsonrpc string              `json:"jsonrpc"`
	Method  string              `json:"method"`
	Params  *SubscriptionResult `json:"params"`
}

type SubscriptionResult struct {
	Subscription gethrpc.ID `json:"subscription"`
	Result       any        `json:"result"`
}

type ErrorResponseJSON struct {
	Jsonrpc string            `json:"jsonrpc"`
	Error   *ErrorMessageJSON `json:"error"`
	ID      *big.Int          `json:"id"`
}

type ErrorMessageJSON struct {
	Code    *big.Int `json:"code"`
	Message string   `json:"message"`
}

type websocketsServer struct {
	rpcAddr  string // listen address of rest-server
	wsAddr   string // listen address of ws server
	certFile string
	keyFile  string
	api      *pubSubAPI
	logger   log.Logger
}

func NewWebsocketsServer(
	clientCtx client.Context,
	logger log.Logger,
	tmWSClient *rpcclient.WSClient,
	cfg *config.Config,
) WebsocketsServer {
	logger = logger.With("api", "websocket-server")
	_, port, _ := net.SplitHostPort(cfg.JSONRPC.Address) // #nosec G703

	return &websocketsServer{
		rpcAddr:  "localhost:" + port, // FIXME: this shouldn't be hardcoded to localhost
		wsAddr:   cfg.JSONRPC.WsAddress,
		certFile: cfg.TLS.CertificatePath,
		keyFile:  cfg.TLS.KeyPath,
		api:      newPubSubAPI(clientCtx, logger, tmWSClient),
		logger:   logger,
	}
}

func (s *websocketsServer) Start() {
	ws := mux.NewRouter()
	ws.Handle("/", s)

	go func() {
		var err error
		if s.certFile == "" || s.keyFile == "" {
			//#nosec G114 -- http functions have no support for timeouts
			err = http.ListenAndServe(s.wsAddr, ws)
		} else {
			//#nosec G114 -- http functions have no support for timeouts
			err = http.ListenAndServeTLS(s.wsAddr, s.certFile, s.keyFile, ws)
		}

		if err != nil {
			if err == http.ErrServerClosed {
				return
			}

			s.logger.Error("failed to start HTTP server for WS", "error", err.Error())
		}
	}()
}

func (s *websocketsServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Debug("websocket upgrade failed", "error", err.Error())
		return
	}

	s.readLoop(&wsConn{
		mux:  new(sync.Mutex),
		conn: conn,
	})
}

func (s *websocketsServer) sendErrResponse(wsConn *wsConn, msg string) {
	res := &ErrorResponseJSON{
		Jsonrpc: "2.0",
		Error: &ErrorMessageJSON{
			Code:    big.NewInt(-32600),
			Message: msg,
		},
		ID: nil,
	}

	_ = wsConn.WriteJSON(res) // #nosec G703
}

type wsConn struct {
	conn *websocket.Conn
	mux  *sync.Mutex
}

func (w *wsConn) WriteJSON(v any) error {
	w.mux.Lock()
	defer w.mux.Unlock()

	return w.conn.WriteJSON(v)
}

func (w *wsConn) Close() error {
	w.mux.Lock()
	defer w.mux.Unlock()

	return w.conn.Close()
}

func (w *wsConn) ReadMessage() (messageType int, p []byte, err error) {
	// not protected by write mutex

	return w.conn.ReadMessage()
}

func (s *websocketsServer) readLoop(wsConn *wsConn) {
	// subscriptions of current connection
	subscriptions := make(map[gethrpc.ID]pubsub.UnsubscribeFunc)
	defer func() {
		// cancel all subscriptions when connection closed
		// #nosec G705
		for _, unsubFn := range subscriptions {
			unsubFn()
		}
	}()

	for {
		_, mb, err := wsConn.ReadMessage()
		if err != nil {
			_ = wsConn.Close() // #nosec G703
			s.logger.Error("read message error, breaking read loop", "error", err.Error())
			return
		}

		if isBatch(mb) {
			if err := s.tcpGetAndSendResponse(wsConn, mb); err != nil {
				s.sendErrResponse(wsConn, err.Error())
			}
			continue
		}

		var msg map[string]any
		if err = json.Unmarshal(mb, &msg); err != nil {
			s.sendErrResponse(wsConn, err.Error())
			continue
		}

		// check if method == eth_subscribe or eth_unsubscribe
		method, ok := msg["method"].(string)
		if !ok {
			// otherwise, call the usual rpc server to respond
			if err := s.tcpGetAndSendResponse(wsConn, mb); err != nil {
				s.sendErrResponse(wsConn, err.Error())
			}

			continue
		}

		var connID float64
		switch id := msg["id"].(type) {
		case string:
			connID, err = strconv.ParseFloat(id, 64)
		case float64:
			connID = id
		default:
			err = fmt.Errorf("unknown type")
		}
		if err != nil {
			s.sendErrResponse(
				wsConn,
				fmt.Errorf("invalid type for connection ID: %T", msg["id"]).Error(),
			)
			continue
		}

		switch method {
		case "eth_subscribe":
			params, ok := s.getParamsAndCheckValid(msg, wsConn)
			if !ok {
				continue
			}

			subID := gethrpc.NewID()
			unsubFn, err := s.api.subscribe(wsConn, subID, params)
			if err != nil {
				s.sendErrResponse(wsConn, err.Error())
				continue
			}
			subscriptions[subID] = unsubFn

			res := &SubscriptionResponseJSON{
				Jsonrpc: "2.0",
				ID:      connID,
				Result:  subID,
			}

			if err := wsConn.WriteJSON(res); err != nil {
				break
			}
		case "eth_unsubscribe":
			params, ok := s.getParamsAndCheckValid(msg, wsConn)
			if !ok {
				continue
			}

			id, ok := params[0].(string)
			if !ok {
				s.sendErrResponse(wsConn, "invalid parameters")
				continue
			}

			subID := gethrpc.ID(id)
			unsubFn, ok := subscriptions[subID]
			if ok {
				delete(subscriptions, subID)
				unsubFn()
			}

			res := &SubscriptionResponseJSON{
				Jsonrpc: "2.0",
				ID:      connID,
				Result:  ok,
			}

			if err := wsConn.WriteJSON(res); err != nil {
				break
			}
		default:
			// otherwise, call the usual rpc server to respond
			if err := s.tcpGetAndSendResponse(wsConn, mb); err != nil {
				s.sendErrResponse(wsConn, err.Error())
			}
		}
	}
}

// tcpGetAndSendResponse sends error response to client if params is invalid
func (s *websocketsServer) getParamsAndCheckValid(msg map[string]any, wsConn *wsConn) ([]any, bool) {
	params, ok := msg["params"].([]any)
	if !ok {
		s.sendErrResponse(wsConn, "invalid parameters")
		return nil, false
	}

	if len(params) == 0 {
		s.sendErrResponse(wsConn, "empty parameters")
		return nil, false
	}

	return params, true
}

// tcpGetAndSendResponse connects to the rest-server over tcp, posts a JSON-RPC request, and sends the response
// to the client over websockets
func (s *websocketsServer) tcpGetAndSendResponse(wsConn *wsConn, mb []byte) error {
	req, err := http.NewRequestWithContext(context.Background(), "POST", "http://"+s.rpcAddr, bytes.NewBuffer(mb))
	if err != nil {
		return pkgerrors.Wrap(err, "Could not build request")
	}

	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return pkgerrors.Wrap(err, "Could not perform request")
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return pkgerrors.Wrap(err, "could not read body from response")
	}

	var wsSend any
	err = json.Unmarshal(body, &wsSend)
	if err != nil {
		return pkgerrors.Wrap(err, "failed to unmarshal rest-server response")
	}

	return wsConn.WriteJSON(wsSend)
}

// pubSubAPI is the eth_ prefixed set of APIs in the Web3 JSON-RPC spec
type pubSubAPI struct {
	events    *EventSubscriber
	logger    log.Logger
	clientCtx client.Context
}

// newPubSubAPI creates an instance of the ethereum PubSub API.
func newPubSubAPI(clientCtx client.Context, logger log.Logger, tmWSClient *rpcclient.WSClient) *pubSubAPI {
	logger = logger.With("module", "websocket-client")
	return &pubSubAPI{
		events:    NewEventSubscriber(logger, tmWSClient),
		logger:    logger,
		clientCtx: clientCtx,
	}
}

func (api *pubSubAPI) subscribe(wsConn *wsConn, subID gethrpc.ID, params []any) (pubsub.UnsubscribeFunc, error) {
	method, ok := params[0].(string)
	if !ok {
		return nil, pkgerrors.New("invalid parameters")
	}

	switch method {
	case "newHeads":
		// TODO: handle extra params
		return api.subscribeNewHeads(wsConn, subID)
	case "logs":
		if len(params) > 1 {
			return api.subscribeLogs(wsConn, subID, params[1])
		}
		return api.subscribeLogs(wsConn, subID, nil)
	case "newPendingTransactions":
		return api.subscribePendingTransactions(wsConn, subID)
	case "syncing":
		return api.subscribeSyncing(wsConn, subID)
	default:
		return nil, pkgerrors.Errorf("unsupported method %s", method)
	}
}

func (api *pubSubAPI) subscribeNewHeads(wsConn *wsConn, subID gethrpc.ID) (pubsub.UnsubscribeFunc, error) {
	sub, unsubFn, err := api.events.SubscribeNewHeads()
	if err != nil {
		return nil, pkgerrors.Wrap(err, "error creating block filter")
	}

	// TODO: use events
	baseFeeWeiPerGas := big.NewInt(params.InitialBaseFee)

	go func() {
		headersCh := sub.EventCh
		errCh := sub.Error()
		for {
			select {
			case event, ok := <-headersCh:
				if !ok {
					return
				}

				data, ok := event.Data.(cmttypes.EventDataNewBlockHeader)
				if !ok {
					api.logger.Debug("event data type mismatch", "type", fmt.Sprintf("%T", event.Data))
					continue
				}

				header := rpc.EthHeaderFromTendermint(data.Header, gethcore.Bloom{}, baseFeeWeiPerGas)

				// write to ws conn
				res := &SubscriptionNotification{
					Jsonrpc: "2.0",
					Method:  "eth_subscription",
					Params: &SubscriptionResult{
						Subscription: subID,
						Result:       header,
					},
				}

				err = wsConn.WriteJSON(res)
				if err != nil {
					api.logger.Error("error writing header, will drop peer", "error", err.Error())

					try(func() {
						if err != websocket.ErrCloseSent {
							_ = wsConn.Close() // #nosec G703
						}
					}, api.logger, "closing websocket peer sub")
				}
			case err, ok := <-errCh:
				if !ok {
					return
				}
				api.logger.Debug("dropping NewHeads WebSocket subscription", "subscription-id", subID, "error", err.Error())
			}
		}
	}()

	return unsubFn, nil
}

func try(fn func(), l log.Logger, desc string) {
	defer func() {
		if x := recover(); x != nil {
			if err, ok := x.(error); ok {
				// debug.PrintStack()
				l.Debug("panic during "+desc, "error", err.Error())
				return
			}

			l.Debug(fmt.Sprintf("panic during %s: %+v", desc, x))
			return
		}
	}()

	fn()
}

func (api *pubSubAPI) subscribeLogs(wsConn *wsConn, subID gethrpc.ID, extra any) (pubsub.UnsubscribeFunc, error) {
	crit := filters.FilterCriteria{}

	if extra != nil {
		params, ok := extra.(map[string]any)
		if !ok {
			err := pkgerrors.New("invalid criteria")
			api.logger.Debug("invalid criteria", "type", fmt.Sprintf("%T", extra))
			return nil, err
		}

		if params["address"] != nil {
			address, isString := params["address"].(string)
			addresses, isSlice := params["address"].([]any)
			if !isString && !isSlice {
				err := pkgerrors.New("invalid addresses; must be address or array of addresses")
				api.logger.Debug("invalid addresses", "type", fmt.Sprintf("%T", params["address"]))
				return nil, err
			}

			if ok {
				crit.Addresses = []common.Address{common.HexToAddress(address)}
			}

			if isSlice {
				crit.Addresses = []common.Address{}
				for _, addr := range addresses {
					address, ok := addr.(string)
					if !ok {
						err := pkgerrors.New("invalid address")
						api.logger.Debug("invalid address", "type", fmt.Sprintf("%T", addr))
						return nil, err
					}

					crit.Addresses = append(crit.Addresses, common.HexToAddress(address))
				}
			}
		}

		if params["topics"] != nil {
			topics, ok := params["topics"].([]any)
			if !ok {
				err := pkgerrors.Errorf("invalid topics: %s", topics)
				api.logger.Error("invalid topics", "type", fmt.Sprintf("%T", topics))
				return nil, err
			}

			crit.Topics = make([][]common.Hash, len(topics))

			addCritTopic := func(topicIdx int, topic any) error {
				tstr, ok := topic.(string)
				if !ok {
					err := pkgerrors.Errorf("invalid topic: %s", topic)
					api.logger.Error("invalid topic", "type", fmt.Sprintf("%T", topic))
					return err
				}

				crit.Topics[topicIdx] = []common.Hash{common.HexToHash(tstr)}
				return nil
			}

			for topicIdx, subtopics := range topics {
				if subtopics == nil {
					continue
				}

				// in case we don't have list, but a single topic value
				if topic, ok := subtopics.(string); ok {
					if err := addCritTopic(topicIdx, topic); err != nil {
						return nil, err
					}

					continue
				}

				// in case we actually have a list of subtopics
				subtopicsList, ok := subtopics.([]any)
				if !ok {
					err := pkgerrors.New("invalid subtopics")
					api.logger.Error("invalid subtopic", "type", fmt.Sprintf("%T", subtopics))
					return nil, err
				}

				subtopicsCollect := make([]common.Hash, len(subtopicsList))
				for idx, subtopic := range subtopicsList {
					tstr, ok := subtopic.(string)
					if !ok {
						err := pkgerrors.Errorf("invalid subtopic: %s", subtopic)
						api.logger.Error("invalid subtopic", "type", fmt.Sprintf("%T", subtopic))
						return nil, err
					}

					subtopicsCollect[idx] = common.HexToHash(tstr)
				}

				crit.Topics[topicIdx] = subtopicsCollect
			}
		}
	}

	sub, unsubFn, err := api.events.SubscribeLogs(crit)
	if err != nil {
		api.logger.Error("failed to subscribe logs", "error", err.Error())
		return nil, err
	}

	go func() {
		ch := sub.EventCh
		errCh := sub.Error()
		for {
			select {
			case event, ok := <-ch:
				if !ok {
					return
				}

				dataTx, ok := event.Data.(cmttypes.EventDataTx)
				if !ok {
					api.logger.Debug("event data type mismatch", "type", fmt.Sprintf("%T", event.Data))
					continue
				}

				txResponse, err := evm.DecodeTxResponse(dataTx.Result.Data)
				if err != nil {
					api.logger.Error("failed to decode tx response", "error", err.Error())
					return
				}

				logs := FilterLogs(evm.LogsToEthereum(txResponse.Logs), crit.FromBlock, crit.ToBlock, crit.Addresses, crit.Topics)
				if len(logs) == 0 {
					continue
				}

				for _, ethLog := range logs {
					res := &SubscriptionNotification{
						Jsonrpc: "2.0",
						Method:  "eth_subscription",
						Params: &SubscriptionResult{
							Subscription: subID,
							Result:       ethLog,
						},
					}

					err = wsConn.WriteJSON(res)
					if err != nil {
						try(func() {
							if err != websocket.ErrCloseSent {
								_ = wsConn.Close() // #nosec G703
							}
						}, api.logger, "closing websocket peer sub")
					}
				}
			case err, ok := <-errCh:
				if !ok {
					return
				}
				api.logger.Debug("dropping Logs WebSocket subscription", "subscription-id", subID, "error", err.Error())
			}
		}
	}()

	return unsubFn, nil
}

func (api *pubSubAPI) subscribePendingTransactions(wsConn *wsConn, subID gethrpc.ID) (pubsub.UnsubscribeFunc, error) {
	sub, unsubFn, err := api.events.SubscribePendingTxs()
	if err != nil {
		return nil, pkgerrors.Wrap(err, "error creating block filter: %s")
	}

	go func() {
		txsCh := sub.EventCh
		errCh := sub.Error()
		for {
			select {
			case ev := <-txsCh:
				data, ok := ev.Data.(cmttypes.EventDataTx)
				if !ok {
					api.logger.Debug("event data type mismatch", "type", fmt.Sprintf("%T", ev.Data))
					continue
				}

				fmt.Printf("data.Tx: %s\n", collections.HumanizeBytes(data.Tx))
				ethTxs, err := rpc.RawTxToEthTx(api.clientCtx, data.Tx)
				if err != nil {
					// not ethereum tx
					continue
				}

				for _, ethTx := range ethTxs {
					// write to ws conn
					res := &SubscriptionNotification{
						Jsonrpc: "2.0",
						Method:  "eth_subscription",
						Params: &SubscriptionResult{
							Subscription: subID,
							Result:       ethTx.Hash,
						},
					}

					err = wsConn.WriteJSON(res)
					if err != nil {
						api.logger.Debug("error writing header, will drop peer", "error", err.Error())

						try(func() {
							if err != websocket.ErrCloseSent {
								_ = wsConn.Close() // #nosec G703
							}
						}, api.logger, "closing websocket peer sub")
					}
				}
			case err, ok := <-errCh:
				if !ok {
					return
				}
				api.logger.Debug("dropping PendingTransactions WebSocket subscription", subID, "error", err.Error())
			}
		}
	}()

	return unsubFn, nil
}

func (api *pubSubAPI) subscribeSyncing(_ *wsConn, _ gethrpc.ID) (pubsub.UnsubscribeFunc, error) {
	return nil, pkgerrors.New("syncing subscription is not implemented")
}

// copy from github.com/ethereum/go-ethereum/rpc/json.go
// isBatch returns true when the first non-whitespace characters is '['
func isBatch(raw []byte) bool {
	for _, c := range raw {
		// skip insignificant whitespace (http://www.ietf.org/rfc/rfc4627.txt)
		if c == 0x20 || c == 0x09 || c == 0x0a || c == 0x0d {
			continue
		}
		return c == '['
	}
	return false
}
