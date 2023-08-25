package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/mock"

	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
)

type MockHooks struct {
	mock.Mock
}

type Hooks interface {
	AfterEpochEnd(ctx sdk.Context, identifier string, epochNumber uint64)
	BeforeEpochStart(ctx sdk.Context, identifier string, epochNumber uint64)
}

func (h *MockHooks) AfterEpochEnd(ctx sdk.Context, identifier string, epochNumber uint64) {
	h.Called(ctx, identifier, epochNumber)
}

func (h *MockHooks) BeforeEpochStart(ctx sdk.Context, identifier string, epochNumber uint64) {
	h.Called(ctx, identifier, epochNumber)
}

func TestAfterEpochEnd(t *testing.T) {
	hooks := new(MockHooks)

	nibiruApp, ctx := testapp.NewNibiruTestAppAndContext()
	identifier := "testID"
	epochNumber := uint64(10)

	nibiruApp.EpochsKeeper.SetHooks(hooks)
	hooks.On("AfterEpochEnd", ctx, identifier, epochNumber)
	nibiruApp.EpochsKeeper.AfterEpochEnd(ctx, identifier, epochNumber)

	hooks.AssertExpectations(t)
}

func TestBeforeEpochStart(t *testing.T) {
	hooks := new(MockHooks)

	nibiruApp, ctx := testapp.NewNibiruTestAppAndContext()
	identifier := "testID"
	epochNumber := uint64(10)

	nibiruApp.EpochsKeeper.SetHooks(hooks)
	hooks.On("BeforeEpochStart", ctx, identifier, epochNumber)
	nibiruApp.EpochsKeeper.BeforeEpochStart(ctx, identifier, epochNumber)

	hooks.AssertExpectations(t)
}
