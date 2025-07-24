package keeper

import (
	"fmt"

	"cosmossdk.io/math"

	"github.com/cometbft/cometbft/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	evmkeeper "github.com/NibiruChain/nibiru/v2/x/evm/keeper"
	"github.com/NibiruChain/nibiru/v2/x/txfees/types"

	gethcommon "github.com/ethereum/go-ethereum/common"
)

type Keeper struct {
	storeKey storetypes.StoreKey
	cdc      codec.BinaryCodec

	accountKeeper types.AccountKeeper
	bankKeeper    types.BankKeeper
	evmKeeper     *evmkeeper.Keeper

	authority string // authority is the x/txfees module authority, which is used to update the fee token.
}

var _ types.TxFeesKeeper = (*Keeper)(nil)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	accountKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper,
	evmKeeper *evmkeeper.Keeper,
	authority string,
) Keeper {
	return Keeper{
		cdc:           cdc,
		storeKey:      storeKey,
		accountKeeper: accountKeeper,
		bankKeeper:    bankKeeper,
		evmKeeper:     evmKeeper,
		authority:     authority,
	}
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k Keeper) GetBaseDenom(ctx sdk.Context) (denom string, err error) {
	store := ctx.KVStore(k.storeKey)

	if !store.Has(types.BaseDenomKey) {
		return "", types.ErrNoBaseDenom
	}

	bz := store.Get(types.BaseDenomKey)

	return string(bz), nil
}

// SetBaseDenom sets the base fee denom for the chain. Should only be used once.
func (k Keeper) SetBaseDenom(ctx sdk.Context, denom string) error {
	store := ctx.KVStore(k.storeKey)

	err := sdk.ValidateDenom(denom)
	if err != nil {
		return err
	}

	store.Set(types.BaseDenomKey, []byte(denom))
	return nil
}

func (k Keeper) SetFeeToken(ctx sdk.Context, feeToken types.FeeToken) error {
	ok := gethcommon.IsHexAddress(feeToken.Address)
	if !ok {
		return fmt.Errorf("invalid fee token address %s: must be a valid hex address", feeToken.Address)
	}

	store := ctx.KVStore(k.storeKey)
	bz, err := k.cdc.Marshal(&feeToken)
	if err != nil {
		return err
	}

	store.Set(types.FeeTokenKey, bz)
	return nil
}

func (k Keeper) GetFeeToken(ctx sdk.Context) (types.FeeToken, error) {
	store := ctx.KVStore(k.storeKey)

	bz := store.Get(types.FeeTokenKey)
	if bz == nil {
		return types.FeeToken{}, nil
	}

	var feeToken types.FeeToken
	if err := k.cdc.Unmarshal(bz, &feeToken); err != nil {
		return types.FeeToken{}, err
	}

	return feeToken, nil
}

// ConvertToBaseToken converts a fee amount in a whitelisted fee token to the base fee token amount.
func (k Keeper) ConvertToBaseToken(ctx sdk.Context, inputFee sdk.Coin) (sdk.Coin, error) {
	baseDenom, err := k.GetBaseDenom(ctx)
	if err != nil {
		return sdk.Coin{}, err
	}

	if inputFee.Denom == baseDenom {
		return inputFee, nil
	}

	// return 1:1
	return sdk.NewCoin(baseDenom, math.OneInt()), nil
}
