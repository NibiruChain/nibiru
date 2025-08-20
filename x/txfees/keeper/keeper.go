package keeper

import (
	"fmt"

	sdkioerrors "cosmossdk.io/errors"

	"github.com/cometbft/cometbft/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	evmkeeper "github.com/NibiruChain/nibiru/v2/x/evm/keeper"
	"github.com/NibiruChain/nibiru/v2/x/txfees/types"

	"github.com/NibiruChain/collections"

	gethcommon "github.com/ethereum/go-ethereum/common"
)

type Keeper struct {
	storeKey storetypes.StoreKey
	cdc      codec.BinaryCodec

	accountKeeper types.AccountKeeper
	bankKeeper    types.BankKeeper
	evmKeeper     *evmkeeper.Keeper
	sudoKeeper    types.SudoKeeper

	Params collections.Item[types.Params]
}

var _ types.TxFeesKeeper = (*Keeper)(nil)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	accountKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper,
	evmKeeper *evmkeeper.Keeper,
	sudoKeeper types.SudoKeeper,
	authority string,
) Keeper {
	return Keeper{
		cdc:           cdc,
		storeKey:      storeKey,
		accountKeeper: accountKeeper,
		bankKeeper:    bankKeeper,
		evmKeeper:     evmKeeper,
		sudoKeeper:    sudoKeeper,
		Params:        collections.NewItem(storeKey, 0, collections.ProtoValueEncoder[types.Params](cdc)),
	}
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k Keeper) GetFeeTokensStore(ctx sdk.Context) sdk.KVStore {
	store := ctx.KVStore(k.storeKey)
	return prefix.NewStore(store, types.FeeTokenKey)
}

func (k Keeper) SetFeeTokens(ctx sdk.Context, feetokens []types.FeeToken) error {
	for _, feeToken := range feetokens {
		err := k.SetFeeToken(ctx, feeToken)
		if err != nil {
			return err
		}
	}
	return nil
}

func (k Keeper) SetFeeToken(ctx sdk.Context, feeToken types.FeeToken) error {
	prefixStore := k.GetFeeTokensStore(ctx)

	ok := gethcommon.IsHexAddress(feeToken.Address)
	if !ok {
		return fmt.Errorf("invalid fee token address %s: must be a valid hex address", feeToken.Address)
	}

	bz, err := k.cdc.Marshal(&feeToken)
	if err != nil {
		return err
	}

	prefixStore.Set([]byte(feeToken.Address), bz)
	return nil
}

func (k Keeper) AddFeeToken(ctx sdk.Context, feeToken types.FeeToken) error {
	prefixStore := k.GetFeeTokensStore(ctx)

	// Validate address format first
	if !gethcommon.IsHexAddress(feeToken.Address) {
		return fmt.Errorf("invalid fee token address %s: must be a valid hex address", feeToken.Address)
	}

	// Check if token already exists
	if prefixStore.Has([]byte(feeToken.Address)) {
		return fmt.Errorf("fee token with address %s already exists", feeToken.Address)
	}

	return k.SetFeeToken(ctx, feeToken)
}

func (k Keeper) RemoveFeeToken(ctx sdk.Context, address string) error {
	prefixStore := k.GetFeeTokensStore(ctx)
	if !gethcommon.IsHexAddress(address) {
		return fmt.Errorf("invalid fee token address %s: must be a valid hex address", address)
	}

	// Check if token already exists
	if !prefixStore.Has([]byte(address)) {
		return fmt.Errorf("fee token with address %s not exists", address)
	}

	store := k.GetFeeTokensStore(ctx)
	store.Delete([]byte(address))
	return nil
}

func (k Keeper) GetFeeToken(ctx sdk.Context, address string) (types.FeeToken, error) {
	prefixStore := k.GetFeeTokensStore(ctx)

	if !prefixStore.Has([]byte(address)) {
		return types.FeeToken{}, sdkioerrors.Wrapf(types.ErrInvalidFeeToken, "%s", address)
	}
	bz := prefixStore.Get([]byte(address))

	var feeToken types.FeeToken
	if err := k.cdc.Unmarshal(bz, &feeToken); err != nil {
		return types.FeeToken{}, err
	}

	return feeToken, nil
}

func (k Keeper) GetFeeTokens(ctx sdk.Context) (feetokens []types.FeeToken) {
	prefixStore := k.GetFeeTokensStore(ctx)

	// this entire store just contains FeeTokens, so iterate over all entries.
	iterator := prefixStore.Iterator(nil, nil)
	defer iterator.Close()

	feeTokens := []types.FeeToken{}

	for ; iterator.Valid(); iterator.Next() {
		feeToken := types.FeeToken{}

		err := k.cdc.Unmarshal(iterator.Value(), &feeToken)
		if err != nil {
			panic(err)
		}

		feeTokens = append(feeTokens, feeToken)
	}
	return feeTokens
}

func (k Keeper) GetBaseToken(ctx sdk.Context, name string) (types.FeeToken, error) {
	feetokens := k.GetFeeTokens(ctx)
	for _, feeToken := range feetokens {
		if feeToken.Name == name {
			return feeToken, nil
		}
	}
	return types.FeeToken{}, sdkioerrors.Wrapf(types.ErrInvalidFeeToken, "base token %s not found", name)
}

func (k Keeper) GetParams(ctx sdk.Context) (params types.Params, err error) {
	params, err = k.Params.Get(ctx)
	if err != nil {
		return types.Params{}, sdkioerrors.Wrapf(err, "failed to get params")
	}

	return params, nil
}

func (k Keeper) SetParams(ctx sdk.Context, params types.Params) error {
	k.Params.Set(ctx, params)
	return nil
}
