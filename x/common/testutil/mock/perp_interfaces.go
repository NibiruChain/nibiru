// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/NibiruChain/nibiru/x/perp/types (interfaces: AccountKeeper,BankKeeper,OracleKeeper,VpoolKeeper,EpochKeeper)

// Package mock is a generated GoMock package.
package mock

import (
	reflect "reflect"
	time "time"

	asset "github.com/NibiruChain/nibiru/x/common/asset"
	types "github.com/NibiruChain/nibiru/x/epochs/types"
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

// SendCoinsFromModuleToModule mocks base method.
func (m *MockBankKeeper) SendCoinsFromModuleToModule(arg0 types1.Context, arg1, arg2 string, arg3 types1.Coins) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendCoinsFromModuleToModule", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(error)
	return ret0
}

// SendCoinsFromModuleToModule indicates an expected call of SendCoinsFromModuleToModule.
func (mr *MockBankKeeperMockRecorder) SendCoinsFromModuleToModule(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendCoinsFromModuleToModule", reflect.TypeOf((*MockBankKeeper)(nil).SendCoinsFromModuleToModule), arg0, arg1, arg2, arg3)
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

// MockOracleKeeper is a mock of OracleKeeper interface.
type MockOracleKeeper struct {
	ctrl     *gomock.Controller
	recorder *MockOracleKeeperMockRecorder
}

// MockOracleKeeperMockRecorder is the mock recorder for MockOracleKeeper.
type MockOracleKeeperMockRecorder struct {
	mock *MockOracleKeeper
}

// NewMockOracleKeeper creates a new mock instance.
func NewMockOracleKeeper(ctrl *gomock.Controller) *MockOracleKeeper {
	mock := &MockOracleKeeper{ctrl: ctrl}
	mock.recorder = &MockOracleKeeperMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockOracleKeeper) EXPECT() *MockOracleKeeperMockRecorder {
	return m.recorder
}

// GetExchangeRate mocks base method.
func (m *MockOracleKeeper) GetExchangeRate(arg0 types1.Context, arg1 asset.Pair) (types1.Dec, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetExchangeRate", arg0, arg1)
	ret0, _ := ret[0].(types1.Dec)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetExchangeRate indicates an expected call of GetExchangeRate.
func (mr *MockOracleKeeperMockRecorder) GetExchangeRate(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetExchangeRate", reflect.TypeOf((*MockOracleKeeper)(nil).GetExchangeRate), arg0, arg1)
}

// GetExchangeRateTwap mocks base method.
func (m *MockOracleKeeper) GetExchangeRateTwap(arg0 types1.Context, arg1 asset.Pair) (types1.Dec, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetExchangeRateTwap", arg0, arg1)
	ret0, _ := ret[0].(types1.Dec)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetExchangeRateTwap indicates an expected call of GetExchangeRateTwap.
func (mr *MockOracleKeeperMockRecorder) GetExchangeRateTwap(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetExchangeRateTwap", reflect.TypeOf((*MockOracleKeeper)(nil).GetExchangeRateTwap), arg0, arg1)
}

// SetPrice mocks base method.
func (m *MockOracleKeeper) SetPrice(arg0 types1.Context, arg1 asset.Pair, arg2 types1.Dec) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetPrice", arg0, arg1, arg2)
}

// SetPrice indicates an expected call of SetPrice.
func (mr *MockOracleKeeperMockRecorder) SetPrice(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetPrice", reflect.TypeOf((*MockOracleKeeper)(nil).SetPrice), arg0, arg1, arg2)
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

// ExistsPool mocks base method.
func (m *MockVpoolKeeper) ExistsPool(arg0 types1.Context, arg1 asset.Pair) bool {
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

// GetAllPools mocks base method.
func (m *MockVpoolKeeper) GetAllPools(arg0 types1.Context) []types0.Vpool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAllPools", arg0)
	ret0, _ := ret[0].([]types0.Vpool)
	return ret0
}

// GetAllPools indicates an expected call of GetAllPools.
func (mr *MockVpoolKeeperMockRecorder) GetAllPools(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAllPools", reflect.TypeOf((*MockVpoolKeeper)(nil).GetAllPools), arg0)
}

// GetBaseAssetPrice mocks base method.
func (m *MockVpoolKeeper) GetBaseAssetPrice(arg0 types0.Vpool, arg1 types0.Direction, arg2 types1.Dec) (types1.Dec, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBaseAssetPrice", arg0, arg1, arg2)
	ret0, _ := ret[0].(types1.Dec)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetBaseAssetPrice indicates an expected call of GetBaseAssetPrice.
func (mr *MockVpoolKeeperMockRecorder) GetBaseAssetPrice(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBaseAssetPrice", reflect.TypeOf((*MockVpoolKeeper)(nil).GetBaseAssetPrice), arg0, arg1, arg2)
}

// GetBaseAssetTWAP mocks base method.
func (m *MockVpoolKeeper) GetBaseAssetTWAP(arg0 types1.Context, arg1 asset.Pair, arg2 types0.Direction, arg3 types1.Dec, arg4 time.Duration) (types1.Dec, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBaseAssetTWAP", arg0, arg1, arg2, arg3, arg4)
	ret0, _ := ret[0].(types1.Dec)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetBaseAssetTWAP indicates an expected call of GetBaseAssetTWAP.
func (mr *MockVpoolKeeperMockRecorder) GetBaseAssetTWAP(arg0, arg1, arg2, arg3, arg4 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBaseAssetTWAP", reflect.TypeOf((*MockVpoolKeeper)(nil).GetBaseAssetTWAP), arg0, arg1, arg2, arg3, arg4)
}

// GetLastSnapshot mocks base method.
func (m *MockVpoolKeeper) GetLastSnapshot(arg0 types1.Context, arg1 types0.Vpool) (types0.ReserveSnapshot, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetLastSnapshot", arg0, arg1)
	ret0, _ := ret[0].(types0.ReserveSnapshot)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetLastSnapshot indicates an expected call of GetLastSnapshot.
func (mr *MockVpoolKeeperMockRecorder) GetLastSnapshot(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetLastSnapshot", reflect.TypeOf((*MockVpoolKeeper)(nil).GetLastSnapshot), arg0, arg1)
}

// GetMaintenanceMarginRatio mocks base method.
func (m *MockVpoolKeeper) GetMaintenanceMarginRatio(arg0 types1.Context, arg1 asset.Pair) (types1.Dec, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetMaintenanceMarginRatio", arg0, arg1)
	ret0, _ := ret[0].(types1.Dec)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetMaintenanceMarginRatio indicates an expected call of GetMaintenanceMarginRatio.
func (mr *MockVpoolKeeperMockRecorder) GetMaintenanceMarginRatio(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetMaintenanceMarginRatio", reflect.TypeOf((*MockVpoolKeeper)(nil).GetMaintenanceMarginRatio), arg0, arg1)
}

// GetMarkPrice mocks base method.
func (m *MockVpoolKeeper) GetMarkPrice(arg0 types1.Context, arg1 asset.Pair) (types1.Dec, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetMarkPrice", arg0, arg1)
	ret0, _ := ret[0].(types1.Dec)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetMarkPrice indicates an expected call of GetMarkPrice.
func (mr *MockVpoolKeeperMockRecorder) GetMarkPrice(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetMarkPrice", reflect.TypeOf((*MockVpoolKeeper)(nil).GetMarkPrice), arg0, arg1)
}

// GetMarkPriceTWAP mocks base method.
func (m *MockVpoolKeeper) GetMarkPriceTWAP(arg0 types1.Context, arg1 asset.Pair, arg2 time.Duration) (types1.Dec, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetMarkPriceTWAP", arg0, arg1, arg2)
	ret0, _ := ret[0].(types1.Dec)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetMarkPriceTWAP indicates an expected call of GetMarkPriceTWAP.
func (mr *MockVpoolKeeperMockRecorder) GetMarkPriceTWAP(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetMarkPriceTWAP", reflect.TypeOf((*MockVpoolKeeper)(nil).GetMarkPriceTWAP), arg0, arg1, arg2)
}

// GetPool mocks base method.
func (m *MockVpoolKeeper) GetPool(arg0 types1.Context, arg1 asset.Pair) (types0.Vpool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetPool", arg0, arg1)
	ret0, _ := ret[0].(types0.Vpool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetPool indicates an expected call of GetPool.
func (mr *MockVpoolKeeperMockRecorder) GetPool(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetPool", reflect.TypeOf((*MockVpoolKeeper)(nil).GetPool), arg0, arg1)
}

// GetQuoteAssetPrice mocks base method.
func (m *MockVpoolKeeper) GetQuoteAssetPrice(arg0 types1.Context, arg1 asset.Pair, arg2 types0.Direction, arg3 types1.Dec) (types1.Dec, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetQuoteAssetPrice", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(types1.Dec)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetQuoteAssetPrice indicates an expected call of GetQuoteAssetPrice.
func (mr *MockVpoolKeeperMockRecorder) GetQuoteAssetPrice(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetQuoteAssetPrice", reflect.TypeOf((*MockVpoolKeeper)(nil).GetQuoteAssetPrice), arg0, arg1, arg2, arg3)
}

// GetSettlementPrice mocks base method.
func (m *MockVpoolKeeper) GetSettlementPrice(arg0 types1.Context, arg1 asset.Pair) (types1.Dec, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSettlementPrice", arg0, arg1)
	ret0, _ := ret[0].(types1.Dec)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetSettlementPrice indicates an expected call of GetSettlementPrice.
func (mr *MockVpoolKeeperMockRecorder) GetSettlementPrice(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSettlementPrice", reflect.TypeOf((*MockVpoolKeeper)(nil).GetSettlementPrice), arg0, arg1)
}

// IsOverSpreadLimit mocks base method.
func (m *MockVpoolKeeper) IsOverSpreadLimit(arg0 types1.Context, arg1 asset.Pair) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsOverSpreadLimit", arg0, arg1)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// IsOverSpreadLimit indicates an expected call of IsOverSpreadLimit.
func (mr *MockVpoolKeeperMockRecorder) IsOverSpreadLimit(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsOverSpreadLimit", reflect.TypeOf((*MockVpoolKeeper)(nil).IsOverSpreadLimit), arg0, arg1)
}

// SwapBaseForQuote mocks base method.
func (m *MockVpoolKeeper) SwapBaseForQuote(arg0 types1.Context, arg1 types0.Vpool, arg2 types0.Direction, arg3, arg4 types1.Dec, arg5 bool) (types0.Vpool, types1.Dec, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SwapBaseForQuote", arg0, arg1, arg2, arg3, arg4, arg5)
	ret0, _ := ret[0].(types0.Vpool)
	ret1, _ := ret[1].(types1.Dec)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// SwapBaseForQuote indicates an expected call of SwapBaseForQuote.
func (mr *MockVpoolKeeperMockRecorder) SwapBaseForQuote(arg0, arg1, arg2, arg3, arg4, arg5 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SwapBaseForQuote", reflect.TypeOf((*MockVpoolKeeper)(nil).SwapBaseForQuote), arg0, arg1, arg2, arg3, arg4, arg5)
}

// SwapQuoteForBase mocks base method.
func (m *MockVpoolKeeper) SwapQuoteForBase(arg0 types1.Context, arg1 types0.Vpool, arg2 types0.Direction, arg3, arg4 types1.Dec, arg5 bool) (types0.Vpool, types1.Dec, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SwapQuoteForBase", arg0, arg1, arg2, arg3, arg4, arg5)
	ret0, _ := ret[0].(types0.Vpool)
	ret1, _ := ret[1].(types1.Dec)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// SwapQuoteForBase indicates an expected call of SwapQuoteForBase.
func (mr *MockVpoolKeeperMockRecorder) SwapQuoteForBase(arg0, arg1, arg2, arg3, arg4, arg5 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SwapQuoteForBase", reflect.TypeOf((*MockVpoolKeeper)(nil).SwapQuoteForBase), arg0, arg1, arg2, arg3, arg4, arg5)
}

// MockEpochKeeper is a mock of EpochKeeper interface.
type MockEpochKeeper struct {
	ctrl     *gomock.Controller
	recorder *MockEpochKeeperMockRecorder
}

// MockEpochKeeperMockRecorder is the mock recorder for MockEpochKeeper.
type MockEpochKeeperMockRecorder struct {
	mock *MockEpochKeeper
}

// NewMockEpochKeeper creates a new mock instance.
func NewMockEpochKeeper(ctrl *gomock.Controller) *MockEpochKeeper {
	mock := &MockEpochKeeper{ctrl: ctrl}
	mock.recorder = &MockEpochKeeperMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockEpochKeeper) EXPECT() *MockEpochKeeperMockRecorder {
	return m.recorder
}

// GetEpochInfo mocks base method.
func (m *MockEpochKeeper) GetEpochInfo(arg0 types1.Context, arg1 string) types.EpochInfo {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetEpochInfo", arg0, arg1)
	ret0, _ := ret[0].(types.EpochInfo)
	return ret0
}

// GetEpochInfo indicates an expected call of GetEpochInfo.
func (mr *MockEpochKeeperMockRecorder) GetEpochInfo(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetEpochInfo", reflect.TypeOf((*MockEpochKeeper)(nil).GetEpochInfo), arg0, arg1)
}
