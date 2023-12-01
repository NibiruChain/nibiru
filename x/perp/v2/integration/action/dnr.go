package action

import (
	"fmt"

	"cosmossdk.io/math"
	"github.com/NibiruChain/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/testutil"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/testutil/action"
	"github.com/NibiruChain/nibiru/x/perp/v2/types"
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
	currentEpoch, err := app.PerpKeeperV2.DnREpoch.Get(ctx)
	if err != nil {
		return ctx, err, true
	}

	v := app.PerpKeeperV2.GetTraderVolumeLastEpoch(ctx, currentEpoch, e.User)
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

type marketOrderFeeIs struct {
	fee sdk.Dec
	*openPositionAction
}

func MarketOrderFeeIs(
	fee sdk.Dec,
	trader sdk.AccAddress,
	pair asset.Pair,
	dir types.Direction,
	margin math.Int,
	leverage sdk.Dec,
	baseAssetLimit sdk.Dec,
	responseCheckers ...MarketOrderResponseChecker,
) action.Action {
	o := openPositionAction{
		trader:           trader,
		pair:             pair,
		dir:              dir,
		margin:           margin,
		leverage:         leverage,
		baseAssetLimit:   baseAssetLimit,
		responseCheckers: responseCheckers,
	}
	return &marketOrderFeeIs{
		fee:                fee,
		openPositionAction: &o,
	}
}

func (o *marketOrderFeeIs) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	collateral, err := app.PerpKeeperV2.Collateral.Get(ctx)
	if err != nil {
		return ctx, err, true
	}

	balanceBefore := app.BankKeeper.GetBalance(ctx, o.trader, collateral).Amount
	resp, err := app.PerpKeeperV2.MarketOrder(
		ctx, o.pair, o.dir, o.trader,
		o.margin, o.leverage, o.baseAssetLimit,
	)
	if err != nil {
		return ctx, err, true
	}

	balanceBefore = balanceBefore.Sub(resp.MarginToVault.TruncateInt())

	expectedFee := math.LegacyNewDecFromInt(o.margin).Mul(o.fee.Add(sdk.MustNewDecFromStr("0.001"))) // we add the ecosystem fund fee
	balanceAfter := app.BankKeeper.GetBalance(ctx, o.trader, collateral).Amount
	paidFees := balanceBefore.Sub(balanceAfter)
	if !paidFees.Equal(expectedFee.TruncateInt()) {
		return ctx, fmt.Errorf("unexpected fee, wanted %s, got %s", expectedFee, paidFees), true
	}
	return ctx, nil, true
}

func SetPreviousEpochUserVolume(user sdk.AccAddress, volume math.Int) action.Action {
	return &setPreviousEpochUserVolumeAction{
		user:   user,
		volume: volume,
	}
}

type setPreviousEpochUserVolumeAction struct {
	user   sdk.AccAddress
	volume math.Int
}

func (s setPreviousEpochUserVolumeAction) Do(app *app.NibiruApp, ctx sdk.Context) (outCtx sdk.Context, err error, isMandatory bool) {
	currentEpoch, err := app.PerpKeeperV2.DnREpoch.Get(ctx)
	if err != nil {
		return ctx, err, true
	}
	app.PerpKeeperV2.TraderVolumes.Insert(ctx, collections.Join(s.user, currentEpoch-1), s.volume)
	return ctx, nil, true
}

func SetGlobalDiscount(fee sdk.Dec, volume math.Int) action.Action {
	return &setGlobalDiscountAction{
		fee:    fee,
		volume: volume,
	}
}

type setGlobalDiscountAction struct {
	fee    sdk.Dec
	volume math.Int
}

func (s setGlobalDiscountAction) Do(app *app.NibiruApp, ctx sdk.Context) (outCtx sdk.Context, err error, isMandatory bool) {
	app.PerpKeeperV2.GlobalDiscounts.Insert(ctx, s.volume, s.fee)
	return ctx, nil, true
}

func SetCustomDiscount(user sdk.AccAddress, fee sdk.Dec, volume math.Int) action.Action {
	return &setCustomDiscountAction{
		fee:    fee,
		volume: volume,
		user:   user,
	}
}

type setCustomDiscountAction struct {
	fee    sdk.Dec
	volume math.Int
	user   sdk.AccAddress
}

func (s *setCustomDiscountAction) Do(app *app.NibiruApp, ctx sdk.Context) (outCtx sdk.Context, err error, isMandatory bool) {
	app.PerpKeeperV2.TraderDiscounts.Insert(ctx, collections.Join(s.user, s.volume), s.fee)
	return ctx, nil, true
}

type fundDnREpoch struct {
	amt sdk.Coins
}

func (f fundDnREpoch) Do(app *app.NibiruApp, ctx sdk.Context) (outCtx sdk.Context, err error, isMandatory bool) {
	tmpAcc := testutil.AccAddress()
	ctx, err, _ = action.FundAccount(tmpAcc, f.amt).Do(app, ctx)
	if err != nil {
		return ctx, err, true
	}
	_, err = app.PerpKeeperV2.AllocateEpochRebates(ctx, tmpAcc, f.amt)
	if err != nil {
		return ctx, err, true
	}
	return ctx, nil, true
}

func FundDnREpoch(amt sdk.Coins) action.Action {
	return fundDnREpoch{amt}
}

type startNewDnRepochAction struct {
}

func (s startNewDnRepochAction) Do(app *app.NibiruApp, ctx sdk.Context) (outCtx sdk.Context, err error, isMandatory bool) {
	currentEpoch, err := app.PerpKeeperV2.DnREpoch.Get(ctx)
	if err != nil {
		return ctx, err, true
	}
	err = app.PerpKeeperV2.StartNewEpoch(ctx, currentEpoch+1)
	if err != nil {
		return ctx, err, true
	}
	return ctx, nil, true
}

func StartNewDnREpoch() action.Action {
	return &startNewDnRepochAction{}
}

type dnrRebateIsAction struct {
	user            sdk.AccAddress
	epoch           uint64
	expectedRewards sdk.Coins
}

func (d dnrRebateIsAction) Do(app *app.NibiruApp, ctx sdk.Context) (outCtx sdk.Context, err error, isMandatory bool) {
	withdrawn, err := app.PerpKeeperV2.WithdrawEpochRebates(ctx, d.epoch, d.user)
	if err != nil {
		return ctx, err, true
	}
	if !withdrawn.IsEqual(d.expectedRewards) {
		return ctx, fmt.Errorf("expected %s, got %s", d.expectedRewards, withdrawn), true
	}
	return ctx, nil, true
}

func DnRRebateIs(user sdk.AccAddress, epoch uint64, expectedRewards sdk.Coins) action.Action {
	return &dnrRebateIsAction{
		user:            user,
		epoch:           epoch,
		expectedRewards: expectedRewards,
	}
}

func DnRRebateFails(user sdk.AccAddress, epoch uint64) action.Action {
	return &dnrRebateFailsAction{
		user:  user,
		epoch: epoch,
	}
}

type dnrRebateFailsAction struct {
	user  sdk.AccAddress
	epoch uint64
}

func (d dnrRebateFailsAction) Do(app *app.NibiruApp, ctx sdk.Context) (outCtx sdk.Context, err error, isMandatory bool) {
	withdrawn, err := app.PerpKeeperV2.WithdrawEpochRebates(ctx, d.epoch, d.user)
	if err == nil {
		return ctx, fmt.Errorf("expected withdrawal error but got instead: %s rewards", withdrawn.String()), true
	}
	return ctx, nil, true
}
