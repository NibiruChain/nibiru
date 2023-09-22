package action

import (
	"fmt"

	"cosmossdk.io/math"
	"github.com/NibiruChain/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/testutil/action"
)

func DnREpochIs(epoch uint64) action.Action {
	return &setEpochAction{
		Epoch: epoch,
	}
}

type setEpochAction struct {
	Epoch uint64
}

func (s setEpochAction) Do(app *app.NibiruApp, ctx sdk.Context) (outCtx sdk.Context, err error, isMandatory bool) {
	app.PerpKeeperV2.DnREpoch.Set(ctx, s.Epoch)
	return ctx, nil, true
}

func DnRCurrentVolumeIs(user sdk.AccAddress, wantVolume math.Int) action.Action {
	return &expectVolumeAction{
		User:   user,
		Volume: wantVolume,
	}
}

type expectVolumeAction struct {
	User   sdk.AccAddress
	Volume math.Int
}

func (e expectVolumeAction) Do(app *app.NibiruApp, ctx sdk.Context) (outCtx sdk.Context, err error, isMandatory bool) {
	currentEpoch, err := app.PerpKeeperV2.DnREpoch.Get(ctx)
	if err != nil {
		return ctx, err, true
	}
	volume, err := app.PerpKeeperV2.TraderVolumes.Get(ctx, collections.Join(e.User, currentEpoch))
	if err != nil {
		return ctx, err, true
	}
	if !volume.Equal(e.Volume) {
		return ctx, fmt.Errorf("unexpected user dnr volume, wanted %s, got %s", e.Volume, volume), true
	}
	return ctx, nil, true
}

func DnRPreviousVolumeIs(user sdk.AccAddress, wantVolume math.Int) action.Action {
	return &expectPreviousVolumeAction{
		User:   user,
		Volume: wantVolume,
	}
}

type expectPreviousVolumeAction struct {
	User   sdk.AccAddress
	Volume math.Int
}

func (e expectPreviousVolumeAction) Do(app *app.NibiruApp, ctx sdk.Context) (outCtx sdk.Context, err error, isMandatory bool) {
	v := app.PerpKeeperV2.GetUserVolumeLastEpoch(ctx, e.User)
	if !v.Equal(e.Volume) {
		return ctx, fmt.Errorf("unexpected user dnr volume, wanted %s, got %s", e.Volume, v), true
	}
	return ctx, nil, true
}

func DnRVolumeNotExist(user sdk.AccAddress, epoch uint64) action.Action {
	return &expectVolumeNotExistAction{
		Epoch: epoch,
		User:  user,
	}
}

type expectVolumeNotExistAction struct {
	Epoch uint64
	User  sdk.AccAddress
}

func (e expectVolumeNotExistAction) Do(app *app.NibiruApp, ctx sdk.Context) (outCtx sdk.Context, err error, isMandatory bool) {
	v, err := app.PerpKeeperV2.TraderVolumes.Get(ctx, collections.Join(e.User, e.Epoch))
	if err == nil {
		return ctx, fmt.Errorf("unexpected user dnr volume, got %s", v), true
	}
	return ctx, nil, true
}
