package simulation

import (
	"fmt"
	"math/rand"

	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
)

var Gas = uint64(20000000)

// Fees returns a random fee by selecting a random amount of bond denomination
// from the account's available balance. If the user doesn't have enough funds for
// paying fees, it returns empty coins.
func Fees(r *rand.Rand, spendableCoins sdk.Coins) (sdk.Coins, error) {
	if spendableCoins.Empty() {
		return nil, nil
	}

	bondDenomAmt := spendableCoins.AmountOf(sdk.DefaultBondDenom)
	if bondDenomAmt.IsZero() {
		return nil, fmt.Errorf("not enough fee tokens")
	}

	amt, err := simtypes.RandPositiveInt(r, bondDenomAmt)
	if err != nil {
		return nil, err
	}

	if amt.IsZero() {
		return nil, nil
	}

	fees := sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, amt))
	return fees, nil
}
