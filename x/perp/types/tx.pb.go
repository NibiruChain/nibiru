// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: perp/v1/tx.proto

package types

import (
	context "context"
	fmt "fmt"
	github_com_cosmos_cosmos_sdk_types "github.com/cosmos/cosmos-sdk/types"
	types "github.com/cosmos/cosmos-sdk/types"
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

// MsgRemoveMargin: Msg to remove margin.
type MsgRemoveMargin struct {
	Sender string     `protobuf:"bytes,1,opt,name=sender,proto3" json:"sender,omitempty"`
	Vpool  string     `protobuf:"bytes,2,opt,name=vpool,proto3" json:"vpool,omitempty"`
	Margin types.Coin `protobuf:"bytes,3,opt,name=margin,proto3" json:"margin"`
}

func (m *MsgRemoveMargin) Reset()         { *m = MsgRemoveMargin{} }
func (m *MsgRemoveMargin) String() string { return proto.CompactTextString(m) }
func (*MsgRemoveMargin) ProtoMessage()    {}
func (*MsgRemoveMargin) Descriptor() ([]byte, []int) {
	return fileDescriptor_28f06b306d51dcfb, []int{0}
}
func (m *MsgRemoveMargin) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgRemoveMargin) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgRemoveMargin.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgRemoveMargin) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgRemoveMargin.Merge(m, src)
}
func (m *MsgRemoveMargin) XXX_Size() int {
	return m.Size()
}
func (m *MsgRemoveMargin) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgRemoveMargin.DiscardUnknown(m)
}

var xxx_messageInfo_MsgRemoveMargin proto.InternalMessageInfo

func (m *MsgRemoveMargin) GetSender() string {
	if m != nil {
		return m.Sender
	}
	return ""
}

func (m *MsgRemoveMargin) GetVpool() string {
	if m != nil {
		return m.Vpool
	}
	return ""
}

func (m *MsgRemoveMargin) GetMargin() types.Coin {
	if m != nil {
		return m.Margin
	}
	return types.Coin{}
}

type MsgRemoveMarginResponse struct {
	// MarginOut: tokens transferred back to the trader
	MarginOut      types.Coin                             `protobuf:"bytes,1,opt,name=margin_out,json=marginOut,proto3" json:"margin_out"`
	FundingPayment github_com_cosmos_cosmos_sdk_types.Dec `protobuf:"bytes,2,opt,name=funding_payment,json=fundingPayment,proto3,customtype=github.com/cosmos/cosmos-sdk/types.Dec" json:"funding_payment"`
}

func (m *MsgRemoveMarginResponse) Reset()         { *m = MsgRemoveMarginResponse{} }
func (m *MsgRemoveMarginResponse) String() string { return proto.CompactTextString(m) }
func (*MsgRemoveMarginResponse) ProtoMessage()    {}
func (*MsgRemoveMarginResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_28f06b306d51dcfb, []int{1}
}
func (m *MsgRemoveMarginResponse) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgRemoveMarginResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgRemoveMarginResponse.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgRemoveMarginResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgRemoveMarginResponse.Merge(m, src)
}
func (m *MsgRemoveMarginResponse) XXX_Size() int {
	return m.Size()
}
func (m *MsgRemoveMarginResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgRemoveMarginResponse.DiscardUnknown(m)
}

var xxx_messageInfo_MsgRemoveMarginResponse proto.InternalMessageInfo

func (m *MsgRemoveMarginResponse) GetMarginOut() types.Coin {
	if m != nil {
		return m.MarginOut
	}
	return types.Coin{}
}

// MsgAddMargin: Msg to remove margin.
type MsgAddMargin struct {
	Sender string     `protobuf:"bytes,1,opt,name=sender,proto3" json:"sender,omitempty"`
	Vpool  string     `protobuf:"bytes,2,opt,name=vpool,proto3" json:"vpool,omitempty"`
	Margin types.Coin `protobuf:"bytes,3,opt,name=margin,proto3" json:"margin"`
}

func (m *MsgAddMargin) Reset()         { *m = MsgAddMargin{} }
func (m *MsgAddMargin) String() string { return proto.CompactTextString(m) }
func (*MsgAddMargin) ProtoMessage()    {}
func (*MsgAddMargin) Descriptor() ([]byte, []int) {
	return fileDescriptor_28f06b306d51dcfb, []int{2}
}
func (m *MsgAddMargin) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgAddMargin) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgAddMargin.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgAddMargin) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgAddMargin.Merge(m, src)
}
func (m *MsgAddMargin) XXX_Size() int {
	return m.Size()
}
func (m *MsgAddMargin) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgAddMargin.DiscardUnknown(m)
}

var xxx_messageInfo_MsgAddMargin proto.InternalMessageInfo

func (m *MsgAddMargin) GetSender() string {
	if m != nil {
		return m.Sender
	}
	return ""
}

func (m *MsgAddMargin) GetVpool() string {
	if m != nil {
		return m.Vpool
	}
	return ""
}

func (m *MsgAddMargin) GetMargin() types.Coin {
	if m != nil {
		return m.Margin
	}
	return types.Coin{}
}

type MsgAddMarginResponse struct {
	// MarginOut: tokens transferred back to the trader
	MarginOut types.Coin `protobuf:"bytes,1,opt,name=margin_out,json=marginOut,proto3" json:"margin_out"`
}

func (m *MsgAddMarginResponse) Reset()         { *m = MsgAddMarginResponse{} }
func (m *MsgAddMarginResponse) String() string { return proto.CompactTextString(m) }
func (*MsgAddMarginResponse) ProtoMessage()    {}
func (*MsgAddMarginResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_28f06b306d51dcfb, []int{3}
}
func (m *MsgAddMarginResponse) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgAddMarginResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgAddMarginResponse.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgAddMarginResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgAddMarginResponse.Merge(m, src)
}
func (m *MsgAddMarginResponse) XXX_Size() int {
	return m.Size()
}
func (m *MsgAddMarginResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgAddMarginResponse.DiscardUnknown(m)
}

var xxx_messageInfo_MsgAddMarginResponse proto.InternalMessageInfo

func (m *MsgAddMarginResponse) GetMarginOut() types.Coin {
	if m != nil {
		return m.MarginOut
	}
	return types.Coin{}
}

func init() {
	proto.RegisterType((*MsgRemoveMargin)(nil), "nibiru.perp.v1.MsgRemoveMargin")
	proto.RegisterType((*MsgRemoveMarginResponse)(nil), "nibiru.perp.v1.MsgRemoveMarginResponse")
	proto.RegisterType((*MsgAddMargin)(nil), "nibiru.perp.v1.MsgAddMargin")
	proto.RegisterType((*MsgAddMarginResponse)(nil), "nibiru.perp.v1.MsgAddMarginResponse")
}

func init() { proto.RegisterFile("perp/v1/tx.proto", fileDescriptor_28f06b306d51dcfb) }

var fileDescriptor_28f06b306d51dcfb = []byte{
	// 461 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xbc, 0x53, 0x41, 0x6b, 0x13, 0x41,
	0x14, 0xce, 0xb6, 0x1a, 0xc8, 0x58, 0x5a, 0x19, 0x82, 0x4d, 0x97, 0xb0, 0x29, 0x8b, 0x68, 0x11,
	0x9c, 0x21, 0xf5, 0xe0, 0x4d, 0x30, 0xed, 0x35, 0x2a, 0x7b, 0x50, 0xf0, 0x12, 0x66, 0xb3, 0xe3,
	0x74, 0xb0, 0x3b, 0x6f, 0xd8, 0x99, 0x5d, 0x52, 0xf0, 0xe4, 0x2f, 0x10, 0xfc, 0x27, 0xfe, 0x8a,
	0x1e, 0x0b, 0x5e, 0xc4, 0x43, 0x91, 0xc4, 0xbf, 0xe0, 0x5d, 0x76, 0x66, 0x5b, 0x9b, 0x20, 0xea,
	0x41, 0x7a, 0xda, 0x99, 0xf7, 0xbe, 0xf7, 0x7d, 0x1f, 0x6f, 0xbf, 0x41, 0xb7, 0x35, 0x2f, 0x34,
	0xad, 0x86, 0xd4, 0xce, 0x88, 0x2e, 0xc0, 0x02, 0xde, 0x54, 0x32, 0x95, 0x45, 0x49, 0xea, 0x06,
	0xa9, 0x86, 0x61, 0x5f, 0x00, 0x88, 0x63, 0x4e, 0x99, 0x96, 0x94, 0x29, 0x05, 0x96, 0x59, 0x09,
	0xca, 0x78, 0x74, 0x18, 0x4d, 0xc1, 0xe4, 0x60, 0x68, 0xca, 0x0c, 0xa7, 0xd5, 0x30, 0xe5, 0x96,
	0x0d, 0xe9, 0x14, 0xa4, 0x6a, 0xfa, 0x5d, 0x01, 0x02, 0xdc, 0x91, 0xd6, 0x27, 0x5f, 0x8d, 0x67,
	0x68, 0x6b, 0x6c, 0x44, 0xc2, 0x73, 0xa8, 0xf8, 0x98, 0x15, 0x42, 0x2a, 0x7c, 0x07, 0xb5, 0x0d,
	0x57, 0x19, 0x2f, 0x7a, 0xc1, 0x6e, 0xb0, 0xd7, 0x49, 0x9a, 0x1b, 0xee, 0xa2, 0x9b, 0x95, 0x06,
	0x38, 0xee, 0xad, 0xb9, 0xb2, 0xbf, 0xe0, 0xc7, 0xa8, 0x9d, 0xbb, 0xb9, 0xde, 0xfa, 0x6e, 0xb0,
	0x77, 0x6b, 0x7f, 0x87, 0x78, 0x1f, 0xa4, 0xf6, 0x41, 0x1a, 0x1f, 0xe4, 0x00, 0xa4, 0x1a, 0xdd,
	0x38, 0x3d, 0x1f, 0xb4, 0x92, 0x06, 0x1e, 0x7f, 0x0a, 0xd0, 0xf6, 0x8a, 0x74, 0xc2, 0x8d, 0x06,
	0x65, 0x38, 0x7e, 0x82, 0x90, 0x47, 0x4d, 0xa0, 0xb4, 0xce, 0xc6, 0x3f, 0x10, 0x77, 0xfc, 0xc8,
	0xf3, 0xd2, 0xe2, 0x57, 0x68, 0xeb, 0x4d, 0xa9, 0x32, 0xa9, 0xc4, 0x44, 0xb3, 0x93, 0x9c, 0x2b,
	0xeb, 0x4d, 0x8f, 0x48, 0x8d, 0xfc, 0x7a, 0x3e, 0xb8, 0x27, 0xa4, 0x3d, 0x2a, 0x53, 0x32, 0x85,
	0x9c, 0x36, 0x7b, 0xf3, 0x9f, 0x87, 0x26, 0x7b, 0x4b, 0xed, 0x89, 0xe6, 0x86, 0x1c, 0xf2, 0x69,
	0xb2, 0xd9, 0xd0, 0xbc, 0xf0, 0x2c, 0x71, 0x89, 0x36, 0xc6, 0x46, 0x3c, 0xcd, 0xb2, 0xeb, 0xdd,
	0xd5, 0x4b, 0xd4, 0xbd, 0x2a, 0xfb, 0xbf, 0xf6, 0xb4, 0xff, 0x23, 0x40, 0xeb, 0x63, 0x23, 0xf0,
	0x3b, 0xb4, 0xb1, 0x14, 0x81, 0x01, 0x59, 0x8e, 0x1e, 0x59, 0xf9, 0x51, 0xe1, 0xfd, 0xbf, 0x00,
	0x2e, 0x1c, 0xc6, 0xf1, 0xfb, 0xcf, 0xdf, 0x3f, 0xae, 0xf5, 0xe3, 0x90, 0xfa, 0x01, 0xea, 0x52,
	0x5e, 0x38, 0xe8, 0xc4, 0x1b, 0xc1, 0x1a, 0x75, 0x7e, 0x6d, 0xb4, 0xff, 0x1b, 0xe6, 0xcb, 0x6e,
	0x78, 0xf7, 0x4f, 0xdd, 0x4b, 0xd1, 0x81, 0x13, 0xdd, 0x89, 0xb7, 0x97, 0x44, 0x59, 0x96, 0x35,
	0x8a, 0xa3, 0xc3, 0xd3, 0x79, 0x14, 0x9c, 0xcd, 0xa3, 0xe0, 0xdb, 0x3c, 0x0a, 0x3e, 0x2c, 0xa2,
	0xd6, 0xd9, 0x22, 0x6a, 0x7d, 0x59, 0x44, 0xad, 0xd7, 0x0f, 0xae, 0x04, 0xe3, 0x99, 0x1b, 0x3e,
	0x38, 0x62, 0x52, 0x5d, 0x10, 0xcd, 0x3c, 0x95, 0x0b, 0x48, 0xda, 0x76, 0x4f, 0xe8, 0xd1, 0xcf,
	0x00, 0x00, 0x00, 0xff, 0xff, 0x00, 0xbd, 0x7e, 0x0b, 0xba, 0x03, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// MsgClient is the client API for Msg service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type MsgClient interface {
	RemoveMargin(ctx context.Context, in *MsgRemoveMargin, opts ...grpc.CallOption) (*MsgRemoveMarginResponse, error)
	AddMargin(ctx context.Context, in *MsgAddMargin, opts ...grpc.CallOption) (*MsgAddMarginResponse, error)
}

type msgClient struct {
	cc grpc1.ClientConn
}

func NewMsgClient(cc grpc1.ClientConn) MsgClient {
	return &msgClient{cc}
}

func (c *msgClient) RemoveMargin(ctx context.Context, in *MsgRemoveMargin, opts ...grpc.CallOption) (*MsgRemoveMarginResponse, error) {
	out := new(MsgRemoveMarginResponse)
	err := c.cc.Invoke(ctx, "/nibiru.perp.v1.Msg/RemoveMargin", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *msgClient) AddMargin(ctx context.Context, in *MsgAddMargin, opts ...grpc.CallOption) (*MsgAddMarginResponse, error) {
	out := new(MsgAddMarginResponse)
	err := c.cc.Invoke(ctx, "/nibiru.perp.v1.Msg/AddMargin", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// MsgServer is the server API for Msg service.
type MsgServer interface {
	RemoveMargin(context.Context, *MsgRemoveMargin) (*MsgRemoveMarginResponse, error)
	AddMargin(context.Context, *MsgAddMargin) (*MsgAddMarginResponse, error)
}

// UnimplementedMsgServer can be embedded to have forward compatible implementations.
type UnimplementedMsgServer struct {
}

func (*UnimplementedMsgServer) RemoveMargin(ctx context.Context, req *MsgRemoveMargin) (*MsgRemoveMarginResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RemoveMargin not implemented")
}
func (*UnimplementedMsgServer) AddMargin(ctx context.Context, req *MsgAddMargin) (*MsgAddMarginResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AddMargin not implemented")
}

func RegisterMsgServer(s grpc1.Server, srv MsgServer) {
	s.RegisterService(&_Msg_serviceDesc, srv)
}

func _Msg_RemoveMargin_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgRemoveMargin)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).RemoveMargin(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/nibiru.perp.v1.Msg/RemoveMargin",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).RemoveMargin(ctx, req.(*MsgRemoveMargin))
	}
	return interceptor(ctx, in, info, handler)
}

func _Msg_AddMargin_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgAddMargin)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).AddMargin(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/nibiru.perp.v1.Msg/AddMargin",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).AddMargin(ctx, req.(*MsgAddMargin))
	}
	return interceptor(ctx, in, info, handler)
}

var _Msg_serviceDesc = grpc.ServiceDesc{
	ServiceName: "nibiru.perp.v1.Msg",
	HandlerType: (*MsgServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "RemoveMargin",
			Handler:    _Msg_RemoveMargin_Handler,
		},
		{
			MethodName: "AddMargin",
			Handler:    _Msg_AddMargin_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "perp/v1/tx.proto",
}

func (m *MsgRemoveMargin) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgRemoveMargin) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgRemoveMargin) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	{
		size, err := m.Margin.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = encodeVarintTx(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x1a
	if len(m.Vpool) > 0 {
		i -= len(m.Vpool)
		copy(dAtA[i:], m.Vpool)
		i = encodeVarintTx(dAtA, i, uint64(len(m.Vpool)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.Sender) > 0 {
		i -= len(m.Sender)
		copy(dAtA[i:], m.Sender)
		i = encodeVarintTx(dAtA, i, uint64(len(m.Sender)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *MsgRemoveMarginResponse) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgRemoveMarginResponse) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgRemoveMarginResponse) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	{
		size := m.FundingPayment.Size()
		i -= size
		if _, err := m.FundingPayment.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintTx(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x12
	{
		size, err := m.MarginOut.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = encodeVarintTx(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0xa
	return len(dAtA) - i, nil
}

func (m *MsgAddMargin) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgAddMargin) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgAddMargin) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	{
		size, err := m.Margin.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = encodeVarintTx(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x1a
	if len(m.Vpool) > 0 {
		i -= len(m.Vpool)
		copy(dAtA[i:], m.Vpool)
		i = encodeVarintTx(dAtA, i, uint64(len(m.Vpool)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.Sender) > 0 {
		i -= len(m.Sender)
		copy(dAtA[i:], m.Sender)
		i = encodeVarintTx(dAtA, i, uint64(len(m.Sender)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *MsgAddMarginResponse) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgAddMarginResponse) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgAddMarginResponse) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	{
		size, err := m.MarginOut.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = encodeVarintTx(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0xa
	return len(dAtA) - i, nil
}

func encodeVarintTx(dAtA []byte, offset int, v uint64) int {
	offset -= sovTx(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *MsgRemoveMargin) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Sender)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = len(m.Vpool)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = m.Margin.Size()
	n += 1 + l + sovTx(uint64(l))
	return n
}

func (m *MsgRemoveMarginResponse) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = m.MarginOut.Size()
	n += 1 + l + sovTx(uint64(l))
	l = m.FundingPayment.Size()
	n += 1 + l + sovTx(uint64(l))
	return n
}

func (m *MsgAddMargin) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Sender)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = len(m.Vpool)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = m.Margin.Size()
	n += 1 + l + sovTx(uint64(l))
	return n
}

func (m *MsgAddMarginResponse) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = m.MarginOut.Size()
	n += 1 + l + sovTx(uint64(l))
	return n
}

func sovTx(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozTx(x uint64) (n int) {
	return sovTx(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *MsgRemoveMargin) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTx
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
			return fmt.Errorf("proto: MsgRemoveMargin: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgRemoveMargin: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Sender", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
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
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Sender = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Vpool", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
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
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Vpool = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Margin", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
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
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Margin.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipTx(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthTx
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
func (m *MsgRemoveMarginResponse) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTx
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
			return fmt.Errorf("proto: MsgRemoveMarginResponse: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgRemoveMarginResponse: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field MarginOut", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
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
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.MarginOut.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field FundingPayment", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
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
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.FundingPayment.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipTx(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthTx
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
func (m *MsgAddMargin) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTx
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
			return fmt.Errorf("proto: MsgAddMargin: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgAddMargin: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Sender", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
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
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Sender = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Vpool", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
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
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Vpool = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Margin", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
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
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Margin.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipTx(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthTx
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
func (m *MsgAddMarginResponse) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTx
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
			return fmt.Errorf("proto: MsgAddMarginResponse: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgAddMarginResponse: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field MarginOut", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
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
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.MarginOut.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipTx(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthTx
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
func skipTx(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowTx
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
					return 0, ErrIntOverflowTx
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
					return 0, ErrIntOverflowTx
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
				return 0, ErrInvalidLengthTx
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupTx
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthTx
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthTx        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowTx          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupTx = fmt.Errorf("proto: unexpected end of group")
)
