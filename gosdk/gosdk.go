package gosdk

import (
	"context"
	"encoding/hex"

	cmtrpcclient "github.com/cometbft/cometbft/rpc/client"
	cmtcoretypes "github.com/cometbft/cometbft/rpc/core/types"
	"google.golang.org/grpc"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/app/appconst"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	csdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

type NibiruSDK struct {
	ChainId          string
	Keyring          keyring.Keyring
	EncCfg           app.EncodingConfig
	Querier          Querier
	CometRPC         cmtrpcclient.Client
	AccountRetriever authtypes.AccountRetriever
	GrpcClient       *grpc.ClientConn
}

func NewNibiruSdk(
	chainId string,
	grpcConn *grpc.ClientConn,
	rpcEndpt string,
) (NibiruSDK, error) {
	EnsureNibiruPrefix()
	encCfg := app.MakeEncodingConfig()
	keyring := keyring.NewInMemory(encCfg.Codec)
	queryClient, err := NewQuerier(grpcConn)
	if err != nil {
		return NibiruSDK{}, err
	}
	cometRpc, err := NewRPCClient(rpcEndpt, "/websocket")
	if err != nil {
		return NibiruSDK{}, err
	}
	return NibiruSDK{
		ChainId:          chainId,
		Keyring:          keyring,
		EncCfg:           encCfg,
		Querier:          queryClient,
		CometRPC:         cometRpc,
		AccountRetriever: authtypes.AccountRetriever{},
		GrpcClient:       grpcConn,
	}, err
}

func EnsureNibiruPrefix() {
	csdkConfig := csdk.GetConfig()
	nibiruPrefix := appconst.AccountAddressPrefix
	if csdkConfig.GetBech32AccountAddrPrefix() != nibiruPrefix {
		app.SetPrefixes(nibiruPrefix)
	}
}

func (nc *NibiruSDK) TxByHash(txHashHex string) (*cmtcoretypes.ResultTx, error) {
	goCtx := context.Background()
	txHashBz, err := TxHashHexToBytes(txHashHex)
	if err != nil {
		return nil, err
	}
	prove := true
	res, err := nc.CometRPC.Tx(goCtx, txHashBz, prove)
	return res, err
}

func TxHashHexToBytes(txHashHex string) ([]byte, error) {
	return hex.DecodeString(txHashHex)
}

func TxHashBytesToHex(txHashBz []byte) (txHashHex string) {
	return hex.EncodeToString(txHashBz)
}

type AccountNumbers struct {
	Number   uint64
	Sequence uint64
}

func GetAccountNumbers(
	address string,
	grpcConn *grpc.ClientConn,
	encCfg app.EncodingConfig,
) (nums AccountNumbers, err error) {
	queryClient := authtypes.NewQueryClient(grpcConn)
	resp, err := queryClient.Account(context.Background(), &authtypes.QueryAccountRequest{
		Address: address,
	})
	if err != nil {
		return nums, err
	}

	// register auth interface
	var acc authtypes.AccountI
	if err := encCfg.InterfaceRegistry.UnpackAny(resp.Account, &acc); err != nil {
		return nums, err
	}

	return AccountNumbers{
		Number:   acc.GetAccountNumber(),
		Sequence: acc.GetSequence(),
	}, err
}

func (nc *NibiruSDK) GetAccountNumbers(
	address string,
) (nums AccountNumbers, err error) {
	return GetAccountNumbers(address, nc.Querier.ClientConn, nc.EncCfg)
}
