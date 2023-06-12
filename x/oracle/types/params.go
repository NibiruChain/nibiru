package types

import (
	"fmt"
	time "time"

	"gopkg.in/yaml.v2"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Parameter keys
var (
	KeyVotePeriod         = []byte("VotePeriod")
	KeyVoteThreshold      = []byte("VoteThreshold")
	KeyMinVoters          = []byte("MinVoters")
	KeyRewardBand         = []byte("RewardBand")
	KeyWhitelist          = []byte("Whitelist")
	KeySlashFraction      = []byte("SlashFraction")
	KeySlashWindow        = []byte("SlashWindow")
	KeyMinValidPerWindow  = []byte("MinValidPerWindow")
	KeyTwapLookbackWindow = []byte("TwapLookbackWindow")
	KeyValidatorFeeRatio  = []byte("ValidatorFeeRatio")
)

// Default parameter values
// Assumes block times are 2s
const (
	DefaultVotePeriod  = 30   // vote every 1 minute
	DefaultSlashWindow = 3600 // 2 hours
	DefaultMinVoters   = 4    // minimum of 4 voters for a pair to become valid
)

// Default parameter values
var (
	DefaultVoteThreshold = sdk.OneDec().Quo(sdk.NewDec(3)) // 33.33%
	DefaultRewardBand    = sdk.NewDecWithPrec(2, 2)        // 2% (-1, 1)
	DefaultWhitelist     = []asset.Pair{

		// paired against NUSD
		asset.Registry.Pair(denoms.NIBI, denoms.NUSD),
		asset.Registry.Pair(denoms.BTC, denoms.NUSD),
		asset.Registry.Pair(denoms.ETH, denoms.NUSD),
		asset.Registry.Pair(denoms.ATOM, denoms.NUSD),
		asset.Registry.Pair(denoms.BNB, denoms.NUSD),
		asset.Registry.Pair(denoms.USDC, denoms.NUSD),
		asset.Registry.Pair(denoms.USDT, denoms.NUSD),
		// asset.Registry.Pair(denoms.OSMO, denoms.NUSD),
		// asset.Registry.Pair(denoms.AVAX, denoms.NUSD),
		// asset.Registry.Pair(denoms.SOL, denoms.NUSD),
		// asset.Registry.Pair(denoms.ADA, denoms.NUSD),

		// paired against the US fiat dollar
		asset.Registry.Pair(denoms.NIBI, denoms.USD),
		asset.Registry.Pair(denoms.BTC, denoms.USD),
		asset.Registry.Pair(denoms.ETH, denoms.USD),
		asset.Registry.Pair(denoms.ATOM, denoms.USD),
		asset.Registry.Pair(denoms.BNB, denoms.USD),
		asset.Registry.Pair(denoms.USDC, denoms.USD),
		asset.Registry.Pair(denoms.USDT, denoms.USD),
		// asset.Registry.Pair(denoms.OSMO, denoms.USD),
		// asset.Registry.Pair(denoms.AVAX, denoms.USD),
		// asset.Registry.Pair(denoms.SOL, denoms.USD),
		// asset.Registry.Pair(denoms.ADA, denoms.USD),
	}
	DefaultSlashFraction      = sdk.NewDecWithPrec(5, 3)        // 0.5%
	DefaultMinValidPerWindow  = sdk.NewDecWithPrec(69, 2)       // 69%
	DefaultTwapLookbackWindow = time.Duration(15 * time.Minute) // 15 minutes
	DefaultValidatorFeeRatio  = sdk.NewDecWithPrec(5, 2)        // 0.05%
)

// DefaultParams creates default oracle module parameters
func DefaultParams() Params {
	return Params{
		VotePeriod:         DefaultVotePeriod,
		VoteThreshold:      DefaultVoteThreshold,
		MinVoters:          DefaultMinVoters,
		RewardBand:         DefaultRewardBand,
		Whitelist:          DefaultWhitelist,
		SlashFraction:      DefaultSlashFraction,
		SlashWindow:        DefaultSlashWindow,
		MinValidPerWindow:  DefaultMinValidPerWindow,
		TwapLookbackWindow: DefaultTwapLookbackWindow,
		ValidatorFeeRatio:  DefaultValidatorFeeRatio,
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

	if p.MinVoters <= 0 {
		return fmt.Errorf("oracle parameter MinVoters must be greater than 0")
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

	if p.ValidatorFeeRatio.GT(sdk.OneDec()) || p.ValidatorFeeRatio.IsNegative() {
		return fmt.Errorf("oracle parameter ValidatorFeeRatio must be between [0, 1]")
	}

	for _, pair := range p.Whitelist {
		if err := pair.Validate(); err != nil {
			return fmt.Errorf("oracle parameter Whitelist Pair invalid format: %w", err)
		}
	}
	return nil
}
