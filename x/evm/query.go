// Copyright (c) 2023-2024 Nibi, Inc.
package evm

import (
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/NibiruChain/nibiru/eth"
	"github.com/NibiruChain/nibiru/x/common"
)

func (m QueryTraceTxRequest) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
	for _, msg := range m.Predecessors {
		if err := msg.UnpackInterfaces(unpacker); err != nil {
			return err
		}
	}
	return m.Msg.UnpackInterfaces(unpacker)
}

func (m QueryTraceBlockRequest) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
	for _, msg := range m.Txs {
		if err := msg.UnpackInterfaces(unpacker); err != nil {
			return err
		}
	}
	return nil
}

func (req *QueryEthAccountRequest) Validate() error {
	if req == nil {
		return common.ErrNilGrpcMsg
	}
	if err := eth.ValidateAddress(req.Address); err != nil {
		return status.Error(
			codes.InvalidArgument, err.Error(),
		)
	}
	return nil
}

func (req *QueryNibiruAccountRequest) Validate() error {
	if req == nil {
		return common.ErrNilGrpcMsg
	}

	if err := eth.ValidateAddress(req.Address); err != nil {
		return status.Error(
			codes.InvalidArgument, err.Error(),
		)
	}
	return nil
}

func (req *QueryValidatorAccountRequest) Validate() (
	consAddr sdk.ConsAddress, err error,
) {
	if req == nil {
		return consAddr, status.Error(codes.InvalidArgument, "empty request")
	}

	consAddr, err = sdk.ConsAddressFromBech32(req.ConsAddress)
	if err != nil {
		return consAddr, status.Error(
			codes.InvalidArgument, err.Error(),
		)
	}
	return consAddr, nil
}

func (req *QueryBalanceRequest) Validate() error {
	if req == nil {
		return common.ErrNilGrpcMsg
	}

	if err := eth.ValidateAddress(req.Address); err != nil {
		return status.Error(
			codes.InvalidArgument,
			ErrZeroAddress.Error(),
		)
	}
	return nil
}

func (req *QueryStorageRequest) Validate() error {
	if req == nil {
		return common.ErrNilGrpcMsg
	}
	if err := eth.ValidateAddress(req.Address); err != nil {
		return status.Error(
			codes.InvalidArgument,
			ErrZeroAddress.Error(),
		)
	}
	return nil
}

func (req *QueryCodeRequest) Validate() error {
	if req == nil {
		return common.ErrNilGrpcMsg
	}

	if err := eth.ValidateAddress(req.Address); err != nil {
		return status.Error(
			codes.InvalidArgument,
			ErrZeroAddress.Error(),
		)
	}
	return nil
}

func (req *EthCallRequest) Validate() error {
	if req == nil {
		return common.ErrNilGrpcMsg
	}
	return nil
}

func (req *QueryTraceTxRequest) Validate() error {
	if req == nil {
		return common.ErrNilGrpcMsg
	}

	if req.TraceConfig != nil && req.TraceConfig.Limit < 0 {
		return status.Errorf(codes.InvalidArgument, "output limit cannot be negative, got %d", req.TraceConfig.Limit)
	}
	return nil
}

func (req *QueryTraceBlockRequest) Validate() error {
	if req == nil {
		return common.ErrNilGrpcMsg
	}

	if req.TraceConfig != nil && req.TraceConfig.Limit < 0 {
		return status.Errorf(codes.InvalidArgument, "output limit cannot be negative, got %d", req.TraceConfig.Limit)
	}
	return nil
}
