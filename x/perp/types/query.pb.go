// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: perp/v1/query.proto

package types

import (
	context "context"
	fmt "fmt"
	_ "github.com/gogo/protobuf/gogoproto"
	grpc1 "github.com/gogo/protobuf/grpc"
	proto "github.com/gogo/protobuf/proto"
	_ "google.golang.org/genproto/googleapis/api/annotations"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	io "io"
	math "math"
	math_bits "math/bits"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

// QueryParamsRequest is request type for the Query/Params RPC method.
type QueryParamsRequest struct {
}

func (m *QueryParamsRequest) Reset()         { *m = QueryParamsRequest{} }
func (m *QueryParamsRequest) String() string { return proto.CompactTextString(m) }
func (*QueryParamsRequest) ProtoMessage()    {}
func (*QueryParamsRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_8212d8958be09421, []int{0}
}
func (m *QueryParamsRequest) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *QueryParamsRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_QueryParamsRequest.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *QueryParamsRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_QueryParamsRequest.Merge(m, src)
}
func (m *QueryParamsRequest) XXX_Size() int {
	return m.Size()
}
func (m *QueryParamsRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_QueryParamsRequest.DiscardUnknown(m)
}

var xxx_messageInfo_QueryParamsRequest proto.InternalMessageInfo

// QueryParamsResponse is response type for the Query/Params RPC method.
type QueryParamsResponse struct {
	// params holds all the parameters of this module.
	Params Params `protobuf:"bytes,1,opt,name=params,proto3" json:"params"`
}

func (m *QueryParamsResponse) Reset()         { *m = QueryParamsResponse{} }
func (m *QueryParamsResponse) String() string { return proto.CompactTextString(m) }
func (*QueryParamsResponse) ProtoMessage()    {}
func (*QueryParamsResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_8212d8958be09421, []int{1}
}
func (m *QueryParamsResponse) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *QueryParamsResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_QueryParamsResponse.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *QueryParamsResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_QueryParamsResponse.Merge(m, src)
}
func (m *QueryParamsResponse) XXX_Size() int {
	return m.Size()
}
func (m *QueryParamsResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_QueryParamsResponse.DiscardUnknown(m)
}

var xxx_messageInfo_QueryParamsResponse proto.InternalMessageInfo

func (m *QueryParamsResponse) GetParams() Params {
	if m != nil {
		return m.Params
	}
	return Params{}
}

// QueryTraderPositionRequest is the request type for the position of the
// x/perp module account.
type QueryTraderPositionRequest struct {
	TokenPair string `protobuf:"bytes,1,opt,name=token_pair,json=tokenPair,proto3" json:"token_pair,omitempty"`
	Trader    string `protobuf:"bytes,2,opt,name=trader,proto3" json:"trader,omitempty"`
}

func (m *QueryTraderPositionRequest) Reset()         { *m = QueryTraderPositionRequest{} }
func (m *QueryTraderPositionRequest) String() string { return proto.CompactTextString(m) }
func (*QueryTraderPositionRequest) ProtoMessage()    {}
func (*QueryTraderPositionRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_8212d8958be09421, []int{2}
}
func (m *QueryTraderPositionRequest) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *QueryTraderPositionRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_QueryTraderPositionRequest.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *QueryTraderPositionRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_QueryTraderPositionRequest.Merge(m, src)
}
func (m *QueryTraderPositionRequest) XXX_Size() int {
	return m.Size()
}
func (m *QueryTraderPositionRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_QueryTraderPositionRequest.DiscardUnknown(m)
}

var xxx_messageInfo_QueryTraderPositionRequest proto.InternalMessageInfo

func (m *QueryTraderPositionRequest) GetTokenPair() string {
	if m != nil {
		return m.TokenPair
	}
	return ""
}

func (m *QueryTraderPositionRequest) GetTrader() string {
	if m != nil {
		return m.Trader
	}
	return ""
}

type QueryTraderPositionResponse struct {
	// TODO:
	Position *Position `protobuf:"bytes,1,opt,name=position,proto3" json:"position,omitempty"`
}

func (m *QueryTraderPositionResponse) Reset()         { *m = QueryTraderPositionResponse{} }
func (m *QueryTraderPositionResponse) String() string { return proto.CompactTextString(m) }
func (*QueryTraderPositionResponse) ProtoMessage()    {}
func (*QueryTraderPositionResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_8212d8958be09421, []int{3}
}
func (m *QueryTraderPositionResponse) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *QueryTraderPositionResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_QueryTraderPositionResponse.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *QueryTraderPositionResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_QueryTraderPositionResponse.Merge(m, src)
}
func (m *QueryTraderPositionResponse) XXX_Size() int {
	return m.Size()
}
func (m *QueryTraderPositionResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_QueryTraderPositionResponse.DiscardUnknown(m)
}

var xxx_messageInfo_QueryTraderPositionResponse proto.InternalMessageInfo

func (m *QueryTraderPositionResponse) GetPosition() *Position {
	if m != nil {
		return m.Position
	}
	return nil
}

// QueryTraderMarginRequest is the request type for the margin of the
// x/perp module account.
type QueryTraderMarginRequest struct {
}

func (m *QueryTraderMarginRequest) Reset()         { *m = QueryTraderMarginRequest{} }
func (m *QueryTraderMarginRequest) String() string { return proto.CompactTextString(m) }
func (*QueryTraderMarginRequest) ProtoMessage()    {}
func (*QueryTraderMarginRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_8212d8958be09421, []int{4}
}
func (m *QueryTraderMarginRequest) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *QueryTraderMarginRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_QueryTraderMarginRequest.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *QueryTraderMarginRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_QueryTraderMarginRequest.Merge(m, src)
}
func (m *QueryTraderMarginRequest) XXX_Size() int {
	return m.Size()
}
func (m *QueryTraderMarginRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_QueryTraderMarginRequest.DiscardUnknown(m)
}

var xxx_messageInfo_QueryTraderMarginRequest proto.InternalMessageInfo

type QueryTraderMarginResponse struct {
}

func (m *QueryTraderMarginResponse) Reset()         { *m = QueryTraderMarginResponse{} }
func (m *QueryTraderMarginResponse) String() string { return proto.CompactTextString(m) }
func (*QueryTraderMarginResponse) ProtoMessage()    {}
func (*QueryTraderMarginResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_8212d8958be09421, []int{5}
}
func (m *QueryTraderMarginResponse) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *QueryTraderMarginResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_QueryTraderMarginResponse.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *QueryTraderMarginResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_QueryTraderMarginResponse.Merge(m, src)
}
func (m *QueryTraderMarginResponse) XXX_Size() int {
	return m.Size()
}
func (m *QueryTraderMarginResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_QueryTraderMarginResponse.DiscardUnknown(m)
}

var xxx_messageInfo_QueryTraderMarginResponse proto.InternalMessageInfo

func init() {
	proto.RegisterType((*QueryParamsRequest)(nil), "nibiru.perp.v1.QueryParamsRequest")
	proto.RegisterType((*QueryParamsResponse)(nil), "nibiru.perp.v1.QueryParamsResponse")
	proto.RegisterType((*QueryTraderPositionRequest)(nil), "nibiru.perp.v1.QueryTraderPositionRequest")
	proto.RegisterType((*QueryTraderPositionResponse)(nil), "nibiru.perp.v1.QueryTraderPositionResponse")
	proto.RegisterType((*QueryTraderMarginRequest)(nil), "nibiru.perp.v1.QueryTraderMarginRequest")
	proto.RegisterType((*QueryTraderMarginResponse)(nil), "nibiru.perp.v1.QueryTraderMarginResponse")
}

func init() { proto.RegisterFile("perp/v1/query.proto", fileDescriptor_8212d8958be09421) }

var fileDescriptor_8212d8958be09421 = []byte{
	// 433 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x8c, 0x93, 0x41, 0x6f, 0xd3, 0x30,
	0x1c, 0xc5, 0x93, 0x01, 0x11, 0x33, 0x68, 0x07, 0x77, 0x4c, 0xc1, 0x2d, 0x01, 0x19, 0x0e, 0x63,
	0x48, 0xb1, 0x36, 0xf8, 0x04, 0x83, 0x1b, 0x02, 0x95, 0x8e, 0x13, 0x97, 0xc9, 0x05, 0x2b, 0xb3,
	0xa0, 0xb6, 0x67, 0x3b, 0x13, 0xbd, 0x72, 0xe1, 0x08, 0x52, 0xbf, 0x54, 0x8f, 0x95, 0xb8, 0x70,
	0x42, 0xa8, 0xe5, 0x83, 0xa0, 0xd8, 0x4e, 0xd5, 0xac, 0x51, 0xd5, 0x5b, 0xfb, 0x7f, 0xef, 0xff,
	0xfe, 0x3f, 0x3d, 0x2b, 0xa0, 0xa3, 0x98, 0x56, 0xe4, 0xea, 0x98, 0x5c, 0x96, 0x4c, 0x8f, 0x73,
	0xa5, 0xa5, 0x95, 0x70, 0x4f, 0xf0, 0x21, 0xd7, 0x65, 0x5e, 0x69, 0xf9, 0xd5, 0x31, 0xda, 0x2f,
	0x64, 0x21, 0x9d, 0x44, 0xaa, 0x5f, 0xde, 0x85, 0x7a, 0x85, 0x94, 0xc5, 0x17, 0x46, 0xa8, 0xe2,
	0x84, 0x0a, 0x21, 0x2d, 0xb5, 0x5c, 0x0a, 0x13, 0xd4, 0x65, 0xb0, 0xb1, 0xd4, 0x32, 0x3f, 0xc4,
	0xfb, 0x00, 0xbe, 0xab, 0xee, 0xf4, 0xa9, 0xa6, 0x23, 0x33, 0x60, 0x97, 0x25, 0x33, 0x16, 0xbf,
	0x06, 0x9d, 0xc6, 0xd4, 0x28, 0x29, 0x0c, 0x83, 0x2f, 0x40, 0xa2, 0xdc, 0x24, 0x8d, 0x1f, 0xc5,
	0x87, 0x77, 0x4e, 0x0e, 0xf2, 0x26, 0x56, 0xee, 0xfd, 0xa7, 0x37, 0xa7, 0x7f, 0x1e, 0x46, 0x83,
	0xe0, 0xc5, 0x67, 0x00, 0xb9, 0xb0, 0xf7, 0x9a, 0x7e, 0x62, 0xba, 0x2f, 0x0d, 0xaf, 0xa8, 0xc2,
	0x29, 0xf8, 0x00, 0x00, 0x2b, 0x3f, 0x33, 0x71, 0xae, 0x28, 0xd7, 0x2e, 0x77, 0x77, 0xb0, 0xeb,
	0x26, 0x7d, 0xca, 0x35, 0x3c, 0x00, 0x89, 0x75, 0x7b, 0xe9, 0x8e, 0x93, 0xc2, 0x3f, 0x7c, 0x06,
	0xba, 0xad, 0xa1, 0x4b, 0xd2, 0xdb, 0x2a, 0xcc, 0x02, 0x6b, 0xba, 0xc6, 0x5a, 0xef, 0x2c, 0x9d,
	0x18, 0x81, 0x74, 0x25, 0xf4, 0x0d, 0xd5, 0x05, 0xaf, 0x39, 0x71, 0x17, 0xdc, 0x6f, 0xd1, 0xfc,
	0xb9, 0x93, 0xc9, 0x0d, 0x70, 0xcb, 0xa9, 0x50, 0x80, 0xc4, 0x97, 0x00, 0xf1, 0xf5, 0x83, 0xeb,
	0x3d, 0xa3, 0xc7, 0x1b, 0x3d, 0x3e, 0x1c, 0x77, 0xbf, 0xfd, 0xfa, 0x37, 0xd9, 0xb9, 0x07, 0x3b,
	0xc4, 0x9b, 0x89, 0x7b, 0x47, 0x5f, 0x2e, 0xfc, 0x11, 0x83, 0xbd, 0x66, 0x07, 0xf0, 0xa8, 0x35,
	0xb4, 0xb5, 0x7d, 0xf4, 0x6c, 0x2b, 0x6f, 0x00, 0x79, 0xe2, 0x40, 0x32, 0xd8, 0x6b, 0x80, 0xf8,
	0x07, 0x39, 0xaf, 0x4b, 0x84, 0xdf, 0x63, 0x70, 0x77, 0xb5, 0x24, 0x78, 0xb8, 0xe1, 0x46, 0xa3,
	0x63, 0xf4, 0x74, 0x0b, 0x67, 0x60, 0xc1, 0x8e, 0xa5, 0x07, 0x51, 0x1b, 0xcb, 0xc8, 0x79, 0x4f,
	0x5f, 0x4d, 0xe7, 0x59, 0x3c, 0x9b, 0x67, 0xf1, 0xdf, 0x79, 0x16, 0xff, 0x5c, 0x64, 0xd1, 0x6c,
	0x91, 0x45, 0xbf, 0x17, 0x59, 0xf4, 0xe1, 0xa8, 0xe0, 0xf6, 0xa2, 0x1c, 0xe6, 0x1f, 0xe5, 0x88,
	0xbc, 0x75, 0xfb, 0x2f, 0x2f, 0x28, 0x17, 0x75, 0xd6, 0xd7, 0x90, 0x36, 0x56, 0xcc, 0x0c, 0x13,
	0xf7, 0xa1, 0x3c, 0xff, 0x1f, 0x00, 0x00, 0xff, 0xff, 0x19, 0xc9, 0xb7, 0x72, 0x98, 0x03, 0x00,
	0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// QueryClient is the client API for Query service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type QueryClient interface {
	// Parameters queries the parameters of the x/perp module.
	Params(ctx context.Context, in *QueryParamsRequest, opts ...grpc.CallOption) (*QueryParamsResponse, error)
	TraderPosition(ctx context.Context, in *QueryTraderPositionRequest, opts ...grpc.CallOption) (*QueryTraderPositionResponse, error)
	TraderMargin(ctx context.Context, in *QueryTraderMarginRequest, opts ...grpc.CallOption) (*QueryTraderMarginResponse, error)
}

type queryClient struct {
	cc grpc1.ClientConn
}

func NewQueryClient(cc grpc1.ClientConn) QueryClient {
	return &queryClient{cc}
}

func (c *queryClient) Params(ctx context.Context, in *QueryParamsRequest, opts ...grpc.CallOption) (*QueryParamsResponse, error) {
	out := new(QueryParamsResponse)
	err := c.cc.Invoke(ctx, "/nibiru.perp.v1.Query/Params", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) TraderPosition(ctx context.Context, in *QueryTraderPositionRequest, opts ...grpc.CallOption) (*QueryTraderPositionResponse, error) {
	out := new(QueryTraderPositionResponse)
	err := c.cc.Invoke(ctx, "/nibiru.perp.v1.Query/TraderPosition", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) TraderMargin(ctx context.Context, in *QueryTraderMarginRequest, opts ...grpc.CallOption) (*QueryTraderMarginResponse, error) {
	out := new(QueryTraderMarginResponse)
	err := c.cc.Invoke(ctx, "/nibiru.perp.v1.Query/TraderMargin", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// QueryServer is the server API for Query service.
type QueryServer interface {
	// Parameters queries the parameters of the x/perp module.
	Params(context.Context, *QueryParamsRequest) (*QueryParamsResponse, error)
	TraderPosition(context.Context, *QueryTraderPositionRequest) (*QueryTraderPositionResponse, error)
	TraderMargin(context.Context, *QueryTraderMarginRequest) (*QueryTraderMarginResponse, error)
}

// UnimplementedQueryServer can be embedded to have forward compatible implementations.
type UnimplementedQueryServer struct {
}

func (*UnimplementedQueryServer) Params(ctx context.Context, req *QueryParamsRequest) (*QueryParamsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Params not implemented")
}
func (*UnimplementedQueryServer) TraderPosition(ctx context.Context, req *QueryTraderPositionRequest) (*QueryTraderPositionResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method TraderPosition not implemented")
}
func (*UnimplementedQueryServer) TraderMargin(ctx context.Context, req *QueryTraderMarginRequest) (*QueryTraderMarginResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method TraderMargin not implemented")
}

func RegisterQueryServer(s grpc1.Server, srv QueryServer) {
	s.RegisterService(&_Query_serviceDesc, srv)
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
		FullMethod: "/nibiru.perp.v1.Query/Params",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Params(ctx, req.(*QueryParamsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_TraderPosition_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryTraderPositionRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).TraderPosition(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/nibiru.perp.v1.Query/TraderPosition",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).TraderPosition(ctx, req.(*QueryTraderPositionRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_TraderMargin_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryTraderMarginRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).TraderMargin(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/nibiru.perp.v1.Query/TraderMargin",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).TraderMargin(ctx, req.(*QueryTraderMarginRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _Query_serviceDesc = grpc.ServiceDesc{
	ServiceName: "nibiru.perp.v1.Query",
	HandlerType: (*QueryServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Params",
			Handler:    _Query_Params_Handler,
		},
		{
			MethodName: "TraderPosition",
			Handler:    _Query_TraderPosition_Handler,
		},
		{
			MethodName: "TraderMargin",
			Handler:    _Query_TraderMargin_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "perp/v1/query.proto",
}

func (m *QueryParamsRequest) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *QueryParamsRequest) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *QueryParamsRequest) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	return len(dAtA) - i, nil
}

func (m *QueryParamsResponse) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *QueryParamsResponse) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *QueryParamsResponse) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	{
		size, err := m.Params.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = encodeVarintQuery(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0xa
	return len(dAtA) - i, nil
}

func (m *QueryTraderPositionRequest) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *QueryTraderPositionRequest) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *QueryTraderPositionRequest) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Trader) > 0 {
		i -= len(m.Trader)
		copy(dAtA[i:], m.Trader)
		i = encodeVarintQuery(dAtA, i, uint64(len(m.Trader)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.TokenPair) > 0 {
		i -= len(m.TokenPair)
		copy(dAtA[i:], m.TokenPair)
		i = encodeVarintQuery(dAtA, i, uint64(len(m.TokenPair)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *QueryTraderPositionResponse) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *QueryTraderPositionResponse) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *QueryTraderPositionResponse) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.Position != nil {
		{
			size, err := m.Position.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintQuery(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *QueryTraderMarginRequest) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *QueryTraderMarginRequest) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *QueryTraderMarginRequest) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	return len(dAtA) - i, nil
}

func (m *QueryTraderMarginResponse) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *QueryTraderMarginResponse) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *QueryTraderMarginResponse) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	return len(dAtA) - i, nil
}

func encodeVarintQuery(dAtA []byte, offset int, v uint64) int {
	offset -= sovQuery(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *QueryParamsRequest) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	return n
}

func (m *QueryParamsResponse) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = m.Params.Size()
	n += 1 + l + sovQuery(uint64(l))
	return n
}

func (m *QueryTraderPositionRequest) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.TokenPair)
	if l > 0 {
		n += 1 + l + sovQuery(uint64(l))
	}
	l = len(m.Trader)
	if l > 0 {
		n += 1 + l + sovQuery(uint64(l))
	}
	return n
}

func (m *QueryTraderPositionResponse) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Position != nil {
		l = m.Position.Size()
		n += 1 + l + sovQuery(uint64(l))
	}
	return n
}

func (m *QueryTraderMarginRequest) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	return n
}

func (m *QueryTraderMarginResponse) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	return n
}

func sovQuery(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozQuery(x uint64) (n int) {
	return sovQuery(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *QueryParamsRequest) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowQuery
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: QueryParamsRequest: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: QueryParamsRequest: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		default:
			iNdEx = preIndex
			skippy, err := skipQuery(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthQuery
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *QueryParamsResponse) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowQuery
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: QueryParamsResponse: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: QueryParamsResponse: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Params", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowQuery
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthQuery
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthQuery
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Params.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipQuery(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthQuery
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *QueryTraderPositionRequest) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowQuery
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: QueryTraderPositionRequest: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: QueryTraderPositionRequest: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field TokenPair", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowQuery
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthQuery
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthQuery
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.TokenPair = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Trader", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowQuery
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthQuery
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthQuery
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Trader = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipQuery(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthQuery
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *QueryTraderPositionResponse) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowQuery
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: QueryTraderPositionResponse: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: QueryTraderPositionResponse: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Position", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowQuery
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthQuery
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthQuery
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.Position == nil {
				m.Position = &Position{}
			}
			if err := m.Position.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipQuery(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthQuery
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *QueryTraderMarginRequest) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowQuery
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: QueryTraderMarginRequest: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: QueryTraderMarginRequest: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		default:
			iNdEx = preIndex
			skippy, err := skipQuery(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthQuery
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *QueryTraderMarginResponse) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowQuery
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: QueryTraderMarginResponse: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: QueryTraderMarginResponse: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		default:
			iNdEx = preIndex
			skippy, err := skipQuery(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthQuery
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipQuery(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowQuery
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowQuery
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
		case 1:
			iNdEx += 8
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowQuery
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthQuery
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupQuery
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthQuery
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthQuery        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowQuery          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupQuery = fmt.Errorf("proto: unexpected end of group")
)
