// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             (unknown)
// source: nibiru/oracle/v1/query.proto

package oraclev1

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	Query_ExchangeRate_FullMethodName      = "/nibiru.oracle.v1.Query/ExchangeRate"
	Query_ExchangeRateTwap_FullMethodName  = "/nibiru.oracle.v1.Query/ExchangeRateTwap"
	Query_ExchangeRates_FullMethodName     = "/nibiru.oracle.v1.Query/ExchangeRates"
	Query_Actives_FullMethodName           = "/nibiru.oracle.v1.Query/Actives"
	Query_VoteTargets_FullMethodName       = "/nibiru.oracle.v1.Query/VoteTargets"
	Query_FeederDelegation_FullMethodName  = "/nibiru.oracle.v1.Query/FeederDelegation"
	Query_MissCounter_FullMethodName       = "/nibiru.oracle.v1.Query/MissCounter"
	Query_AggregatePrevote_FullMethodName  = "/nibiru.oracle.v1.Query/AggregatePrevote"
	Query_AggregatePrevotes_FullMethodName = "/nibiru.oracle.v1.Query/AggregatePrevotes"
	Query_AggregateVote_FullMethodName     = "/nibiru.oracle.v1.Query/AggregateVote"
	Query_AggregateVotes_FullMethodName    = "/nibiru.oracle.v1.Query/AggregateVotes"
	Query_Params_FullMethodName            = "/nibiru.oracle.v1.Query/Params"
)

// QueryClient is the client API for Query service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
//
// Query defines the gRPC querier service.
type QueryClient interface {
	// ExchangeRate returns exchange rate of a pair along with the block height and
	// block time that the exchange rate was set by the oracle module.
	ExchangeRate(ctx context.Context, in *QueryExchangeRateRequest, opts ...grpc.CallOption) (*QueryExchangeRateResponse, error)
	// ExchangeRateTwap returns twap exchange rate of a pair
	ExchangeRateTwap(ctx context.Context, in *QueryExchangeRateRequest, opts ...grpc.CallOption) (*QueryExchangeRateResponse, error)
	// ExchangeRates returns exchange rates of all pairs
	ExchangeRates(ctx context.Context, in *QueryExchangeRatesRequest, opts ...grpc.CallOption) (*QueryExchangeRatesResponse, error)
	// Actives returns all active pairs
	Actives(ctx context.Context, in *QueryActivesRequest, opts ...grpc.CallOption) (*QueryActivesResponse, error)
	// VoteTargets returns all vote target for pairs
	VoteTargets(ctx context.Context, in *QueryVoteTargetsRequest, opts ...grpc.CallOption) (*QueryVoteTargetsResponse, error)
	// FeederDelegation returns feeder delegation of a validator
	FeederDelegation(ctx context.Context, in *QueryFeederDelegationRequest, opts ...grpc.CallOption) (*QueryFeederDelegationResponse, error)
	// MissCounter returns oracle miss counter of a validator
	MissCounter(ctx context.Context, in *QueryMissCounterRequest, opts ...grpc.CallOption) (*QueryMissCounterResponse, error)
	// AggregatePrevote returns an aggregate prevote of a validator
	AggregatePrevote(ctx context.Context, in *QueryAggregatePrevoteRequest, opts ...grpc.CallOption) (*QueryAggregatePrevoteResponse, error)
	// AggregatePrevotes returns aggregate prevotes of all validators
	AggregatePrevotes(ctx context.Context, in *QueryAggregatePrevotesRequest, opts ...grpc.CallOption) (*QueryAggregatePrevotesResponse, error)
	// AggregateVote returns an aggregate vote of a validator
	AggregateVote(ctx context.Context, in *QueryAggregateVoteRequest, opts ...grpc.CallOption) (*QueryAggregateVoteResponse, error)
	// AggregateVotes returns aggregate votes of all validators
	AggregateVotes(ctx context.Context, in *QueryAggregateVotesRequest, opts ...grpc.CallOption) (*QueryAggregateVotesResponse, error)
	// Params queries all parameters.
	Params(ctx context.Context, in *QueryParamsRequest, opts ...grpc.CallOption) (*QueryParamsResponse, error)
}

type queryClient struct {
	cc grpc.ClientConnInterface
}

func NewQueryClient(cc grpc.ClientConnInterface) QueryClient {
	return &queryClient{cc}
}

func (c *queryClient) ExchangeRate(ctx context.Context, in *QueryExchangeRateRequest, opts ...grpc.CallOption) (*QueryExchangeRateResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(QueryExchangeRateResponse)
	err := c.cc.Invoke(ctx, Query_ExchangeRate_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) ExchangeRateTwap(ctx context.Context, in *QueryExchangeRateRequest, opts ...grpc.CallOption) (*QueryExchangeRateResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(QueryExchangeRateResponse)
	err := c.cc.Invoke(ctx, Query_ExchangeRateTwap_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) ExchangeRates(ctx context.Context, in *QueryExchangeRatesRequest, opts ...grpc.CallOption) (*QueryExchangeRatesResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(QueryExchangeRatesResponse)
	err := c.cc.Invoke(ctx, Query_ExchangeRates_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) Actives(ctx context.Context, in *QueryActivesRequest, opts ...grpc.CallOption) (*QueryActivesResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(QueryActivesResponse)
	err := c.cc.Invoke(ctx, Query_Actives_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) VoteTargets(ctx context.Context, in *QueryVoteTargetsRequest, opts ...grpc.CallOption) (*QueryVoteTargetsResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(QueryVoteTargetsResponse)
	err := c.cc.Invoke(ctx, Query_VoteTargets_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) FeederDelegation(ctx context.Context, in *QueryFeederDelegationRequest, opts ...grpc.CallOption) (*QueryFeederDelegationResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(QueryFeederDelegationResponse)
	err := c.cc.Invoke(ctx, Query_FeederDelegation_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) MissCounter(ctx context.Context, in *QueryMissCounterRequest, opts ...grpc.CallOption) (*QueryMissCounterResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(QueryMissCounterResponse)
	err := c.cc.Invoke(ctx, Query_MissCounter_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) AggregatePrevote(ctx context.Context, in *QueryAggregatePrevoteRequest, opts ...grpc.CallOption) (*QueryAggregatePrevoteResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(QueryAggregatePrevoteResponse)
	err := c.cc.Invoke(ctx, Query_AggregatePrevote_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) AggregatePrevotes(ctx context.Context, in *QueryAggregatePrevotesRequest, opts ...grpc.CallOption) (*QueryAggregatePrevotesResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(QueryAggregatePrevotesResponse)
	err := c.cc.Invoke(ctx, Query_AggregatePrevotes_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) AggregateVote(ctx context.Context, in *QueryAggregateVoteRequest, opts ...grpc.CallOption) (*QueryAggregateVoteResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(QueryAggregateVoteResponse)
	err := c.cc.Invoke(ctx, Query_AggregateVote_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) AggregateVotes(ctx context.Context, in *QueryAggregateVotesRequest, opts ...grpc.CallOption) (*QueryAggregateVotesResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(QueryAggregateVotesResponse)
	err := c.cc.Invoke(ctx, Query_AggregateVotes_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) Params(ctx context.Context, in *QueryParamsRequest, opts ...grpc.CallOption) (*QueryParamsResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(QueryParamsResponse)
	err := c.cc.Invoke(ctx, Query_Params_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// QueryServer is the server API for Query service.
// All implementations must embed UnimplementedQueryServer
// for forward compatibility.
//
// Query defines the gRPC querier service.
type QueryServer interface {
	// ExchangeRate returns exchange rate of a pair along with the block height and
	// block time that the exchange rate was set by the oracle module.
	ExchangeRate(context.Context, *QueryExchangeRateRequest) (*QueryExchangeRateResponse, error)
	// ExchangeRateTwap returns twap exchange rate of a pair
	ExchangeRateTwap(context.Context, *QueryExchangeRateRequest) (*QueryExchangeRateResponse, error)
	// ExchangeRates returns exchange rates of all pairs
	ExchangeRates(context.Context, *QueryExchangeRatesRequest) (*QueryExchangeRatesResponse, error)
	// Actives returns all active pairs
	Actives(context.Context, *QueryActivesRequest) (*QueryActivesResponse, error)
	// VoteTargets returns all vote target for pairs
	VoteTargets(context.Context, *QueryVoteTargetsRequest) (*QueryVoteTargetsResponse, error)
	// FeederDelegation returns feeder delegation of a validator
	FeederDelegation(context.Context, *QueryFeederDelegationRequest) (*QueryFeederDelegationResponse, error)
	// MissCounter returns oracle miss counter of a validator
	MissCounter(context.Context, *QueryMissCounterRequest) (*QueryMissCounterResponse, error)
	// AggregatePrevote returns an aggregate prevote of a validator
	AggregatePrevote(context.Context, *QueryAggregatePrevoteRequest) (*QueryAggregatePrevoteResponse, error)
	// AggregatePrevotes returns aggregate prevotes of all validators
	AggregatePrevotes(context.Context, *QueryAggregatePrevotesRequest) (*QueryAggregatePrevotesResponse, error)
	// AggregateVote returns an aggregate vote of a validator
	AggregateVote(context.Context, *QueryAggregateVoteRequest) (*QueryAggregateVoteResponse, error)
	// AggregateVotes returns aggregate votes of all validators
	AggregateVotes(context.Context, *QueryAggregateVotesRequest) (*QueryAggregateVotesResponse, error)
	// Params queries all parameters.
	Params(context.Context, *QueryParamsRequest) (*QueryParamsResponse, error)
	mustEmbedUnimplementedQueryServer()
}

// UnimplementedQueryServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedQueryServer struct{}

func (UnimplementedQueryServer) ExchangeRate(context.Context, *QueryExchangeRateRequest) (*QueryExchangeRateResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ExchangeRate not implemented")
}
func (UnimplementedQueryServer) ExchangeRateTwap(context.Context, *QueryExchangeRateRequest) (*QueryExchangeRateResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ExchangeRateTwap not implemented")
}
func (UnimplementedQueryServer) ExchangeRates(context.Context, *QueryExchangeRatesRequest) (*QueryExchangeRatesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ExchangeRates not implemented")
}
func (UnimplementedQueryServer) Actives(context.Context, *QueryActivesRequest) (*QueryActivesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Actives not implemented")
}
func (UnimplementedQueryServer) VoteTargets(context.Context, *QueryVoteTargetsRequest) (*QueryVoteTargetsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method VoteTargets not implemented")
}
func (UnimplementedQueryServer) FeederDelegation(context.Context, *QueryFeederDelegationRequest) (*QueryFeederDelegationResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method FeederDelegation not implemented")
}
func (UnimplementedQueryServer) MissCounter(context.Context, *QueryMissCounterRequest) (*QueryMissCounterResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method MissCounter not implemented")
}
func (UnimplementedQueryServer) AggregatePrevote(context.Context, *QueryAggregatePrevoteRequest) (*QueryAggregatePrevoteResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AggregatePrevote not implemented")
}
func (UnimplementedQueryServer) AggregatePrevotes(context.Context, *QueryAggregatePrevotesRequest) (*QueryAggregatePrevotesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AggregatePrevotes not implemented")
}
func (UnimplementedQueryServer) AggregateVote(context.Context, *QueryAggregateVoteRequest) (*QueryAggregateVoteResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AggregateVote not implemented")
}
func (UnimplementedQueryServer) AggregateVotes(context.Context, *QueryAggregateVotesRequest) (*QueryAggregateVotesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AggregateVotes not implemented")
}
func (UnimplementedQueryServer) Params(context.Context, *QueryParamsRequest) (*QueryParamsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Params not implemented")
}
func (UnimplementedQueryServer) mustEmbedUnimplementedQueryServer() {}
func (UnimplementedQueryServer) testEmbeddedByValue()               {}

// UnsafeQueryServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to QueryServer will
// result in compilation errors.
type UnsafeQueryServer interface {
	mustEmbedUnimplementedQueryServer()
}

func RegisterQueryServer(s grpc.ServiceRegistrar, srv QueryServer) {
	// If the following call pancis, it indicates UnimplementedQueryServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&Query_ServiceDesc, srv)
}

func _Query_ExchangeRate_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryExchangeRateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).ExchangeRate(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Query_ExchangeRate_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).ExchangeRate(ctx, req.(*QueryExchangeRateRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_ExchangeRateTwap_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryExchangeRateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).ExchangeRateTwap(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Query_ExchangeRateTwap_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).ExchangeRateTwap(ctx, req.(*QueryExchangeRateRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_ExchangeRates_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryExchangeRatesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).ExchangeRates(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Query_ExchangeRates_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).ExchangeRates(ctx, req.(*QueryExchangeRatesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_Actives_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryActivesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Actives(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Query_Actives_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Actives(ctx, req.(*QueryActivesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_VoteTargets_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryVoteTargetsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).VoteTargets(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Query_VoteTargets_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).VoteTargets(ctx, req.(*QueryVoteTargetsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_FeederDelegation_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryFeederDelegationRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).FeederDelegation(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Query_FeederDelegation_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).FeederDelegation(ctx, req.(*QueryFeederDelegationRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_MissCounter_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryMissCounterRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).MissCounter(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Query_MissCounter_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).MissCounter(ctx, req.(*QueryMissCounterRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_AggregatePrevote_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryAggregatePrevoteRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).AggregatePrevote(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Query_AggregatePrevote_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).AggregatePrevote(ctx, req.(*QueryAggregatePrevoteRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_AggregatePrevotes_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryAggregatePrevotesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).AggregatePrevotes(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Query_AggregatePrevotes_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).AggregatePrevotes(ctx, req.(*QueryAggregatePrevotesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_AggregateVote_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryAggregateVoteRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).AggregateVote(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Query_AggregateVote_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).AggregateVote(ctx, req.(*QueryAggregateVoteRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_AggregateVotes_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryAggregateVotesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).AggregateVotes(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Query_AggregateVotes_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).AggregateVotes(ctx, req.(*QueryAggregateVotesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_Params_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryParamsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Params(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Query_Params_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Params(ctx, req.(*QueryParamsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Query_ServiceDesc is the grpc.ServiceDesc for Query service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Query_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "nibiru.oracle.v1.Query",
	HandlerType: (*QueryServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "ExchangeRate",
			Handler:    _Query_ExchangeRate_Handler,
		},
		{
			MethodName: "ExchangeRateTwap",
			Handler:    _Query_ExchangeRateTwap_Handler,
		},
		{
			MethodName: "ExchangeRates",
			Handler:    _Query_ExchangeRates_Handler,
		},
		{
			MethodName: "Actives",
			Handler:    _Query_Actives_Handler,
		},
		{
			MethodName: "VoteTargets",
			Handler:    _Query_VoteTargets_Handler,
		},
		{
			MethodName: "FeederDelegation",
			Handler:    _Query_FeederDelegation_Handler,
		},
		{
			MethodName: "MissCounter",
			Handler:    _Query_MissCounter_Handler,
		},
		{
			MethodName: "AggregatePrevote",
			Handler:    _Query_AggregatePrevote_Handler,
		},
		{
			MethodName: "AggregatePrevotes",
			Handler:    _Query_AggregatePrevotes_Handler,
		},
		{
			MethodName: "AggregateVote",
			Handler:    _Query_AggregateVote_Handler,
		},
		{
			MethodName: "AggregateVotes",
			Handler:    _Query_AggregateVotes_Handler,
		},
		{
			MethodName: "Params",
			Handler:    _Query_Params_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "nibiru/oracle/v1/query.proto",
}
