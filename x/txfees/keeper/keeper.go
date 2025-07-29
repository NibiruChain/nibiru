package keeper

import (
	"fmt"

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
