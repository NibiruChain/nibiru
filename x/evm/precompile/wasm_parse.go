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

func parseSdkCoins(unparsed []struct {
	Denom  string   `json:"denom"`
	Amount *big.Int `json:"amount"`
},
) sdk.Coins {
	parsed := sdk.Coins{}
	for _, coin := range unparsed {
		parsed = append(
			parsed,
			sdk.NewCoin(coin.Denom, sdk.NewIntFromBigInt(coin.Amount)),
		)
	}
	return parsed
}

// Parses [sdk.Coins] from a "BankCoin[]" solidity argument:
//
//	```solidity
//	    BankCoin[] memory funds
//	```
func parseFundsArg(arg any) (funds sdk.Coins, err error) {
	bankCoinsUnparsed, ok := arg.([]struct {
		Denom  string   `json:"denom"`
		Amount *big.Int `json:"amount"`
	})
	switch {
	case arg == nil:
		bankCoinsUnparsed = []struct {
			Denom  string   `json:"denom"`
			Amount *big.Int `json:"amount"`
		}{}
	case !ok:
		err = ErrArgTypeValidation("BankCoin[] funds", arg)
		return
	case ok:
		// Type assertion succeeded
	}
	funds = parseSdkCoins(bankCoinsUnparsed)
	return
}

// Parses [sdk.AccAddress] from a "string" solidity argument:
func parseContraAddrArg(arg any) (addr sdk.AccAddress, err error) {
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

func (p precompileWasm) parseInstantiateArgs(args []any, sender string) (
	txMsg wasm.MsgInstantiateContract,
	err error,
) {
	if e := assertNumArgs(len(args), 5); e != nil {
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

func (p precompileWasm) parseExecuteArgs(args []any) (
	wasmContract sdk.AccAddress,
	msgArgs []byte,
	funds sdk.Coins,
	err error,
) {
	wantArgsLen := 3
	if len(args) != wantArgsLen {
		err = fmt.Errorf("expected %d arguments but got %d", wantArgsLen, len(args))
		return
	}

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

	argIdx++
	msgArgs, ok = args[argIdx].([]byte)
	if !ok {
		err = ErrArgTypeValidation("bytes msgArgs", args[argIdx])
		return
	}

	argIdx++
	funds, e := parseFundsArg(args[argIdx])
	if e != nil {
		err = e
		return
	}

	return contractAddr, msgArgs, funds, nil
}

func (p precompileWasm) parseQueryArgs(args []any) (
	contractAddr sdk.AccAddress,
	req wasm.RawContractMessage,
	err error,
) {
	wantArgsLen := 2
	if len(args) != wantArgsLen {
		err = fmt.Errorf("expected %d arguments but got %d", wantArgsLen, len(args))
		return
	}

	argsIdx := 0
	contractAddr, e := parseContraAddrArg(args[argsIdx])
	if e != nil {
		err = e
		return
	}

	argsIdx++
	reqBz := args[argsIdx].([]byte)
	req = wasm.RawContractMessage(reqBz)
	if e := req.ValidateBasic(); e != nil {
		err = e
		return
	}

	return contractAddr, req, nil
}

func (p precompileWasm) parseExecuteMultiArgs(args []any) (
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
	wantArgsLen := 1
	if len(args) != wantArgsLen {
		err = fmt.Errorf("expected %d arguments but got %d", wantArgsLen, len(args))
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
