// Code generated by mockery v2.14.0. DO NOT EDIT.

package mocks

import (
	context "context"

	grpc "google.golang.org/grpc"

	mock "github.com/stretchr/testify/mock"

	"github.com/NibiruChain/nibiru/v2/x/evm"
)

// EVMQueryClient is an autogenerated mock type for the EVMQueryClient type
type EVMQueryClient struct {
	mock.Mock
}

// EthAccount provides a mock function with given fields: ctx, in, opts
func (_m *EVMQueryClient) EthAccount(ctx context.Context, in *evm.QueryEthAccountRequest, opts ...grpc.CallOption) (*evm.QueryEthAccountResponse, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, in)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 *evm.QueryEthAccountResponse
	if rf, ok := ret.Get(0).(func(context.Context, *evm.QueryEthAccountRequest, ...grpc.CallOption) *evm.QueryEthAccountResponse); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*evm.QueryEthAccountResponse)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *evm.QueryEthAccountRequest, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, in, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Balance provides a mock function with given fields: ctx, in, opts
func (_m *EVMQueryClient) Balance(ctx context.Context, in *evm.QueryBalanceRequest, opts ...grpc.CallOption) (*evm.QueryBalanceResponse, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, in)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 *evm.QueryBalanceResponse
	if rf, ok := ret.Get(0).(func(context.Context, *evm.QueryBalanceRequest, ...grpc.CallOption) *evm.QueryBalanceResponse); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*evm.QueryBalanceResponse)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *evm.QueryBalanceRequest, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, in, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// BaseFee provides a mock function with given fields: ctx, in, opts
func (_m *EVMQueryClient) BaseFee(ctx context.Context, in *evm.QueryBaseFeeRequest, opts ...grpc.CallOption) (*evm.QueryBaseFeeResponse, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, in)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 *evm.QueryBaseFeeResponse
	if rf, ok := ret.Get(0).(func(context.Context, *evm.QueryBaseFeeRequest, ...grpc.CallOption) *evm.QueryBaseFeeResponse); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*evm.QueryBaseFeeResponse)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *evm.QueryBaseFeeRequest, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, in, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Code provides a mock function with given fields: ctx, in, opts
func (_m *EVMQueryClient) Code(ctx context.Context, in *evm.QueryCodeRequest, opts ...grpc.CallOption) (*evm.QueryCodeResponse, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, in)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 *evm.QueryCodeResponse
	if rf, ok := ret.Get(0).(func(context.Context, *evm.QueryCodeRequest, ...grpc.CallOption) *evm.QueryCodeResponse); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*evm.QueryCodeResponse)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *evm.QueryCodeRequest, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, in, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// EstimateGas provides a mock function with given fields: ctx, in, opts
func (_m *EVMQueryClient) EstimateGas(ctx context.Context, in *evm.EthCallRequest, opts ...grpc.CallOption) (*evm.EstimateGasResponse, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, in)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 *evm.EstimateGasResponse
	if rf, ok := ret.Get(0).(func(context.Context, *evm.EthCallRequest, ...grpc.CallOption) *evm.EstimateGasResponse); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*evm.EstimateGasResponse)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *evm.EthCallRequest, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, in, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// EthCall provides a mock function with given fields: ctx, in, opts
func (_m *EVMQueryClient) EthCall(ctx context.Context, in *evm.EthCallRequest, opts ...grpc.CallOption) (*evm.MsgEthereumTxResponse, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, in)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 *evm.MsgEthereumTxResponse
	if rf, ok := ret.Get(0).(func(context.Context, *evm.EthCallRequest, ...grpc.CallOption) *evm.MsgEthereumTxResponse); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*evm.MsgEthereumTxResponse)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *evm.EthCallRequest, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, in, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Params provides a mock function with given fields: ctx, in, opts
func (_m *EVMQueryClient) Params(ctx context.Context, in *evm.QueryParamsRequest, opts ...grpc.CallOption) (*evm.QueryParamsResponse, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, in)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 *evm.QueryParamsResponse
	if rf, ok := ret.Get(0).(func(context.Context, *evm.QueryParamsRequest, ...grpc.CallOption) *evm.QueryParamsResponse); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*evm.QueryParamsResponse)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *evm.QueryParamsRequest, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, in, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Storage provides a mock function with given fields: ctx, in, opts
func (_m *EVMQueryClient) Storage(ctx context.Context, in *evm.QueryStorageRequest, opts ...grpc.CallOption) (*evm.QueryStorageResponse, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, in)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 *evm.QueryStorageResponse
	if rf, ok := ret.Get(0).(func(context.Context, *evm.QueryStorageRequest, ...grpc.CallOption) *evm.QueryStorageResponse); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*evm.QueryStorageResponse)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *evm.QueryStorageRequest, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, in, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// TraceBlock provides a mock function with given fields: ctx, in, opts
func (_m *EVMQueryClient) TraceBlock(ctx context.Context, in *evm.QueryTraceBlockRequest, opts ...grpc.CallOption) (*evm.QueryTraceBlockResponse, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, in)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 *evm.QueryTraceBlockResponse
	if rf, ok := ret.Get(0).(func(context.Context, *evm.QueryTraceBlockRequest, ...grpc.CallOption) *evm.QueryTraceBlockResponse); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*evm.QueryTraceBlockResponse)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *evm.QueryTraceBlockRequest, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, in, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// TraceTx provides a mock function with given fields: ctx, in, opts
func (_m *EVMQueryClient) TraceTx(ctx context.Context, in *evm.QueryTraceTxRequest, opts ...grpc.CallOption) (*evm.QueryTraceTxResponse, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, in)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 *evm.QueryTraceTxResponse
	if rf, ok := ret.Get(0).(func(context.Context, *evm.QueryTraceTxRequest, ...grpc.CallOption) *evm.QueryTraceTxResponse); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*evm.QueryTraceTxResponse)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *evm.QueryTraceTxRequest, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, in, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ValidatorAccount provides a mock function with given fields: ctx, in, opts
func (_m *EVMQueryClient) ValidatorAccount(ctx context.Context, in *evm.QueryValidatorAccountRequest, opts ...grpc.CallOption) (*evm.QueryValidatorAccountResponse, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, in)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 *evm.QueryValidatorAccountResponse
	if rf, ok := ret.Get(0).(func(context.Context, *evm.QueryValidatorAccountRequest, ...grpc.CallOption) *evm.QueryValidatorAccountResponse); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*evm.QueryValidatorAccountResponse)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *evm.QueryValidatorAccountRequest, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, in, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FunTokenMapping provides a mock function with given fields: ctx, in, opts
func (_m *EVMQueryClient) FunTokenMapping(ctx context.Context, in *evm.QueryFunTokenMappingRequest, opts ...grpc.CallOption) (*evm.QueryFunTokenMappingResponse, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, in)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 *evm.QueryFunTokenMappingResponse
	if rf, ok := ret.Get(0).(func(context.Context, *evm.QueryFunTokenMappingRequest, ...grpc.CallOption) *evm.QueryFunTokenMappingResponse); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*evm.QueryFunTokenMappingResponse)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *evm.QueryFunTokenMappingRequest, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, in, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewEVMQueryClient interface {
	mock.TestingT
	Cleanup(func())
}

// NewEVMQueryClient creates a new instance of EVMQueryClient. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewEVMQueryClient(t mockConstructorTestingTNewEVMQueryClient) *EVMQueryClient {
	mock := &EVMQueryClient{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
