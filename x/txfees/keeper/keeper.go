package keeper

import (
	"fmt"

	"github.com/cosmos/gogoproto/proto"

	"cosmossdk.io/store/prefix"
	"github.com/cometbft/cometbft/libs/log"

	sdk "github.com/cosmos/cosmos-sdk/types"

	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	consensustypes "github.com/cosmos/cosmos-sdk/x/consensus/types"

	"github.com/NibiruChain/nibiru/v2/x/txfees/types"
)

type Keeper struct {
	storeKey storetypes.StoreKey

	accountKeeper      types.AccountKeeper
	bankKeeper         types.BankKeeper
	protorevKeeper     types.ProtorevKeeper
	distributionKeeper types.DistributionKeeper
	consensusKeeper    types.ConsensusKeeper
	dataDir            string

	paramSpace paramtypes.Subspace
}

var _ types.TxFeesKeeper = (*Keeper)(nil)

func NewKeeper(
	accountKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper,
	storeKey storetypes.StoreKey,
	distributionKeeper types.DistributionKeeper,
	consensusKeeper types.ConsensusKeeper,
	dataDir string,
	paramSpace paramtypes.Subspace,
) Keeper {
	// set KeyTable if it has not already been set
	if !paramSpace.HasKeyTable() {
		paramSpace = paramSpace.WithKeyTable(types.ParamKeyTable())
	}

	return Keeper{
		accountKeeper:      accountKeeper,
		bankKeeper:         bankKeeper,
		storeKey:           storeKey,
		distributionKeeper: distributionKeeper,
		consensusKeeper:    consensusKeeper,
		dataDir:            dataDir,
		paramSpace:         paramSpace,
	}
}

// GetParams returns the total set of txfees parameters.
func (k Keeper) GetParams(ctx sdk.Context) (params types.Params) {
	k.paramSpace.GetParamSet(ctx, &params)
	return params
}

// SetParams sets the total set of txfees parameters.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.paramSpace.SetParamSet(ctx, &params)
}

// SetParam sets a specific txfees module's parameter with the provided parameter.
func (k Keeper) SetParam(ctx sdk.Context, key []byte, value interface{}) {
	k.paramSpace.Set(ctx, key, value)
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k Keeper) GetFeeTokensStore(ctx sdk.Context) storetypes.KVStore {
	store := ctx.KVStore(k.storeKey)
	return prefix.NewStore(store, types.FeeTokensStorePrefix)
}

// GetConsParams returns the current consensus parameters from the consensus params store.
func (k Keeper) GetConsParams(ctx sdk.Context) (*consensustypes.QueryParamsResponse, error) {
	return k.consensusKeeper.Params(ctx, &consensustypes.QueryParamsRequest{})
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

func (k Keeper) SetFeeTokens(ctx sdk.Context, feetokens []types.FeeToken) error {
	for _, feeToken := range feetokens {
		err := k.setFeeToken(ctx, feeToken)
		if err != nil {
			return err
		}
	}
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

	feeToken, err := k.GetFeeToken(ctx, inputFee.Denom)
	if err != nil {
		return sdk.Coin{}, err
	}

	spotPrice, err := k.CalcFeeSpotPrice(ctx, feeToken.Denom)
	if err != nil {
		return sdk.Coin{}, err
	}

	// Note: spotPrice truncation is done here for maintaining state-compatibility with v19.x
	// It should be changed to support full spot price precision before
	// https://github.com/osmosis-labs/osmosis/issues/6064 is complete
	return sdk.NewCoin(baseDenom, spotPrice.Dec().MulIntMut(inputFee.Amount).RoundInt()), nil
}