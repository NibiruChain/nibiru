package keeper

import (
	"fmt"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"

	dexkeeper "github.com/NibiruChain/nibiru/x/dex/keeper"
	"github.com/NibiruChain/nibiru/x/incentivization/types"
	ltypes "github.com/NibiruChain/nibiru/x/lockup/types"
)

const (
	// MinLockupDuration defines the lockup minimum time
	// TODO(mercilex): maybe module param
	MinLockupDuration = 24 * time.Hour
	// MinEpochs defines the minimum number of epochs
	// TODO(mercilex): maybe module param
	MinEpochs int64 = 7
)

const (
	// FundsModuleAccountAddressPrefix defines the prefix
	// of module accounts created that contain an
	// incentivization program funds.
	FundsModuleAccountAddressPrefix = "incentivization_escrow_"
)

func NewKeeper(cdc codec.Codec, storeKey sdk.StoreKey, ak authkeeper.AccountKeeper, bk bankkeeper.Keeper, dk dexkeeper.Keeper, lk types.LockupKeeper) Keeper {
	return Keeper{
		cdc:      cdc,
		storeKey: storeKey,
		ak:       ak,
		bk:       bk,
		dk:       dk,
		lk:       lk,
	}
}

type Keeper struct {
	cdc      codec.Codec
	storeKey sdk.StoreKey

	ak authkeeper.AccountKeeper
	bk bankkeeper.Keeper
	dk dexkeeper.Keeper
	lk types.LockupKeeper
}

func (k Keeper) CreateIncentivizationProgram(
	ctx sdk.Context,
	lpDenom string, minLockupDuration time.Duration, startTime time.Time, epochs int64) (*types.IncentivizationProgram, error) {
	// TODO(mercilex): assert lp denom from dex keeper

	if epochs < MinEpochs {
		return nil, types.ErrEpochsTooLow.Wrapf("%d is lower than minimum allowed %d", epochs, MinEpochs)
	}

	if minLockupDuration < MinLockupDuration {
		return nil, types.ErrMinLockupDurationTooLow.Wrapf("%s is lower than minimum allowed %s", minLockupDuration, MinLockupDuration)
	}

	if ctx.BlockTime().After(startTime) {
		return nil, types.ErrStartTimeInPast.Wrapf("current time %s, got: %s", ctx.BlockTime(), startTime)
	}

	// we create a new instance of an incentivization program

	nextID := k.IncentivizationProgramsState(ctx).PeekNextID()                                           // we need to peek the next ID to create a new
	escrowAccount := k.ak.NewAccount(ctx, authtypes.NewEmptyModuleAccount(NewEscrowAccountName(nextID))) // module account that holds the escrowed funds.
	k.ak.SetAccount(ctx, escrowAccount)

	program := &types.IncentivizationProgram{
		EscrowAddress:     escrowAccount.GetAddress().String(),
		RemainingEpochs:   epochs,
		LpDenom:           lpDenom,
		MinLockupDuration: minLockupDuration,
		StartTime:         startTime,
	}

	k.IncentivizationProgramsState(ctx).Create(program)

	return program, nil
}

func (k Keeper) FundIncentivizationProgram(ctx sdk.Context, id uint64, funder sdk.AccAddress, funds sdk.Coins) error {
	program, err := k.IncentivizationProgramsState(ctx).Get(id)
	if err != nil {
		return err
	}

	escrowAddr, err := sdk.AccAddressFromBech32(program.EscrowAddress)
	if err != nil {
		panic(err)
	}

	// we transfer money from funder to the program escrow address
	// NOTE(mercilex): can't use send coins from module to account because
	// due to how GetModuleAccount works, which fetches information in a
	// stateless way. TRAGEDY. ABSOLUTE TRAGEDY.
	if err := k.bk.SendCoins(ctx, funder, escrowAddr, funds); err != nil {
		return err
	}

	return nil
}

// Distribute distributes incentivization rewards to accounts
// that meet incentivization program criteria.
func (k Keeper) Distribute(ctx sdk.Context) error {
	activePrograms := k.getActivePrograms(ctx)
	now := ctx.BlockTime()
	for _, p := range activePrograms {
		escrowAddr, err := sdk.AccAddressFromBech32(p.EscrowAddress)
		if err != nil {
			panic(err)
		}
		balance := k.bk.GetBalance(ctx, escrowAddr, p.LpDenom)
		if balance.IsZero() {
			// TODO: should we panic or simply ignore
			continue
		}
		locks := []*ltypes.Lock{}
		totalLockedAmt := sdk.NewInt(0)
		k.lk.LocksByDenomUnlockingAfter(ctx, p.LpDenom, p.MinLockupDuration, func(lock *ltypes.Lock) bool {
			if lock.Coins.Empty() || lock.EndTime.Before(now) {
				return false
			}
			locks = append(locks, lock)
			totalLockedAmt = totalLockedAmt.Add(lock.Coins.AmountOf(p.LpDenom))
			return false
		})
		for _, lock := range locks {
			amt := balance.Amount.Mul(lock.Coins.AmountOf(p.LpDenom)).Quo(totalLockedAmt.Mul(sdk.NewInt(p.RemainingEpochs)))
			if amt.IsPositive() {
				coins := sdk.Coins{{
					Denom:  p.LpDenom,
					Amount: amt,
				}}
				k.bk.SendCoins(ctx, escrowAddr, lock.OwnerAddress(), coins)
			}
		}
		// Can this be calculated from the current time or do we need to update??
		p.RemainingEpochs -= 1
		if err := k.IncentivizationProgramsState(ctx).Update(p.Id, p.RemainingEpochs); err != nil {
			return err
		}
	}
	return nil
}

// NewEscrowAccountName returns the escrow module account name
func NewEscrowAccountName(id uint64) string {
	return fmt.Sprintf("%s%d", FundsModuleAccountAddressPrefix, id)
}

func (k Keeper) getActivePrograms(ctx sdk.Context) []*types.IncentivizationProgram {
	programs := []*types.IncentivizationProgram{}
	now := ctx.BlockTime()

	k.IncentivizationProgramsState(ctx).IteratePrograms(func(p *types.IncentivizationProgram) bool {
		if p.RemainingEpochs <= 0 || p.StartTime.Add(p.MinLockupDuration).After(now) {
			return false
		}
		programs = append(programs, p)
		return false
	})
	return programs
}
