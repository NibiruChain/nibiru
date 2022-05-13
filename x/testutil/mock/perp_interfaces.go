// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/NibiruChain/nibiru/x/perp/types (interfaces: AccountKeeper,BankKeeper,PriceKeeper,VpoolKeeper)

// Package mock is a generated GoMock package.
package mock

import (
	reflect "reflect"

	common "github.com/NibiruChain/nibiru/x/common"
	types "github.com/NibiruChain/nibiru/x/pricefeed/types"
	types0 "github.com/NibiruChain/nibiru/x/vpool/types"
	types1 "github.com/cosmos/cosmos-sdk/types"
	types2 "github.com/cosmos/cosmos-sdk/x/auth/types"
	gomock "github.com/golang/mock/gomock"
)

// MockAccountKeeper is a mock of AccountKeeper interface.
type MockAccountKeeper struct {
	ctrl     *gomock.Controller
	recorder *MockAccountKeeperMockRecorder
}

// MockAccountKeeperMockRecorder is the mock recorder for MockAccountKeeper.
type MockAccountKeeperMockRecorder struct {
	mock *MockAccountKeeper
}

// NewMockAccountKeeper creates a new mock instance.
func NewMockAccountKeeper(ctrl *gomock.Controller) *MockAccountKeeper {
	mock := &MockAccountKeeper{ctrl: ctrl}
	mock.recorder = &MockAccountKeeperMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockAccountKeeper) EXPECT() *MockAccountKeeperMockRecorder {
	return m.recorder
}

// GetAccount mocks base method.
func (m *MockAccountKeeper) GetAccount(arg0 types1.Context, arg1 types1.AccAddress) types2.AccountI {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAccount", arg0, arg1)
	ret0, _ := ret[0].(types2.AccountI)
	return ret0
}

// GetAccount indicates an expected call of GetAccount.
func (mr *MockAccountKeeperMockRecorder) GetAccount(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAccount", reflect.TypeOf((*MockAccountKeeper)(nil).GetAccount), arg0, arg1)
}

// GetModuleAccount mocks base method.
func (m *MockAccountKeeper) GetModuleAccount(arg0 types1.Context, arg1 string) types2.ModuleAccountI {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetModuleAccount", arg0, arg1)
	ret0, _ := ret[0].(types2.ModuleAccountI)
	return ret0
}

// GetModuleAccount indicates an expected call of GetModuleAccount.
func (mr *MockAccountKeeperMockRecorder) GetModuleAccount(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetModuleAccount", reflect.TypeOf((*MockAccountKeeper)(nil).GetModuleAccount), arg0, arg1)
}

// GetModuleAddress mocks base method.
func (m *MockAccountKeeper) GetModuleAddress(arg0 string) types1.AccAddress {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetModuleAddress", arg0)
	ret0, _ := ret[0].(types1.AccAddress)
	return ret0
}

// GetModuleAddress indicates an expected call of GetModuleAddress.
func (mr *MockAccountKeeperMockRecorder) GetModuleAddress(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetModuleAddress", reflect.TypeOf((*MockAccountKeeper)(nil).GetModuleAddress), arg0)
}

// MockBankKeeper is a mock of BankKeeper interface.
type MockBankKeeper struct {
	ctrl     *gomock.Controller
	recorder *MockBankKeeperMockRecorder
}

// MockBankKeeperMockRecorder is the mock recorder for MockBankKeeper.
type MockBankKeeperMockRecorder struct {
	mock *MockBankKeeper
}

// NewMockBankKeeper creates a new mock instance.
func NewMockBankKeeper(ctrl *gomock.Controller) *MockBankKeeper {
	mock := &MockBankKeeper{ctrl: ctrl}
	mock.recorder = &MockBankKeeperMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockBankKeeper) EXPECT() *MockBankKeeperMockRecorder {
	return m.recorder
}

// BurnCoins mocks base method.
func (m *MockBankKeeper) BurnCoins(arg0 types1.Context, arg1 string, arg2 types1.Coins) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BurnCoins", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// BurnCoins indicates an expected call of BurnCoins.
func (mr *MockBankKeeperMockRecorder) BurnCoins(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BurnCoins", reflect.TypeOf((*MockBankKeeper)(nil).BurnCoins), arg0, arg1, arg2)
}

// GetBalance mocks base method.
func (m *MockBankKeeper) GetBalance(arg0 types1.Context, arg1 types1.AccAddress, arg2 string) types1.Coin {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBalance", arg0, arg1, arg2)
	ret0, _ := ret[0].(types1.Coin)
	return ret0
}

// GetBalance indicates an expected call of GetBalance.
func (mr *MockBankKeeperMockRecorder) GetBalance(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBalance", reflect.TypeOf((*MockBankKeeper)(nil).GetBalance), arg0, arg1, arg2)
}

// MintCoins mocks base method.
func (m *MockBankKeeper) MintCoins(arg0 types1.Context, arg1 string, arg2 types1.Coins) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "MintCoins", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// MintCoins indicates an expected call of MintCoins.
func (mr *MockBankKeeperMockRecorder) MintCoins(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MintCoins", reflect.TypeOf((*MockBankKeeper)(nil).MintCoins), arg0, arg1, arg2)
}

// SendCoinsFromAccountToModule mocks base method.
func (m *MockBankKeeper) SendCoinsFromAccountToModule(arg0 types1.Context, arg1 types1.AccAddress, arg2 string, arg3 types1.Coins) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendCoinsFromAccountToModule", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(error)
	return ret0
}

// SendCoinsFromAccountToModule indicates an expected call of SendCoinsFromAccountToModule.
func (mr *MockBankKeeperMockRecorder) SendCoinsFromAccountToModule(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendCoinsFromAccountToModule", reflect.TypeOf((*MockBankKeeper)(nil).SendCoinsFromAccountToModule), arg0, arg1, arg2, arg3)
}

// SendCoinsFromModuleToAccount mocks base method.
func (m *MockBankKeeper) SendCoinsFromModuleToAccount(arg0 types1.Context, arg1 string, arg2 types1.AccAddress, arg3 types1.Coins) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendCoinsFromModuleToAccount", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(error)
	return ret0
}

// SendCoinsFromModuleToAccount indicates an expected call of SendCoinsFromModuleToAccount.
func (mr *MockBankKeeperMockRecorder) SendCoinsFromModuleToAccount(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendCoinsFromModuleToAccount", reflect.TypeOf((*MockBankKeeper)(nil).SendCoinsFromModuleToAccount), arg0, arg1, arg2, arg3)
}

// SpendableCoins mocks base method.
func (m *MockBankKeeper) SpendableCoins(arg0 types1.Context, arg1 types1.AccAddress) types1.Coins {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SpendableCoins", arg0, arg1)
	ret0, _ := ret[0].(types1.Coins)
	return ret0
}

// SpendableCoins indicates an expected call of SpendableCoins.
func (mr *MockBankKeeperMockRecorder) SpendableCoins(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SpendableCoins", reflect.TypeOf((*MockBankKeeper)(nil).SpendableCoins), arg0, arg1)
}

// MockPriceKeeper is a mock of PriceKeeper interface.
type MockPriceKeeper struct {
	ctrl     *gomock.Controller
	recorder *MockPriceKeeperMockRecorder
}

// MockPriceKeeperMockRecorder is the mock recorder for MockPriceKeeper.
type MockPriceKeeperMockRecorder struct {
	mock *MockPriceKeeper
}

// NewMockPriceKeeper creates a new mock instance.
func NewMockPriceKeeper(ctrl *gomock.Controller) *MockPriceKeeper {
	mock := &MockPriceKeeper{ctrl: ctrl}
	mock.recorder = &MockPriceKeeperMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockPriceKeeper) EXPECT() *MockPriceKeeperMockRecorder {
	return m.recorder
}

// GetCurrentPrice mocks base method.
func (m *MockPriceKeeper) GetCurrentPrice(arg0 types1.Context, arg1, arg2 string) (types.CurrentPrice, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCurrentPrice", arg0, arg1, arg2)
	ret0, _ := ret[0].(types.CurrentPrice)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetCurrentPrice indicates an expected call of GetCurrentPrice.
func (mr *MockPriceKeeperMockRecorder) GetCurrentPrice(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCurrentPrice", reflect.TypeOf((*MockPriceKeeper)(nil).GetCurrentPrice), arg0, arg1, arg2)
}

// GetCurrentPrices mocks base method.
func (m *MockPriceKeeper) GetCurrentPrices(arg0 types1.Context) types.CurrentPrices {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCurrentPrices", arg0)
	ret0, _ := ret[0].(types.CurrentPrices)
	return ret0
}

// GetCurrentPrices indicates an expected call of GetCurrentPrices.
func (mr *MockPriceKeeperMockRecorder) GetCurrentPrices(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCurrentPrices", reflect.TypeOf((*MockPriceKeeper)(nil).GetCurrentPrices), arg0)
}

// GetOracle mocks base method.
func (m *MockPriceKeeper) GetOracle(arg0 types1.Context, arg1 string, arg2 types1.AccAddress) (types1.AccAddress, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetOracle", arg0, arg1, arg2)
	ret0, _ := ret[0].(types1.AccAddress)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetOracle indicates an expected call of GetOracle.
func (mr *MockPriceKeeperMockRecorder) GetOracle(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetOracle", reflect.TypeOf((*MockPriceKeeper)(nil).GetOracle), arg0, arg1, arg2)
}

// GetOracles mocks base method.
func (m *MockPriceKeeper) GetOracles(arg0 types1.Context, arg1 string) ([]types1.AccAddress, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetOracles", arg0, arg1)
	ret0, _ := ret[0].([]types1.AccAddress)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetOracles indicates an expected call of GetOracles.
func (mr *MockPriceKeeperMockRecorder) GetOracles(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetOracles", reflect.TypeOf((*MockPriceKeeper)(nil).GetOracles), arg0, arg1)
}

// GetPair mocks base method.
func (m *MockPriceKeeper) GetPair(arg0 types1.Context, arg1 string) (types.Pair, bool) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetPair", arg0, arg1)
	ret0, _ := ret[0].(types.Pair)
	ret1, _ := ret[1].(bool)
	return ret0, ret1
}

// GetPair indicates an expected call of GetPair.
func (mr *MockPriceKeeperMockRecorder) GetPair(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetPair", reflect.TypeOf((*MockPriceKeeper)(nil).GetPair), arg0, arg1)
}

// GetPairs mocks base method.
func (m *MockPriceKeeper) GetPairs(arg0 types1.Context) types.Pairs {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetPairs", arg0)
	ret0, _ := ret[0].(types.Pairs)
	return ret0
}

// GetPairs indicates an expected call of GetPairs.
func (mr *MockPriceKeeperMockRecorder) GetPairs(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetPairs", reflect.TypeOf((*MockPriceKeeper)(nil).GetPairs), arg0)
}

// GetRawPrices mocks base method.
func (m *MockPriceKeeper) GetRawPrices(arg0 types1.Context, arg1 string) types.PostedPrices {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRawPrices", arg0, arg1)
	ret0, _ := ret[0].(types.PostedPrices)
	return ret0
}

// GetRawPrices indicates an expected call of GetRawPrices.
func (mr *MockPriceKeeperMockRecorder) GetRawPrices(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRawPrices", reflect.TypeOf((*MockPriceKeeper)(nil).GetRawPrices), arg0, arg1)
}

// SetCurrentPrices mocks base method.
func (m *MockPriceKeeper) SetCurrentPrices(arg0 types1.Context, arg1, arg2 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetCurrentPrices", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetCurrentPrices indicates an expected call of SetCurrentPrices.
func (mr *MockPriceKeeperMockRecorder) SetCurrentPrices(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetCurrentPrices", reflect.TypeOf((*MockPriceKeeper)(nil).SetCurrentPrices), arg0, arg1, arg2)
}

// MockVpoolKeeper is a mock of VpoolKeeper interface.
type MockVpoolKeeper struct {
	ctrl     *gomock.Controller
	recorder *MockVpoolKeeperMockRecorder
}

// MockVpoolKeeperMockRecorder is the mock recorder for MockVpoolKeeper.
type MockVpoolKeeperMockRecorder struct {
	mock *MockVpoolKeeper
}

// NewMockVpoolKeeper creates a new mock instance.
func NewMockVpoolKeeper(ctrl *gomock.Controller) *MockVpoolKeeper {
	mock := &MockVpoolKeeper{ctrl: ctrl}
	mock.recorder = &MockVpoolKeeperMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockVpoolKeeper) EXPECT() *MockVpoolKeeperMockRecorder {
	return m.recorder
}

// CalcFee mocks base method.
func (m *MockVpoolKeeper) CalcFee(arg0 types1.Context, arg1 common.TokenPair, arg2 types1.Int) (types1.Int, types1.Int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CalcFee", arg0, arg1, arg2)
	ret0, _ := ret[0].(types1.Int)
	ret1, _ := ret[1].(types1.Int)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// CalcFee indicates an expected call of CalcFee.
func (mr *MockVpoolKeeperMockRecorder) CalcFee(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CalcFee", reflect.TypeOf((*MockVpoolKeeper)(nil).CalcFee), arg0, arg1, arg2)
}

// ExistsPool mocks base method.
func (m *MockVpoolKeeper) ExistsPool(arg0 types1.Context, arg1 common.TokenPair) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ExistsPool", arg0, arg1)
	ret0, _ := ret[0].(bool)
	return ret0
}

// ExistsPool indicates an expected call of ExistsPool.
func (mr *MockVpoolKeeperMockRecorder) ExistsPool(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ExistsPool", reflect.TypeOf((*MockVpoolKeeper)(nil).ExistsPool), arg0, arg1)
}

// GetOutputPrice mocks base method.
func (m *MockVpoolKeeper) GetOutputPrice(arg0 types1.Context, arg1 common.TokenPair, arg2 types0.Direction, arg3 types1.Dec) (types1.Dec, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetOutputPrice", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(types1.Dec)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetOutputPrice indicates an expected call of GetOutputPrice.
func (mr *MockVpoolKeeperMockRecorder) GetOutputPrice(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetOutputPrice", reflect.TypeOf((*MockVpoolKeeper)(nil).GetOutputPrice), arg0, arg1, arg2, arg3)
}

// GetOutputTWAP mocks base method.
func (m *MockVpoolKeeper) GetOutputTWAP(arg0 types1.Context, arg1 common.TokenPair, arg2 types0.Direction, arg3 types1.Int) (types1.Dec, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetOutputTWAP", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(types1.Dec)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetOutputTWAP indicates an expected call of GetOutputTWAP.
func (mr *MockVpoolKeeperMockRecorder) GetOutputTWAP(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetOutputTWAP", reflect.TypeOf((*MockVpoolKeeper)(nil).GetOutputTWAP), arg0, arg1, arg2, arg3)
}

// GetSpotPrice mocks base method.
func (m *MockVpoolKeeper) GetSpotPrice(arg0 types1.Context, arg1 common.TokenPair) (types1.Dec, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSpotPrice", arg0, arg1)
	ret0, _ := ret[0].(types1.Dec)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetSpotPrice indicates an expected call of GetSpotPrice.
func (mr *MockVpoolKeeperMockRecorder) GetSpotPrice(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSpotPrice", reflect.TypeOf((*MockVpoolKeeper)(nil).GetSpotPrice), arg0, arg1)
}

// GetUnderlyingPrice mocks base method.
func (m *MockVpoolKeeper) GetUnderlyingPrice(arg0 types1.Context, arg1 common.TokenPair) (types1.Dec, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUnderlyingPrice", arg0, arg1)
	ret0, _ := ret[0].(types1.Dec)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetUnderlyingPrice indicates an expected call of GetUnderlyingPrice.
func (mr *MockVpoolKeeperMockRecorder) GetUnderlyingPrice(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUnderlyingPrice", reflect.TypeOf((*MockVpoolKeeper)(nil).GetUnderlyingPrice), arg0, arg1)
}

// SwapInput mocks base method.
func (m *MockVpoolKeeper) SwapInput(arg0 types1.Context, arg1 common.TokenPair, arg2 types0.Direction, arg3, arg4 types1.Dec) (types1.Dec, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SwapInput", arg0, arg1, arg2, arg3, arg4)
	ret0, _ := ret[0].(types1.Dec)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SwapInput indicates an expected call of SwapInput.
func (mr *MockVpoolKeeperMockRecorder) SwapInput(arg0, arg1, arg2, arg3, arg4 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SwapInput", reflect.TypeOf((*MockVpoolKeeper)(nil).SwapInput), arg0, arg1, arg2, arg3, arg4)
}

// SwapOutput mocks base method.
func (m *MockVpoolKeeper) SwapOutput(arg0 types1.Context, arg1 common.TokenPair, arg2 types0.Direction, arg3, arg4 types1.Dec) (types1.Int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SwapOutput", arg0, arg1, arg2, arg3, arg4)
	ret0, _ := ret[0].(types1.Int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SwapOutput mocks base method.
func (m *MockVpoolKeeper) IsOverSpreadLimit(arg0 types1.Context, arg1 common.TokenPair) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsOverSpreadLimit", arg0, arg1)
	ret0, _ := ret[0].(bool)
	return ret0
}

// SwapOutput indicates an expected call of SwapOutput.
func (mr *MockVpoolKeeperMockRecorder) SwapOutput(arg0, arg1, arg2, arg3, arg4 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SwapOutput", reflect.TypeOf((*MockVpoolKeeper)(nil).SwapOutput), arg0, arg1, arg2, arg3, arg4)
}
