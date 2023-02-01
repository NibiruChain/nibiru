package types

import (
	"fmt"
	time "time"

	"gopkg.in/yaml.v2"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"

	sdk "github.com/cosmos/cosmos-sdk/types"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

// Parameter keys
var (
	KeyVotePeriod         = []byte("VotePeriod")
	KeyVoteThreshold      = []byte("VoteThreshold")
	KeyRewardBand         = []byte("RewardBand")
	KeyWhitelist          = []byte("Whitelist")
	KeySlashFraction      = []byte("SlashFraction")
	KeySlashWindow        = []byte("SlashWindow")
	KeyMinValidPerWindow  = []byte("MinValidPerWindow")
	KeyTwapLookbackWindow = []byte("TwapLookbackWindow")
)

// Default parameter values
// TODO(mercilex): need to adjust this based on expected genesis parameters, this assumes block times are 1s
// DefaultVotePeriod: 10s
// DefaultSlashWindow: 1 Week
const (
	DefaultVotePeriod  = 10     // vote every 10s
	DefaultSlashWindow = 604800 // 1 week
)

// Default parameter values
var (
	DefaultVoteThreshold = sdk.NewDecWithPrec(50, 2) // 50%
	DefaultRewardBand    = sdk.NewDecWithPrec(2, 2)  // 2% (-1, 1)
	DefaultWhitelist     = []common.AssetPair{

		// paired against NUSD
		asset.Registry.Pair(denoms.NIBI, denoms.NUSD),
		asset.Registry.Pair(denoms.BTC, denoms.NUSD),
		asset.Registry.Pair(denoms.ETH, denoms.NUSD),
		asset.Registry.Pair(denoms.ATOM, denoms.NUSD),
		asset.Registry.Pair(denoms.OSMO, denoms.NUSD),
		asset.Registry.Pair(denoms.AVAX, denoms.NUSD),
		asset.Registry.Pair(denoms.SOL, denoms.NUSD),
		asset.Registry.Pair(denoms.ADA, denoms.NUSD),
		asset.Registry.Pair(denoms.BNB, denoms.NUSD),
		asset.Registry.Pair(denoms.USDC, denoms.NUSD),
		asset.Registry.Pair(denoms.USDT, denoms.NUSD),
		asset.Registry.Pair(denoms.NIBI, denoms.NUSD),

		// paired against the US fiat dollar
		asset.Registry.Pair(denoms.BTC, denoms.USD),
		asset.Registry.Pair(denoms.ETH, denoms.USD),
		asset.Registry.Pair(denoms.ATOM, denoms.USD),
		asset.Registry.Pair(denoms.OSMO, denoms.USD),
		asset.Registry.Pair(denoms.AVAX, denoms.USD),
		asset.Registry.Pair(denoms.SOL, denoms.USD),
		asset.Registry.Pair(denoms.ADA, denoms.USD),
		asset.Registry.Pair(denoms.BNB, denoms.USD),
		asset.Registry.Pair(denoms.USDC, denoms.USD),
		asset.Registry.Pair(denoms.USDT, denoms.USD),
	}
	DefaultSlashFraction      = sdk.NewDecWithPrec(1, 4)        // 0.01%
	DefaultMinValidPerWindow  = sdk.NewDecWithPrec(5, 2)        // 5%
	DefaultTwapLookbackWindow = time.Duration(15 * time.Minute) // 15 minutes
)

var _ paramstypes.ParamSet = &Params{}

// DefaultParams creates default oracle module parameters
func DefaultParams() Params {
	return Params{
		VotePeriod:         DefaultVotePeriod,
		VoteThreshold:      DefaultVoteThreshold,
		RewardBand:         DefaultRewardBand,
		Whitelist:          DefaultWhitelist,
		SlashFraction:      DefaultSlashFraction,
		SlashWindow:        DefaultSlashWindow,
		MinValidPerWindow:  DefaultMinValidPerWindow,
		TwapLookbackWindow: DefaultTwapLookbackWindow,
	}
}

// ParamKeyTable returns the parameter key table.
func ParamKeyTable() paramstypes.KeyTable {
	return paramstypes.NewKeyTable().RegisterParamSet(&Params{})
}

// ParamSetPairs implements the ParamSet interface and returns all the key/value pairs
// pairs of oracle module's parameters.
func (p *Params) ParamSetPairs() paramstypes.ParamSetPairs {
	return paramstypes.ParamSetPairs{
		paramstypes.NewParamSetPair(KeyVotePeriod, &p.VotePeriod, validateVotePeriod),
		paramstypes.NewParamSetPair(KeyVoteThreshold, &p.VoteThreshold, validateVoteThreshold),
		paramstypes.NewParamSetPair(KeyRewardBand, &p.RewardBand, validateRewardBand),
		paramstypes.NewParamSetPair(KeyWhitelist, &p.Whitelist, validateWhitelist),
		paramstypes.NewParamSetPair(KeySlashFraction, &p.SlashFraction, validateSlashFraction),
		paramstypes.NewParamSetPair(KeySlashWindow, &p.SlashWindow, validateSlashWindow),
		paramstypes.NewParamSetPair(KeyMinValidPerWindow, &p.MinValidPerWindow, validateMinValidPerWindow),
		paramstypes.NewParamSetPair(KeyTwapLookbackWindow, &p.TwapLookbackWindow, validateTwapLookbackWindow),
	}
}

// String implements fmt.Stringer interface
func (p Params) String() string {
	out, _ := yaml.Marshal(p)
	return string(out)
}

// Validate performs basic validation on oracle parameters.
func (p Params) Validate() error {
	if p.VotePeriod == 0 {
		return fmt.Errorf("oracle parameter VotePeriod must be > 0, is %d", p.VotePeriod)
	}
	if p.VoteThreshold.LTE(sdk.NewDecWithPrec(33, 2)) {
		return fmt.Errorf("oracle parameter VoteThreshold must be greater than 33 percent")
	}

	if p.RewardBand.GT(sdk.OneDec()) || p.RewardBand.IsNegative() {
		return fmt.Errorf("oracle parameter RewardBand must be between [0, 1]")
	}

	if p.SlashFraction.GT(sdk.OneDec()) || p.SlashFraction.IsNegative() {
		return fmt.Errorf("oracle parameter SlashFraction must be between [0, 1]")
	}

	if p.SlashWindow < p.VotePeriod {
		return fmt.Errorf("oracle parameter SlashWindow must be greater than or equal with VotePeriod")
	}

	if p.MinValidPerWindow.GT(sdk.OneDec()) || p.MinValidPerWindow.IsNegative() {
		return fmt.Errorf("oracle parameter MinValidPerWindow must be between [0, 1]")
	}

	for _, pair := range p.Whitelist {
		if err := pair.Validate(); err != nil {
			return fmt.Errorf("oracle parameter Whitelist Pair invalid format: %w", err)
		}
	}
	return nil
}

func validateVotePeriod(i interface{}) error {
	v, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v == 0 {
		return fmt.Errorf("vote period must be positive: %d", v)
	}

	return nil
}

func validateVoteThreshold(i interface{}) error {
	v, ok := i.(sdk.Dec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v.LT(sdk.NewDecWithPrec(33, 2)) {
		return fmt.Errorf("vote threshold must be bigger than 33%%: %s", v)
	}

	if v.GT(sdk.OneDec()) {
		return fmt.Errorf("vote threshold too large: %s", v)
	}

	return nil
}

func validateRewardBand(i interface{}) error {
	v, ok := i.(sdk.Dec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v.IsNegative() {
		return fmt.Errorf("reward band must be positive: %s", v)
	}

	if v.GT(sdk.OneDec()) {
		return fmt.Errorf("reward band is too large: %s", v)
	}

	return nil
}

func validateWhitelist(i interface{}) error {
	v, ok := i.([]common.AssetPair)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	for _, d := range v {
		if err := d.Validate(); err != nil {
			return fmt.Errorf("oracle parameter Whitelist Pair invalid format: %w", err)
		}
	}

	return nil
}

func validateSlashFraction(i interface{}) error {
	v, ok := i.(sdk.Dec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v.IsNegative() {
		return fmt.Errorf("slash fraction must be positive: %s", v)
	}

	if v.GT(sdk.OneDec()) {
		return fmt.Errorf("slash fraction is too large: %s", v)
	}

	return nil
}

func validateSlashWindow(i interface{}) error {
	v, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v == 0 {
		return fmt.Errorf("slash window must be positive: %d", v)
	}

	return nil
}

func validateMinValidPerWindow(i interface{}) error {
	v, ok := i.(sdk.Dec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v.IsNegative() {
		return fmt.Errorf("min valid per window must be positive: %s", v)
	}

	if v.GT(sdk.OneDec()) {
		return fmt.Errorf("min valid per window is too large: %s", v)
	}

	return nil
}

func validateTwapLookbackWindow(i interface{}) error {
	v, ok := i.(time.Duration)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if v < 0 {
		return fmt.Errorf("look back twap duration should be positive: %s", v)
	}
	return nil
}
