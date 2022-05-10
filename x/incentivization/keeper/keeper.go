package keeper

import (
	"fmt"
	dexkeeper "github.com/NibiruChain/nibiru/x/dex/keeper"
	"github.com/NibiruChain/nibiru/x/incentivization/types"
	lockupkeeper "github.com/NibiruChain/nibiru/x/lockup/keeper"
	lockuptypes "github.com/NibiruChain/nibiru/x/lockup/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	"time"
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

func NewKeeper(cdc codec.Codec, storeKey sdk.StoreKey, ak authkeeper.AccountKeeper, bk bankkeeper.Keeper, dk dexkeeper.Keeper, lk lockupkeeper.LockupKeeper) Keeper {
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
	lk lockupkeeper.LockupKeeper
}

func (k Keeper) CreateIncentivizationProgram(
	ctx sdk.Context,
	lpDenom string, minLockupDuration time.Duration, starTime time.Time, epochs int64) (*types.IncentivizationProgram, error) {
	// TODO(mercilex): assert lp denom from dex keeper

	if epochs < MinEpochs {
		return nil, types.ErrEpochsTooLow.Wrapf("%d is lower than minimum allowed %d", epochs, MinEpochs)
	}

	if minLockupDuration < MinLockupDuration {
		return nil, types.ErrMinLockupDurationTooLow.Wrapf("%s is lower than minimum allowed %s", minLockupDuration, MinLockupDuration)
	}

	if ctx.BlockTime().Before(starTime) {
		return nil, types.ErrStartTimeInPast.Wrapf("current time %s, got: %s", ctx.BlockTime(), starTime)
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
		StartTime:         starTime,
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
	// TODO(mercilex): maybe some extra checks on minimum sendable
	// TODO(mercilex): protect through a deposit of NIBI coins

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
func (k Keeper) Distribute(ctx sdk.Context) {
	// TODO(mercilex): this is highly inefficient, better algo needed.
	state := k.IncentivizationProgramsState(ctx)
	// iterate over every active program
	state.IteratePrograms(func(program *types.IncentivizationProgram) (stop bool) {
		// we get the escrow address balance
		escrowAddr, err := sdk.AccAddressFromBech32(program.EscrowAddress)
		if err != nil {
			panic(err)
		}
		// we get the balance
		balance := k.bk.GetAllBalances(ctx, escrowAddr)
		// basically this balance needs to be divided by epochs remaining
		toDistribute := sdk.NewCoins()
		for i := range balance { // iterate over every coin in balance
			coin := balance[i]
			amountToDistribute := coin.Amount.QuoRaw(program.RemainingEpochs) // divide amount by number of epochs remaining
			coinToDistribute := sdk.NewCoin(coin.Denom, amountToDistribute)   // then add the single coin to the coins to distribute
			toDistribute = toDistribute.Add(coinToDistribute)
		}
		// iterate over every account with locked coins
		totalLocked := sdk.NewCoins()
		var coinsByLocker []struct {
			addr  string
			coins sdk.Coins
		}
		k.lk.LocksByDenomUnlockingAfter(ctx, program.LpDenom, program.MinLockupDuration, func(lock *lockuptypes.Lock) (stop bool) {
			totalLocked = totalLocked.Add(lock.Coins...) // we add to the total locked coins
			coinsByLocker = append(coinsByLocker, struct {
				addr  string
				coins sdk.Coins
			}{addr: lock.Owner, coins: lock.Coins})
			return false
		})

		// we calculate weights based on amounts
		for _, lockedCoins := range coinsByLocker {
			percentageToDistr := calcWeight(totalLocked, lockedCoins.coins)
			addr, err := sdk.AccAddressFromBech32(lockedCoins.addr)
			if err != nil {
				panic(err)
			}
			err = k.bk.SendCoins(ctx, escrowAddr, addr, coinsPercentage(toDistribute, percentageToDistr))
			if err != nil {
				panic(err)
			}
		}

		// update program
		program.RemainingEpochs -= 1
		switch program.RemainingEpochs {
		case 0:
			// TODO(mercilex): delete program
		default:
			// todo(mercilex): update program
		}
		return false
	})
}

func coinsPercentage(distribute sdk.Coins, percentage sdk.Int) sdk.Coins {
	// TODO(mercilex): this not right because of precision
	c := sdk.NewCoins()
	for _, coin := range distribute {
		c = c.Add(sdk.NewCoin(coin.Denom, coin.Amount.QuoRaw(100).Mul(percentage)))
	}

	return c
}

func calcWeight(total sdk.Coins, owned sdk.Coins) sdk.Int {
	sumWeights := sdk.ZeroInt()
	n := int64(0)
	for _, totalCoin := range total {
		n++
		// check if the lock has the denom
		ownedCoin := owned.AmountOfNoDenomValidation(totalCoin.Denom)
		if ownedCoin.IsZero() {
			continue
		}
		// totalCoin : 100 = ownedCoin : x
		weight := ownedCoin.MulRaw(100).Quo(totalCoin.Amount)
		sumWeights = sumWeights.Add(weight)
	}

	return sumWeights.QuoRaw(n)
}

// NewEscrowAccountName returns the escrow module account name
func NewEscrowAccountName(id uint64) string {
	return fmt.Sprintf("%s%d", FundsModuleAccountAddressPrefix, id)
}
