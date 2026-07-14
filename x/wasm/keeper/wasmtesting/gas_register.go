package wasmtesting

import (
	"github.com/NibiruChain/nibiru/v2/lib/wasmvm/wvm"

	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
)

// MockGasRegister mock that implements keeper.GasRegister
type MockGasRegister struct {
	CompileCostFn             func(byteLength int) sdk.Gas
	NewContractInstanceCostFn func(pinned bool, msgLen int) sdk.Gas
	InstantiateContractCostFn func(pinned bool, msgLen int) sdk.Gas
	ReplyCostFn               func(pinned bool, reply wvm.Reply) sdk.Gas
	EventCostsFn              func(evts []wvm.EventAttribute) sdk.Gas
	ToWasmVMGasFn             func(source sdk.Gas) uint64
	FromWasmVMGasFn           func(source uint64) sdk.Gas
	UncompressCostsFn         func(byteLength int) sdk.Gas
}

func (m MockGasRegister) NewContractInstanceCosts(pinned bool, msgLen int) sdk.Gas {
	if m.NewContractInstanceCostFn == nil {
		panic("not expected to be called")
	}
	return m.NewContractInstanceCostFn(pinned, msgLen)
}

func (m MockGasRegister) CompileCosts(byteLength int) sdk.Gas {
	if m.CompileCostFn == nil {
		panic("not expected to be called")
	}
	return m.CompileCostFn(byteLength)
}

func (m MockGasRegister) UncompressCosts(byteLength int) sdk.Gas {
	if m.UncompressCostsFn == nil {
		panic("not expected to be called")
	}
	return m.UncompressCostsFn(byteLength)
}

func (m MockGasRegister) InstantiateContractCosts(pinned bool, msgLen int) sdk.Gas {
	if m.InstantiateContractCostFn == nil {
		panic("not expected to be called")
	}
	return m.InstantiateContractCostFn(pinned, msgLen)
}

func (m MockGasRegister) ReplyCosts(pinned bool, reply wvm.Reply) sdk.Gas {
	if m.ReplyCostFn == nil {
		panic("not expected to be called")
	}
	return m.ReplyCostFn(pinned, reply)
}

func (m MockGasRegister) EventCosts(evts []wvm.EventAttribute, _ wvm.Events) sdk.Gas {
	if m.EventCostsFn == nil {
		panic("not expected to be called")
	}
	return m.EventCostsFn(evts)
}

func (m MockGasRegister) ToWasmVMGas(source sdk.Gas) uint64 {
	if m.ToWasmVMGasFn == nil {
		panic("not expected to be called")
	}
	return m.ToWasmVMGasFn(source)
}

func (m MockGasRegister) FromWasmVMGas(source uint64) sdk.Gas {
	if m.FromWasmVMGasFn == nil {
		panic("not expected to be called")
	}
	return m.FromWasmVMGasFn(source)
}
