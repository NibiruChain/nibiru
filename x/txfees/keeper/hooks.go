package keeper

import (
	"fmt"
	"math/big"

	sdkioerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	gethcore "github.com/ethereum/go-ethereum/core/types"

	"github.com/NibiruChain/nibiru/v2/eth"
	epochtypes "github.com/NibiruChain/nibiru/v2/x/epochs/types"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
	evmKeeper "github.com/NibiruChain/nibiru/v2/x/evm/keeper"
	"github.com/NibiruChain/nibiru/v2/x/txfees/types"
)

var _ epochtypes.EpochHooks = Hooks{}

// Hooks implements module-speecific calls that will occur in the ABCI
// BeginBlock logic.
func (k Keeper) Hooks() Hooks {
	return Hooks{k}
}

// Hooks implements module-specific calls ([epochstypes.EpochHooks]) that will
// occur at the end of every epoch. Hooks is meant for use with
// `EpochsKeeper.SetHooks`. These functions run outside the normal body of
// transactions.
type Hooks struct {
	K Keeper
}

// BeforeEpochStart is a hook that runs just prior to the start of a new epoch.
func (h Hooks) BeforeEpochStart(ctx sdk.Context, epochIdentifier string, epochNumber uint64) {
	// Perform no operations; we don't need to do anything here
	_, _, _ = ctx, epochIdentifier, epochNumber
}

// AfterEpochEnd convert all fees collected in the previous epoch to the base token
func (h Hooks) AfterEpochEnd(ctx sdk.Context, epochIdentifier string, _ uint64) {
	feeCollector := eth.NibiruAddrToEthAddr(h.K.accountKeeper.GetModuleAddress(types.ModuleName))

	unusedBigInt := big.NewInt(0)
	feeTokens := h.K.GetFeeTokens(ctx)

	for _, feeToken := range feeTokens {
		if feeToken.TokenType == types.FeeTokenType_FEE_TOKEN_TYPE_CONVERTIBLE {
			h.withdrawFeeToken(ctx, gethcommon.HexToAddress(feeToken.Address), feeCollector, unusedBigInt)
		}
		if feeToken.TokenType == types.FeeTokenType_FEE_TOKEN_TYPE_SWAPPABLE {
			// TODO: implement swappable fee token withdrawal
			return
		}
	}
	// TODO: events
}

func (h Hooks) withdrawFeeToken(ctx sdk.Context, contract, feeCollector gethcommon.Address, unusedBigInt *big.Int) {
	txConfig := h.K.evmKeeper.TxConfig(ctx, gethcommon.Hash{})
	stateDB := h.K.evmKeeper.Bank.StateDB
	if stateDB == nil {
		stateDB = h.K.evmKeeper.NewStateDB(ctx, txConfig)
	}
	defer func() {
		h.K.evmKeeper.Bank.StateDB = nil
	}()
	evmObj := h.K.evmKeeper.NewEVM(ctx, MOCK_GETH_MESSAGE, h.K.evmKeeper.GetEVMConfig(ctx), nil /*tracer*/, stateDB)

	input, err := embeds.SmartContract_WNIBI.ABI.Pack("balanceOf", feeCollector)
	if err != nil {
		panic(err)
	}
	evmResp, err := h.K.evmKeeper.CallContractWithInput(
		ctx,
		evmObj,
		evm.EVM_MODULE_ADDRESS,
		&contract,
		false,
		input,
		evmKeeper.GetCallGasWithLimit(ctx, evmKeeper.Erc20GasLimitQuery),
	)
	if err != nil {
		panic(err)
	}

	if evmResp.Failed() {
		panic(fmt.Errorf("failed to get balance of %s with VM error: %s", feeCollector, evmResp.VmError))
	}

	if len(evmResp.Ret) == 0 {
		return // No fees collected in the previous epoch
	}

	// Unpack the response to get the balance of the fee collector
	erc20BigInt := new(evmKeeper.ERC20BigInt)
	err = embeds.SmartContract_WNIBI.ABI.UnpackIntoInterface(
		erc20BigInt, "balanceOf", evmResp.Ret,
	)
	if err != nil {
		panic(err)
	}

	input, err = embeds.SmartContract_WNIBI.ABI.Pack(
		"withdraw", erc20BigInt.Value,
	)
	if err != nil {
		panic(sdkioerrors.Wrap(err, "failed to pack ABI args for withdraw"))
	}

	nonce := h.K.evmKeeper.GetAccNonce(ctx, feeCollector)
	evmMsg := core.Message{
		To:               &contract,
		From:             feeCollector,
		Nonce:            nonce,
		Value:            unusedBigInt, // amount
		GasLimit:         5_500_000,
		GasPrice:         unusedBigInt,
		GasFeeCap:        unusedBigInt,
		GasTipCap:        unusedBigInt,
		Data:             input,
		AccessList:       gethcore.AccessList{},
		SkipNonceChecks:  false,
		SkipFromEOACheck: false,
	}
	evmObj = h.K.evmKeeper.NewEVM(ctx, evmMsg, h.K.evmKeeper.GetEVMConfig(ctx), nil /*tracer*/, stateDB)

	resp, err := h.K.evmKeeper.CallContractWithInput(ctx, evmObj, feeCollector, &contract, false /*commit*/, input, evmKeeper.GetCallGasWithLimit(ctx, evmKeeper.Erc20GasLimitExecute))
	if err != nil {
		panic(sdkioerrors.Wrap(err, "failed to call WNIBI contract transfer"))
	}

	if resp.Failed() {
		panic(sdkioerrors.Wrap(err, "failed to call WNIBI contract transfer with VM error"))
	}

	if err := stateDB.Commit(); err != nil {
		panic(sdkioerrors.Wrap(err, "failed to commit stateDB"))
	}

	acc := h.K.accountKeeper.GetModuleAccount(ctx, types.FeeCollectorName)
	if err := acc.SetSequence(nonce); err != nil {
		panic(sdkioerrors.Wrapf(err, "failed to set sequence to %d", nonce))
	}
}

var MOCK_GETH_MESSAGE = core.Message{
	To:               nil,
	From:             evm.EVM_MODULE_ADDRESS,
	Nonce:            0,
	Value:            evm.Big0, // amount
	GasLimit:         0,
	GasPrice:         evm.Big0,
	GasFeeCap:        evm.Big0,
	GasTipCap:        evm.Big0,
	Data:             []byte{},
	AccessList:       gethcore.AccessList{},
	SkipNonceChecks:  false,
	SkipFromEOACheck: false,
}
