// nolint
package types

import (
	"context"
	storetypes "cosmossdk.io/store/types"
	"errors"
	"math"
	"math/rand"
	"time"

	sdkmath "cosmossdk.io/math"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/cometbft/cometbft/crypto/secp256k1"
	tmprotocrypto "github.com/cometbft/cometbft/proto/tendermint/crypto"
)

// OracleDecPrecision nolint
const OracleDecPrecision = 8

// GenerateRandomTestCase nolint
func GenerateRandomTestCase() (rates []float64, valAddrs []sdk.ValAddress, stakingKeeper DummyStakingKeeper) {
	valAddrs = []sdk.ValAddress{}
	mockValidators := []MockValidator{}

	base := math.Pow10(OracleDecPrecision)

	rand.Seed(int64(time.Now().Nanosecond()))
	numInputs := 10 + (rand.Int() % 100)
	for i := 0; i < numInputs; i++ {
		rate := float64(int64(rand.Float64()*base)) / base
		rates = append(rates, rate)

		pubKey := secp256k1.GenPrivKey().PubKey()
		valAddr := sdk.ValAddress(pubKey.Address())
		valAddrs = append(valAddrs, valAddr)

		power := rand.Int63()%1000 + 1
		mockValidator := NewMockValidator(valAddr, power)
		mockValidators = append(mockValidators, mockValidator)
	}

	stakingKeeper = NewDummyStakingKeeper(mockValidators)

	return
}

var _ StakingKeeper = DummyStakingKeeper{}

// DummyStakingKeeper dummy staking keeper to test votes
type DummyStakingKeeper struct {
	validators []MockValidator
}

// NewDummyStakingKeeper returns new DummyStakingKeeper instance
func NewDummyStakingKeeper(validators []MockValidator) DummyStakingKeeper {
	return DummyStakingKeeper{
		validators: validators,
	}
}

// Validators nolint
func (sk DummyStakingKeeper) Validators() []MockValidator {
	return sk.validators
}

// Validator nolint
func (sk DummyStakingKeeper) Validator(ctx context.Context, address sdk.ValAddress) (stakingtypes.ValidatorI, error) {
	for _, validator := range sk.validators {
		if validator.GetOperator() == address.String() {
			return validator, nil
		}
	}

	return nil, errors.New("validator not found")
}

// TotalBondedTokens nolint
func (DummyStakingKeeper) TotalBondedTokens(context.Context) (sdkmath.Int, error) {
	return sdkmath.ZeroInt(), nil
}

// Slash nolint
func (DummyStakingKeeper) Slash(context.Context, sdk.ConsAddress, int64, int64, sdkmath.LegacyDec) (sdkmath.Int, error) {
	return sdkmath.ZeroInt(), nil
}

// ValidatorsPowerStoreIterator nolint
func (DummyStakingKeeper) ValidatorsPowerStoreIterator(context context.Context) (storetypes.Iterator, error) {
	return storetypes.KVStoreReversePrefixIterator(nil, nil), nil
}

// Jail nolint
func (DummyStakingKeeper) Jail(context.Context, sdk.ConsAddress) error {
	return nil
}

// GetLastValidatorPower nolint
func (sk DummyStakingKeeper) GetLastValidatorPower(ctx context.Context, operator sdk.ValAddress) (power int64) {
	val, _ := sk.Validator(ctx, operator)
	return val.GetConsensusPower(sdk.DefaultPowerReduction)
}

// MaxValidators returns the maximum amount of bonded validators
func (DummyStakingKeeper) MaxValidators(context.Context) (uint32, error) {
	return 100, nil
}

// PowerReduction - is the amount of staking tokens required for 1 unit of consensus-engine power
func (DummyStakingKeeper) PowerReduction(ctx context.Context) (res sdkmath.Int) {
	res = sdk.DefaultPowerReduction
	return
}

// MockValidator nolint
type MockValidator struct {
	power       int64
	valOperAddr sdk.ValAddress
}

var _ stakingtypes.ValidatorI = MockValidator{}

func (MockValidator) IsJailed() bool                          { return false }
func (MockValidator) GetMoniker() string                      { return "" }
func (MockValidator) GetStatus() stakingtypes.BondStatus      { return stakingtypes.Bonded }
func (MockValidator) IsBonded() bool                          { return true }
func (MockValidator) IsUnbonded() bool                        { return false }
func (MockValidator) IsUnbonding() bool                       { return false }
func (v MockValidator) GetOperator() string                   { return v.valOperAddr.String() }
func (MockValidator) ConsPubKey() (cryptotypes.PubKey, error) { return nil, nil }
func (MockValidator) TmConsPublicKey() (tmprotocrypto.PublicKey, error) {
	return tmprotocrypto.PublicKey{}, nil
}
func (MockValidator) GetConsAddr() ([]byte, error) { return nil, nil }
func (v MockValidator) GetTokens() sdkmath.Int {
	return sdk.TokensFromConsensusPower(v.power, sdk.DefaultPowerReduction)
}

func (v MockValidator) GetBondedTokens() sdkmath.Int {
	return sdk.TokensFromConsensusPower(v.power, sdk.DefaultPowerReduction)
}
func (v MockValidator) GetConsensusPower(powerReduction sdkmath.Int) int64 { return v.power }
func (v *MockValidator) SetConsensusPower(power int64)                     { v.power = power }
func (v MockValidator) GetCommission() sdkmath.LegacyDec                   { return sdkmath.LegacyZeroDec() }
func (v MockValidator) GetMinSelfDelegation() sdkmath.Int                  { return sdkmath.OneInt() }
func (v MockValidator) GetDelegatorShares() sdkmath.LegacyDec              { return sdkmath.LegacyNewDec(v.power) }
func (v MockValidator) TokensFromShares(sdkmath.LegacyDec) sdkmath.LegacyDec {
	return sdkmath.LegacyZeroDec()
}
func (v MockValidator) TokensFromSharesTruncated(sdkmath.LegacyDec) sdkmath.LegacyDec {
	return sdkmath.LegacyZeroDec()
}
func (v MockValidator) TokensFromSharesRoundUp(sdkmath.LegacyDec) sdkmath.LegacyDec {
	return sdkmath.LegacyZeroDec()
}
func (v MockValidator) SharesFromTokens(amt sdkmath.Int) (sdkmath.LegacyDec, error) {
	return sdkmath.LegacyZeroDec(), nil
}
func (v MockValidator) SharesFromTokensTruncated(amt sdkmath.Int) (sdkmath.LegacyDec, error) {
	return sdkmath.LegacyZeroDec(), nil
}

func NewMockValidator(valAddr sdk.ValAddress, power int64) MockValidator {
	return MockValidator{
		power:       power,
		valOperAddr: valAddr,
	}
}
