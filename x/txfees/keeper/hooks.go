package keeper

import (
	"math/big"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	gethcore "github.com/ethereum/go-ethereum/core/types"

	sdkioerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/NibiruChain/nibiru/v2/eth"
	epochtypes "github.com/NibiruChain/nibiru/v2/x/epochs/types"
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
	out, err := h.K.evmKeeper.GetErc20Balance(ctx, feeCollector, contract)
	if err != nil {
		return
	}

	input, err := embeds.SmartContract_WNIBI.ABI.Pack(
		"withdraw", out,
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
	txConfig := h.K.evmKeeper.TxConfig(ctx, gethcommon.Hash{})
	stateDB := h.K.evmKeeper.Bank.StateDB
	if stateDB == nil {
		stateDB = h.K.evmKeeper.NewStateDB(ctx, txConfig)
	}
	defer func() {
		h.K.evmKeeper.Bank.StateDB = nil
	}()
	evmObj := h.K.evmKeeper.NewEVM(ctx, evmMsg, h.K.evmKeeper.GetEVMConfig(ctx), nil /*tracer*/, stateDB)

	resp, err := h.K.evmKeeper.CallContractWithInput(ctx, evmObj, feeCollector, &contract, false /*commit*/, input, evmKeeper.GetCallGasWithLimit(ctx, evmKeeper.Erc20GasLimitExecute))
	if err != nil {
		panic(sdkioerrors.Wrap(err, "failed to call WNIBI contract withdraw"))
	}

	if resp.Failed() {
		panic(sdkioerrors.Wrap(err, "failed to call WNIBI contract withdraw with VM error"))
	}

	if err := stateDB.Commit(); err != nil {
		panic(sdkioerrors.Wrap(err, "failed to commit stateDB"))
	}

	acc := h.K.accountKeeper.GetModuleAccount(ctx, authtypes.FeeCollectorName)
	if err := acc.SetSequence(nonce); err != nil {
		panic(sdkioerrors.Wrapf(err, "failed to set sequence to %d", nonce))
	}
}
