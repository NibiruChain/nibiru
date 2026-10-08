package evmtrader

import (
	"testing"

	"github.com/NibiruChain/nibiru/v2/evm"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	abci "github.com/cometbft/cometbft/abci/types"
)

func TestEVMResultDistinguishesRevertFromCosmosSuccess(t *testing.T) {
	for _, vmError := range []string{"", "precompile error: execute wasm contract failed"} {
		event, err := sdk.TypedEventToEvent(&evm.EventEthereumTx{VmError: vmError})
		if err != nil {
			t.Fatal(err)
		}
		err = validateEVMResult(&sdk.TxResponse{Code: 0, Events: []abci.Event{abci.Event(event)}})
		if (err != nil) != (vmError != "") {
			t.Fatalf("vm_error=%q, got error %v", vmError, err)
		}
	}
	if err := validateEVMResult(&sdk.TxResponse{Code: 1, RawLog: "failed"}); err == nil {
		t.Fatal("accepted failed Cosmos transaction")
	}
	if err := validateEVMResult(&sdk.TxResponse{Code: 0}); err == nil {
		t.Fatal("accepted missing EVM result")
	}
}
