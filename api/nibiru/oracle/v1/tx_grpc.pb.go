// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             (unknown)
// source: nibiru/oracle/v1/tx.proto

package oraclev1

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// MsgClient is the client API for Msg service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type MsgClient interface {
	// AggregateExchangeRatePrevote defines a method for submitting
	// aggregate exchange rate prevote
	AggregateExchangeRatePrevote(ctx context.Context, in *MsgAggregateExchangeRatePrevote, opts ...grpc.CallOption) (*MsgAggregateExchangeRatePrevoteResponse, error)
	// AggregateExchangeRateVote defines a method for submitting
	// aggregate exchange rate vote
	AggregateExchangeRateVote(ctx context.Context, in *MsgAggregateExchangeRateVote, opts ...grpc.CallOption) (*MsgAggregateExchangeRateVoteResponse, error)
	// DelegateFeedConsent defines a method for delegating oracle voting rights
	// to another address known as a price feeder.
	// See https://github.com/NibiruChain/pricefeeder.
	DelegateFeedConsent(ctx context.Context, in *MsgDelegateFeedConsent, opts ...grpc.CallOption) (*MsgDelegateFeedConsentResponse, error)
	EditOracleParams(ctx context.Context, in *MsgEditOracleParams, opts ...grpc.CallOption) (*MsgEditOracleParamsResponse, error)
}

type msgClient struct {
	cc grpc.ClientConnInterface
}

func NewMsgClient(cc grpc.ClientConnInterface) MsgClient {
	return &msgClient{cc}
}

func (c *msgClient) AggregateExchangeRatePrevote(ctx context.Context, in *MsgAggregateExchangeRatePrevote, opts ...grpc.CallOption) (*MsgAggregateExchangeRatePrevoteResponse, error) {
	out := new(MsgAggregateExchangeRatePrevoteResponse)
	err := c.cc.Invoke(ctx, "/nibiru.oracle.v1.Msg/AggregateExchangeRatePrevote", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *msgClient) AggregateExchangeRateVote(ctx context.Context, in *MsgAggregateExchangeRateVote, opts ...grpc.CallOption) (*MsgAggregateExchangeRateVoteResponse, error) {
	out := new(MsgAggregateExchangeRateVoteResponse)
	err := c.cc.Invoke(ctx, "/nibiru.oracle.v1.Msg/AggregateExchangeRateVote", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *msgClient) DelegateFeedConsent(ctx context.Context, in *MsgDelegateFeedConsent, opts ...grpc.CallOption) (*MsgDelegateFeedConsentResponse, error) {
	out := new(MsgDelegateFeedConsentResponse)
	err := c.cc.Invoke(ctx, "/nibiru.oracle.v1.Msg/DelegateFeedConsent", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *msgClient) EditOracleParams(ctx context.Context, in *MsgEditOracleParams, opts ...grpc.CallOption) (*MsgEditOracleParamsResponse, error) {
	out := new(MsgEditOracleParamsResponse)
	err := c.cc.Invoke(ctx, "/nibiru.oracle.v1.Msg/EditOracleParams", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// MsgServer is the server API for Msg service.
// All implementations must embed UnimplementedMsgServer
// for forward compatibility
type MsgServer interface {
	// AggregateExchangeRatePrevote defines a method for submitting
	// aggregate exchange rate prevote
	AggregateExchangeRatePrevote(context.Context, *MsgAggregateExchangeRatePrevote) (*MsgAggregateExchangeRatePrevoteResponse, error)
	// AggregateExchangeRateVote defines a method for submitting
	// aggregate exchange rate vote
	AggregateExchangeRateVote(context.Context, *MsgAggregateExchangeRateVote) (*MsgAggregateExchangeRateVoteResponse, error)
	// DelegateFeedConsent defines a method for delegating oracle voting rights
	// to another address known as a price feeder.
	// See https://github.com/NibiruChain/pricefeeder.
	DelegateFeedConsent(context.Context, *MsgDelegateFeedConsent) (*MsgDelegateFeedConsentResponse, error)
	EditOracleParams(context.Context, *MsgEditOracleParams) (*MsgEditOracleParamsResponse, error)
	mustEmbedUnimplementedMsgServer()
}

// UnimplementedMsgServer must be embedded to have forward compatible implementations.
type UnimplementedMsgServer struct {
}

func (UnimplementedMsgServer) AggregateExchangeRatePrevote(context.Context, *MsgAggregateExchangeRatePrevote) (*MsgAggregateExchangeRatePrevoteResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AggregateExchangeRatePrevote not implemented")
}
func (UnimplementedMsgServer) AggregateExchangeRateVote(context.Context, *MsgAggregateExchangeRateVote) (*MsgAggregateExchangeRateVoteResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AggregateExchangeRateVote not implemented")
}
func (UnimplementedMsgServer) DelegateFeedConsent(context.Context, *MsgDelegateFeedConsent) (*MsgDelegateFeedConsentResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DelegateFeedConsent not implemented")
}
func (UnimplementedMsgServer) EditOracleParams(context.Context, *MsgEditOracleParams) (*MsgEditOracleParamsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method EditOracleParams not implemented")
}
func (UnimplementedMsgServer) mustEmbedUnimplementedMsgServer() {}

// UnsafeMsgServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to MsgServer will
// result in compilation errors.
type UnsafeMsgServer interface {
	mustEmbedUnimplementedMsgServer()
}

func RegisterMsgServer(s grpc.ServiceRegistrar, srv MsgServer) {
	s.RegisterService(&Msg_ServiceDesc, srv)
}

func _Msg_AggregateExchangeRatePrevote_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgAggregateExchangeRatePrevote)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).AggregateExchangeRatePrevote(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/nibiru.oracle.v1.Msg/AggregateExchangeRatePrevote",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).AggregateExchangeRatePrevote(ctx, req.(*MsgAggregateExchangeRatePrevote))
	}
	return interceptor(ctx, in, info, handler)
}

func _Msg_AggregateExchangeRateVote_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgAggregateExchangeRateVote)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).AggregateExchangeRateVote(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/nibiru.oracle.v1.Msg/AggregateExchangeRateVote",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).AggregateExchangeRateVote(ctx, req.(*MsgAggregateExchangeRateVote))
	}
	return interceptor(ctx, in, info, handler)
}

func _Msg_DelegateFeedConsent_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgDelegateFeedConsent)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).DelegateFeedConsent(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/nibiru.oracle.v1.Msg/DelegateFeedConsent",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).DelegateFeedConsent(ctx, req.(*MsgDelegateFeedConsent))
	}
	return interceptor(ctx, in, info, handler)
}

func _Msg_EditOracleParams_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgEditOracleParams)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).EditOracleParams(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/nibiru.oracle.v1.Msg/EditOracleParams",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).EditOracleParams(ctx, req.(*MsgEditOracleParams))
	}
	return interceptor(ctx, in, info, handler)
}

// Msg_ServiceDesc is the grpc.ServiceDesc for Msg service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Msg_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "nibiru.oracle.v1.Msg",
	HandlerType: (*MsgServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "AggregateExchangeRatePrevote",
			Handler:    _Msg_AggregateExchangeRatePrevote_Handler,
		},
		{
			MethodName: "AggregateExchangeRateVote",
			Handler:    _Msg_AggregateExchangeRateVote_Handler,
		},
		{
			MethodName: "DelegateFeedConsent",
			Handler:    _Msg_DelegateFeedConsent_Handler,
		},
		{
			MethodName: "EditOracleParams",
			Handler:    _Msg_EditOracleParams_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "nibiru/oracle/v1/tx.proto",
}
