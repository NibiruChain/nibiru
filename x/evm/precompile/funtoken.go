package precompile

import (
	"fmt"
	"math/big"
	"sync"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	gethabi "github.com/ethereum/go-ethereum/accounts/abi"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/NibiruChain/nibiru/v2/app/keepers"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
	evmkeeper "github.com/NibiruChain/nibiru/v2/x/evm/keeper"
)

var _ vm.PrecompiledContract = (*precompileFunToken)(nil)

// Precompile address for "FunToken.sol", the contract that
// enables transfers of ERC20 tokens to "nibi" addresses as bank coins
// using the ERC20's `FunToken` mapping.
var PrecompileAddr_FunToken = gethcommon.HexToAddress("0x0000000000000000000000000000000000000800")

func (p precompileFunToken) Address() gethcommon.Address {
	return PrecompileAddr_FunToken
}

func (p precompileFunToken) ABI() *gethabi.ABI {
	return embeds.SmartContract_FunToken.ABI
}

// RequiredGas calculates the cost of calling the precompile in gas units.
func (p precompileFunToken) RequiredGas(input []byte) (gasCost uint64) {
	return RequiredGas(input, p.ABI())
}

const (
	FunTokenMethod_BankSend PrecompileMethod = "bankSend"
)

type PrecompileMethod string

// Run runs the precompiled contract
func (p precompileFunToken) Run(
	evm *vm.EVM, contract *vm.Contract, readonly bool,
) (bz []byte, err error) {
	defer func() {
		err = ErrPrecompileRun(err, p)
	}()

	res, err := OnRunStart(evm, contract, p.ABI())
	if err != nil {
		return nil, err
	}
	method, args, ctx := res.Method, res.Args, res.Ctx

	switch PrecompileMethod(method.Name) {
	case FunTokenMethod_BankSend:
		bz, err = p.bankSend(ctx, contract.CallerAddress, method, args, readonly)
	default:
		// Note that this code path should be impossible to reach since
		// "DecomposeInput" parses methods directly from the ABI.
		err = fmt.Errorf("invalid method called with name \"%s\"", method.Name)
		return
	}
	if err != nil {
		return nil, err
	}
	if err := OnRunEnd(res.StateDB, res.SnapshotBeforeRun, p.Address()); err != nil {
		return nil, err
	}
	res.WriteCtx()
	return bz, nil
}

func PrecompileFunToken(keepers keepers.PublicKeepers) vm.PrecompiledContract {
	return precompileFunToken{
		bankKeeper: keepers.BankKeeper,
		evmKeeper:  keepers.EvmKeeper,
	}
}

type precompileFunToken struct {
	bankKeeper bankkeeper.Keeper
	evmKeeper  evmkeeper.Keeper
}

var executionGuard sync.Mutex

// bankSend: Implements "IFunToken.bankSend"
//
// The "args" populate the following function signature in Solidity:
//
//	```solidity
//	/// @dev bankSend sends ERC20 tokens as coins to a Nibiru base account
//	/// @param erc20 the address of the ERC20 token contract
//	/// @param amount the amount of tokens to send
//	/// @param to the receiving Nibiru base account address as a string
//	function bankSend(address erc20, uint256 amount, string memory to) external;
//	```
func (p precompileFunToken) bankSend(
	ctx sdk.Context,
	caller gethcommon.Address,
	method *gethabi.Method,
	args []any,
	readOnly bool,
) (bz []byte, err error) {
	if e := assertNotReadonlyTx(readOnly, true); e != nil {
		err = e
		return
	}
	if !executionGuard.TryLock() {
		return nil, fmt.Errorf("bankSend is already in progress")
	}
	defer executionGuard.Unlock()

	erc20, amount, to, err := p.decomposeBankSendArgs(args)
	if err != nil {
		return
	}

	// ERC20 must have FunToken mapping
	funtokens := p.evmKeeper.FunTokens.Collect(
		ctx, p.evmKeeper.FunTokens.Indexes.ERC20Addr.ExactMatch(ctx, erc20),
	)
	if len(funtokens) != 1 {
		err = fmt.Errorf("no FunToken mapping exists for ERC20 \"%s\"", erc20.Hex())
		return
	}
	funtoken := funtokens[0]

	// Amount should be positive
	if amount == nil || amount.Cmp(big.NewInt(0)) != 1 {
		return nil, fmt.Errorf("transfer amount must be positive")
	}

	// The "to" argument must be a valid Nibiru address
	toAddr, err := sdk.AccAddressFromBech32(to)
	if err != nil {
		return nil, fmt.Errorf("\"to\" is not a valid address (%s): %w", to, err)
	}

	// Caller transfers ERC20 to the EVM account
	transferTo := evm.EVM_MODULE_ADDRESS
	_, err = p.evmKeeper.ERC20().Transfer(erc20, caller, transferTo, amount, ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to send from caller to the EVM account: %w", err)
	}

	// EVM account mints FunToken.BankDenom to module account
	amt := math.NewIntFromBigInt(amount)
	coins := sdk.NewCoins(sdk.NewCoin(funtoken.BankDenom, amt))
	if funtoken.IsMadeFromCoin {
		// If the FunToken mapping was created from a bank coin, then the EVM account
		// owns the ERC20 contract and was the original minter of the ERC20 tokens.
		// Since we're sending them away and want accurate total supply tracking, the
		// tokens need to be burned.
		_, err = p.evmKeeper.ERC20().Burn(erc20, evm.EVM_MODULE_ADDRESS, amount, ctx)
		if err != nil {
			err = fmt.Errorf("ERC20.Burn: %w", err)
			return
		}
	} else {
		err = p.bankKeeper.MintCoins(ctx, evm.ModuleName, coins)
		if err != nil {
			return nil, fmt.Errorf("mint failed for module \"%s\" (%s): contract caller %s: %w",
				evm.ModuleName, evm.EVM_MODULE_ADDRESS.Hex(), caller.Hex(), err,
			)
		}
	}

	// Transfer the bank coin
	err = p.bankKeeper.SendCoinsFromModuleToAccount(ctx, evm.ModuleName, toAddr, coins)
	if err != nil {
		return nil, fmt.Errorf("send failed for module \"%s\" (%s): contract caller %s: %w",
			evm.ModuleName, evm.EVM_MODULE_ADDRESS.Hex(), caller.Hex(), err,
		)
	}

	// TODO: UD-DEBUG: feat: Emit EVM events

	return method.Outputs.Pack()
}

func (p precompileFunToken) decomposeBankSendArgs(args []any) (
	erc20 gethcommon.Address,
	amount *big.Int,
	to string,
	err error,
) {
	if e := assertNumArgs(len(args), 3); e != nil {
		err = e
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
