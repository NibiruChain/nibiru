package precompile

import (
	"bytes"
	"fmt"
	"sync"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/collections"
	gethabi "github.com/ethereum/go-ethereum/accounts/abi"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/NibiruChain/nibiru/app/keepers"
	"github.com/NibiruChain/nibiru/x/common/set"
	"github.com/NibiruChain/nibiru/x/evm/statedb"
)

type Info struct {
	ABI  gethabi.ABI
	Addr gethcommon.Address
}

func InitPrecompiles(
	k keepers.PublicKeepers,
) {
	initMutex.Lock()
	defer initMutex.Unlock()

	for _, precompileSetupFn := range []func(k keepers.PublicKeepers) vm.PrecompiledContract{
		PrecompileFunTokenGateway,
	} {
		pc := precompileSetupFn(k)
		addPrecompileToVM(pc)
	}
}

// initMutex: Mutual exclusion lock (mutex) to prevent race conditions with
// consecutive calls of InitPrecompiles.
var initMutex = &sync.Mutex{}

// addPrecompileToVM
func addPrecompileToVM(p vm.PrecompiledContract) {
	addr := p.Address()
	for _, precompileMap := range []map[gethcommon.Address]vm.PrecompiledContract{
		vm.PrecompiledContractsHomestead,
		vm.PrecompiledContractsByzantium,
		vm.PrecompiledContractsIstanbul,
		vm.PrecompiledContractsBerlin,
		vm.PrecompiledContractsBLS,
		// TODO: 2024-07-05 feat: Cancun after go-ethereum upgrade
		// https://github.com/NibiruChain/nibiru/issues/1921
		// vm.PrecompiledContractsCancun,
	} {
		precompileMap[addr] = p
	}

	// Done if the precompiled contracts are already added
	// This check is only relevant during tests to prevent races. The iteration
	// doesn't get repeated in production.
	vmSet := set.New(vm.PrecompiledAddressesHomestead...)
	vmSet.AddMulti(vm.PrecompiledAddressesByzantium...)
	vmSet.AddMulti(vm.PrecompiledAddressesBerlin...)
	vmSet.AddMulti(vm.PrecompiledAddressesIstanbul...)
	if vmSet.Has(addr) {
		return
	}

	vm.PrecompiledAddressesHomestead = append(vm.PrecompiledAddressesHomestead, addr)
	vm.PrecompiledAddressesByzantium = append(vm.PrecompiledAddressesByzantium, addr)
	vm.PrecompiledAddressesIstanbul = append(vm.PrecompiledAddressesIstanbul, addr)
	vm.PrecompiledAddressesBerlin = append(vm.PrecompiledAddressesBerlin, addr)
	// TODO: 2024-07-05 feat: Cancun after go-ethereum upgrade
	// https://github.com/NibiruChain/nibiru/issues/1921
	// vm.PrecompiledAddressesCancun,

	vmSet.Add(addr)
	Addresses = vmSet
}

// Addresses is the set of all known precompile addresses. It includes defaults
// from go-ethereum and the custom ones specific to the Nibiru EVM.
var Addresses set.Set[gethcommon.Address]

type NibiruPrecompile interface {
	ABI() gethabi.ABI
}

// ABIMethodByID: Looks up an ABI method by the 4-byte id.
// Copy of "ABI.MethodById" from go-ethereum version > 1.10
func ABIMethodByID(abi gethabi.ABI, sigdata []byte) (*gethabi.Method, error) {
	if len(sigdata) < 4 {
		return nil, fmt.Errorf("data too short (%d bytes) for abi method lookup", len(sigdata))
	}
	for _, method := range abi.Methods {
		if bytes.Equal(method.ID, sigdata[:4]) {
			return &method, nil
		}
	}
	return nil, fmt.Errorf("no method with id: %#x", sigdata[:4])
}

func OnStart(
	p NibiruPrecompile, evm *vm.EVM, input []byte,
) (ctx sdk.Context, method *gethabi.Method, args []interface{}, err error) {
	// 1 | Get context from StateDB
	stateDB, ok := evm.StateDB.(*statedb.StateDB)
	if !ok {
		err = fmt.Errorf("failed to load the sdk.Context from the EVM StateDB")
		return
	}
	ctx = stateDB.GetContext()

	// 2 | Parse the ABI method
	// ABI method IDs are at least 4 bytes according to "gethabi.ABI.MethodByID".
	methodIdBytes := 4
	if len(input) < methodIdBytes {
		readableBz := collections.HumanizeBytes(input)
		err = fmt.Errorf("input \"%s\" too short to extract method ID (less than 4 bytes)", readableBz)
		return
	}
	methodID := input[:methodIdBytes]
	abi := p.ABI()
	method, err = ABIMethodByID(abi, methodID)
	if err != nil {
		err = fmt.Errorf("unable to parse ABI method by its 4-byte ID: %w", err)
		return
	}

	argsBz := input[methodIdBytes:]
	args, err = method.Inputs.Unpack(argsBz)
	if err != nil {
		err = fmt.Errorf("unable to unpack input args: %w", err)
		return
	}

	return ctx, method, args, nil
}
