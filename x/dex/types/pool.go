package types

import (
	"errors"
	fmt "fmt"
	math "math"

	"github.com/holiman/uint256"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
)

/*
Returns the *base* denomination of a pool share token for a given poolId.

args:

	poolId: the pool id number

ret:

	poolDenom: the pool denomination name of the poolId
*/
func GetPoolShareBaseDenom(poolId uint64) (poolDenom string) {
	return fmt.Sprintf("nibiru/pool/%d", poolId)
}

/*
Returns the *display* denomination of a pool share token for a given poolId.
Display denom means the denomination showed to the user, which could be many exponents greater than the base denom.
e.g. 1 atom is the display denom, but 10^6 uatom is the base denom.

In Nibiru, a display denom is 10^18 base denoms.

args:

	poolId: the pool id number

ret:

	poolDenom: the pool denomination name of the poolId
*/
func GetPoolShareDisplayDenom(poolId uint64) (poolDenom string) {
	return fmt.Sprintf("NIBIRU-POOL-%d", poolId)
}

/*
Creates a new pool and sets the initial assets.

args:

	poolId: the pool numeric id
	poolAccountAddr: the pool's account address for holding funds
	poolParams: pool configuration options
	poolAssets: the initial pool assets and weights

ret:

	pool: a new pool
	err: error if any
*/
func NewPool(
	poolId uint64,
	poolAccountAddr sdk.Address,
	poolParams PoolParams,
	poolAssets []PoolAsset,
) (pool Pool, err error) {
	pool = Pool{
		Id:          poolId,
		Address:     poolAccountAddr.String(),
		PoolParams:  poolParams,
		PoolAssets:  nil,
		TotalWeight: sdk.ZeroInt(),
		TotalShares: sdk.NewCoin(GetPoolShareBaseDenom(poolId), InitPoolSharesSupply),
	}

	err = pool.setInitialPoolAssets(poolAssets)
	if err != nil {
		return Pool{}, err
	}

	return pool, nil
}

/*
Adds tokens to a pool and updates the pool balances (i.e. liquidity).

args:
  - tokensIn: the tokens to add to the pool

ret:
  - numShares: the number of LP shares given to the user for the deposit
  - remCoins: the number of coins remaining after the deposit
  - err: error if any
*/
func (pool *Pool) AddTokensToPool(tokensIn sdk.Coins) (
	numShares sdk.Int, remCoins sdk.Coins, err error,
) {
	if pool.TotalShares.Amount.IsZero() {
		// Mint the initial 100.000000000000000000 pool share tokens to the sender
		numShares = InitPoolSharesSupply
		remCoins = sdk.Coins{}
	} else if pool.PoolParams.PoolType == PoolType_STABLESWAP {
		numShares, err = pool.numSharesOutFromTokensInStableSwap(tokensIn)
		remCoins = sdk.Coins{}
	} else {
		numShares, remCoins, err = pool.numSharesOutFromTokensIn(tokensIn)
	}
	if err != nil {
		return sdk.ZeroInt(), sdk.Coins{}, err
	}

	tokensIn.Sort()
	if err := pool.incrementBalances(numShares, tokensIn.Sub(remCoins)); err != nil {
		return sdk.ZeroInt(), sdk.Coins{}, err
	}

	return numShares, remCoins, nil
}

/*
Adds tokens to a pool optimizing the amount of shares (swap + join) and updates the pool balances (i.e. liquidity).
We maximally join with both tokens first, and then perform a single asset join with the remaining assets.

This function is only necessary for balancer pool. Stableswap pool already takes all the deposit from the user.

args:
  - tokensIn: the tokens to add to the pool

ret:
  - numShares: the number of LP shares given to the user for the deposit
  - remCoins: the number of coins remaining after the deposit
  - err: error if any
*/
func (pool *Pool) AddAllTokensToPool(tokensIn sdk.Coins) (
	numShares sdk.Int, remCoins sdk.Coins, err error,
) {
	if pool.PoolParams.PoolType == PoolType_STABLESWAP {
		err = ErrInvalidPoolType
		return
	}

	remCoins = tokensIn
	if tokensIn.Len() > 1 {
		numShares, remCoins, err = pool.AddTokensToPool(tokensIn)
	} else {
		numShares = sdk.ZeroInt()
	}

	if remCoins.Empty() {
		return
	}

	numShares2nd, _, err := pool.AddTokensToPool(remCoins)
	if err != nil {
		return
	}

	numShares = numShares2nd.Add(numShares)
	remCoins = sdk.NewCoins()
	return
}

/*
Fetch the pool's address as an sdk.Address.
*/
func (pool Pool) GetAddress() (addr sdk.AccAddress) {
	addr, err := sdk.AccAddressFromBech32(pool.Address)
	if err != nil {
		panic(fmt.Sprintf("could not bech32 decode address of pool with id: %d", pool.Id))
	}
	return addr
}

/*
Given the amount of pool shares to exit, calculates the amount of coins to exit
from the pool and modifies the pool. Accounts for an exit fee, if any, on the pool.

args:
  - exitingShares: the number of pool shares to exit from the pool
*/
func (pool *Pool) ExitPool(exitingShares sdk.Int) (
	exitedCoins sdk.Coins, err error,
) {
	if exitingShares.GT(pool.TotalShares.Amount) {
		return sdk.Coins{}, errors.New("too many shares out")
	}

	exitedCoins, err = pool.TokensOutFromPoolSharesIn(exitingShares)
	if err != nil {
		return sdk.Coins{}, err
	}

	if !exitedCoins.IsValid() {
		return sdk.Coins{}, errors.New("not enough pool shares to withdraw")
	}

	// update the pool's balances
	for _, exitedCoin := range exitedCoins {
		err = pool.SubtractPoolAssetBalance(exitedCoin.Denom, exitedCoin.Amount)
		if err != nil {
			return sdk.Coins{}, err
		}
	}

	pool.TotalShares = sdk.NewCoin(pool.TotalShares.Denom, pool.TotalShares.Amount.Sub(exitingShares))
	return exitedCoins, nil
}

/*
Updates the pool's asset liquidity using the provided tokens.

args:
  - tokens: the new token liquidity in the pool

ret:
  - err: error if any
*/
func (pool *Pool) updatePoolAssetBalances(tokens sdk.Coins) (err error) {
	// Ensures that there are no duplicate denoms, all denom's are valid,
	// and amount is > 0
	if len(tokens) != len(pool.PoolAssets) {
		return errors.New("provided tokens do not match number of assets in pool")
	}
	if err = tokens.Validate(); err != nil {
		return fmt.Errorf("provided coins are invalid, %v", err)
	}

	for _, coin := range tokens {
		assetIndex, existingAsset, err := pool.getPoolAssetAndIndex(coin.Denom)
		if err != nil {
			return err
		}
		existingAsset.Token = coin
		pool.PoolAssets[assetIndex].Token = coin
	}

	return nil
}

// setInitialPoolAssets sets the PoolAssets in the pool.
// It is only designed to be called at the pool's creation.
// If the same denom's PoolAsset exists, will return error.
// The list of PoolAssets must be sorted. This is done to enable fast searching for a PoolAsset by denomination.
func (pool *Pool) setInitialPoolAssets(poolAssets []PoolAsset) (err error) {
	exists := make(map[string]bool)

	newTotalWeight := sdk.ZeroInt()
	scaledPoolAssets := make([]PoolAsset, 0, len(poolAssets))

	for _, asset := range poolAssets {
		if err = asset.Validate(); err != nil {
			return err
		}

		if exists[asset.Token.Denom] {
			return fmt.Errorf("same PoolAsset already exists")
		}
		exists[asset.Token.Denom] = true

		// Scale weight from the user provided weight to the correct internal weight
		asset.Weight = asset.Weight.MulRaw(GuaranteedWeightPrecision)
		scaledPoolAssets = append(scaledPoolAssets, asset)
		newTotalWeight = newTotalWeight.Add(asset.Weight)
	}

	pool.PoolAssets = scaledPoolAssets
	sortPoolAssetsByDenom(pool.PoolAssets)

	pool.TotalWeight = newTotalWeight

	return nil
}

// For a stableswap pool, compute the D invariant value  in non-overflowing integer operations iteratively
// A * sum(x_i) * n**n + D = A * D * n**n + D**(n+1) / (n**n * prod(x_i))
// Converging solution:
// D[j+1] = (A * n**n * sum(x_i) - D[j]**(n+1) / (n**n prod(x_i))) / (A * n**n - 1)
func (pool Pool) getD(poolAssets []PoolAsset) (*uint256.Int, error) {
	nCoins := uint256.NewInt().SetUint64(uint64(len(poolAssets)))

	S := uint256.NewInt()
	A_Precision := common.APrecision

	Amp := uint256.NewInt().SetUint64(uint64(pool.PoolParams.A.Int64()))
	Amp.Mul(Amp, A_Precision)

	Ann := uint256.NewInt()

	nCoinsFloat := float64(len(poolAssets))
	Ann.Mul(Amp, uint256.NewInt().SetUint64(uint64(math.Pow(nCoinsFloat, nCoinsFloat))))

	var poolAssetsTokens []*uint256.Int
	for _, token := range poolAssets {
		amount := uint256.NewInt().SetUint64(token.Token.Amount.Uint64())
		poolAssetsTokens = append(poolAssetsTokens, amount)
		S.Add(S, amount)
	}

	D := uint256.NewInt().Set(S)

	for i := 0; i < 255; i++ {
		D_P := uint256.NewInt().Set(D)
		for _, token := range poolAssetsTokens {
			D_P.Div(
				uint256.NewInt().Mul(D_P, D),
				uint256.NewInt().Mul(token, nCoins),
			)
		}
		previousD := uint256.NewInt().Set(D)

		// D = (Ann * S + D_P * N_COINS) * D / ((Ann - 1) * D + (N_COINS + 1) * D_P)
		num := uint256.NewInt().Mul(
			uint256.NewInt().Add(
				uint256.NewInt().Mul(Ann, S),
				uint256.NewInt().Mul(D_P, nCoins),
			),
			D,
		)
		denom := uint256.NewInt().Add(
			uint256.NewInt().Mul(
				uint256.NewInt().Add(
					nCoins,
					uint256.NewInt().SetOne(),
				),
				D_P,
			),
			uint256.NewInt().Mul(
				uint256.NewInt().Sub(Ann, uint256.NewInt().SetOne()),
				D,
			),
		)

		// D = (Ann * S / A_PRECISION + D_P * N_COINS) * D / ((Ann - A_PRECISION) * D / A_PRECISION + (N_COINS + 1) * D_P)
		absDifference := uint256.NewInt()
		D.Div(num, denom)

		absDifference.Abs(uint256.NewInt().Sub(D, previousD))
		if absDifference.Lt(uint256.NewInt().SetUint64(2)) { // absDifference LTE 1 -> absDifference LT 2
			return D, nil
		}
	}

	// convergence typically occurs in 4 rounds or less, this should be unreachable!
	// if it does happen the pool is borked and LPs can withdraw via `remove_liquidity`
	return uint256.NewInt(), ErrBorkedPool
}

// getA returns the amplification factor of the pool
func (pool Pool) getA() (Amp *uint256.Int) {
	Amp = uint256.NewInt().SetUint64(uint64(pool.PoolParams.A.Int64()))
	return
}

// Search for the i and j indices for a swap like x[j] if one makes x[i] = x
func (pool Pool) getIJforSwap(denomIn, denomOut string) (i int, j int, err error) {
	i, _, err = pool.getPoolAssetAndIndex(denomIn)
	if err != nil {
		return
	}

	j, _, err = pool.getPoolAssetAndIndex(denomOut)
	if err != nil {
		return
	}

	return i, j, nil
}

func MustSdkIntToUint256(num sdk.Int) *uint256.Int {
	return uint256.NewInt().SetUint64(uint64(num.Int64()))
}

// Calculate the amount of token out
func (pool Pool) Exchange(tokenIn sdk.Coin, tokenOutDenom string) (dy sdk.Int, err error) {
	_, poolAssetIn, err := pool.getPoolAssetAndIndex(tokenIn.Denom)
	if err != nil {
		return
	}
	_, poolAssetOut, err := pool.getPoolAssetAndIndex(tokenOutDenom)
	if err != nil {
		return
	}

	dx := poolAssetIn.Token.Add(tokenIn)
	yAmount, err := pool.SolveStableswapInvariant(dx, tokenOutDenom)

	y := sdk.NewCoin(tokenOutDenom, yAmount)
	dy = poolAssetOut.Token.Sub(y).Amount
	return
}

// Calculate y if one makes x = tokenIn
// Done by solving quadratic equation iteratively.
// x_1**2 + x1 * (sum' - (A*n**n - 1) * D / (A * n**n)) = D ** (n+1)/(n ** (2 * n) * prod' * A)
// x_1**2 + b*x_1 = c
// x_1 = (x_1**2 + c) / (2*x_1 + b - D)
func (pool Pool) SolveStableswapInvariant(tokenIn sdk.Coin, tokenOutDenom string) (yAmount sdk.Int, err error) {
	A := pool.getA()
	D, err := pool.getD(pool.PoolAssets)
	if err != nil {
		return
	}

	Ann := uint256.NewInt()
	nCoins := uint256.NewInt().SetUint64(uint64(len(pool.PoolAssets)))

	nCoinsFloat := float64(len(pool.PoolAssets))
	Ann.Mul(A, uint256.NewInt().SetUint64(uint64(math.Pow(nCoinsFloat, nCoinsFloat))))

	c := uint256.NewInt().Set(D)
	S := uint256.NewInt()
	var _x *uint256.Int

	i, j, err := pool.getIJforSwap(tokenIn.Denom, tokenOutDenom)
	if err != nil {
		return
	}

	for _i := 0; _i < len(pool.PoolAssets); _i++ {
		if _i == i {
			_x = MustSdkIntToUint256(tokenIn.Amount)
		} else if _i != j {
			_x = MustSdkIntToUint256(pool.PoolAssets[_i].Token.Amount)
		} else {
			continue
		}

		S.Add(S, _x)

		c.Div(
			uint256.NewInt().Mul(c, D),
			uint256.NewInt().Mul(_x, nCoins),
		)
	}

	// c = c * D * A_PRECISION / (Ann * N_COINS)
	c.Div(
		uint256.NewInt().Mul(c, uint256.NewInt().Mul(D, common.APrecision)),
		uint256.NewInt().Mul(Ann, nCoins),
	)

	b := uint256.NewInt().Add(
		S,
		uint256.NewInt().Div(
			uint256.NewInt().Mul(D, common.APrecision),
			Ann,
		),
	)

	y := uint256.NewInt().Set(D)
	y_prev := uint256.NewInt()

	for _i := 0; _i < 255; _i++ {
		y_prev.Set(y)

		y.Div(
			uint256.NewInt().Add(uint256.NewInt().Mul(y, y), c),
			uint256.NewInt().Sub(
				uint256.NewInt().Add(
					uint256.NewInt().Mul(uint256.NewInt().SetUint64(2),
						y,
					),
					b,
				),
				D,
			),
		)

		absDifference := uint256.NewInt()
		absDifference.Abs(uint256.NewInt().Sub(y, y_prev))
		if absDifference.Lt(uint256.NewInt().SetUint64(2)) { // LTE 1
			return sdk.NewIntFromUint64(y.Uint64()), nil
		}
	}

	errvals := fmt.Sprintf(
		"y=%v\ny_prev=%v\nb=%v\nD=%v\nc=%v\nS=%v\n",
		y, y_prev, b, D, c, S,
	)

	// Should converge in a couple of round unless pool is borked
	err = fmt.Errorf("%w: unable to compute the SolveStableswapInvariant for values %s", ErrBorkedPool, errvals)
	return
}
