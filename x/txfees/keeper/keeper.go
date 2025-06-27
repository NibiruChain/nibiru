package keeper

import (
	"fmt"

	"github.com/cosmos/gogoproto/proto"

	sdkioerrors "cosmossdk.io/errors"
	"cosmossdk.io/math"

	"github.com/cometbft/cometbft/libs/log"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/x/txfees/types"
)

type Keeper struct {
	storeKey storetypes.StoreKey

	accountKeeper      types.AccountKeeper
	bankKeeper         types.BankKeeper
	protorevKeeper     types.ProtorevKeeper
	distributionKeeper types.DistributionKeeper

	WhitelistedFeeTokenSetters []string
}

var _ types.TxFeesKeeper = (*Keeper)(nil)

func NewKeeper(
	accountKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper,
	storeKey storetypes.StoreKey,
	distributionKeeper types.DistributionKeeper,
) Keeper {
	return Keeper{
		accountKeeper:      accountKeeper,
		bankKeeper:         bankKeeper,
		storeKey:           storeKey,
		distributionKeeper: distributionKeeper,
	}
}

// GetParams returns the total set of txfees parameters.
func (k Keeper) GetParams(ctx sdk.Context) (params types.Params) {
	return types.Params{
		WhitelistedFeeTokenSetters: k.WhitelistedFeeTokenSetters,
	}
}

// SetParams sets the total set of txfees parameters.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.WhitelistedFeeTokenSetters = params.WhitelistedFeeTokenSetters
}

// // SetParam sets a specific txfees module's parameter with the provided parameter.
// func (k Keeper) SetParam(ctx sdk.Context, key []byte, value interface{}) {
// 	k.paramSpace.Set(ctx, key, value)
// }

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k Keeper) GetFeeTokensStore(ctx sdk.Context) storetypes.KVStore {
	store := ctx.KVStore(k.storeKey)
	return prefix.NewStore(store, types.FeeTokensStorePrefix)
}

func (k Keeper) GetFeeTokens(ctx sdk.Context) (feetokens []types.FeeToken) {
	prefixStore := k.GetFeeTokensStore(ctx)

	// this entire store just contains FeeTokens, so iterate over all entries.
	iterator := prefixStore.Iterator(nil, nil)
	defer iterator.Close()

	feeTokens := []types.FeeToken{}

	for ; iterator.Valid(); iterator.Next() {
		feeToken := types.FeeToken{}

		err := proto.Unmarshal(iterator.Value(), &feeToken)
		if err != nil {
			panic(err)
		}

		feeTokens = append(feeTokens, feeToken)
	}
	return feeTokens
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

func (k Keeper) GetFeeToken(ctx sdk.Context, denom string) (types.FeeToken, error) {
	prefixStore := k.GetFeeTokensStore(ctx)
	if !prefixStore.Has([]byte(denom)) {
		return types.FeeToken{}, sdkioerrors.Wrapf(types.ErrInvalidFeeToken, "%s", denom)
	}
	bz := prefixStore.Get([]byte(denom))

	feeToken := types.FeeToken{}
	err := proto.Unmarshal(bz, &feeToken)
	if err != nil {
		return types.FeeToken{}, err
	}

	return feeToken, nil
}

func (k Keeper) SetFeeTokens(ctx sdk.Context, feetokens []types.FeeToken) error {
	for _, feeToken := range feetokens {
		err := k.setFeeToken(ctx, feeToken)
		if err != nil {
			return err
		}
	}
	return nil
}

func (k Keeper) setFeeToken(ctx sdk.Context, feeToken types.FeeToken) error {
	prefixStore := k.GetFeeTokensStore(ctx)

	baseDenom, err := k.GetBaseDenom(ctx)
	if err != nil {
		return err
	}
	if baseDenom == feeToken.Denom {
		return sdkioerrors.Wrap(types.ErrInvalidFeeToken, "cannot add basedenom as a whitelisted fee token")
	}

	bz, err := proto.Marshal(&feeToken)
	if err != nil {
		return err
	}

	prefixStore.Set([]byte(feeToken.Denom), bz)
	return nil
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
