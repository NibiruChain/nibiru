package evmstate

import (
	"fmt"
	"math/big"

	sdkioerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bank "github.com/cosmos/cosmos-sdk/x/bank/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
)

func (k *Keeper) createFunTokenFromCoin(
	ctx sdk.Context, bankDenom string, allowZeroDecimals bool,
) (funtoken *evm.FunToken, evmResp *evm.MsgEthereumTxResponse, err error) {
	// 1 | Coin already registered with FunToken?
	if funtokens := k.FunTokens.Collect(ctx, k.FunTokens.Indexes.BankDenom.ExactMatch(ctx, bankDenom)); len(funtokens) > 0 {
		return nil, nil, fmt.Errorf("funtoken mapping already created for bank denom \"%s\"", bankDenom)
	}

	// 2 | Check for denom metadata in bank state
	bankMetadata, isFound := k.Bank.GetDenomMetaData(ctx, bankDenom)
	if !isFound {
		return nil, nil, fmt.Errorf("bank coin denom should have bank metadata for denom \"%s\"", bankDenom)
	}

	// 3 | deploy ERC20 for metadata
	var erc20Addr gethcommon.Address
	erc20Addr, evmResp, err = k.deployERC20ForBankCoin(
		ctx,
		bankMetadata,
		allowZeroDecimals,
	)
	if err != nil {
		return nil, evmResp, sdkioerrors.Wrap(err, "failed to deploy ERC20 for bank coin")
	}

	// 4 | ERC20 already registered with FunToken?
	if funtokens := k.FunTokens.Collect(ctx, k.FunTokens.Indexes.ERC20Addr.ExactMatch(ctx, erc20Addr)); len(funtokens) > 0 {
		return nil, evmResp, fmt.Errorf("funtoken mapping already created for ERC20 \"%s\"", erc20Addr.Hex())
	}

	// 5 | Officially create the funtoken mapping
	funtoken = &evm.FunToken{
		Erc20Addr: eth.EIP55Addr{
			Address: erc20Addr,
		},
		BankDenom:      bankDenom,
		IsMadeFromCoin: true,
	}

	return funtoken, evmResp, k.FunTokens.SafeInsert(
		ctx, erc20Addr,
		funtoken.BankDenom,
		funtoken.IsMadeFromCoin,
	)
}

func (k *Keeper) deployERC20ForBankCoin(
	ctx sdk.Context, bankCoin bank.Metadata, allowZeroDecimals bool,
) (erc20Addr gethcommon.Address, evmResp *evm.MsgEthereumTxResponse, err error) {
	erc20Addr = crypto.CreateAddress(evm.EVM_MODULE_ADDRESS, k.GetAccNonce(ctx, evm.EVM_MODULE_ADDRESS))

	erc20Info, err := evm.ValidateFunTokenBankMetadata(bankCoin, allowZeroDecimals)
	if err != nil {
		err = fmt.Errorf(`metadata unsuitable to create FunToken mapping for Bank Coin "%s": %w. Fix this with "MsgSudoSetDenomMetadata" or "MsgSetDenomMetadata"`, bankCoin.Base, err)
		return
	}

	// pass empty method name to deploy the contract
	packedArgs, err := embeds.SmartContract_ERC20MinterWithMetadataUpdates.ABI.Pack(
		"", erc20Info.Name, erc20Info.Symbol, erc20Info.Decimals,
	)
	if err != nil {
		return gethcommon.Address{}, nil, sdkioerrors.Wrap(err, "failed to pack ABI args")
	}
	input := append(embeds.SmartContract_ERC20MinterWithMetadataUpdates.Bytecode, packedArgs...)

	unusedBigInt := big.NewInt(0)
	evmMsg := core.Message{
		To:               nil,
		From:             evm.EVM_MODULE_ADDRESS,
		Nonce:            k.GetAccNonce(ctx, evm.EVM_MODULE_ADDRESS),
		Value:            unusedBigInt, // amount
		GasLimit:         Erc20GasLimitDeploy,
		GasPrice:         unusedBigInt,
		GasFeeCap:        unusedBigInt,
		GasTipCap:        unusedBigInt,
		Data:             input,
		AccessList:       gethcore.AccessList{},
		SkipNonceChecks:  false,
		SkipFromEOACheck: false,
	}
	evmCfg := k.GetEVMConfig(ctx)
	sdb := k.NewSDB(ctx, k.TxConfig(ctx, ctx.EvmTxHash()))
	evmObj := k.NewEVM(ctx, evmMsg, evmCfg, nil /*tracer*/, sdb)
	evmResp, err = k.CallContract(
		evmObj, evm.EVM_MODULE_ADDRESS, nil, input, Erc20GasLimitDeploy,
		evm.COMMIT_ETH_TX, /*commit*/
		nil,
	)
	if err != nil {
		return gethcommon.Address{}, evmResp, sdkioerrors.Wrap(err, "failed to deploy ERC20 contract")
	}

	return erc20Addr, evmResp, nil
}
