package gosdk

import (
	"context"

	cmtrpc "github.com/cometbft/cometbft/rpc/client"
	sdkclient "github.com/cosmos/cosmos-sdk/client"
	sdkclienttx "github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdktypestx "github.com/cosmos/cosmos-sdk/types/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"google.golang.org/grpc"

	"github.com/NibiruChain/nibiru/v2/x/common"
	"github.com/NibiruChain/nibiru/v2/x/common/denoms"
)

func BroadcastMsgsWithSeq(
	args BroadcastArgs,
	from sdk.AccAddress,
	seq uint64,
	msgs ...sdk.Msg,
) (*sdk.TxResponse, error) {
	broadcaster := args.Broadcaster

	info, err := args.kring.KeyByAddress(from)
	if err != nil {
		return nil, err
	}

	txBuilder := args.txCfg.NewTxBuilder()
	err = txBuilder.SetMsgs(msgs...)
	if err != nil {
		return nil, err
	}

	bondDenom := denoms.NIBI
	txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(bondDenom, sdk.NewInt(1000))))
	txBuilder.SetGasLimit(uint64(2 * common.TO_MICRO))

	nums, err := args.gosdk.GetAccountNumbers(from.String())
	if err != nil {
		return nil, err
	}

	var accRetriever sdkclient.AccountRetriever = authtypes.AccountRetriever{}
	txFactory := sdkclienttx.Factory{}.
		WithChainID(args.chainID).
		WithKeybase(args.kring).
		WithTxConfig(args.txCfg).
		WithAccountRetriever(accRetriever).
		WithAccountNumber(nums.Number).
		WithSequence(seq)

	overwriteSig := true
	err = sdkclienttx.Sign(txFactory, info.Name, txBuilder, overwriteSig)
	if err != nil {
		return nil, err
	}

	txBytes, err := args.txCfg.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		return nil, err
	}

	return broadcaster.BroadcastTxSync(txBytes)
}

func BroadcastMsgs(
	args BroadcastArgs,
	from sdk.AccAddress,
	msgs ...sdk.Msg,
) (*sdk.TxResponse, error) {
	nums, err := args.gosdk.GetAccountNumbers(from.String())
	if err != nil {
		return nil, err
	}
	return BroadcastMsgsWithSeq(args, from, nums.Sequence, msgs...)
}

type Broadcaster interface {
	BroadcastTxSync(txBytes []byte) (*sdk.TxResponse, error)
}

var (
	_ Broadcaster = (*BroadcasterTmRpc)(nil)
	_ Broadcaster = (*BroadcasterGrpc)(nil)
)

type BroadcasterTmRpc struct {
	RPC cmtrpc.Client
}

func (b BroadcasterTmRpc) BroadcastTxSync(
	txBytes []byte,
) (*sdk.TxResponse, error) {
	respRaw, err := b.RPC.BroadcastTxSync(context.Background(), txBytes)
	if err != nil {
		return nil, err
	}

	return sdk.NewResponseFormatBroadcastTx(respRaw), err
}

type BroadcasterGrpc struct {
	GRPC *grpc.ClientConn
}

func (b BroadcasterGrpc) BroadcastTx(
	txBytes []byte, mode sdktypestx.BroadcastMode,
) (*sdk.TxResponse, error) {
	txClient := sdktypestx.NewServiceClient(b.GRPC)
	respRaw, err := txClient.BroadcastTx(
		context.Background(), &sdktypestx.BroadcastTxRequest{
			TxBytes: txBytes,
			Mode:    mode,
		})
	return respRaw.TxResponse, err
}

func (b BroadcasterGrpc) BroadcastTxSync(
	txBytes []byte,
) (*sdk.TxResponse, error) {
	return b.BroadcastTx(txBytes, sdktypestx.BroadcastMode_BROADCAST_MODE_SYNC)
}

func (b BroadcasterGrpc) BroadcastTxAsync(
	txBytes []byte,
) (*sdk.TxResponse, error) {
	return b.BroadcastTx(txBytes, sdktypestx.BroadcastMode_BROADCAST_MODE_ASYNC)
}

// func GetTxBytes() ([]byte, error) {
// 	return txBytes, err
// }

type BroadcastArgs struct {
	kring keyring.Keyring
	txCfg sdkclient.TxConfig
	gosdk NibiruSDK
	// clientCtx   sdkclient.Context // TODO: implement
	Broadcaster Broadcaster
	rpc         cmtrpc.Client
	chainID     string
}

func initBroadcastArgs(
	nc *NibiruSDK, broadcaster Broadcaster,
) (args BroadcastArgs) {
	txConfig := nc.EncCfg.TxConfig
	return BroadcastArgs{
		kring:       nc.Keyring,
		txCfg:       txConfig,
		gosdk:       *nc,
		Broadcaster: broadcaster,
		rpc:         nc.CometRPC,
		chainID:     nc.ChainId,
	}
}

func (nc *NibiruSDK) BroadcastMsgs(
	from sdk.AccAddress,
	msgs ...sdk.Msg,
) (*sdk.TxResponse, error) {
	broadcaster := BroadcasterTmRpc{RPC: nc.CometRPC}
	args := initBroadcastArgs(nc, broadcaster)
	return BroadcastMsgs(args, from, msgs...)
}

func (nc *NibiruSDK) BroadcastMsgsWithSeq(
	from sdk.AccAddress,
	seq uint64,
	msgs ...sdk.Msg,
) (*sdk.TxResponse, error) {
	broadcaster := BroadcasterTmRpc{RPC: nc.CometRPC}
	args := initBroadcastArgs(nc, broadcaster)
	return BroadcastMsgsWithSeq(args, from, seq, msgs...)
}

func (nc *NibiruSDK) BroadcastMsgsGrpc(
	from sdk.AccAddress,
	msgs ...sdk.Msg,
) (*sdk.TxResponse, error) {
	broadcaster := BroadcasterGrpc{GRPC: nc.Querier.ClientConn}
	args := initBroadcastArgs(nc, broadcaster)
	return BroadcastMsgs(args, from, msgs...)
}

func (nc *NibiruSDK) BroadcastMsgsGrpcWithSeq(
	from sdk.AccAddress,
	seq uint64,
	msgs ...sdk.Msg,
) (*sdk.TxResponse, error) {
	broadcaster := BroadcasterGrpc{GRPC: nc.Querier.ClientConn}
	args := initBroadcastArgs(nc, broadcaster)
	return BroadcastMsgsWithSeq(args, from, seq, msgs...)
}
