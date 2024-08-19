package gosdk

type NetworkInfo struct {
	GrpcEndpoint      string
	LcdEndpoint       string
	TmRpcEndpoint     string
	WebsocketEndpoint string
	ChainID           string
}

var DefaultNetworkInfo = NetworkInfo{
	GrpcEndpoint:      "localhost:9090",
	LcdEndpoint:       "http://localhost:1317",
	TmRpcEndpoint:     "http://localhost:26657",
	WebsocketEndpoint: "ws://localhost:26657/websocket",
	ChainID:           "nibiru-localnet-0",
}
