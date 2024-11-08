package precompile

import (
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	wasm "github.com/CosmWasm/wasmd/x/wasm/types"
)

// WasmBankCoin is a naked struct for the "BankCoin" type from Wasm.sol.
// The ABI parser requires an unnamed strict, so this type is only used in tests.
type WasmBankCoin struct {
	Denom  string   `json:"denom"`
	Amount *big.Int `json:"amount"`
}

// Parses [sdk.Coins] from a "BankCoin[]" solidity argument:
//
//	```solidity
//	    BankCoin[] memory funds
//	```
func parseFundsArg(arg any) (funds sdk.Coins, err error) {
	if arg == nil {
		return funds, nil
	}

	raw, ok := arg.([]struct {
		Denom  string   `json:"denom"`
		Amount *big.Int `json:"amount"`
	})

	if !ok {
		return funds, ErrArgTypeValidation("BankCoin[] funds", arg)
	}

	for _, coin := range raw {
		funds = append(
			funds,
			// Favor the sdk.Coin constructor over sdk.NewCoin because sdk.NewCoin
			// is not panic-safe. Validation will be handled when the coin is used
			// as an argument during the execution of a transaction.
			sdk.Coin{
				Denom:  coin.Denom,
				Amount: sdk.NewIntFromBigInt(coin.Amount),
			},
		)
	}
	return funds, nil
}

// Parses [sdk.AccAddress] from a "string" solidity argument:
func parseArgContractAddr(arg any) (addr sdk.AccAddress, err error) {
	addrStr, ok := arg.(string)
	if !ok {
		err = ErrArgTypeValidation("string contractAddr", arg)
		return
	}
	addr, err = sdk.AccAddressFromBech32(addrStr)
	if err != nil {
		err = fmt.Errorf("%w: %s",
			ErrArgTypeValidation("string contractAddr", arg), err,
		)
		return
	}
	return addr, nil
}

func (p precompileWasm) parseArgsWasmInstantiate(args []any, sender string) (
	txMsg wasm.MsgInstantiateContract,
	err error,
) {
	if e := assertNumArgs(args, 5); e != nil {
		err = e
		return
	}

	argIdx := 0
	admin, ok := args[argIdx].(string)
	if !ok {
		err = ErrArgTypeValidation("string admin", args[argIdx])
		return
	}

	argIdx++
	codeID, ok := args[argIdx].(uint64)
	if !ok {
		err = ErrArgTypeValidation("uint64 codeID", args[argIdx])
		return
	}

	argIdx++
	msgArgs, ok := args[argIdx].([]byte)
	if !ok {
		err = ErrArgTypeValidation("bytes msgArgs", args[argIdx])
		return
	}

	argIdx++
	label, ok := args[argIdx].(string)
	if !ok {
		err = ErrArgTypeValidation("string label", args[argIdx])
		return
	}

	argIdx++
	funds, e := parseFundsArg(args[argIdx])
	if e != nil {
		err = e
		return
	}

	txMsg = wasm.MsgInstantiateContract{
		Sender: sender,
		CodeID: codeID,
		Label:  label,
		Msg:    msgArgs,
		Funds:  funds,
	}
	if len(admin) > 0 {
		txMsg.Admin = admin
	}
	return txMsg, txMsg.ValidateBasic()
}

func (p precompileWasm) parseArgsWasmExecute(args []any) (
	wasmContract sdk.AccAddress,
	msgArgs []byte,
	funds sdk.Coins,
	err error,
) {
	if e := assertNumArgs(args, 3); e != nil {
		err = e
		return
	}

	// contract address
	argIdx := 0
	contractAddrStr, ok := args[argIdx].(string)
	if !ok {
		err = ErrArgTypeValidation("string contractAddr", args[argIdx])
		return
	}
	contractAddr, err := sdk.AccAddressFromBech32(contractAddrStr)
	if err != nil {
		err = fmt.Errorf("%w: %s",
			ErrArgTypeValidation("string contractAddr", args[argIdx]), err,
		)
		return
	}

	// msg args
	argIdx++
	msgArgs, ok = args[argIdx].([]byte)
	if !ok {
		err = ErrArgTypeValidation("bytes msgArgs", args[argIdx])
		return
	}
	msgArgsCopy := wasm.RawContractMessage(msgArgs)
	if e := msgArgsCopy.ValidateBasic(); e != nil {
		err = ErrArgTypeValidation(e.Error(), args[argIdx])
		return
	}

	// funds
	argIdx++
	funds, e := parseFundsArg(args[argIdx])
	if e != nil {
		err = ErrArgTypeValidation(e.Error(), args[argIdx])
		return
	}

	return contractAddr, msgArgs, funds, nil
}

func (p precompileWasm) parseArgsWasmQuery(args []any) (
	wasmContract sdk.AccAddress,
	req wasm.RawContractMessage,
	err error,
) {
	if e := assertNumArgs(args, 2); e != nil {
		err = e
		return
	}

	argsIdx := 0
	wasmContract, e := parseArgContractAddr(args[argsIdx])
	if e != nil {
		err = e
		return
	}

	argsIdx++
	reqBz, ok := args[argsIdx].([]byte)
	if !ok {
		err = ErrArgTypeValidation("bytes req", args[argsIdx])
		return
	}
	req = wasm.RawContractMessage(reqBz)
	if e := req.ValidateBasic(); e != nil {
		err = e
		return
	}

	return wasmContract, req, nil
}

func (p precompileWasm) parseArgsWasmExecuteMulti(args []any) (
	wasmExecMsgs []struct {
		ContractAddr string `json:"contractAddr"`
		MsgArgs      []byte `json:"msgArgs"`
		Funds        []struct {
			Denom  string   `json:"denom"`
			Amount *big.Int `json:"amount"`
		} `json:"funds"`
	},
	err error,
) {
	if e := assertNumArgs(args, 1); e != nil {
		err = e
		return
	}

	arg := args[0]
	execMsgs, ok := arg.([]struct {
		ContractAddr string `json:"contractAddr"`
		MsgArgs      []byte `json:"msgArgs"`
		Funds        []struct {
			Denom  string   `json:"denom"`
			Amount *big.Int `json:"amount"`
		} `json:"funds"`
	})
	if !ok {
		err = ErrArgTypeValidation("BankCoin[] funds", arg)
		return
	}

	return execMsgs, nil
}
