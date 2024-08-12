package precompile

import (
	"fmt"
	"math/big"
	"reflect"
	"sync"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gethabi "github.com/ethereum/go-ethereum/accounts/abi"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/NibiruChain/nibiru/app/keepers"
	"github.com/NibiruChain/nibiru/x/evm"
	"github.com/NibiruChain/nibiru/x/evm/embeds"
)

var (
	_ vm.PrecompiledContract = (*precompileFunToken)(nil)
	_ NibiruPrecompile       = (*precompileFunToken)(nil)
)

// Precompile address for "FunToken.sol", the contract that
// enables transfers of ERC20 tokens to "nibi" addresses as bank coins
// using the ERC20's `FunToken` mapping.
var PrecompileAddr_FuntokenGateway = gethcommon.HexToAddress("0x0000000000000000000000000000000000000800")

func (p precompileFunToken) Address() gethcommon.Address {
	return PrecompileAddr_FuntokenGateway
}

func (p precompileFunToken) RequiredGas(input []byte) (gasPrice uint64) {
	// TODO: https://github.com/NibiruChain/nibiru/issues/1990
	// We need to determine an appropriate gas value for the transaction to
	// configure this function and add a assertions around the gas usage to
	// the precompile's test suite. UD-DEBUG: not implemented yet. Currently
	// set to 0 gasPrice
	return 22
}

const (
	FunTokenMethod_BankSend FunTokenMethod = "bankSend"
)

type FunTokenMethod string

// Run runs the precompiled contract
func (p precompileFunToken) Run(
	evm *vm.EVM, contract *vm.Contract, readonly bool,
) (bz []byte, err error) {
	// This is a `defer` pattern to add behavior that runs in the case that the error is
	// non-nil, creating a concise way to add extra information.
	defer func() {
		if err != nil {
			precompileType := reflect.TypeOf(p).Name()
			err = fmt.Errorf("precompile error: failed to run %s: %w", precompileType, err)
		}
	}()

	contractInput := contract.Input
	ctx, method, args, err := OnRunStart(p, evm, contractInput)
	if err != nil {
		return nil, err
	}

	caller := contract.CallerAddress
	switch FunTokenMethod(method.Name) {
	case FunTokenMethod_BankSend:
		// TODO: UD-DEBUG: Test that calling non-method on the right address does
		// nothing.
		bz, err = p.bankSend(ctx, caller, method, args, readonly)
	default:
		// TODO: UD-DEBUG: test invalid method called
		err = fmt.Errorf("invalid method called with name \"%s\"", method.Name)
		return
	}
	return
}

func PrecompileFunToken(keepers keepers.PublicKeepers) vm.PrecompiledContract {
	return precompileFunToken{
		PublicKeepers: keepers,
	}
}

func (p precompileFunToken) ABI() *gethabi.ABI {
	return embeds.SmartContract_FunToken.ABI
}

type precompileFunToken struct {
	keepers.PublicKeepers
	NibiruPrecompile
}

var executionGuard sync.Mutex

/*
bankSend: Implements "IFunToken.bankSend"

The "args" populate the following function signature in Solidity:
```solidity
/// @dev bankSend sends ERC20 tokens as coins to a Nibiru base account
/// @param erc20 the address of the ERC20 token contract
/// @param amount the amount of tokens to send
/// @param to the receiving Nibiru base account address as a string
function bankSend(address erc20, uint256 amount, string memory to) external;
```
*/
func (p precompileFunToken) bankSend(
	ctx sdk.Context,
	caller gethcommon.Address,
	method *gethabi.Method,
	args []interface{},
	readOnly bool,
) (bz []byte, err error) {
	if readOnly {
		// Check required for transactions but not needed for queries
		return nil, fmt.Errorf("cannot write state from staticcall (a read-only call)")
	}
	if !executionGuard.TryLock() {
		return nil, fmt.Errorf("bankSend is already in progress")
	}
	defer executionGuard.Unlock()

	erc20, amount, to, err := p.AssertArgTypesBankSend(args)
	if err != nil {
		return
	}

	// ERC20 must have FunToken mapping
	funtokens := p.EvmKeeper.FunTokens.Collect(
		ctx, p.EvmKeeper.FunTokens.Indexes.ERC20Addr.ExactMatch(ctx, erc20),
	)
	if len(funtokens) != 1 {
		err = fmt.Errorf("no FunToken mapping exists for ERC20 \"%s\"", erc20.Hex())
		return
	}
	funtoken := funtokens[0]

	// Amount should be positive
	if amount == nil || amount.Cmp(big.NewInt(0)) != 1 {
		err = fmt.Errorf("transfer amount must be positive")
		return
	}

	// The "to" argument must be a valid Nibiru address
	toAddr, err := sdk.AccAddressFromBech32(to)
	if err != nil {
		err = fmt.Errorf("\"to\" is not a valid address (%s): %w", to, err)
		return
	}

	// Caller transfers ERC20 to the EVM account
	transferTo := evm.EVM_MODULE_ADDRESS
	_, err = p.EvmKeeper.ERC20().Transfer(erc20, caller, transferTo, amount, ctx)
	if err != nil {
		err = fmt.Errorf("failed to send from caller to the EVM account: %w", err)
		return
	}

	// EVM account mints FunToken.BankDenom to module account
	amt := math.NewIntFromBigInt(amount)
	coins := sdk.NewCoins(sdk.NewCoin(funtoken.BankDenom, amt))
	err = p.BankKeeper.MintCoins(ctx, evm.ModuleName, coins)
	if err != nil {
		err = fmt.Errorf("mint failed for module \"%s\" (%s): contract caller %s: %w",
			evm.ModuleName, evm.EVM_MODULE_ADDRESS.Hex(), caller.Hex(), err,
		)
		return
	}

	err = p.BankKeeper.SendCoinsFromModuleToAccount(ctx, evm.ModuleName, toAddr, coins)
	if err != nil {
		err = fmt.Errorf("send failed for module \"%s\" (%s): contract caller %s: %w",
			evm.ModuleName, evm.EVM_MODULE_ADDRESS.Hex(), caller.Hex(), err,
		)
		return
	}

	// If the FunToken mapping was created from a bank coin, then the EVM account
	// owns the ERC20 contract and was the original minter of the ERC20 tokens.
	// Since we're sending them away and want accurate total supply tracking, the
	// tokens need to be burned.
	if funtoken.IsMadeFromCoin {
		caller := evm.EVM_MODULE_ADDRESS
		_, err = p.EvmKeeper.ERC20().Burn(erc20, caller, amount, ctx)
		if err != nil {
			err = fmt.Errorf("ERC20.Burn: %w", err)
			return
		}
	}

	// TODO: UD-DEBUG: feat: Emit EVM events
	// TODO: UD-DEBUG: feat: Emit ABCI events

	return method.Outputs.Pack() // TODO: change interface
}

func (p precompileFunToken) AssertArgTypesBankSend(args []any) (
	erc20 gethcommon.Address,
	amount *big.Int,
	to string,
	err error,
) {
	if len(args) != 3 {
		err = fmt.Errorf("expected 3 arguments but got %d", len(args))
		return
	}

	erc20, ok := args[0].(gethcommon.Address)
	if !ok {
		err = fmt.Errorf("type validation for failed for (address erc20) argument")
		return
	}

	amount, ok = args[1].(*big.Int)
	if !ok {
		err = fmt.Errorf("type validation for failed for (uint256 amount) argument")
		return
	}

	to, ok = args[2].(string)
	if !ok {
		err = fmt.Errorf("type validation for failed for (string to) argument")
		return
	}

	return
}
