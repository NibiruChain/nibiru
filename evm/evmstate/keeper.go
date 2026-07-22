package evmstate

// Copyright (c) 2023-2024 Nibi, Inc.

import (
	"bytes"
	"fmt"
	"math/big"
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"

	"github.com/cometbft/cometbft/libs/log"

	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/codec"
	storetypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/store/types"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"

	bankkeeper "github.com/NibiruChain/nibiru/v2/x/bank/keeper"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/evm"
)

type Keeper struct {
	cdc codec.BinaryCodec
	// storeKey: For persistent storage of EVM state.
	storeKey storetypes.StoreKey
	// transientKey: Store key that resets every block during Commit
	transientKey storetypes.StoreKey

	// EvmState isolates the key-value stores (collections) for the x/evm module.
	EvmState EvmState

	// FunTokens isolates the key-value stores (collections) for fungible token
	// mappings.
	FunTokens FunTokenState

	// the address capable of executing a MsgUpdateParams message. Typically,
	// this should be the x/gov module account.
	authority sdk.AccAddress

	Bank          *NibiruBankKeeper
	accountKeeper evm.AccountKeeper
	stakingKeeper evm.StakingKeeper
	SudoKeeper    evm.SudoKeeper

	// tracer: Configures the output type for a geth `vm.EVMLogger`. Tracer types
	// include "access_list", "json", "struct", and "markdown". If any other
	// value is used, a no operation tracer is set.
	tracer string

	// pendingTxCounts points at the current epoch's per-sender admission map.
	// Values are *atomic.Uint64. EndBlock stores a fresh map; the old one is GC'd.
	// Never consensus state.
	pendingTxCounts atomic.Pointer[sync.Map]
}

func (k *Keeper) BK() bankkeeper.Keeper {
	return k.Bank
}

var (
	_ bankkeeper.NibiruExtKeeper = (*NibiruBankKeeper)(nil)
	_ bankkeeper.Keeper          = (*NibiruBankKeeper)(nil)
)

type NibiruBankKeeper struct {
	bankkeeper.BaseKeeper
}

// NewKeeper is a constructor for an x/evm [Keeper]. This function is necessary
// because the [Keeper] struct has private fields.
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey, transientKey storetypes.StoreKey,
	authority sdk.AccAddress,
	accKeeper evm.AccountKeeper,
	bankKeeper *NibiruBankKeeper,
	stakingKeeper evm.StakingKeeper,
	sudoKeeper evm.SudoKeeper,
	tracer string,
) Keeper {
	if err := sdk.VerifyAddressFormat(authority); err != nil {
		panic(err)
	}

	k := Keeper{
		cdc:           cdc,
		storeKey:      storeKey,
		transientKey:  transientKey,
		authority:     authority,
		EvmState:      NewEvmState(cdc, storeKey, transientKey),
		FunTokens:     NewFunTokenState(cdc, storeKey),
		accountKeeper: accKeeper,
		Bank:          bankKeeper,
		stakingKeeper: stakingKeeper,
		SudoKeeper:    sudoKeeper,
		tracer:        tracer,
	}
	k.pendingTxCounts.Store(&sync.Map{})
	return k
}

// GetWeiBalance: Used in the EVM Ante Handler,
// "github.com/NibiruChain/nibiru/v2/evm/evmante": Load account's balance of gas
// tokens for EVM execution in EVM denom units.
func (k *Keeper) GetWeiBalance(ctx sdk.Context, addr gethcommon.Address) (balance *uint256.Int) {
	nibiruAddr := sdk.AccAddress(addr.Bytes())
	return k.Bank.GetWeiBalance(ctx, nibiruAddr)
}

func (k Keeper) EthChainID(ctx sdk.Context) *big.Int {
	return appconst.GetEthChainID(ctx.ChainID())
}

// BaseFeeMicronibiPerGas returns the gas base fee in units of the EVM denom. Note
// that this function is currently constant/stateless.
func (k Keeper) BaseFeeMicronibiPerGas(_ sdk.Context) *big.Int {
	// TODO: (someday maybe):  Consider making base fee dynamic based on
	// congestion in the previous block.
	return evm.BASE_FEE_MICRONIBI
}

// BaseFeeWeiPerGas is the same as BaseFeeMicronibiPerGas, except its in units of
// wei per gas.
func (k Keeper) BaseFeeWeiPerGas(_ sdk.Context) *big.Int {
	return evm.NativeToWei(k.BaseFeeMicronibiPerGas(sdk.Context{}))
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", evm.ModuleName)
}

// IsSimulation checks if the context is a simulation context.
func IsSimulation(ctx sdk.Context) bool {
	if val := ctx.Value(evm.CtxKeyEvmSimulation); val != nil {
		if simulation, ok := val.(bool); ok && simulation {
			return true
		}
	}
	return false
}

// IsDeliverTx checks if we're in DeliverTx, NOT in CheckTx, ReCheckTx, or simulation
func IsDeliverTx(ctx sdk.Context) bool {
	return !ctx.IsCheckTx() && !ctx.IsReCheckTx() && !IsSimulation(ctx)
}

func (sdb SDB) IsDeliverTx() bool {
	return IsDeliverTx(sdb.Ctx())
}

// IsReCheckTxOnly is true only in ABCI ReCheckTx (ctx.IsReCheckTx()).
// New CheckTx has IsCheckTx() && !IsReCheckTx(); ReCheckTx has both flags set
// because the SDK sets checkTx=true whenever recheckTx=true.
func IsReCheckTxOnly(ctx sdk.Context) bool {
	return ctx.IsReCheckTx()
}

func (sdb SDB) IsReCheckTxOnly() bool {
	return IsReCheckTxOnly(sdb.Ctx())
}

func (k *Keeper) pendingTxCountMap() *sync.Map {
	return k.pendingTxCounts.Load()
}

// pendingTxCount returns the per-sender admission counter, creating it if needed.
func (k *Keeper) pendingTxCount(addr gethcommon.Address) *atomic.Uint64 {
	m := k.pendingTxCountMap()
	if v, ok := m.Load(addr); ok {
		return v.(*atomic.Uint64)
	}
	c := new(atomic.Uint64)
	actual, loaded := m.LoadOrStore(addr, c)
	if loaded {
		return actual.(*atomic.Uint64)
	}
	return c
}

// PendingTxCount returns the node-local New CheckTx admission count for addr.
func (k *Keeper) PendingTxCount(addr gethcommon.Address) uint64 {
	if v, ok := k.pendingTxCountMap().Load(addr); ok {
		return v.(*atomic.Uint64).Load()
	}
	return 0
}

// IncrementPendingTxCount increments and returns the new pending admission count.
func (k *Keeper) IncrementPendingTxCount(addr gethcommon.Address) uint64 {
	return k.pendingTxCount(addr).Add(1)
}

// ResetPendingTxCount replaces the pending admission map with an empty one.
// The previous map is left for GC; in-flight increments on the old pointer are
// discarded with that epoch and do not affect the new window.
func (k *Keeper) ResetPendingTxCount() {
	k.pendingTxCounts.Store(&sync.Map{})
}

func (k *Keeper) ImportGenesisAccount(ctx sdk.Context, account evm.GenesisAccount) (err error) {
	address := gethcommon.HexToAddress(account.Address)
	accAddress := sdk.AccAddress(address.Bytes())
	// check that the EVM balance the matches the account balance
	acc := k.accountKeeper.GetAccount(ctx, accAddress)
	if acc == nil {
		err = fmt.Errorf("account not found for address %s", account.Address)
		return
	}

	ethAcct, ok := acc.(eth.EthAccountI)
	if !ok {
		err = fmt.Errorf("account %s must be an EthAccount interface, got %T",
			account.Address, acc,
		)
		return
	}
	code := gethcommon.Hex2Bytes(account.Code)
	codeHash := crypto.Keccak256Hash(code)

	// we ignore the empty Code hash checking, see ethermint PR#1234
	if len(account.Code) != 0 && !bytes.Equal(ethAcct.GetCodeHash().Bytes(), codeHash.Bytes()) {
		err = fmt.Errorf(
			`the evm state code doesn't match with the codehash: account: "%s" , evm state codehash: "%v", ethAccount codehash: "%v", evm state code: "%s"`,
			account.Address, codeHash.Hex(), ethAcct.GetCodeHash().Hex(), account.Code)
		return
	}
	k.SetCode(ctx, codeHash.Bytes(), code)

	for _, storage := range account.Storage {
		k.SetState(ctx, address, gethcommon.HexToHash(storage.Key), gethcommon.HexToHash(storage.Value).Bytes())
	}

	return nil
}
