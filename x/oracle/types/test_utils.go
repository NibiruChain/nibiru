// nolint
package types

import (
	"math"
	"math/rand"
	"time"

	sdkmath "cosmossdk.io/math"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	tmprotocrypto "github.com/cometbft/cometbft/api/cometbft/crypto/v1"
	"github.com/cometbft/cometbft/crypto/secp256k1"
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
func (sk DummyStakingKeeper) Validator(ctx sdk.Context, address sdk.ValAddress) stakingtypes.ValidatorI {
	for _, validator := range sk.validators {
		if validator.GetOperator().Equals(address) {
			return validator
		}
	}

	return nil
}

// TotalBondedTokens nolint
func (DummyStakingKeeper) TotalBondedTokens(_ sdk.Context) sdk.Int {
	return sdkmath.ZeroInt()
}

// Slash nolint
func (DummyStakingKeeper) Slash(sdk.Context, sdk.ConsAddress, int64, int64, math.LegacyDec) sdkmath.Int {
	return sdkmath.ZeroInt()
}

// ValidatorsPowerStoreIterator nolint
func (DummyStakingKeeper) ValidatorsPowerStoreIterator(ctx sdk.Context) sdk.Iterator {
	return sdk.KVStoreReversePrefixIterator(nil, nil)
}

// Jail nolint
func (DummyStakingKeeper) Jail(sdk.Context, sdk.ConsAddress) {
}

// GetLastValidatorPower nolint
func (sk DummyStakingKeeper) GetLastValidatorPower(ctx sdk.Context, operator sdk.ValAddress) (power int64) {
	return sk.Validator(ctx, operator).GetConsensusPower(sdk.DefaultPowerReduction)
}

// MaxValidators returns the maximum amount of bonded validators
func (DummyStakingKeeper) MaxValidators(sdk.Context) uint32 {
	return 100
}

// PowerReduction - is the amount of staking tokens required for 1 unit of consensus-engine power
func (DummyStakingKeeper) PowerReduction(ctx sdk.Context) (res sdk.Int) {
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
func (v MockValidator) GetOperator() sdk.ValAddress           { return v.valOperAddr }
func (MockValidator) ConsPubKey() (cryptotypes.PubKey, error) { return nil, nil }
func (MockValidator) TmConsPublicKey() (tmprotocrypto.PublicKey, error) {
	return tmprotocrypto.PublicKey{}, nil
}
func (MockValidator) GetConsAddr() (sdk.ConsAddress, error) { return nil, nil }
func (v MockValidator) GetTokens() sdk.Int {
	return sdk.TokensFromConsensusPower(v.power, sdk.DefaultPowerReduction)
}

func (v MockValidator) GetBondedTokens() sdk.Int {
	return sdk.TokensFromConsensusPower(v.power, sdk.DefaultPowerReduction)
}
func (v MockValidator) GetConsensusPower(powerReduction sdk.Int) int64 { return v.power }
func (v *MockValidator) SetConsensusPower(power int64)                 { v.power = power }
func (v MockValidator) GetCommission() math.LegacyDec                  { return sdkmath.LegacyZeroDec() }
func (v MockValidator) GetMinSelfDelegation() sdk.Int                  { return sdkmath.OneInt() }
func (v MockValidator) GetDelegatorShares() math.LegacyDec             { return sdkmath.LegacyNewDec(v.power) }
func (v MockValidator) TokensFromShares(math.LegacyDec) math.LegacyDec {
	return sdkmath.LegacyZeroDec()
}
func (v MockValidator) TokensFromSharesTruncated(math.LegacyDec) math.LegacyDec {
	return sdkmath.LegacyZeroDec()
}
func (v MockValidator) TokensFromSharesRoundUp(math.LegacyDec) math.LegacyDec {
	return sdkmath.LegacyZeroDec()
}
func (v MockValidator) SharesFromTokens(amt sdk.Int) (math.LegacyDec, error) {
	return sdkmath.LegacyZeroDec(), nil
}

func (v MockValidator) SharesFromTokensTruncated(amt sdk.Int) (math.LegacyDec, error) {
	return sdkmath.LegacyZeroDec(), nil
}

func NewMockValidator(valAddr sdk.ValAddress, power int64) MockValidator {
	return MockValidator{
		power:       power,
		valOperAddr: valAddr,
	}
}
