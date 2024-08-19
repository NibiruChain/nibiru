package types_test

import (
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/NibiruChain/nibiru/v2/x/epochs/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Define a MockEpochHooks type
type MockEpochHooks struct {
	mock.Mock
}

func (h *MockEpochHooks) AfterEpochEnd(ctx sdk.Context, epochIdentifier string, epochNumber uint64) {
	h.Called(ctx, epochIdentifier, epochNumber)
}

func (h *MockEpochHooks) BeforeEpochStart(ctx sdk.Context, epochIdentifier string, epochNumber uint64) {
	h.Called(ctx, epochIdentifier, epochNumber)
}

func TestAfterEpochEnd(t *testing.T) {
	hook1 := new(MockEpochHooks)
	hook2 := new(MockEpochHooks)
	hooks := types.NewMultiEpochHooks(hook1, hook2)

	ctx := sdk.Context{}
	epochIdentifier := "testID"
	epochNumber := uint64(10)

	hook1.On("AfterEpochEnd", ctx, epochIdentifier, epochNumber)
	hook2.On("AfterEpochEnd", ctx, epochIdentifier, epochNumber)

	hooks.AfterEpochEnd(ctx, epochIdentifier, epochNumber)

	hook1.AssertExpectations(t)
	hook2.AssertExpectations(t)
}

func TestBeforeEpochStart(t *testing.T) {
	hook1 := new(MockEpochHooks)
	hook2 := new(MockEpochHooks)
	hooks := types.NewMultiEpochHooks(hook1, hook2)

	ctx := sdk.Context{}
	epochIdentifier := "testID"
	epochNumber := uint64(10)

	hook1.On("BeforeEpochStart", ctx, epochIdentifier, epochNumber)
	hook2.On("BeforeEpochStart", ctx, epochIdentifier, epochNumber)

	hooks.BeforeEpochStart(ctx, epochIdentifier, epochNumber)

	hook1.AssertExpectations(t)
	hook2.AssertExpectations(t)
}
