// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: stablecoin/v1/genesis.proto

package types

import (
	fmt "fmt"
	types "github.com/cosmos/cosmos-sdk/types"
	_ "github.com/cosmos/gogoproto/gogoproto"
	proto "github.com/gogo/protobuf/proto"
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

// GenesisState defines the stablecoin module's genesis state.
type GenesisState struct {
	Params               Params     `protobuf:"bytes,1,opt,name=params,proto3" json:"params"`
	ModuleAccountBalance types.Coin `protobuf:"bytes,2,opt,name=module_account_balance,json=moduleAccountBalance,proto3" json:"module_account_balance" yaml:"module_account_balance"`
}

func (m *GenesisState) Reset()         { *m = GenesisState{} }
func (m *GenesisState) String() string { return proto.CompactTextString(m) }
func (*GenesisState) ProtoMessage()    {}
func (*GenesisState) Descriptor() ([]byte, []int) {
	return fileDescriptor_d4ea16ec0b847ae6, []int{0}
}
func (m *GenesisState) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *GenesisState) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_GenesisState.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *GenesisState) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GenesisState.Merge(m, src)
}
func (m *GenesisState) XXX_Size() int {
	return m.Size()
}
func (m *GenesisState) XXX_DiscardUnknown() {
	xxx_messageInfo_GenesisState.DiscardUnknown(m)
}

var xxx_messageInfo_GenesisState proto.InternalMessageInfo

func (m *GenesisState) GetParams() Params {
	if m != nil {
		return m.Params
	}
	return Params{}
}

func (m *GenesisState) GetModuleAccountBalance() types.Coin {
	if m != nil {
		return m.ModuleAccountBalance
	}
	return types.Coin{}
}

func init() {
	proto.RegisterType((*GenesisState)(nil), "nibiru.stablecoin.v1.GenesisState")
}

func init() { proto.RegisterFile("stablecoin/v1/genesis.proto", fileDescriptor_d4ea16ec0b847ae6) }

var fileDescriptor_d4ea16ec0b847ae6 = []byte{
	// 297 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x6c, 0x50, 0x31, 0x4b, 0xc4, 0x30,
	0x18, 0x6d, 0x44, 0x6e, 0xa8, 0x4e, 0xa5, 0xc8, 0x59, 0x35, 0x27, 0x05, 0xc1, 0x29, 0xb1, 0xba,
	0xdd, 0x66, 0x6f, 0x10, 0x1c, 0x44, 0xce, 0xcd, 0xe5, 0x48, 0x62, 0xe8, 0x05, 0xda, 0xa4, 0x34,
	0x69, 0xf1, 0xfe, 0x85, 0xbf, 0xc9, 0xe9, 0xc6, 0x1b, 0x9d, 0x0e, 0x69, 0xff, 0x81, 0xbf, 0x40,
	0x9a, 0x14, 0xf4, 0xe0, 0xb6, 0x8f, 0xf7, 0xbd, 0xef, 0xbd, 0xf7, 0x3d, 0xff, 0x4c, 0x1b, 0x42,
	0x73, 0xce, 0x94, 0x90, 0xb8, 0x49, 0x70, 0xc6, 0x25, 0xd7, 0x42, 0xa3, 0xb2, 0x52, 0x46, 0x05,
	0xa1, 0x14, 0x54, 0x54, 0x35, 0xfa, 0xe3, 0xa0, 0x26, 0x89, 0x20, 0x53, 0xba, 0x50, 0x1a, 0x53,
	0xa2, 0x39, 0x6e, 0x12, 0xca, 0x0d, 0x49, 0xb0, 0x5d, 0xda, 0xab, 0x28, 0xcc, 0x54, 0xa6, 0xec,
	0x88, 0xfb, 0x69, 0x40, 0xa3, 0x5d, 0xa3, 0x92, 0x54, 0xa4, 0x18, 0x7c, 0xe2, 0x4f, 0xe0, 0x1f,
	0x3f, 0x38, 0xe7, 0x17, 0x43, 0x0c, 0x0f, 0xa6, 0xfe, 0xc8, 0x11, 0xc6, 0xe0, 0x12, 0x5c, 0x1f,
	0xdd, 0x9e, 0xa3, 0x7d, 0x49, 0xd0, 0xb3, 0xe5, 0xa4, 0x87, 0xeb, 0xed, 0xc4, 0x9b, 0x0f, 0x17,
	0x41, 0xe3, 0x9f, 0x14, 0xea, 0xad, 0xce, 0xf9, 0x82, 0x30, 0xa6, 0x6a, 0x69, 0x16, 0x94, 0xe4,
	0x44, 0x32, 0x3e, 0x3e, 0xb0, 0x5a, 0xa7, 0xc8, 0xe5, 0x47, 0x7d, 0x7e, 0x34, 0xe4, 0x47, 0x33,
	0x25, 0x64, 0x7a, 0xd5, 0x0b, 0xfd, 0x6c, 0x27, 0x17, 0x2b, 0x52, 0xe4, 0xd3, 0x78, 0xbf, 0x4c,
	0x3c, 0x0f, 0xdd, 0xe2, 0xde, 0xe1, 0xa9, 0x83, 0xd3, 0xc7, 0x75, 0x0b, 0xc1, 0xa6, 0x85, 0xe0,
	0xbb, 0x85, 0xe0, 0xa3, 0x83, 0xde, 0xa6, 0x83, 0xde, 0x57, 0x07, 0xbd, 0xd7, 0x9b, 0x4c, 0x98,
	0x65, 0x4d, 0x11, 0x53, 0x05, 0x7e, 0xb2, 0x7f, 0xcc, 0x96, 0x44, 0x48, 0xec, 0x7e, 0xc2, 0xef,
	0xf8, 0x5f, 0x35, 0x66, 0x55, 0x72, 0x4d, 0x47, 0xb6, 0x97, 0xbb, 0xdf, 0x00, 0x00, 0x00, 0xff,
	0xff, 0x32, 0xcc, 0xe0, 0x24, 0x9e, 0x01, 0x00, 0x00,
}

func (m *GenesisState) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *GenesisState) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *GenesisState) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	{
		size, err := m.ModuleAccountBalance.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = encodeVarintGenesis(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x12
	{
		size, err := m.Params.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = encodeVarintGenesis(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0xa
	return len(dAtA) - i, nil
}

func encodeVarintGenesis(dAtA []byte, offset int, v uint64) int {
	offset -= sovGenesis(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *GenesisState) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = m.Params.Size()
	n += 1 + l + sovGenesis(uint64(l))
	l = m.ModuleAccountBalance.Size()
	n += 1 + l + sovGenesis(uint64(l))
	return n
}

func sovGenesis(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozGenesis(x uint64) (n int) {
	return sovGenesis(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *GenesisState) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowGenesis
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
			return fmt.Errorf("proto: GenesisState: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: GenesisState: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Params", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenesis
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
				return ErrInvalidLengthGenesis
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthGenesis
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Params.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ModuleAccountBalance", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenesis
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
				return ErrInvalidLengthGenesis
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthGenesis
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.ModuleAccountBalance.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipGenesis(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthGenesis
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
func skipGenesis(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowGenesis
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
					return 0, ErrIntOverflowGenesis
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
					return 0, ErrIntOverflowGenesis
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
				return 0, ErrInvalidLengthGenesis
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupGenesis
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthGenesis
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthGenesis        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowGenesis          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupGenesis = fmt.Errorf("proto: unexpected end of group")
)
