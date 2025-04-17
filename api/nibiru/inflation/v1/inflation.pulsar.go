// Code generated by protoc-gen-go-pulsar. DO NOT EDIT.
package inflationv1

import (
	fmt "fmt"
	_ "github.com/cosmos/cosmos-proto"
	runtime "github.com/cosmos/cosmos-proto/runtime"
	_ "github.com/cosmos/gogoproto/gogoproto"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoiface "google.golang.org/protobuf/runtime/protoiface"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	io "io"
	reflect "reflect"
	sync "sync"
)

var (
	md_InflationDistribution                    protoreflect.MessageDescriptor
	fd_InflationDistribution_staking_rewards    protoreflect.FieldDescriptor
	fd_InflationDistribution_community_pool     protoreflect.FieldDescriptor
	fd_InflationDistribution_strategic_reserves protoreflect.FieldDescriptor
)

func init() {
	file_nibiru_inflation_v1_inflation_proto_init()
	md_InflationDistribution = File_nibiru_inflation_v1_inflation_proto.Messages().ByName("InflationDistribution")
	fd_InflationDistribution_staking_rewards = md_InflationDistribution.Fields().ByName("staking_rewards")
	fd_InflationDistribution_community_pool = md_InflationDistribution.Fields().ByName("community_pool")
	fd_InflationDistribution_strategic_reserves = md_InflationDistribution.Fields().ByName("strategic_reserves")
}

var _ protoreflect.Message = (*fastReflection_InflationDistribution)(nil)

type fastReflection_InflationDistribution InflationDistribution

func (x *InflationDistribution) ProtoReflect() protoreflect.Message {
	return (*fastReflection_InflationDistribution)(x)
}

func (x *InflationDistribution) slowProtoReflect() protoreflect.Message {
	mi := &file_nibiru_inflation_v1_inflation_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

var _fastReflection_InflationDistribution_messageType fastReflection_InflationDistribution_messageType
var _ protoreflect.MessageType = fastReflection_InflationDistribution_messageType{}

type fastReflection_InflationDistribution_messageType struct{}

func (x fastReflection_InflationDistribution_messageType) Zero() protoreflect.Message {
	return (*fastReflection_InflationDistribution)(nil)
}
func (x fastReflection_InflationDistribution_messageType) New() protoreflect.Message {
	return new(fastReflection_InflationDistribution)
}
func (x fastReflection_InflationDistribution_messageType) Descriptor() protoreflect.MessageDescriptor {
	return md_InflationDistribution
}

// Descriptor returns message descriptor, which contains only the protobuf
// type information for the message.
func (x *fastReflection_InflationDistribution) Descriptor() protoreflect.MessageDescriptor {
	return md_InflationDistribution
}

// Type returns the message type, which encapsulates both Go and protobuf
// type information. If the Go type information is not needed,
// it is recommended that the message descriptor be used instead.
func (x *fastReflection_InflationDistribution) Type() protoreflect.MessageType {
	return _fastReflection_InflationDistribution_messageType
}

// New returns a newly allocated and mutable empty message.
func (x *fastReflection_InflationDistribution) New() protoreflect.Message {
	return new(fastReflection_InflationDistribution)
}

// Interface unwraps the message reflection interface and
// returns the underlying ProtoMessage interface.
func (x *fastReflection_InflationDistribution) Interface() protoreflect.ProtoMessage {
	return (*InflationDistribution)(x)
}

// Range iterates over every populated field in an undefined order,
// calling f for each field descriptor and value encountered.
// Range returns immediately if f returns false.
// While iterating, mutating operations may only be performed
// on the current field descriptor.
func (x *fastReflection_InflationDistribution) Range(f func(protoreflect.FieldDescriptor, protoreflect.Value) bool) {
	if x.StakingRewards != "" {
		value := protoreflect.ValueOfString(x.StakingRewards)
		if !f(fd_InflationDistribution_staking_rewards, value) {
			return
		}
	}
	if x.CommunityPool != "" {
		value := protoreflect.ValueOfString(x.CommunityPool)
		if !f(fd_InflationDistribution_community_pool, value) {
			return
		}
	}
	if x.StrategicReserves != "" {
		value := protoreflect.ValueOfString(x.StrategicReserves)
		if !f(fd_InflationDistribution_strategic_reserves, value) {
			return
		}
	}
}

// Has reports whether a field is populated.
//
// Some fields have the property of nullability where it is possible to
// distinguish between the default value of a field and whether the field
// was explicitly populated with the default value. Singular message fields,
// member fields of a oneof, and proto2 scalar fields are nullable. Such
// fields are populated only if explicitly set.
//
// In other cases (aside from the nullable cases above),
// a proto3 scalar field is populated if it contains a non-zero value, and
// a repeated field is populated if it is non-empty.
func (x *fastReflection_InflationDistribution) Has(fd protoreflect.FieldDescriptor) bool {
	switch fd.FullName() {
	case "nibiru.inflation.v1.InflationDistribution.staking_rewards":
		return x.StakingRewards != ""
	case "nibiru.inflation.v1.InflationDistribution.community_pool":
		return x.CommunityPool != ""
	case "nibiru.inflation.v1.InflationDistribution.strategic_reserves":
		return x.StrategicReserves != ""
	default:
		if fd.IsExtension() {
			panic(fmt.Errorf("proto3 declared messages do not support extensions: nibiru.inflation.v1.InflationDistribution"))
		}
		panic(fmt.Errorf("message nibiru.inflation.v1.InflationDistribution does not contain field %s", fd.FullName()))
	}
}

// Clear clears the field such that a subsequent Has call reports false.
//
// Clearing an extension field clears both the extension type and value
// associated with the given field number.
//
// Clear is a mutating operation and unsafe for concurrent use.
func (x *fastReflection_InflationDistribution) Clear(fd protoreflect.FieldDescriptor) {
	switch fd.FullName() {
	case "nibiru.inflation.v1.InflationDistribution.staking_rewards":
		x.StakingRewards = ""
	case "nibiru.inflation.v1.InflationDistribution.community_pool":
		x.CommunityPool = ""
	case "nibiru.inflation.v1.InflationDistribution.strategic_reserves":
		x.StrategicReserves = ""
	default:
		if fd.IsExtension() {
			panic(fmt.Errorf("proto3 declared messages do not support extensions: nibiru.inflation.v1.InflationDistribution"))
		}
		panic(fmt.Errorf("message nibiru.inflation.v1.InflationDistribution does not contain field %s", fd.FullName()))
	}
}

// Get retrieves the value for a field.
//
// For unpopulated scalars, it returns the default value, where
// the default value of a bytes scalar is guaranteed to be a copy.
// For unpopulated composite types, it returns an empty, read-only view
// of the value; to obtain a mutable reference, use Mutable.
func (x *fastReflection_InflationDistribution) Get(descriptor protoreflect.FieldDescriptor) protoreflect.Value {
	switch descriptor.FullName() {
	case "nibiru.inflation.v1.InflationDistribution.staking_rewards":
		value := x.StakingRewards
		return protoreflect.ValueOfString(value)
	case "nibiru.inflation.v1.InflationDistribution.community_pool":
		value := x.CommunityPool
		return protoreflect.ValueOfString(value)
	case "nibiru.inflation.v1.InflationDistribution.strategic_reserves":
		value := x.StrategicReserves
		return protoreflect.ValueOfString(value)
	default:
		if descriptor.IsExtension() {
			panic(fmt.Errorf("proto3 declared messages do not support extensions: nibiru.inflation.v1.InflationDistribution"))
		}
		panic(fmt.Errorf("message nibiru.inflation.v1.InflationDistribution does not contain field %s", descriptor.FullName()))
	}
}

// Set stores the value for a field.
//
// For a field belonging to a oneof, it implicitly clears any other field
// that may be currently set within the same oneof.
// For extension fields, it implicitly stores the provided ExtensionType.
// When setting a composite type, it is unspecified whether the stored value
// aliases the source's memory in any way. If the composite value is an
// empty, read-only value, then it panics.
//
// Set is a mutating operation and unsafe for concurrent use.
func (x *fastReflection_InflationDistribution) Set(fd protoreflect.FieldDescriptor, value protoreflect.Value) {
	switch fd.FullName() {
	case "nibiru.inflation.v1.InflationDistribution.staking_rewards":
		x.StakingRewards = value.Interface().(string)
	case "nibiru.inflation.v1.InflationDistribution.community_pool":
		x.CommunityPool = value.Interface().(string)
	case "nibiru.inflation.v1.InflationDistribution.strategic_reserves":
		x.StrategicReserves = value.Interface().(string)
	default:
		if fd.IsExtension() {
			panic(fmt.Errorf("proto3 declared messages do not support extensions: nibiru.inflation.v1.InflationDistribution"))
		}
		panic(fmt.Errorf("message nibiru.inflation.v1.InflationDistribution does not contain field %s", fd.FullName()))
	}
}

// Mutable returns a mutable reference to a composite type.
//
// If the field is unpopulated, it may allocate a composite value.
// For a field belonging to a oneof, it implicitly clears any other field
// that may be currently set within the same oneof.
// For extension fields, it implicitly stores the provided ExtensionType
// if not already stored.
// It panics if the field does not contain a composite type.
//
// Mutable is a mutating operation and unsafe for concurrent use.
func (x *fastReflection_InflationDistribution) Mutable(fd protoreflect.FieldDescriptor) protoreflect.Value {
	switch fd.FullName() {
	case "nibiru.inflation.v1.InflationDistribution.staking_rewards":
		panic(fmt.Errorf("field staking_rewards of message nibiru.inflation.v1.InflationDistribution is not mutable"))
	case "nibiru.inflation.v1.InflationDistribution.community_pool":
		panic(fmt.Errorf("field community_pool of message nibiru.inflation.v1.InflationDistribution is not mutable"))
	case "nibiru.inflation.v1.InflationDistribution.strategic_reserves":
		panic(fmt.Errorf("field strategic_reserves of message nibiru.inflation.v1.InflationDistribution is not mutable"))
	default:
		if fd.IsExtension() {
			panic(fmt.Errorf("proto3 declared messages do not support extensions: nibiru.inflation.v1.InflationDistribution"))
		}
		panic(fmt.Errorf("message nibiru.inflation.v1.InflationDistribution does not contain field %s", fd.FullName()))
	}
}

// NewField returns a new value that is assignable to the field
// for the given descriptor. For scalars, this returns the default value.
// For lists, maps, and messages, this returns a new, empty, mutable value.
func (x *fastReflection_InflationDistribution) NewField(fd protoreflect.FieldDescriptor) protoreflect.Value {
	switch fd.FullName() {
	case "nibiru.inflation.v1.InflationDistribution.staking_rewards":
		return protoreflect.ValueOfString("")
	case "nibiru.inflation.v1.InflationDistribution.community_pool":
		return protoreflect.ValueOfString("")
	case "nibiru.inflation.v1.InflationDistribution.strategic_reserves":
		return protoreflect.ValueOfString("")
	default:
		if fd.IsExtension() {
			panic(fmt.Errorf("proto3 declared messages do not support extensions: nibiru.inflation.v1.InflationDistribution"))
		}
		panic(fmt.Errorf("message nibiru.inflation.v1.InflationDistribution does not contain field %s", fd.FullName()))
	}
}

// WhichOneof reports which field within the oneof is populated,
// returning nil if none are populated.
// It panics if the oneof descriptor does not belong to this message.
func (x *fastReflection_InflationDistribution) WhichOneof(d protoreflect.OneofDescriptor) protoreflect.FieldDescriptor {
	switch d.FullName() {
	default:
		panic(fmt.Errorf("%s is not a oneof field in nibiru.inflation.v1.InflationDistribution", d.FullName()))
	}
	panic("unreachable")
}

// GetUnknown retrieves the entire list of unknown fields.
// The caller may only mutate the contents of the RawFields
// if the mutated bytes are stored back into the message with SetUnknown.
func (x *fastReflection_InflationDistribution) GetUnknown() protoreflect.RawFields {
	return x.unknownFields
}

// SetUnknown stores an entire list of unknown fields.
// The raw fields must be syntactically valid according to the wire format.
// An implementation may panic if this is not the case.
// Once stored, the caller must not mutate the content of the RawFields.
// An empty RawFields may be passed to clear the fields.
//
// SetUnknown is a mutating operation and unsafe for concurrent use.
func (x *fastReflection_InflationDistribution) SetUnknown(fields protoreflect.RawFields) {
	x.unknownFields = fields
}

// IsValid reports whether the message is valid.
//
// An invalid message is an empty, read-only value.
//
// An invalid message often corresponds to a nil pointer of the concrete
// message type, but the details are implementation dependent.
// Validity is not part of the protobuf data model, and may not
// be preserved in marshaling or other operations.
func (x *fastReflection_InflationDistribution) IsValid() bool {
	return x != nil
}

// ProtoMethods returns optional fastReflectionFeature-path implementations of various operations.
// This method may return nil.
//
// The returned methods type is identical to
// "google.golang.org/protobuf/runtime/protoiface".Methods.
// Consult the protoiface package documentation for details.
func (x *fastReflection_InflationDistribution) ProtoMethods() *protoiface.Methods {
	size := func(input protoiface.SizeInput) protoiface.SizeOutput {
		x := input.Message.Interface().(*InflationDistribution)
		if x == nil {
			return protoiface.SizeOutput{
				NoUnkeyedLiterals: input.NoUnkeyedLiterals,
				Size:              0,
			}
		}
		options := runtime.SizeInputToOptions(input)
		_ = options
		var n int
		var l int
		_ = l
		l = len(x.StakingRewards)
		if l > 0 {
			n += 1 + l + runtime.Sov(uint64(l))
		}
		l = len(x.CommunityPool)
		if l > 0 {
			n += 1 + l + runtime.Sov(uint64(l))
		}
		l = len(x.StrategicReserves)
		if l > 0 {
			n += 1 + l + runtime.Sov(uint64(l))
		}
		if x.unknownFields != nil {
			n += len(x.unknownFields)
		}
		return protoiface.SizeOutput{
			NoUnkeyedLiterals: input.NoUnkeyedLiterals,
			Size:              n,
		}
	}

	marshal := func(input protoiface.MarshalInput) (protoiface.MarshalOutput, error) {
		x := input.Message.Interface().(*InflationDistribution)
		if x == nil {
			return protoiface.MarshalOutput{
				NoUnkeyedLiterals: input.NoUnkeyedLiterals,
				Buf:               input.Buf,
			}, nil
		}
		options := runtime.MarshalInputToOptions(input)
		_ = options
		size := options.Size(x)
		dAtA := make([]byte, size)
		i := len(dAtA)
		_ = i
		var l int
		_ = l
		if x.unknownFields != nil {
			i -= len(x.unknownFields)
			copy(dAtA[i:], x.unknownFields)
		}
		if len(x.StrategicReserves) > 0 {
			i -= len(x.StrategicReserves)
			copy(dAtA[i:], x.StrategicReserves)
			i = runtime.EncodeVarint(dAtA, i, uint64(len(x.StrategicReserves)))
			i--
			dAtA[i] = 0x1a
		}
		if len(x.CommunityPool) > 0 {
			i -= len(x.CommunityPool)
			copy(dAtA[i:], x.CommunityPool)
			i = runtime.EncodeVarint(dAtA, i, uint64(len(x.CommunityPool)))
			i--
			dAtA[i] = 0x12
		}
		if len(x.StakingRewards) > 0 {
			i -= len(x.StakingRewards)
			copy(dAtA[i:], x.StakingRewards)
			i = runtime.EncodeVarint(dAtA, i, uint64(len(x.StakingRewards)))
			i--
			dAtA[i] = 0xa
		}
		if input.Buf != nil {
			input.Buf = append(input.Buf, dAtA...)
		} else {
			input.Buf = dAtA
		}
		return protoiface.MarshalOutput{
			NoUnkeyedLiterals: input.NoUnkeyedLiterals,
			Buf:               input.Buf,
		}, nil
	}
	unmarshal := func(input protoiface.UnmarshalInput) (protoiface.UnmarshalOutput, error) {
		x := input.Message.Interface().(*InflationDistribution)
		if x == nil {
			return protoiface.UnmarshalOutput{
				NoUnkeyedLiterals: input.NoUnkeyedLiterals,
				Flags:             input.Flags,
			}, nil
		}
		options := runtime.UnmarshalInputToOptions(input)
		_ = options
		dAtA := input.Buf
		l := len(dAtA)
		iNdEx := 0
		for iNdEx < l {
			preIndex := iNdEx
			var wire uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrIntOverflow
				}
				if iNdEx >= l {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
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
				return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, fmt.Errorf("proto: InflationDistribution: wiretype end group for non-group")
			}
			if fieldNum <= 0 {
				return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, fmt.Errorf("proto: InflationDistribution: illegal tag %d (wire type %d)", fieldNum, wire)
			}
			switch fieldNum {
			case 1:
				if wireType != 2 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, fmt.Errorf("proto: wrong wireType = %d for field StakingRewards", wireType)
				}
				var stringLen uint64
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrIntOverflow
					}
					if iNdEx >= l {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
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
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrInvalidLength
				}
				postIndex := iNdEx + intStringLen
				if postIndex < 0 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrInvalidLength
				}
				if postIndex > l {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
				}
				x.StakingRewards = string(dAtA[iNdEx:postIndex])
				iNdEx = postIndex
			case 2:
				if wireType != 2 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, fmt.Errorf("proto: wrong wireType = %d for field CommunityPool", wireType)
				}
				var stringLen uint64
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrIntOverflow
					}
					if iNdEx >= l {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
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
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrInvalidLength
				}
				postIndex := iNdEx + intStringLen
				if postIndex < 0 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrInvalidLength
				}
				if postIndex > l {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
				}
				x.CommunityPool = string(dAtA[iNdEx:postIndex])
				iNdEx = postIndex
			case 3:
				if wireType != 2 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, fmt.Errorf("proto: wrong wireType = %d for field StrategicReserves", wireType)
				}
				var stringLen uint64
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrIntOverflow
					}
					if iNdEx >= l {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
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
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrInvalidLength
				}
				postIndex := iNdEx + intStringLen
				if postIndex < 0 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrInvalidLength
				}
				if postIndex > l {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
				}
				x.StrategicReserves = string(dAtA[iNdEx:postIndex])
				iNdEx = postIndex
			default:
				iNdEx = preIndex
				skippy, err := runtime.Skip(dAtA[iNdEx:])
				if err != nil {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, err
				}
				if (skippy < 0) || (iNdEx+skippy) < 0 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrInvalidLength
				}
				if (iNdEx + skippy) > l {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
				}
				if !options.DiscardUnknown {
					x.unknownFields = append(x.unknownFields, dAtA[iNdEx:iNdEx+skippy]...)
				}
				iNdEx += skippy
			}
		}

		if iNdEx > l {
			return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
		}
		return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, nil
	}
	return &protoiface.Methods{
		NoUnkeyedLiterals: struct{}{},
		Flags:             protoiface.SupportMarshalDeterministic | protoiface.SupportUnmarshalDiscardUnknown,
		Size:              size,
		Marshal:           marshal,
		Unmarshal:         unmarshal,
		Merge:             nil,
		CheckInitialized:  nil,
	}
}

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.27.0
// 	protoc        (unknown)
// source: nibiru/inflation/v1/inflation.proto

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// InflationDistribution defines the distribution in which inflation is
// allocated through minting on each epoch (staking, community, strategic). It
// excludes the team vesting distribution.
type InflationDistribution struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// staking_rewards defines the proportion of the minted_denom that is
	// to be allocated as staking rewards
	StakingRewards string `protobuf:"bytes,1,opt,name=staking_rewards,json=stakingRewards,proto3" json:"staking_rewards,omitempty"`
	// community_pool defines the proportion of the minted_denom that is to
	// be allocated to the community pool
	CommunityPool string `protobuf:"bytes,2,opt,name=community_pool,json=communityPool,proto3" json:"community_pool,omitempty"`
	// strategic_reserves defines the proportion of the minted_denom that
	// is to be allocated to the strategic reserves module address
	StrategicReserves string `protobuf:"bytes,3,opt,name=strategic_reserves,json=strategicReserves,proto3" json:"strategic_reserves,omitempty"`
}

func (x *InflationDistribution) Reset() {
	*x = InflationDistribution{}
	if protoimpl.UnsafeEnabled {
		mi := &file_nibiru_inflation_v1_inflation_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *InflationDistribution) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*InflationDistribution) ProtoMessage() {}

// Deprecated: Use InflationDistribution.ProtoReflect.Descriptor instead.
func (*InflationDistribution) Descriptor() ([]byte, []int) {
	return file_nibiru_inflation_v1_inflation_proto_rawDescGZIP(), []int{0}
}

func (x *InflationDistribution) GetStakingRewards() string {
	if x != nil {
		return x.StakingRewards
	}
	return ""
}

func (x *InflationDistribution) GetCommunityPool() string {
	if x != nil {
		return x.CommunityPool
	}
	return ""
}

func (x *InflationDistribution) GetStrategicReserves() string {
	if x != nil {
		return x.StrategicReserves
	}
	return ""
}

var File_nibiru_inflation_v1_inflation_proto protoreflect.FileDescriptor

var file_nibiru_inflation_v1_inflation_proto_rawDesc = []byte{
	0x0a, 0x23, 0x6e, 0x69, 0x62, 0x69, 0x72, 0x75, 0x2f, 0x69, 0x6e, 0x66, 0x6c, 0x61, 0x74, 0x69,
	0x6f, 0x6e, 0x2f, 0x76, 0x31, 0x2f, 0x69, 0x6e, 0x66, 0x6c, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x13, 0x6e, 0x69, 0x62, 0x69, 0x72, 0x75, 0x2e, 0x69, 0x6e,
	0x66, 0x6c, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x76, 0x31, 0x1a, 0x14, 0x67, 0x6f, 0x67, 0x6f,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x67, 0x6f, 0x67, 0x6f, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x1a, 0x19, 0x63, 0x6f, 0x73, 0x6d, 0x6f, 0x73, 0x5f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x63,
	0x6f, 0x73, 0x6d, 0x6f, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xaf, 0x02, 0x0a, 0x15,
	0x49, 0x6e, 0x66, 0x6c, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x44, 0x69, 0x73, 0x74, 0x72, 0x69, 0x62,
	0x75, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x5a, 0x0a, 0x0f, 0x73, 0x74, 0x61, 0x6b, 0x69, 0x6e, 0x67,
	0x5f, 0x72, 0x65, 0x77, 0x61, 0x72, 0x64, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x42, 0x31,
	0xc8, 0xde, 0x1f, 0x00, 0xda, 0xde, 0x1f, 0x1b, 0x63, 0x6f, 0x73, 0x6d, 0x6f, 0x73, 0x73, 0x64,
	0x6b, 0x2e, 0x69, 0x6f, 0x2f, 0x6d, 0x61, 0x74, 0x68, 0x2e, 0x4c, 0x65, 0x67, 0x61, 0x63, 0x79,
	0x44, 0x65, 0x63, 0xd2, 0xb4, 0x2d, 0x0a, 0x63, 0x6f, 0x73, 0x6d, 0x6f, 0x73, 0x2e, 0x44, 0x65,
	0x63, 0x52, 0x0e, 0x73, 0x74, 0x61, 0x6b, 0x69, 0x6e, 0x67, 0x52, 0x65, 0x77, 0x61, 0x72, 0x64,
	0x73, 0x12, 0x58, 0x0a, 0x0e, 0x63, 0x6f, 0x6d, 0x6d, 0x75, 0x6e, 0x69, 0x74, 0x79, 0x5f, 0x70,
	0x6f, 0x6f, 0x6c, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x42, 0x31, 0xc8, 0xde, 0x1f, 0x00, 0xda,
	0xde, 0x1f, 0x1b, 0x63, 0x6f, 0x73, 0x6d, 0x6f, 0x73, 0x73, 0x64, 0x6b, 0x2e, 0x69, 0x6f, 0x2f,
	0x6d, 0x61, 0x74, 0x68, 0x2e, 0x4c, 0x65, 0x67, 0x61, 0x63, 0x79, 0x44, 0x65, 0x63, 0xd2, 0xb4,
	0x2d, 0x0a, 0x63, 0x6f, 0x73, 0x6d, 0x6f, 0x73, 0x2e, 0x44, 0x65, 0x63, 0x52, 0x0d, 0x63, 0x6f,
	0x6d, 0x6d, 0x75, 0x6e, 0x69, 0x74, 0x79, 0x50, 0x6f, 0x6f, 0x6c, 0x12, 0x60, 0x0a, 0x12, 0x73,
	0x74, 0x72, 0x61, 0x74, 0x65, 0x67, 0x69, 0x63, 0x5f, 0x72, 0x65, 0x73, 0x65, 0x72, 0x76, 0x65,
	0x73, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x42, 0x31, 0xc8, 0xde, 0x1f, 0x00, 0xda, 0xde, 0x1f,
	0x1b, 0x63, 0x6f, 0x73, 0x6d, 0x6f, 0x73, 0x73, 0x64, 0x6b, 0x2e, 0x69, 0x6f, 0x2f, 0x6d, 0x61,
	0x74, 0x68, 0x2e, 0x4c, 0x65, 0x67, 0x61, 0x63, 0x79, 0x44, 0x65, 0x63, 0xd2, 0xb4, 0x2d, 0x0a,
	0x63, 0x6f, 0x73, 0x6d, 0x6f, 0x73, 0x2e, 0x44, 0x65, 0x63, 0x52, 0x11, 0x73, 0x74, 0x72, 0x61,
	0x74, 0x65, 0x67, 0x69, 0x63, 0x52, 0x65, 0x73, 0x65, 0x72, 0x76, 0x65, 0x73, 0x42, 0xc9, 0x01,
	0x0a, 0x17, 0x63, 0x6f, 0x6d, 0x2e, 0x6e, 0x69, 0x62, 0x69, 0x72, 0x75, 0x2e, 0x69, 0x6e, 0x66,
	0x6c, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x76, 0x31, 0x42, 0x0e, 0x49, 0x6e, 0x66, 0x6c, 0x61,
	0x74, 0x69, 0x6f, 0x6e, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x50, 0x01, 0x5a, 0x30, 0x63, 0x6f, 0x73,
	0x6d, 0x6f, 0x73, 0x73, 0x64, 0x6b, 0x2e, 0x69, 0x6f, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x6e, 0x69,
	0x62, 0x69, 0x72, 0x75, 0x2f, 0x69, 0x6e, 0x66, 0x6c, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2f, 0x76,
	0x31, 0x3b, 0x69, 0x6e, 0x66, 0x6c, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x76, 0x31, 0xa2, 0x02, 0x03,
	0x4e, 0x49, 0x58, 0xaa, 0x02, 0x13, 0x4e, 0x69, 0x62, 0x69, 0x72, 0x75, 0x2e, 0x49, 0x6e, 0x66,
	0x6c, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x56, 0x31, 0xca, 0x02, 0x13, 0x4e, 0x69, 0x62, 0x69,
	0x72, 0x75, 0x5c, 0x49, 0x6e, 0x66, 0x6c, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x5c, 0x56, 0x31, 0xe2,
	0x02, 0x1f, 0x4e, 0x69, 0x62, 0x69, 0x72, 0x75, 0x5c, 0x49, 0x6e, 0x66, 0x6c, 0x61, 0x74, 0x69,
	0x6f, 0x6e, 0x5c, 0x56, 0x31, 0x5c, 0x47, 0x50, 0x42, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74,
	0x61, 0xea, 0x02, 0x15, 0x4e, 0x69, 0x62, 0x69, 0x72, 0x75, 0x3a, 0x3a, 0x49, 0x6e, 0x66, 0x6c,
	0x61, 0x74, 0x69, 0x6f, 0x6e, 0x3a, 0x3a, 0x56, 0x31, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x33,
}

var (
	file_nibiru_inflation_v1_inflation_proto_rawDescOnce sync.Once
	file_nibiru_inflation_v1_inflation_proto_rawDescData = file_nibiru_inflation_v1_inflation_proto_rawDesc
)

func file_nibiru_inflation_v1_inflation_proto_rawDescGZIP() []byte {
	file_nibiru_inflation_v1_inflation_proto_rawDescOnce.Do(func() {
		file_nibiru_inflation_v1_inflation_proto_rawDescData = protoimpl.X.CompressGZIP(file_nibiru_inflation_v1_inflation_proto_rawDescData)
	})
	return file_nibiru_inflation_v1_inflation_proto_rawDescData
}

var file_nibiru_inflation_v1_inflation_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_nibiru_inflation_v1_inflation_proto_goTypes = []interface{}{
	(*InflationDistribution)(nil), // 0: nibiru.inflation.v1.InflationDistribution
}
var file_nibiru_inflation_v1_inflation_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_nibiru_inflation_v1_inflation_proto_init() }
func file_nibiru_inflation_v1_inflation_proto_init() {
	if File_nibiru_inflation_v1_inflation_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_nibiru_inflation_v1_inflation_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*InflationDistribution); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_nibiru_inflation_v1_inflation_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_nibiru_inflation_v1_inflation_proto_goTypes,
		DependencyIndexes: file_nibiru_inflation_v1_inflation_proto_depIdxs,
		MessageInfos:      file_nibiru_inflation_v1_inflation_proto_msgTypes,
	}.Build()
	File_nibiru_inflation_v1_inflation_proto = out.File
	file_nibiru_inflation_v1_inflation_proto_rawDesc = nil
	file_nibiru_inflation_v1_inflation_proto_goTypes = nil
	file_nibiru_inflation_v1_inflation_proto_depIdxs = nil
}
