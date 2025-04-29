package gosdk

import "github.com/NibiruChain/nibiru/v2/app/appconst"

type NetworkInfo struct {
	// Nibiru EVM EIP-155 chain ID as an integer
	EvmChainID int64 `json:"evmChainID"`
	// Nibiru EVM EIP-155 chain ID
	EvmChainIDHex string `json:"evmChainIDHex"`
	// Nibiru EVM JSON-RPC server endpoint
	EvmRpc string `json:"evmRpc"`
	// Nibiru EVM JSON-RPC server endpoint from an archive node (no pruning)
	EvmRpcArchive string `json:"evmRpcArchive"`
	EvmWebsocket  string `json:"evmWebsocket"`

	CmtChainID        string `json:"cmtChainID"`
	GrpcEndpoint      string `json:"grpcEndpoint"`
	LcdEndpoint       string `json:"lcdEndpoint"`
	TmRpcEndpoint     string `json:"tmRpcEndpoint"`
	WebsocketEndpoint string `json:"websocketEndpoint"`
}

var (
	NETWORK_INFO_DEFAULT = NetworkInfo{
		EvmChainID:        appconst.ETH_CHAIN_ID_LOCALNET_0,
		EvmChainIDHex:     "0x1B0A",
		EvmRpc:            "http://127.0.0.1:8545",
		EvmRpcArchive:     "",
		EvmWebsocket:      "http://127.0.0.1:8546",
		CmtChainID:        "nibiru-localnet-0",
		GrpcEndpoint:      "localhost:9090",
		LcdEndpoint:       "http://localhost:1317",
		TmRpcEndpoint:     "http://localhost:26657",
		WebsocketEndpoint: "ws://localhost:26657/websocket",
	}
	NETWORK_INFO_NIBIRU_MAINNET = NetworkInfo{
		EvmChainID:        appconst.ETH_CHAIN_ID_MAINNET,
		EvmChainIDHex:     "0x1AF4",
		EvmRpc:            "https://evm-rpc.nibiru.fi:443",
		EvmRpcArchive:     "https://evm-rpc.archive.nibiru.fi:443",
		EvmWebsocket:      "wss://evm-rpc-ws.nibiru.fi",
		CmtChainID:        "cataclysm-1",
		GrpcEndpoint:      "grpc.nibiru.fi:443",
		TmRpcEndpoint:     "https://rpc.nibiru.fi:443",
		WebsocketEndpoint: "",
	}
)
