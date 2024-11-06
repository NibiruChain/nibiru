package precompile

import (
	"fmt"
	"math/big"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gethabi "github.com/ethereum/go-ethereum/accounts/abi"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/NibiruChain/nibiru/v2/app/keepers"
	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
	evmkeeper "github.com/NibiruChain/nibiru/v2/x/evm/keeper"
)

var _ vm.PrecompiledContract = (*precompileFunToken)(nil)

// Precompile address for "IFunToken.sol", the contract that
// enables transfers of ERC20 tokens to "nibi" addresses as bank coins
// using the ERC20's `FunToken` mapping.
var PrecompileAddr_FunToken = gethcommon.HexToAddress("0x0000000000000000000000000000000000000800")

func (p precompileFunToken) Address() gethcommon.Address {
	return PrecompileAddr_FunToken
}

// RequiredGas calculates the cost of calling the precompile in gas units.
func (p precompileFunToken) RequiredGas(input []byte) (gasCost uint64) {
	return requiredGas(input, p.ABI())
}

func (p precompileFunToken) ABI() *gethabi.ABI {
	return embeds.SmartContract_FunToken.ABI
}

const (
	FunTokenMethod_sendToBank  PrecompileMethod = "sendToBank"
	FunTokenMethod_balance     PrecompileMethod = "balance"
	FunTokenMethod_bankBalance PrecompileMethod = "bankBalance"
)

// Run runs the precompiled contract
func (p precompileFunToken) Run(
	evm *vm.EVM, contract *vm.Contract, readonly bool,
) (bz []byte, err error) {
	defer func() {
		err = ErrPrecompileRun(err, p)
	}()
	startResult, err := OnRunStart(evm, contract.Input, p.ABI(), contract.Gas)
	if err != nil {
		return nil, err
	}
	p.evmKeeper.Bank.StateDB = startResult.StateDB

	// Gracefully handles "out of gas"
	defer HandleOutOfGasPanic(&err)()

	method := startResult.Method
	switch PrecompileMethod(method.Name) {
	case FunTokenMethod_sendToBank:
		bz, err = p.sendToBank(startResult, contract.CallerAddress, readonly)
	case FunTokenMethod_balance:
		bz, err = p.balance(startResult, contract)
	case FunTokenMethod_bankBalance:
		bz, err = p.bankBalance(startResult, contract)
	default:
		// Note that this code path should be impossible to reach since
		// "DecomposeInput" parses methods directly from the ABI.
		err = fmt.Errorf("invalid method called with name \"%s\"", method.Name)
		return
	}
	if err != nil {
		return nil, err
	}

	// Gas consumed by a local gas meter
	contract.UseGas(startResult.CacheCtx.GasMeter().GasConsumed())
	return bz, err
}

func PrecompileFunToken(keepers keepers.PublicKeepers) vm.PrecompiledContract {
	return precompileFunToken{
		evmKeeper: keepers.EvmKeeper,
	}
}

type precompileFunToken struct {
	evmKeeper *evmkeeper.Keeper
}

// sendToBank: Implements "IFunToken.sendToBank"
//
// The "args" populate the following function signature in Solidity:
//
//	```solidity
//	/// @dev sendToBank sends ERC20 tokens as coins to a Nibiru base account
//	/// @param erc20 the address of the ERC20 token contract
//	/// @param amount the amount of tokens to send
//	/// @param to the receiving Nibiru base account address as a string
//	function sendToBank(address erc20, uint256 amount, string memory to) external;
//	```
func (p precompileFunToken) sendToBank(
	startResult OnRunStartResult,
	caller gethcommon.Address,
	readOnly bool,
) (bz []byte, err error) {
	ctx, method, args := startResult.CacheCtx, startResult.Method, startResult.Args
	if err := assertNotReadonlyTx(readOnly, method); err != nil {
		return nil, err
	}

	erc20, amount, to, err := p.parseSendToBankArgs(args)
	if err != nil {
		return
	}

	var evmResponses []*evm.MsgEthereumTxResponse

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
	gotAmount, transferResp, err := p.evmKeeper.ERC20().Transfer(erc20, caller, transferTo, amount, ctx)
	if err != nil {
		return nil, fmt.Errorf("error in ERC20.transfer from caller to EVM account: %w", err)
	}
	evmResponses = append(evmResponses, transferResp)

	// EVM account mints FunToken.BankDenom to module account
	coinToSend := sdk.NewCoin(funtoken.BankDenom, math.NewIntFromBigInt(gotAmount))
	if funtoken.IsMadeFromCoin {
		// If the FunToken mapping was created from a bank coin, then the EVM account
		// owns the ERC20 contract and was the original minter of the ERC20 tokens.
		// Since we're sending them away and want accurate total supply tracking, the
		// tokens need to be burned.
		burnResp, e := p.evmKeeper.ERC20().Burn(erc20, evm.EVM_MODULE_ADDRESS, gotAmount, ctx)
		if e != nil {
			err = fmt.Errorf("ERC20.Burn: %w", e)
			return
		}
		evmResponses = append(evmResponses, burnResp)
	} else {
		// NOTE: The NibiruBankKeeper needs to reference the current [vm.StateDB] before
		// any operation that has the potential to use Bank send methods. This will
		// guarantee that [evmkeeper.Keeper.SetAccBalance] journal changes are
		// recorded if wei (NIBI) is transferred.
		p.evmKeeper.Bank.StateDB = startResult.StateDB
		err = p.evmKeeper.Bank.MintCoins(ctx, evm.ModuleName, sdk.NewCoins(coinToSend))
		if err != nil {
			return nil, fmt.Errorf("mint failed for module \"%s\" (%s): contract caller %s: %w",
				evm.ModuleName, evm.EVM_MODULE_ADDRESS.Hex(), caller.Hex(), err,
			)
		}
	}

	// Transfer the bank coin
	//
	// NOTE: The NibiruBankKeeper needs to reference the current [vm.StateDB] before
	// any operation that has the potential to use Bank send methods. This will
	// guarantee that [evmkeeper.Keeper.SetAccBalance] journal changes are
	// recorded if wei (NIBI) is transferred.
	p.evmKeeper.Bank.StateDB = startResult.StateDB
	err = p.evmKeeper.Bank.SendCoinsFromModuleToAccount(
		ctx,
		evm.ModuleName,
		toAddr,
		sdk.NewCoins(coinToSend),
	)
	if err != nil {
		return nil, fmt.Errorf("send failed for module \"%s\" (%s): contract caller %s: %w",
			evm.ModuleName, evm.EVM_MODULE_ADDRESS.Hex(), caller.Hex(), err,
		)
	}
	for _, resp := range evmResponses {
		for _, log := range resp.Logs {
			startResult.StateDB.AddLog(log.ToEthereum())
		}
	}

	// TODO: UD-DEBUG: feat: Emit EVM events

	return method.Outputs.Pack(gotAmount)
}

func (p precompileFunToken) parseSendToBankArgs(args []any) (
	erc20 gethcommon.Address,
	amount *big.Int,
	to string,
	err error,
) {
	if e := assertNumArgs(args, 3); e != nil {
		err = e
		return
	}

	argIdx := 0
	erc20, ok := args[argIdx].(gethcommon.Address)
	if !ok {
		err = ErrArgTypeValidation("address erc20", args[argIdx])
		return
	}

	argIdx++
	amount, ok = args[argIdx].(*big.Int)
	if !ok {
		err = ErrArgTypeValidation("uint256 amount", args[argIdx])
		return
	}

	argIdx++
	to, ok = args[argIdx].(string)
	if !ok {
		err = ErrArgTypeValidation("string to", args[argIdx])
		return
	}

	return
}

// TODO: UD-DEBUG: impl
// TODO: UD-DEBUG: test
// TODO: UD-DEBUG: docs
func (p precompileFunToken) balance(
	start OnRunStartResult,
	contract *vm.Contract,
) (bz []byte, err error) {
	method, args, ctx := start.Method, start.Args, start.CacheCtx
	defer func() {
		if err != nil {
			err = ErrMethodCalled(method, err)
		}
	}()
	if err := assertContractQuery(contract); err != nil {
		return bz, err
	}

	addrEth, addrBech32, funtoken, err := p.parseArgsBalance(args, ctx)
	if err != nil {
		err = ErrInvalidArgs(err)
		return
	}

	erc20Bal, err := p.evmKeeper.ERC20().BalanceOf(funtoken.Erc20Addr.Address, addrEth, ctx)
	if err != nil {
		return
	}
	bankBal := p.evmKeeper.Bank.GetBalance(ctx, addrBech32, funtoken.BankDenom).Amount.BigInt()

	// TODO: emit event for balance change of sender
	// TODO: emit event for balance change of recipient

	return method.Outputs.Pack([]any{
		erc20Bal,
		bankBal,
		struct {
			Erc20     gethcommon.Address `json:"erc20"`
			BankDenom string             `json:"bankDenom"`
		}{
			Erc20:     funtoken.Erc20Addr.Address,
			BankDenom: funtoken.BankDenom,
		},
		struct {
			EthAddr    gethcommon.Address `json:"ethAddr"`
			Bech32Addr string             `json:"bech32Addr"`
		}{
			EthAddr:    addrEth,
			Bech32Addr: addrBech32.String(),
		},
	}...)
}

func (p precompileFunToken) parseArgsBalance(args []any, ctx sdk.Context) (
	addrEth gethcommon.Address,
	addrBech32 sdk.AccAddress,
	funtoken evm.FunToken,
	err error,
) {
	if e := assertNumArgs(args, 2); e != nil {
		err = e
		return
	}

	argIdx := 0
	who, ok := args[argIdx].(gethcommon.Address)
	if !ok {
		err = ErrArgTypeValidation("bytes who", args[argIdx])
		return
	}
	req := &evm.QueryEthAccountRequest{Address: who.Hex()}
	_, e := req.Validate()
	if e != nil {
		err = e
		return
	}
	addrEth = gethcommon.HexToAddress(req.Address)
	addrBech32 = eth.EthAddrToNibiruAddr(addrEth)

	argIdx++
	funtokenErc20, ok := args[argIdx].(gethcommon.Address)
	if !ok {
		err = ErrArgTypeValidation("bytes funtoken", args[argIdx])
		return
	}
	resp, e := p.evmKeeper.FunTokenMapping(ctx, &evm.QueryFunTokenMappingRequest{
		Token: funtokenErc20.Hex(),
	})
	if e != nil {
		err = e
		return
	}

	return addrEth, addrBech32, *resp.FunToken, nil
}

// TODO: UD-DEBUG: impl
// TODO: UD-DEBUG: test
// TODO: UD-DEBUG: docs
func (p precompileFunToken) bankBalance(
	start OnRunStartResult,
	contract *vm.Contract,
) (bz []byte, err error) {
	method, args, ctx := start.Method, start.Args, start.CacheCtx
	defer func() {
		if err != nil {
			err = ErrMethodCalled(method, err)
		}
	}()
	if err := assertContractQuery(contract); err != nil {
		return bz, err
	}

	addrEth, addrBech32, bankDenom, err := p.parseArgsBankBalance(args)
	bankBal := p.evmKeeper.Bank.GetBalance(ctx, addrBech32, bankDenom).Amount.BigInt()

	// TODO: emit event for balance change of sender
	// TODO: emit event for balance change of recipient

	return method.Outputs.Pack([]any{
		bankBal,
		struct {
			EthAddr    gethcommon.Address `json:"ethAddr"`
			Bech32Addr string             `json:"bech32Addr"`
		}{
			EthAddr:    addrEth,
			Bech32Addr: addrBech32.String(),
		},
	}...)
}

func (p precompileFunToken) parseArgsBankBalance(args []any) (
	addrEth gethcommon.Address,
	addrBech32 sdk.AccAddress,
	bankDenom string,
	err error,
) {
	if e := assertNumArgs(args, 2); e != nil {
		err = e
		return
	}

	argIdx := 0
	who, ok := args[argIdx].(gethcommon.Address)
	if !ok {
		err = ErrArgTypeValidation("bytes who", args[argIdx])
		return
	}
	req := &evm.QueryEthAccountRequest{Address: who.Hex()}
	_, e := req.Validate()
	if e != nil {
		err = e
		return
	}
	addrEth = gethcommon.HexToAddress(req.Address)
	addrBech32 = eth.EthAddrToNibiruAddr(addrEth)

	argIdx++
	bankDenom, ok = args[argIdx].(string)
	if !ok {
		err = ErrArgTypeValidation("string bankDenom", args[argIdx])
		return
	}
	if e := sdk.ValidateDenom(bankDenom); e != nil {
		err = e
		return
	}

	return addrEth, addrBech32, bankDenom, nil
}
