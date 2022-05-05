package mock

import (
	reflect "reflect"

	v1 "github.com/NibiruChain/nibiru/x/perp/types/v1"
	types "github.com/cosmos/cosmos-sdk/types"
	gomock "github.com/golang/mock/gomock"
)

// MockIClearingHouse is a mock of IClearingHouse interface.
type MockIClearingHouse struct {
	ctrl     *gomock.Controller
	recorder *MockIClearingHouseMockRecorder
}

// MockIClearingHouseMockRecorder is the mock recorder for MockIClearingHouse.
type MockIClearingHouseMockRecorder struct {
	mock *MockIClearingHouse
}

// NewMockIClearingHouse creates a new mock instance.
func NewMockIClearingHouse(ctrl *gomock.Controller) *MockIClearingHouse {
	mock := &MockIClearingHouse{ctrl: ctrl}
	mock.recorder = &MockIClearingHouseMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockIClearingHouse) EXPECT() *MockIClearingHouseMockRecorder {
	return m.recorder
}

// ClearPosition mocks base method.
func (m *MockIClearingHouse) ClearPosition(ctx types.Context, vpool v1.IVirtualPool, owner string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ClearPosition", ctx, vpool, owner)
	ret0, _ := ret[0].(error)
	return ret0
}

// ClearPosition indicates an expected call of ClearPosition.
func (mr *MockIClearingHouseMockRecorder) ClearPosition(ctx, vpool, owner interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ClearPosition", reflect.TypeOf((*MockIClearingHouse)(nil).ClearPosition), ctx, vpool, owner)
}

// GetPosition mocks base method.
func (m *MockIClearingHouse) GetPosition(ctx types.Context, vpool v1.IVirtualPool, owner string) (*v1.Position, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetPosition", ctx, vpool, owner)
	ret0, _ := ret[0].(*v1.Position)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetPosition indicates an expected call of GetPosition.
func (mr *MockIClearingHouseMockRecorder) GetPosition(ctx, vpool, owner interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetPosition", reflect.TypeOf((*MockIClearingHouse)(nil).GetPosition), ctx, vpool, owner)
}

// SetPosition mocks base method.
func (m *MockIClearingHouse) SetPosition(ctx types.Context, vpool v1.IVirtualPool, owner string, position *v1.Position) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetPosition", ctx, vpool, owner, position)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetPosition indicates an expected call of SetPosition.
func (mr *MockIClearingHouseMockRecorder) SetPosition(ctx, vpool, owner, position interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetPosition", reflect.TypeOf((*MockIClearingHouse)(nil).SetPosition), ctx, vpool, owner, position)
}

// MockIVirtualPool is a mock of IVirtualPool interface.
type MockIVirtualPool struct {
	ctrl     *gomock.Controller
	recorder *MockIVirtualPoolMockRecorder
}

// MockIVirtualPoolMockRecorder is the mock recorder for MockIVirtualPool.
type MockIVirtualPoolMockRecorder struct {
	mock *MockIVirtualPool
}

// NewMockIVirtualPool creates a new mock instance.
func NewMockIVirtualPool(ctrl *gomock.Controller) *MockIVirtualPool {
	mock := &MockIVirtualPool{ctrl: ctrl}
	mock.recorder = &MockIVirtualPoolMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockIVirtualPool) EXPECT() *MockIVirtualPoolMockRecorder {
	return m.recorder
}

// CalcFee mocks base method.
func (m *MockIVirtualPool) CalcFee(quoteAmt types.Int) (types.Int, types.Int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CalcFee", quoteAmt)
	ret0, _ := ret[0].(types.Int)
	ret1, _ := ret[1].(types.Int)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// CalcFee indicates an expected call of CalcFee.
func (mr *MockIVirtualPoolMockRecorder) CalcFee(quoteAmt interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CalcFee", reflect.TypeOf((*MockIVirtualPool)(nil).CalcFee), quoteAmt)
}

// GetMaxHoldingBaseAsset mocks base method.
func (m *MockIVirtualPool) GetMaxHoldingBaseAsset(ctx types.Context) (types.Int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetMaxHoldingBaseAsset", ctx)
	ret0, _ := ret[0].(types.Int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetMaxHoldingBaseAsset indicates an expected call of GetMaxHoldingBaseAsset.
func (mr *MockIVirtualPoolMockRecorder) GetMaxHoldingBaseAsset(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetMaxHoldingBaseAsset", reflect.TypeOf((*MockIVirtualPool)(nil).GetMaxHoldingBaseAsset), ctx)
}

// GetOpenInterestNotionalCap mocks base method.
func (m *MockIVirtualPool) GetOpenInterestNotionalCap(ctx types.Context) (types.Int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetOpenInterestNotionalCap", ctx)
	ret0, _ := ret[0].(types.Int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetOpenInterestNotionalCap indicates an expected call of GetOpenInterestNotionalCap.
func (mr *MockIVirtualPoolMockRecorder) GetOpenInterestNotionalCap(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetOpenInterestNotionalCap", reflect.TypeOf((*MockIVirtualPool)(nil).GetOpenInterestNotionalCap), ctx)
}

// GetOutputPrice mocks base method.
func (m *MockIVirtualPool) GetOutputPrice(ctx types.Context, dir v1.VirtualPoolDirection, abs types.Int) (types.Int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetOutputPrice", ctx, dir, abs)
	ret0, _ := ret[0].(types.Int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetOutputPrice indicates an expected call of GetOutputPrice.
func (mr *MockIVirtualPoolMockRecorder) GetOutputPrice(ctx, dir, abs interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetOutputPrice", reflect.TypeOf((*MockIVirtualPool)(nil).GetOutputPrice), ctx, dir, abs)
}

// GetOutputTWAP mocks base method.
func (m *MockIVirtualPool) GetOutputTWAP(ctx types.Context, dir v1.VirtualPoolDirection, abs types.Int) (types.Int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetOutputTWAP", ctx, dir, abs)
	ret0, _ := ret[0].(types.Int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetOutputTWAP indicates an expected call of GetOutputTWAP.
func (mr *MockIVirtualPoolMockRecorder) GetOutputTWAP(ctx, dir, abs interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetOutputTWAP", reflect.TypeOf((*MockIVirtualPool)(nil).GetOutputTWAP), ctx, dir, abs)
}

// GetSpotPrice mocks base method.
func (m *MockIVirtualPool) GetSpotPrice(ctx types.Context) (types.Int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSpotPrice", ctx)
	ret0, _ := ret[0].(types.Int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetSpotPrice indicates an expected call of GetSpotPrice.
func (mr *MockIVirtualPoolMockRecorder) GetSpotPrice(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSpotPrice", reflect.TypeOf((*MockIVirtualPool)(nil).GetSpotPrice), ctx)
}

// GetUnderlyingPrice mocks base method.
func (m *MockIVirtualPool) GetUnderlyingPrice(ctx types.Context) (types.Dec, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUnderlyingPrice", ctx)
	ret0, _ := ret[0].(types.Dec)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetUnderlyingPrice indicates an expected call of GetUnderlyingPrice.
func (mr *MockIVirtualPoolMockRecorder) GetUnderlyingPrice(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUnderlyingPrice", reflect.TypeOf((*MockIVirtualPool)(nil).GetUnderlyingPrice), ctx)
}

// Pair mocks base method.
func (m *MockIVirtualPool) Pair() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Pair")
	ret0, _ := ret[0].(string)
	return ret0
}

// Pair indicates an expected call of Pair.
func (mr *MockIVirtualPoolMockRecorder) Pair() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Pair", reflect.TypeOf((*MockIVirtualPool)(nil).Pair))
}

// QuoteTokenDenom mocks base method.
func (m *MockIVirtualPool) QuoteTokenDenom() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "QuoteTokenDenom")
	ret0, _ := ret[0].(string)
	return ret0
}

// QuoteTokenDenom indicates an expected call of QuoteTokenDenom.
func (mr *MockIVirtualPoolMockRecorder) QuoteTokenDenom() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "QuoteTokenDenom", reflect.TypeOf((*MockIVirtualPool)(nil).QuoteTokenDenom))
}

// SwapInput mocks base method.
func (m *MockIVirtualPool) SwapInput(ctx types.Context, ammDir v1.VirtualPoolDirection, inputAmount, minOutputAmount types.Int, canOverFluctuationLimit bool) (types.Int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SwapInput", ctx, ammDir, inputAmount, minOutputAmount, canOverFluctuationLimit)
	ret0, _ := ret[0].(types.Int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SwapInput indicates an expected call of SwapInput.
func (mr *MockIVirtualPoolMockRecorder) SwapInput(ctx, ammDir, inputAmount, minOutputAmount, canOverFluctuationLimit interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SwapInput", reflect.TypeOf((*MockIVirtualPool)(nil).SwapInput), ctx, ammDir, inputAmount, minOutputAmount, canOverFluctuationLimit)
}

// SwapOutput mocks base method.
func (m *MockIVirtualPool) SwapOutput(ctx types.Context, dir v1.VirtualPoolDirection, abs, limit types.Int) (types.Int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SwapOutput", ctx, dir, abs, limit)
	ret0, _ := ret[0].(types.Int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SwapOutput indicates an expected call of SwapOutput.
func (mr *MockIVirtualPoolMockRecorder) SwapOutput(ctx, dir, abs, limit interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SwapOutput", reflect.TypeOf((*MockIVirtualPool)(nil).SwapOutput), ctx, dir, abs, limit)
}
