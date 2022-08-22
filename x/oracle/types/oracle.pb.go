// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: oracle/v1beta1/oracle.proto

package types

import (
	fmt "fmt"
	_ "github.com/cosmos/cosmos-sdk/types"
	github_com_cosmos_cosmos_sdk_types "github.com/cosmos/cosmos-sdk/types"
	_ "github.com/gogo/protobuf/gogoproto"
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

// Params defines the parameters for the oracle module.
type Params struct {
	VotePeriod        uint64                                 `protobuf:"varint,1,opt,name=vote_period,json=votePeriod,proto3" json:"vote_period,omitempty" yaml:"vote_period"`
	VoteThreshold     github_com_cosmos_cosmos_sdk_types.Dec `protobuf:"bytes,2,opt,name=vote_threshold,json=voteThreshold,proto3,customtype=github.com/cosmos/cosmos-sdk/types.Dec" json:"vote_threshold" yaml:"vote_threshold"`
	RewardBand        github_com_cosmos_cosmos_sdk_types.Dec `protobuf:"bytes,3,opt,name=reward_band,json=rewardBand,proto3,customtype=github.com/cosmos/cosmos-sdk/types.Dec" json:"reward_band" yaml:"reward_band"`
	Whitelist         PairList                               `protobuf:"bytes,4,rep,name=whitelist,proto3,castrepeated=PairList" json:"whitelist" yaml:"whitelist"`
	SlashFraction     github_com_cosmos_cosmos_sdk_types.Dec `protobuf:"bytes,5,opt,name=slash_fraction,json=slashFraction,proto3,customtype=github.com/cosmos/cosmos-sdk/types.Dec" json:"slash_fraction" yaml:"slash_fraction"`
	SlashWindow       uint64                                 `protobuf:"varint,6,opt,name=slash_window,json=slashWindow,proto3" json:"slash_window,omitempty" yaml:"slash_window"`
	MinValidPerWindow github_com_cosmos_cosmos_sdk_types.Dec `protobuf:"bytes,7,opt,name=min_valid_per_window,json=minValidPerWindow,proto3,customtype=github.com/cosmos/cosmos-sdk/types.Dec" json:"min_valid_per_window" yaml:"min_valid_per_window"`
}

func (m *Params) Reset()      { *m = Params{} }
func (*Params) ProtoMessage() {}
func (*Params) Descriptor() ([]byte, []int) {
	return fileDescriptor_2784fd4b0e83b02f, []int{0}
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

func (m *Params) GetVotePeriod() uint64 {
	if m != nil {
		return m.VotePeriod
	}
	return 0
}

func (m *Params) GetWhitelist() PairList {
	if m != nil {
		return m.Whitelist
	}
	return nil
}

func (m *Params) GetSlashWindow() uint64 {
	if m != nil {
		return m.SlashWindow
	}
	return 0
}

// Pair is the object that holds configuration of each pair.
type Pair struct {
	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty" yaml:"name"`
}

func (m *Pair) Reset()      { *m = Pair{} }
func (*Pair) ProtoMessage() {}
func (*Pair) Descriptor() ([]byte, []int) {
	return fileDescriptor_2784fd4b0e83b02f, []int{1}
}
func (m *Pair) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Pair) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Pair.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Pair) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Pair.Merge(m, src)
}
func (m *Pair) XXX_Size() int {
	return m.Size()
}
func (m *Pair) XXX_DiscardUnknown() {
	xxx_messageInfo_Pair.DiscardUnknown(m)
}

var xxx_messageInfo_Pair proto.InternalMessageInfo

// struct for aggregate prevoting on the ExchangeRateVote.
// The purpose of aggregate prevote is to hide vote exchange rates with hash
// which is formatted as hex string in SHA256("{salt}:({pair},{exchange_rate})|...|({pair},{exchange_rate}):{voter}")
type AggregateExchangeRatePrevote struct {
	Hash        string `protobuf:"bytes,1,opt,name=hash,proto3" json:"hash,omitempty" yaml:"hash"`
	Voter       string `protobuf:"bytes,2,opt,name=voter,proto3" json:"voter,omitempty" yaml:"voter"`
	SubmitBlock uint64 `protobuf:"varint,3,opt,name=submit_block,json=submitBlock,proto3" json:"submit_block,omitempty" yaml:"submit_block"`
}

func (m *AggregateExchangeRatePrevote) Reset()      { *m = AggregateExchangeRatePrevote{} }
func (*AggregateExchangeRatePrevote) ProtoMessage() {}
func (*AggregateExchangeRatePrevote) Descriptor() ([]byte, []int) {
	return fileDescriptor_2784fd4b0e83b02f, []int{2}
}
func (m *AggregateExchangeRatePrevote) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *AggregateExchangeRatePrevote) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_AggregateExchangeRatePrevote.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *AggregateExchangeRatePrevote) XXX_Merge(src proto.Message) {
	xxx_messageInfo_AggregateExchangeRatePrevote.Merge(m, src)
}
func (m *AggregateExchangeRatePrevote) XXX_Size() int {
	return m.Size()
}
func (m *AggregateExchangeRatePrevote) XXX_DiscardUnknown() {
	xxx_messageInfo_AggregateExchangeRatePrevote.DiscardUnknown(m)
}

var xxx_messageInfo_AggregateExchangeRatePrevote proto.InternalMessageInfo

// MsgAggregateExchangeRateVote - struct for voting on
// the exchange rates different assets.
type AggregateExchangeRateVote struct {
	ExchangeRateTuples ExchangeRateTuples `protobuf:"bytes,1,rep,name=exchange_rate_tuples,json=exchangeRateTuples,proto3,castrepeated=ExchangeRateTuples" json:"exchange_rate_tuples" yaml:"exchange_rate_tuples"`
	Voter              string             `protobuf:"bytes,2,opt,name=voter,proto3" json:"voter,omitempty" yaml:"voter"`
}

func (m *AggregateExchangeRateVote) Reset()      { *m = AggregateExchangeRateVote{} }
func (*AggregateExchangeRateVote) ProtoMessage() {}
func (*AggregateExchangeRateVote) Descriptor() ([]byte, []int) {
	return fileDescriptor_2784fd4b0e83b02f, []int{3}
}
func (m *AggregateExchangeRateVote) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *AggregateExchangeRateVote) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_AggregateExchangeRateVote.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *AggregateExchangeRateVote) XXX_Merge(src proto.Message) {
	xxx_messageInfo_AggregateExchangeRateVote.Merge(m, src)
}
func (m *AggregateExchangeRateVote) XXX_Size() int {
	return m.Size()
}
func (m *AggregateExchangeRateVote) XXX_DiscardUnknown() {
	xxx_messageInfo_AggregateExchangeRateVote.DiscardUnknown(m)
}

var xxx_messageInfo_AggregateExchangeRateVote proto.InternalMessageInfo

// ExchangeRateTuple - struct to store interpreted exchange rates data to store
type ExchangeRateTuple struct {
	Pair         string                                 `protobuf:"bytes,1,opt,name=pair,proto3" json:"pair,omitempty" yaml:"pair"`
	ExchangeRate github_com_cosmos_cosmos_sdk_types.Dec `protobuf:"bytes,2,opt,name=exchange_rate,json=exchangeRate,proto3,customtype=github.com/cosmos/cosmos-sdk/types.Dec" json:"exchange_rate" yaml:"exchange_rate"`
}

func (m *ExchangeRateTuple) Reset()      { *m = ExchangeRateTuple{} }
func (*ExchangeRateTuple) ProtoMessage() {}
func (*ExchangeRateTuple) Descriptor() ([]byte, []int) {
	return fileDescriptor_2784fd4b0e83b02f, []int{4}
}
func (m *ExchangeRateTuple) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *ExchangeRateTuple) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_ExchangeRateTuple.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *ExchangeRateTuple) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ExchangeRateTuple.Merge(m, src)
}
func (m *ExchangeRateTuple) XXX_Size() int {
	return m.Size()
}
func (m *ExchangeRateTuple) XXX_DiscardUnknown() {
	xxx_messageInfo_ExchangeRateTuple.DiscardUnknown(m)
}

var xxx_messageInfo_ExchangeRateTuple proto.InternalMessageInfo

func init() {
	proto.RegisterType((*Params)(nil), "nibiru.oracle.v1beta1.Params")
	proto.RegisterType((*Pair)(nil), "nibiru.oracle.v1beta1.Pair")
	proto.RegisterType((*AggregateExchangeRatePrevote)(nil), "nibiru.oracle.v1beta1.AggregateExchangeRatePrevote")
	proto.RegisterType((*AggregateExchangeRateVote)(nil), "nibiru.oracle.v1beta1.AggregateExchangeRateVote")
	proto.RegisterType((*ExchangeRateTuple)(nil), "nibiru.oracle.v1beta1.ExchangeRateTuple")
}

func init() { proto.RegisterFile("oracle/v1beta1/oracle.proto", fileDescriptor_2784fd4b0e83b02f) }

var fileDescriptor_2784fd4b0e83b02f = []byte{
	// 699 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xa4, 0x54, 0xbf, 0x6f, 0xd3, 0x4c,
	0x18, 0x8e, 0xbf, 0xa6, 0xfd, 0xda, 0x4b, 0xfa, 0x7d, 0xad, 0xbf, 0xf4, 0x23, 0xb4, 0x28, 0xae,
	0x0e, 0x54, 0x65, 0x80, 0x58, 0x85, 0x01, 0x35, 0x1b, 0xa6, 0x94, 0x05, 0x50, 0x74, 0xaa, 0x8a,
	0xc4, 0x12, 0x9d, 0xed, 0xc3, 0x3e, 0xd5, 0xf6, 0x45, 0x77, 0x4e, 0xd3, 0x2e, 0xcc, 0x8c, 0x2c,
	0x48, 0x8c, 0x9d, 0x59, 0x18, 0x10, 0xff, 0x43, 0xc7, 0x8e, 0x88, 0xc1, 0xa0, 0x76, 0x81, 0x35,
	0x7f, 0x01, 0xba, 0x3b, 0xb7, 0x75, 0x9a, 0x20, 0x51, 0x31, 0xb5, 0xcf, 0xfb, 0xe3, 0x79, 0xdf,
	0x7b, 0xde, 0x27, 0x06, 0x2b, 0x8c, 0x63, 0x2f, 0x22, 0xf6, 0xde, 0xba, 0x4b, 0x52, 0xbc, 0x6e,
	0x6b, 0xd8, 0xea, 0x71, 0x96, 0x32, 0x73, 0x29, 0xa1, 0x2e, 0xe5, 0xfd, 0x56, 0x1e, 0xcc, 0x6b,
	0x96, 0x6b, 0x01, 0x0b, 0x98, 0xaa, 0xb0, 0xe5, 0x7f, 0xba, 0x78, 0xb9, 0xe1, 0x31, 0x11, 0x33,
	0x61, 0xbb, 0x58, 0x5c, 0xd0, 0x79, 0x8c, 0x26, 0x3a, 0x0f, 0x3f, 0x4c, 0x83, 0x99, 0x0e, 0xe6,
	0x38, 0x16, 0xe6, 0x7d, 0x50, 0xd9, 0x63, 0x29, 0xe9, 0xf6, 0x08, 0xa7, 0xcc, 0xaf, 0x1b, 0xab,
	0x46, 0xb3, 0xec, 0xfc, 0x3f, 0xcc, 0x2c, 0xf3, 0x00, 0xc7, 0x51, 0x1b, 0x16, 0x92, 0x10, 0x01,
	0x89, 0x3a, 0x0a, 0x98, 0x09, 0xf8, 0x47, 0xe5, 0xd2, 0x90, 0x13, 0x11, 0xb2, 0xc8, 0xaf, 0xff,
	0xb5, 0x6a, 0x34, 0xe7, 0x9c, 0xc7, 0x47, 0x99, 0x55, 0xfa, 0x92, 0x59, 0x6b, 0x01, 0x4d, 0xc3,
	0xbe, 0xdb, 0xf2, 0x58, 0x6c, 0xe7, 0xeb, 0xe8, 0x3f, 0x77, 0x84, 0xbf, 0x6b, 0xa7, 0x07, 0x3d,
	0x22, 0x5a, 0x9b, 0xc4, 0x1b, 0x66, 0xd6, 0x52, 0x61, 0xd2, 0x39, 0x1b, 0x44, 0xf3, 0x32, 0xb0,
	0x7d, 0x86, 0x4d, 0x02, 0x2a, 0x9c, 0x0c, 0x30, 0xf7, 0xbb, 0x2e, 0x4e, 0xfc, 0xfa, 0x94, 0x1a,
	0xb6, 0x79, 0xe5, 0x61, 0xf9, 0xb3, 0x0a, 0x54, 0x10, 0x01, 0x8d, 0x1c, 0x9c, 0xf8, 0x66, 0x17,
	0xcc, 0x0d, 0x42, 0x9a, 0x92, 0x88, 0x8a, 0xb4, 0x5e, 0x5e, 0x9d, 0x6a, 0x56, 0xee, 0xae, 0xb4,
	0x26, 0x6a, 0xdf, 0xea, 0x60, 0xca, 0x9d, 0x5b, 0x72, 0x83, 0x61, 0x66, 0x2d, 0x68, 0xde, 0xf3,
	0x5e, 0xf8, 0xfe, 0xab, 0x35, 0x2b, 0x2b, 0x9e, 0x50, 0x91, 0xa2, 0x0b, 0x4e, 0xa9, 0x9b, 0x88,
	0xb0, 0x08, 0xbb, 0x2f, 0x39, 0xf6, 0x52, 0xca, 0x92, 0xfa, 0xf4, 0x9f, 0xe9, 0x36, 0xca, 0x06,
	0xd1, 0xbc, 0x0a, 0x6c, 0xe5, 0xd8, 0x6c, 0x83, 0xaa, 0xae, 0x18, 0xd0, 0xc4, 0x67, 0x83, 0xfa,
	0x8c, 0xba, 0xf0, 0xb5, 0x61, 0x66, 0xfd, 0x57, 0xec, 0xd7, 0x59, 0x88, 0x2a, 0x0a, 0x3e, 0x57,
	0xc8, 0x7c, 0x05, 0x6a, 0x31, 0x4d, 0xba, 0x7b, 0x38, 0xa2, 0xbe, 0x34, 0xc1, 0x19, 0xc7, 0xdf,
	0x6a, 0xe3, 0xa7, 0x57, 0xde, 0x78, 0x45, 0x4f, 0x9c, 0xc4, 0x09, 0xd1, 0x62, 0x4c, 0x93, 0x1d,
	0x19, 0xed, 0x10, 0xae, 0xe7, 0xb7, 0x67, 0xdf, 0x1d, 0x5a, 0xa5, 0xef, 0x87, 0x96, 0x01, 0x37,
	0x40, 0x59, 0x8a, 0x69, 0xde, 0x04, 0xe5, 0x04, 0xc7, 0x44, 0xf9, 0x74, 0xce, 0xf9, 0x77, 0x98,
	0x59, 0x15, 0xcd, 0x29, 0xa3, 0x10, 0xa9, 0x64, 0xbb, 0xfa, 0xfa, 0xd0, 0x2a, 0xe5, 0xad, 0x25,
	0xf8, 0xc9, 0x00, 0x37, 0x1e, 0x04, 0x01, 0x27, 0x01, 0x4e, 0xc9, 0xa3, 0x7d, 0x2f, 0xc4, 0x49,
	0x40, 0x10, 0x4e, 0x49, 0x87, 0x13, 0x69, 0x31, 0xc9, 0x19, 0x62, 0x11, 0x8e, 0x73, 0xca, 0x28,
	0x44, 0x2a, 0x69, 0xae, 0x81, 0x69, 0x59, 0xcc, 0x73, 0x97, 0x2f, 0x0c, 0x33, 0xab, 0x7a, 0xe1,
	0x5b, 0x0e, 0x91, 0x4e, 0x2b, 0xb9, 0xfb, 0x6e, 0x4c, 0xd3, 0xae, 0x1b, 0x31, 0x6f, 0x57, 0xf9,
	0x74, 0x54, 0xee, 0x42, 0x56, 0xca, 0xad, 0xa0, 0x23, 0xd1, 0xa5, 0xbd, 0x7f, 0x18, 0xe0, 0xfa,
	0xc4, 0xbd, 0x77, 0xe4, 0xd2, 0x6f, 0x0d, 0x50, 0x23, 0x79, 0xb0, 0xcb, 0xb1, 0xfc, 0xe9, 0xf4,
	0x7b, 0x11, 0x11, 0x75, 0x43, 0x79, 0xb6, 0xf9, 0x0b, 0xcf, 0x16, 0x79, 0xb6, 0x65, 0x83, 0xb3,
	0x91, 0x1b, 0x38, 0xbf, 0xcd, 0x24, 0x4e, 0xe9, 0x65, 0x73, 0xac, 0x53, 0x20, 0x93, 0x8c, 0xc5,
	0x7e, 0x57, 0xa7, 0x4b, 0x6f, 0xfd, 0x68, 0x80, 0xc5, 0xb1, 0x01, 0xf2, 0x30, 0x3d, 0x4c, 0xf9,
	0xf8, 0x61, 0x64, 0x14, 0x22, 0x95, 0x34, 0x77, 0xc1, 0xfc, 0xc8, 0xce, 0xf9, 0xe0, 0xad, 0x2b,
	0x9b, 0xb3, 0x36, 0x41, 0x00, 0x88, 0xaa, 0xc5, 0x37, 0x8e, 0x6e, 0xed, 0x6c, 0x1d, 0x9d, 0x34,
	0x8c, 0xe3, 0x93, 0x86, 0xf1, 0xed, 0xa4, 0x61, 0xbc, 0x39, 0x6d, 0x94, 0x8e, 0x4f, 0x1b, 0xa5,
	0xcf, 0xa7, 0x8d, 0xd2, 0x8b, 0xdb, 0x85, 0xa9, 0xcf, 0xd4, 0x21, 0x1e, 0x86, 0x98, 0x26, 0xb6,
	0x3e, 0x8a, 0xbd, 0x9f, 0x7f, 0xdb, 0xf5, 0x7c, 0x77, 0x46, 0x7d, 0x95, 0xef, 0xfd, 0x0c, 0x00,
	0x00, 0xff, 0xff, 0x80, 0x7d, 0x31, 0x24, 0x01, 0x06, 0x00, 0x00,
}

func (this *Params) Equal(that interface{}) bool {
	if that == nil {
		return this == nil
	}

	that1, ok := that.(*Params)
	if !ok {
		that2, ok := that.(Params)
		if ok {
			that1 = &that2
		} else {
			return false
		}
	}
	if that1 == nil {
		return this == nil
	} else if this == nil {
		return false
	}
	if this.VotePeriod != that1.VotePeriod {
		return false
	}
	if !this.VoteThreshold.Equal(that1.VoteThreshold) {
		return false
	}
	if !this.RewardBand.Equal(that1.RewardBand) {
		return false
	}
	if len(this.Whitelist) != len(that1.Whitelist) {
		return false
	}
	for i := range this.Whitelist {
		if !this.Whitelist[i].Equal(&that1.Whitelist[i]) {
			return false
		}
	}
	if !this.SlashFraction.Equal(that1.SlashFraction) {
		return false
	}
	if this.SlashWindow != that1.SlashWindow {
		return false
	}
	if !this.MinValidPerWindow.Equal(that1.MinValidPerWindow) {
		return false
	}
	return true
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
	{
		size := m.MinValidPerWindow.Size()
		i -= size
		if _, err := m.MinValidPerWindow.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintOracle(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x3a
	if m.SlashWindow != 0 {
		i = encodeVarintOracle(dAtA, i, uint64(m.SlashWindow))
		i--
		dAtA[i] = 0x30
	}
	{
		size := m.SlashFraction.Size()
		i -= size
		if _, err := m.SlashFraction.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintOracle(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x2a
	if len(m.Whitelist) > 0 {
		for iNdEx := len(m.Whitelist) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.Whitelist[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintOracle(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x22
		}
	}
	{
		size := m.RewardBand.Size()
		i -= size
		if _, err := m.RewardBand.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintOracle(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x1a
	{
		size := m.VoteThreshold.Size()
		i -= size
		if _, err := m.VoteThreshold.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintOracle(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x12
	if m.VotePeriod != 0 {
		i = encodeVarintOracle(dAtA, i, uint64(m.VotePeriod))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func (m *Pair) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Pair) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Pair) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Name) > 0 {
		i -= len(m.Name)
		copy(dAtA[i:], m.Name)
		i = encodeVarintOracle(dAtA, i, uint64(len(m.Name)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *AggregateExchangeRatePrevote) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *AggregateExchangeRatePrevote) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *AggregateExchangeRatePrevote) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.SubmitBlock != 0 {
		i = encodeVarintOracle(dAtA, i, uint64(m.SubmitBlock))
		i--
		dAtA[i] = 0x18
	}
	if len(m.Voter) > 0 {
		i -= len(m.Voter)
		copy(dAtA[i:], m.Voter)
		i = encodeVarintOracle(dAtA, i, uint64(len(m.Voter)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.Hash) > 0 {
		i -= len(m.Hash)
		copy(dAtA[i:], m.Hash)
		i = encodeVarintOracle(dAtA, i, uint64(len(m.Hash)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *AggregateExchangeRateVote) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *AggregateExchangeRateVote) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *AggregateExchangeRateVote) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Voter) > 0 {
		i -= len(m.Voter)
		copy(dAtA[i:], m.Voter)
		i = encodeVarintOracle(dAtA, i, uint64(len(m.Voter)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.ExchangeRateTuples) > 0 {
		for iNdEx := len(m.ExchangeRateTuples) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.ExchangeRateTuples[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintOracle(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0xa
		}
	}
	return len(dAtA) - i, nil
}

func (m *ExchangeRateTuple) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *ExchangeRateTuple) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *ExchangeRateTuple) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	{
		size := m.ExchangeRate.Size()
		i -= size
		if _, err := m.ExchangeRate.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintOracle(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x12
	if len(m.Pair) > 0 {
		i -= len(m.Pair)
		copy(dAtA[i:], m.Pair)
		i = encodeVarintOracle(dAtA, i, uint64(len(m.Pair)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintOracle(dAtA []byte, offset int, v uint64) int {
	offset -= sovOracle(v)
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
	if m.VotePeriod != 0 {
		n += 1 + sovOracle(uint64(m.VotePeriod))
	}
	l = m.VoteThreshold.Size()
	n += 1 + l + sovOracle(uint64(l))
	l = m.RewardBand.Size()
	n += 1 + l + sovOracle(uint64(l))
	if len(m.Whitelist) > 0 {
		for _, e := range m.Whitelist {
			l = e.Size()
			n += 1 + l + sovOracle(uint64(l))
		}
	}
	l = m.SlashFraction.Size()
	n += 1 + l + sovOracle(uint64(l))
	if m.SlashWindow != 0 {
		n += 1 + sovOracle(uint64(m.SlashWindow))
	}
	l = m.MinValidPerWindow.Size()
	n += 1 + l + sovOracle(uint64(l))
	return n
}

func (m *Pair) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Name)
	if l > 0 {
		n += 1 + l + sovOracle(uint64(l))
	}
	return n
}

func (m *AggregateExchangeRatePrevote) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Hash)
	if l > 0 {
		n += 1 + l + sovOracle(uint64(l))
	}
	l = len(m.Voter)
	if l > 0 {
		n += 1 + l + sovOracle(uint64(l))
	}
	if m.SubmitBlock != 0 {
		n += 1 + sovOracle(uint64(m.SubmitBlock))
	}
	return n
}

func (m *AggregateExchangeRateVote) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if len(m.ExchangeRateTuples) > 0 {
		for _, e := range m.ExchangeRateTuples {
			l = e.Size()
			n += 1 + l + sovOracle(uint64(l))
		}
	}
	l = len(m.Voter)
	if l > 0 {
		n += 1 + l + sovOracle(uint64(l))
	}
	return n
}

func (m *ExchangeRateTuple) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Pair)
	if l > 0 {
		n += 1 + l + sovOracle(uint64(l))
	}
	l = m.ExchangeRate.Size()
	n += 1 + l + sovOracle(uint64(l))
	return n
}

func sovOracle(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozOracle(x uint64) (n int) {
	return sovOracle(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *Params) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowOracle
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
				return fmt.Errorf("proto: wrong wireType = %d for field VotePeriod", wireType)
			}
			m.VotePeriod = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowOracle
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.VotePeriod |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field VoteThreshold", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowOracle
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
				return ErrInvalidLengthOracle
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthOracle
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.VoteThreshold.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field RewardBand", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowOracle
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
				return ErrInvalidLengthOracle
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthOracle
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.RewardBand.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Whitelist", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowOracle
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
				return ErrInvalidLengthOracle
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthOracle
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Whitelist = append(m.Whitelist, Pair{})
			if err := m.Whitelist[len(m.Whitelist)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 5:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field SlashFraction", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowOracle
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
				return ErrInvalidLengthOracle
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthOracle
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.SlashFraction.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 6:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field SlashWindow", wireType)
			}
			m.SlashWindow = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowOracle
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.SlashWindow |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 7:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field MinValidPerWindow", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowOracle
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
				return ErrInvalidLengthOracle
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthOracle
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.MinValidPerWindow.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipOracle(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthOracle
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
func (m *Pair) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowOracle
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
			return fmt.Errorf("proto: Pair: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Pair: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Name", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowOracle
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
				return ErrInvalidLengthOracle
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthOracle
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Name = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipOracle(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthOracle
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
func (m *AggregateExchangeRatePrevote) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowOracle
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
			return fmt.Errorf("proto: AggregateExchangeRatePrevote: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: AggregateExchangeRatePrevote: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Hash", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowOracle
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
				return ErrInvalidLengthOracle
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthOracle
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Hash = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Voter", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowOracle
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
				return ErrInvalidLengthOracle
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthOracle
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Voter = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field SubmitBlock", wireType)
			}
			m.SubmitBlock = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowOracle
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.SubmitBlock |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipOracle(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthOracle
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
func (m *AggregateExchangeRateVote) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowOracle
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
			return fmt.Errorf("proto: AggregateExchangeRateVote: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: AggregateExchangeRateVote: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ExchangeRateTuples", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowOracle
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
				return ErrInvalidLengthOracle
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthOracle
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ExchangeRateTuples = append(m.ExchangeRateTuples, ExchangeRateTuple{})
			if err := m.ExchangeRateTuples[len(m.ExchangeRateTuples)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Voter", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowOracle
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
				return ErrInvalidLengthOracle
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthOracle
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Voter = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipOracle(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthOracle
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
func (m *ExchangeRateTuple) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowOracle
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
			return fmt.Errorf("proto: ExchangeRateTuple: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: ExchangeRateTuple: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Pair", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowOracle
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
				return ErrInvalidLengthOracle
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthOracle
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Pair = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ExchangeRate", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowOracle
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
				return ErrInvalidLengthOracle
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthOracle
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.ExchangeRate.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipOracle(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthOracle
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
func skipOracle(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowOracle
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
					return 0, ErrIntOverflowOracle
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
					return 0, ErrIntOverflowOracle
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
				return 0, ErrInvalidLengthOracle
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupOracle
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthOracle
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthOracle        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowOracle          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupOracle = fmt.Errorf("proto: unexpected end of group")
)
