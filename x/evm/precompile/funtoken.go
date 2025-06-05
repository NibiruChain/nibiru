package precompile

import (
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	bank "github.com/cosmos/cosmos-sdk/x/bank/types"
	gethabi "github.com/ethereum/go-ethereum/accounts/abi"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/vm"

	tftypes "github.com/NibiruChain/nibiru/v2/x/tokenfactory/types"

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
	FunTokenMethod_sendToBank      PrecompileMethod = "sendToBank"
	FunTokenMethod_balance         PrecompileMethod = "balance"
	FunTokenMethod_bankBalance     PrecompileMethod = "bankBalance"
	FunTokenMethod_whoAmI          PrecompileMethod = "whoAmI"
	FunTokenMethod_sendToEvm       PrecompileMethod = "sendToEvm"
	FunTokenMethod_bankMsgSend     PrecompileMethod = "bankMsgSend"
	FunTokenMethod_getErc20Address PrecompileMethod = "getErc20Address"
)

// Run runs the precompiled contract
func (p precompileFunToken) Run(
	evm *vm.EVM,
	trueCaller gethcommon.Address,
	// Note that we use "trueCaller" here to differentiate between a delegate
	// caller ("parent.CallerAddress" in geth) and "contract.CallerAddress"
	// because these two addresses may differ.
	contract *vm.Contract,
	readonly bool,
	// isDelegatedCall: Flag to add conditional logic specific to delegate calls
	isDelegatedCall bool,
) (bz []byte, err error) {
	defer func() {
		err = ErrPrecompileRun(err, p)
	}()
	startResult, err := OnRunStart(evm, contract.Input, p.ABI(), contract.Gas)
	if err != nil {
		return nil, err
	}

	// Gracefully handles "out of gas"
	defer HandleOutOfGasPanic(&err)()

	abciEventsStartIdx := len(startResult.CacheCtx.EventManager().Events())

	method := startResult.Method
	switch PrecompileMethod(method.Name) {
	case FunTokenMethod_sendToBank:
		bz, err = p.sendToBank(startResult, trueCaller, readonly, evm)
	case FunTokenMethod_balance:
		bz, err = p.balance(startResult, contract, evm)
	case FunTokenMethod_bankBalance:
		bz, err = p.bankBalance(startResult, contract)
	case FunTokenMethod_whoAmI:
		bz, err = p.whoAmI(startResult, contract)
	case FunTokenMethod_sendToEvm:
		bz, err = p.sendToEvm(startResult, trueCaller, readonly, evm)
	case FunTokenMethod_bankMsgSend:
		bz, err = p.bankMsgSend(startResult, trueCaller, readonly)
	case FunTokenMethod_getErc20Address:
		bz, err = p.getErc20Address(startResult, contract)
	default:
		// Note that this code path should be impossible to reach since
		// "[decomposeInput]" parses methods directly from the ABI.
		err = fmt.Errorf("invalid method called with name \"%s\"", method.Name)
		return
	}
	// Gas consumed by a local gas meter
	contract.UseGas(
		startResult.CacheCtx.GasMeter().GasConsumed(),
		evm.Config.Tracer,
		tracing.GasChangeCallPrecompiledContract,
	)
	if err != nil {
		return nil, err
	}

	// Emit extra events for the EVM if this is a transaction
	// https://github.com/NibiruChain/nibiru/issues/2121
	if isMutation[PrecompileMethod(startResult.Method.Name)] {
		EmitEventAbciEvents(
			startResult.CacheCtx,
			startResult.StateDB,
			startResult.CacheCtx.EventManager().Events()[abciEventsStartIdx:],
			p.Address(),
		)
	}

	return bz, err
}

func PrecompileFunToken(keepers keepers.PublicKeepers) NibiruCustomPrecompile {
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
//
// Because [sendToBank] uses "SendCoinsFromModuleToAccount" to send Bank Coins,
// this method correctly avoids sending funds to addresses blocked by the Bank
// module.
func (p precompileFunToken) sendToBank(
	startResult OnRunStartResult,
	caller gethcommon.Address,
	readOnly bool,
	evmObj *vm.EVM,
) (bz []byte, err error) {
	ctx, method, args := startResult.CacheCtx, startResult.Method, startResult.Args
	if err := assertNotReadonlyTx(readOnly, method); err != nil {
		return nil, err
	}

	erc20, amount, to, err := p.parseArgsSendToBank(args)
	if err != nil {
		return nil, ErrInvalidArgs(err)
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

	// The "to" argument must be a valid nibi or EVM address
	toAddr, err := parseToAddr(to)
	if err != nil {
		return nil, fmt.Errorf("recipient address invalid (%s): %w", to, err)
	}

	// Caller transfers ERC20 to the EVM module account
	gotAmount, _, err := p.evmKeeper.ERC20().Transfer(
		erc20,                  /*erc20*/
		caller,                 /*from*/
		evm.EVM_MODULE_ADDRESS, /*to*/
		amount,                 /*value*/
		ctx,
		evmObj,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"error in ERC20.transfer from caller to EVM account: %w: from %s, erc20 %s, amount: %s",
			err, caller, erc20, amount,
		)
	}

	// EVM account mints FunToken.BankDenom to module account
	coinToSend := sdk.NewCoin(funtoken.BankDenom, sdkmath.NewIntFromBigInt(gotAmount))
	if funtoken.IsMadeFromCoin {
		// If the FunToken mapping was created from a bank coin, then the EVM account
		// owns the ERC20 contract and was the original minter of the ERC20 tokens.
		// Since we're sending them away and want accurate total supply tracking, the
		// tokens need to be burned.
		_, err := p.evmKeeper.ERC20().Burn(erc20, evm.EVM_MODULE_ADDRESS, gotAmount, ctx, evmObj)
		if err != nil {
			return nil, fmt.Errorf("ERC20.Burn: %w", err)
		}
	} else {
		// NOTE: The NibiruBankKeeper needs to reference the current [vm.StateDB] before
		// any operation that has the potential to use Bank send methods. This will
		// guarantee that [evmkeeper.Keeper.SetAccBalance] journal changes are
		// recorded if wei (NIBI) is transferred.
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
	err = p.evmKeeper.Bank.SendCoinsFromModuleToAccount(
		ctx,
		evm.ModuleName,
		eth.EthAddrToNibiruAddr(toAddr),
		sdk.NewCoins(coinToSend),
	)
	if err != nil {
		return nil, fmt.Errorf("send failed for module \"%s\" (%s): contract caller %s: %w",
			evm.ModuleName, evm.EVM_MODULE_ADDRESS.Hex(), caller.Hex(), err,
		)
	}

	return method.Outputs.Pack(gotAmount)
}

func (p precompileFunToken) parseArgsSendToBank(args []any) (
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

// balance: Implements "IFunToken.balance"
//
// The "args" populate the following function signature in Solidity:
//
//	```solidity
//	function balance(
//	    address who,
//	    address funtoken
//	)
//	    external
//	    returns (
//	        uint256 erc20Balance,
//	        uint256 bankBalance,
//	        FunToken memory token,
//	        NibiruAccount memory whoAddrs
//	    );
//	```
func (p precompileFunToken) balance(
	start OnRunStartResult,
	contract *vm.Contract,
	evmObj *vm.EVM,
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

	erc20Bal, err := p.evmKeeper.ERC20().BalanceOf(funtoken.Erc20Addr.Address, addrEth, ctx, evmObj)
	if err != nil {
		return
	}
	bankBal := p.evmKeeper.Bank.GetBalance(ctx, addrBech32, funtoken.BankDenom).Amount.BigInt()

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

// bankBalance: Implements "IFunToken.bankBalance"
//
// The "args" populate the following function signature in Solidity:
//
//	```solidity
//	function bankBalance(
//	    address who,
//	    string calldata bankDenom
//	) external returns (uint256 bankBalance, NibiruAccount memory whoAddrs);
//	```
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
	if err != nil {
		err = ErrInvalidArgs(err)
		return
	}
	bankBal := p.evmKeeper.Bank.GetBalance(ctx, addrBech32, bankDenom).Amount.BigInt()

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

// whoAmI: Implements "IFunToken.whoAmI"
//
// The "args" populate the following function signature in Solidity:
//
//	```solidity
//	function whoAmI(
//	    string calldata who
//	) external returns (NibiruAccount memory whoAddrs);
//	```
func (p precompileFunToken) whoAmI(
	start OnRunStartResult,
	contract *vm.Contract,
) (bz []byte, err error) {
	method, args := start.Method, start.Args
	defer func() {
		if err != nil {
			err = ErrMethodCalled(method, err)
		}
	}()
	if err := assertContractQuery(contract); err != nil {
		return bz, err
	}
	addrEth, addrBech32, err := p.parseArgsWhoAmI(args)
	if err != nil {
		err = ErrInvalidArgs(err)
		return
	}
	bz, err = method.Outputs.Pack([]any{
		struct {
			EthAddr    gethcommon.Address `json:"ethAddr"`
			Bech32Addr string             `json:"bech32Addr"`
		}{
			EthAddr:    addrEth,
			Bech32Addr: addrBech32.String(),
		},
	}...)
	return bz, err
}

func (p precompileFunToken) parseArgsWhoAmI(args []any) (
	addrEth gethcommon.Address,
	addrBech32 sdk.AccAddress,
	err error,
) {
	if e := assertNumArgs(args, 1); e != nil {
		err = e
		return
	}

	argIdx := 0
	who, ok := args[argIdx].(string)
	if !ok {
		err = ErrArgTypeValidation("string calldata who", args[argIdx])
		return
	}
	req := &evm.QueryEthAccountRequest{Address: who}
	isBech32, e := req.Validate()
	if e != nil {
		err = e
		return
	}
	if isBech32 {
		addrBech32 = sdk.MustAccAddressFromBech32(req.Address)
		addrEth = eth.NibiruAddrToEthAddr(addrBech32)
	} else {
		addrEth = gethcommon.HexToAddress(req.Address)
		addrBech32 = eth.EthAddrToNibiruAddr(addrEth)
	}
	return addrEth, addrBech32, nil
}

// SendToEvm: Implements "IFunToken.sendToEvm"
// Transfers the caller's bank coin `denom` to its ERC-20 representation on
// the EVM side.
func (p precompileFunToken) sendToEvm(
	startResult OnRunStartResult,
	caller gethcommon.Address,
	readOnly bool,
	evmObj *vm.EVM,
) ([]byte, error) {
	ctx, method, args := startResult.CacheCtx, startResult.Method, startResult.Args
	if err := assertNotReadonlyTx(readOnly, method); err != nil {
		return nil, err
	}
	// parse call: (string bankDenom, uint256 amount, string to)
	bankDenom, amount, toStr, err := parseArgsSendToEvm(args)
	if err != nil {
		return nil, ErrInvalidArgs(err)
	}

	// load the FunToken mapping
	//   For bankDenom, check if there's an existing funtoken
	funtokens := p.evmKeeper.FunTokens.Collect(
		ctx, p.evmKeeper.FunTokens.Indexes.BankDenom.ExactMatch(ctx, bankDenom),
	)
	if len(funtokens) == 0 {
		return nil, fmt.Errorf("no funtoken found for bank denom \"%s\"", bankDenom)
	}
	funtoken := funtokens[0]

	if amount == nil || amount.Sign() != 1 {
		return nil, fmt.Errorf("transfer amount must be positive")
	}

	// parse the `to` argument as hex or bech32 address
	toEthAddr, err := parseToAddr(toStr)
	if err != nil {
		return nil, fmt.Errorf("recipient address invalid: %w", err)
	}

	// 1) remove (burn or escrow) the bank coin from caller
	coinToSend := sdk.NewCoin(funtoken.BankDenom, sdkmath.NewIntFromBigInt(amount))
	callerBech32 := eth.EthAddrToNibiruAddr(caller)

	// bank send from account => module
	if err := p.evmKeeper.Bank.SendCoinsFromAccountToModule(
		ctx, callerBech32, evm.ModuleName, sdk.NewCoins(coinToSend),
	); err != nil {
		return nil, fmt.Errorf("failed to send coins to module: %w", err)
	}

	// 2) mint (or unescrow) the ERC20
	erc20Addr := funtoken.Erc20Addr.Address
	actualAmt, err := p.mintOrUnescrowERC20(
		ctx,
		erc20Addr,                  /*erc20Contract*/
		toEthAddr,                  /*to*/
		coinToSend.Amount.BigInt(), /*amount*/
		funtoken,                   /*funtoken*/
		evmObj,
	)
	if err != nil {
		return nil, err
	}

	if !funtoken.IsMadeFromCoin {
		// If the tokens is from an ERC20, we need to burn the cosmos coin
		// and unescrow the ERC20 tokens to the recipient.
		err := p.evmKeeper.Bank.BurnCoins(ctx, evm.ModuleName, sdk.NewCoins(coinToSend))
		if err != nil {
			return nil, fmt.Errorf("failed to burn coins: %w", err)
		}
	}

	// return the number of tokens minted
	return method.Outputs.Pack(actualAmt)
}

func (p precompileFunToken) mintOrUnescrowERC20(
	ctx sdk.Context,
	erc20Addr gethcommon.Address,
	to gethcommon.Address,
	amount *big.Int,
	funtoken evm.FunToken,
	evmObj *vm.EVM,
) (*big.Int, error) {
	// If funtoken is "IsMadeFromCoin", we own the ERC20 contract, so we can mint.
	// If not, we do a transfer from EVM module to 'to' address using escrowed tokens.
	if funtoken.IsMadeFromCoin {
		_, err := p.evmKeeper.ERC20().Mint(
			erc20Addr,              /*erc20Contract*/
			evm.EVM_MODULE_ADDRESS, /*from*/
			to,                     /*to*/
			amount,
			ctx,
			evmObj,
		)
		if err != nil {
			return nil, fmt.Errorf("mint erc20 error: %w", err)
		}
		// For an owner-minted contract, the entire `amount` is minted.
		return amount, nil
	} else {
		balanceIncrease, _, err := p.evmKeeper.ERC20().Transfer(
			erc20Addr, evm.EVM_MODULE_ADDRESS, to, amount, ctx, evmObj,
		)
		if err != nil {
			return nil, fmt.Errorf("erc20.transfer from module to user: %w", err)
		}
		return balanceIncrease, nil
	}
}

// parse the arguments: (string bankDenom, uint256 amount, string to)
func parseArgsSendToEvm(args []any) (bankDenom string, amount *big.Int, to string, err error) {
	if e := assertNumArgs(args, 3); e != nil {
		return "", nil, "", e
	}
	var ok bool
	// 0) bankDenom
	bankDenom, ok = args[0].(string)
	if !ok {
		err = ErrArgTypeValidation("string bankDenom", args[0])
		return
	}
	// 1) amount
	amount, ok = args[1].(*big.Int)
	if !ok {
		err = ErrArgTypeValidation("uint256 amount", args[1])
		return
	}
	// 2) to
	to, ok = args[2].(string)
	if !ok {
		err = ErrArgTypeValidation("string to", args[2])
		return
	}
	return
}

// parse a user-specified address "toStr" that might be bech32 or hex
func parseToAddr(toStr string) (gethcommon.Address, error) {
	// check if bech32 or hex
	if err := eth.ValidateAddress(toStr); err == nil {
		// hex address
		return gethcommon.HexToAddress(toStr), nil
	}
	// else try bech32
	nibAddr, e := sdk.AccAddressFromBech32(toStr)
	if e != nil {
		return gethcommon.Address{}, fmt.Errorf("invalid bech32 or hex address: %w", e)
	}
	return eth.NibiruAddrToEthAddr(nibAddr), nil
}

func (p precompileFunToken) bankMsgSend(
	startResult OnRunStartResult,
	caller gethcommon.Address,
	readOnly bool,
) ([]byte, error) {
	ctx, method, args := startResult.CacheCtx, startResult.Method, startResult.Args
	if err := assertNotReadonlyTx(readOnly, method); err != nil {
		return nil, err
	}

	// parse call: (string to, string denom, uint256 amount)
	toStr, denom, amount, err := parseArgsBankMsgSend(args)
	if err != nil {
		return nil, ErrInvalidArgs(err)
	}

	// parse toStr (bech32 or hex)
	toEthAddr, e := parseToAddr(toStr)
	if e != nil {
		return nil, e
	}
	fromBech32 := eth.EthAddrToNibiruAddr(caller)
	toBech32 := eth.EthAddrToNibiruAddr(toEthAddr)

	// do the bank send
	coin := sdk.NewCoins(sdk.NewCoin(denom, sdkmath.NewIntFromBigInt(amount)))
	bankMsg := &bank.MsgSend{
		FromAddress: fromBech32.String(),
		ToAddress:   toBech32.String(),
		Amount:      coin,
	}
	if err := bankMsg.ValidateBasic(); err != nil {
		return nil, err
	}
	if _, err := bankkeeper.NewMsgServerImpl(p.evmKeeper.Bank).Send(
		sdk.WrapSDKContext(ctx), bankMsg,
	); err != nil {
		return nil, fmt.Errorf("bankMsgSend: %w", err)
	}
	// Return bool success
	return method.Outputs.Pack(true)
}

// getErc20Address implements "IFunToken.getErc20Address"
// It looks up the FunToken mapping by the bank denomination and returns the associated ERC20 address.
//
//	```solidity
//	function getErc20Address(string memory bankDenom) external view returns (address erc20Address);
//	```
func (p precompileFunToken) getErc20Address(
	start OnRunStartResult,
	contract *vm.Contract, // Needed for assertContractQuery
) (bz []byte, err error) {
	method, args, ctx := start.Method, start.Args, start.CacheCtx
	defer func() {
		if err != nil {
			err = ErrMethodCalled(method, err)
		}
	}()
	// Ensure this is called in a read-only context (like view or pure)
	if err := assertContractQuery(contract); err != nil {
		return bz, err
	}

	bankDenom, err := p.parseArgsGetErc20Address(args)
	if err != nil {
		err = ErrInvalidArgs(err)
		return
	}

	// Perform lookup using the BankDenom index
	iterator := p.evmKeeper.FunTokens.Indexes.BankDenom.ExactMatch(ctx, bankDenom)
	mappings := p.evmKeeper.FunTokens.Collect(ctx, iterator)

	var erc20ResultAddress gethcommon.Address // Default to address(0)

	if len(mappings) == 1 {
		erc20ResultAddress = mappings[0].Erc20Addr.Address
	} else if len(mappings) > 1 {
		err = fmt.Errorf(
			"multiple FunToken mappings found for bank denom \"%s\": %d",
			bankDenom, len(mappings),
		)
		return
	} else {
		// No mapping found, erc20ResultAddress remains address(0)
		err = fmt.Errorf(
			"no FunToken mapping found for bank denom \"%s\"", bankDenom,
		)
		return
	}

	// Pack the result (either the found address or address(0))
	return method.Outputs.Pack(erc20ResultAddress)
}

// parseArgsGetErc20Address parses the arguments for the getErc20Address method.
// Expected arguments: (string memory bankDenom)
func (p precompileFunToken) parseArgsGetErc20Address(args []any) (
	bankDenom string,
	err error,
) {
	if e := assertNumArgs(args, 1); e != nil {
		err = e
		return
	}

	argIdx := 0
	bankDenom, ok := args[argIdx].(string)
	if !ok {
		err = ErrArgTypeValidation("string bankDenom", args[argIdx])
		return
	}

	// Validate the bank denomination format using Cosmos SDK validation
	if err = sdk.ValidateDenom(bankDenom); err != nil {
		// maybe it's a tf denom
		tfDenom := tftypes.DenomStr(bankDenom)

		if err = tfDenom.Validate(); err != nil {
			err = fmt.Errorf("invalid bank denomination format: %w", err)
			return
		}
	}

	return bankDenom, nil
}

func parseArgsBankMsgSend(args []any) (toStr, denom string, amount *big.Int, err error) {
	if e := assertNumArgs(args, 3); e != nil {
		err = e
		return
	}
	// 0) to
	var ok bool
	toStr, ok = args[0].(string)
	if !ok {
		err = ErrArgTypeValidation("string to", args[0])
		return
	}

	// 1) denom
	denom, ok = args[1].(string)
	if !ok {
		err = ErrArgTypeValidation("string bankDenom", args[1])
		return
	}

	// 2) amount
	tmpAmount, ok := args[2].(*big.Int)
	if !ok {
		err = ErrArgTypeValidation("uint256 amount", args[2])
		return
	}
	amount = tmpAmount

	return
}
