package gosdk

// import (
// 	"github.com/NibiruChain/nibiru/v2/app"
// 	sdkclient "github.com/cosmos/cosmos-sdk/client"
// 	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
// )

// TODO: https://github.com/NibiruChain/nibiru/issues/1894
// Make a way to instantiate a NibiruSDK from a (*cli.Network, *cli.Validator)

// ClientCtx: Docs for args
//
//   - tmCfgRootDir: /node0/simd
//   - Validator.Dir: /node0
//   - Validator.ClientCtx.KeyringDir: /node0/simcli

// func NewNibiruSDKFromClientCtx(
// 	clientCtx sdkclient.Context, grpcUrl, cometRpcUrl string,
// ) (gosdk NibiruSDK, err error) {
// 	grpcConn, err := GetGRPCConnection(grpcUrl, true, 5)
// 	if err != nil {
// 		return
// 	}
// 	cometRpc, err := NewRPCClient(cometRpcUrl, "/websocket")
// 	if err != nil {
// 		return
// 	}
// 	querier, err := NewQuerier(grpcConn)
// 	if err != nil {
// 		return
// 	}
// 	return NibiruSDK{
// 		ChainId:          clientCtx.ChainID,
// 		Keyring:          clientCtx.Keyring,
// 		EncCfg:           app.MakeEncodingConfig(),
// 		Querier:          querier,
// 		CometRPC:         cometRpc,
// 		AccountRetriever: authtypes.AccountRetriever{},
// 		GrpcClient:       grpcConn,
// 	}, err
// }
