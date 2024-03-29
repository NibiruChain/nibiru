// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: nibiru/spot/v1/params.proto

package types

import (
	fmt "fmt"
	_ "github.com/cosmos/cosmos-proto"
	github_com_cosmos_cosmos_sdk_types "github.com/cosmos/cosmos-sdk/types"
	types "github.com/cosmos/cosmos-sdk/types"
	_ "github.com/cosmos/gogoproto/gogoproto"
	proto "github.com/cosmos/gogoproto/proto"
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

// Params defines the parameters for the module.
type Params struct {
	// The start pool number, i.e. the first pool number that isn't taken yet.
	StartingPoolNumber uint64 `protobuf:"varint,1,opt,name=starting_pool_number,json=startingPoolNumber,proto3" json:"starting_pool_number,omitempty"`
	// The cost of creating a pool, taken from the pool creator's account.
	PoolCreationFee github_com_cosmos_cosmos_sdk_types.Coins `protobuf:"bytes,2,rep,name=pool_creation_fee,json=poolCreationFee,proto3,castrepeated=github.com/cosmos/cosmos-sdk/types.Coins" json:"pool_creation_fee" yaml:"pool_creation_fee"`
	// The assets that can be used to create liquidity pools
	WhitelistedAsset []string `protobuf:"bytes,3,rep,name=whitelisted_asset,json=whitelistedAsset,proto3" json:"whitelisted_asset,omitempty"`
}

func (m *Params) Reset()      { *m = Params{} }
func (*Params) ProtoMessage() {}
func (*Params) Descriptor() ([]byte, []int) {
	return fileDescriptor_532c93f2cfe0dc59, []int{0}
}
func (m *Params) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Params) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Params.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Params) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Params.Merge(m, src)
}
func (m *Params) XXX_Size() int {
	return m.Size()
}
func (m *Params) XXX_DiscardUnknown() {
	xxx_messageInfo_Params.DiscardUnknown(m)
}

var xxx_messageInfo_Params proto.InternalMessageInfo

func (m *Params) GetStartingPoolNumber() uint64 {
	if m != nil {
		return m.StartingPoolNumber
	}
	return 0
}

func (m *Params) GetPoolCreationFee() github_com_cosmos_cosmos_sdk_types.Coins {
	if m != nil {
		return m.PoolCreationFee
	}
	return nil
}

func (m *Params) GetWhitelistedAsset() []string {
	if m != nil {
		return m.WhitelistedAsset
	}
	return nil
}

func init() {
	proto.RegisterType((*Params)(nil), "nibiru.spot.v1.Params")
}

func init() { proto.RegisterFile("nibiru/spot/v1/params.proto", fileDescriptor_532c93f2cfe0dc59) }

var fileDescriptor_532c93f2cfe0dc59 = []byte{
	// 351 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x64, 0x91, 0x3f, 0x4f, 0xc2, 0x40,
	0x18, 0xc6, 0x5b, 0x20, 0x24, 0xd6, 0x44, 0xa5, 0x61, 0x28, 0x98, 0xb4, 0x84, 0xa9, 0xd1, 0xd8,
	0xb3, 0xba, 0xb1, 0x09, 0xc6, 0xc9, 0x10, 0xc2, 0xe8, 0xd2, 0x5c, 0xcb, 0x59, 0x2e, 0xb6, 0x7d,
	0x9b, 0xde, 0x81, 0x32, 0xfb, 0x05, 0x4c, 0x5c, 0x1c, 0x9d, 0xfd, 0x24, 0x8c, 0x8c, 0x4e, 0x68,
	0xe0, 0x1b, 0xf8, 0x09, 0x4c, 0xef, 0xce, 0x84, 0xc4, 0xe9, 0xfe, 0xfc, 0xde, 0xe7, 0x7d, 0x9f,
	0xbc, 0x8f, 0x71, 0x9c, 0xd1, 0x90, 0x16, 0x33, 0xc4, 0x72, 0xe0, 0x68, 0xee, 0xa3, 0x1c, 0x17,
	0x38, 0x65, 0x5e, 0x5e, 0x00, 0x07, 0xf3, 0x40, 0x42, 0xaf, 0x84, 0xde, 0xdc, 0x6f, 0x37, 0x63,
	0x88, 0x41, 0x20, 0x54, 0xde, 0x64, 0x55, 0xdb, 0x8e, 0x80, 0xa5, 0xc0, 0x50, 0x88, 0x19, 0x41,
	0x73, 0x3f, 0x24, 0x1c, 0xfb, 0x28, 0x02, 0x9a, 0x29, 0xde, 0x92, 0x3c, 0x90, 0x42, 0xf9, 0x90,
	0xa8, 0xfb, 0x5c, 0x31, 0xea, 0x23, 0x31, 0xd1, 0x3c, 0x37, 0x9a, 0x8c, 0xe3, 0x82, 0xd3, 0x2c,
	0x0e, 0x72, 0x80, 0x24, 0xc8, 0x66, 0x69, 0x48, 0x0a, 0x4b, 0xef, 0xe8, 0x6e, 0x6d, 0x6c, 0xfe,
	0xb1, 0x11, 0x40, 0x32, 0x14, 0xc4, 0x7c, 0xd5, 0x8d, 0x86, 0xa8, 0x8c, 0x0a, 0x82, 0x39, 0x85,
	0x2c, 0xb8, 0x27, 0xc4, 0xaa, 0x74, 0xaa, 0xee, 0xfe, 0x45, 0xcb, 0x53, 0x73, 0x4a, 0x53, 0x9e,
	0x32, 0xe5, 0x0d, 0x80, 0x66, 0xfd, 0xdb, 0xe5, 0xda, 0xd1, 0x7e, 0xd6, 0x8e, 0xb5, 0xc0, 0x69,
	0xd2, 0xeb, 0xfe, 0xeb, 0xd0, 0xfd, 0xf8, 0x72, 0xdc, 0x98, 0xf2, 0xe9, 0x2c, 0xf4, 0x22, 0x48,
	0x95, 0x61, 0x75, 0x9c, 0xb1, 0xc9, 0x03, 0xe2, 0x8b, 0x9c, 0x30, 0xd1, 0x8c, 0x8d, 0x0f, 0x4b,
	0xfd, 0x40, 0xc9, 0x6f, 0x08, 0x31, 0x4f, 0x8d, 0xc6, 0xe3, 0x94, 0x72, 0x92, 0x50, 0xc6, 0xc9,
	0x24, 0xc0, 0x8c, 0x11, 0x6e, 0x55, 0x3b, 0x55, 0x77, 0x6f, 0x7c, 0xb4, 0x03, 0xae, 0xca, 0xff,
	0x5e, 0xed, 0xed, 0xdd, 0xd1, 0xfa, 0xd7, 0xcb, 0x8d, 0xad, 0xaf, 0x36, 0xb6, 0xfe, 0xbd, 0xb1,
	0xf5, 0x97, 0xad, 0xad, 0xad, 0xb6, 0xb6, 0xf6, 0xb9, 0xb5, 0xb5, 0xbb, 0x93, 0x1d, 0x1f, 0x43,
	0x91, 0xc5, 0x60, 0x8a, 0x69, 0x86, 0x54, 0x68, 0x4f, 0x32, 0x36, 0xe1, 0x27, 0xac, 0x8b, 0x95,
	0x5e, 0xfe, 0x06, 0x00, 0x00, 0xff, 0xff, 0xe7, 0xf4, 0x89, 0x01, 0xd2, 0x01, 0x00, 0x00,
}

func (m *Params) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Params) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Params) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.WhitelistedAsset) > 0 {
		for iNdEx := len(m.WhitelistedAsset) - 1; iNdEx >= 0; iNdEx-- {
			i -= len(m.WhitelistedAsset[iNdEx])
			copy(dAtA[i:], m.WhitelistedAsset[iNdEx])
			i = encodeVarintParams(dAtA, i, uint64(len(m.WhitelistedAsset[iNdEx])))
			i--
			dAtA[i] = 0x1a
		}
	}
	if len(m.PoolCreationFee) > 0 {
		for iNdEx := len(m.PoolCreationFee) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.PoolCreationFee[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintParams(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x12
		}
	}
	if m.StartingPoolNumber != 0 {
		i = encodeVarintParams(dAtA, i, uint64(m.StartingPoolNumber))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func encodeVarintParams(dAtA []byte, offset int, v uint64) int {
	offset -= sovParams(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *Params) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.StartingPoolNumber != 0 {
		n += 1 + sovParams(uint64(m.StartingPoolNumber))
	}
	if len(m.PoolCreationFee) > 0 {
		for _, e := range m.PoolCreationFee {
			l = e.Size()
			n += 1 + l + sovParams(uint64(l))
		}
	}
	if len(m.WhitelistedAsset) > 0 {
		for _, s := range m.WhitelistedAsset {
			l = len(s)
			n += 1 + l + sovParams(uint64(l))
		}
	}
	return n
}

func sovParams(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozParams(x uint64) (n int) {
	return sovParams(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *Params) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowParams
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
			return fmt.Errorf("proto: Params: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Params: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field StartingPoolNumber", wireType)
			}
			m.StartingPoolNumber = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowParams
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.StartingPoolNumber |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field PoolCreationFee", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowParams
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
				return ErrInvalidLengthParams
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthParams
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.PoolCreationFee = append(m.PoolCreationFee, types.Coin{})
			if err := m.PoolCreationFee[len(m.PoolCreationFee)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field WhitelistedAsset", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowParams
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
				return ErrInvalidLengthParams
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthParams
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.WhitelistedAsset = append(m.WhitelistedAsset, string(dAtA[iNdEx:postIndex]))
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipParams(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthParams
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
func skipParams(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowParams
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
					return 0, ErrIntOverflowParams
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
					return 0, ErrIntOverflowParams
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
				return 0, ErrInvalidLengthParams
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupParams
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthParams
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthParams        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowParams          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupParams = fmt.Errorf("proto: unexpected end of group")
)
