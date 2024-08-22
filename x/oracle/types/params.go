package types

import (
	"fmt"
	time "time"

	"cosmossdk.io/math"

	"github.com/NibiruChain/nibiru/v2/x/common/asset"
	"github.com/NibiruChain/nibiru/v2/x/common/denoms"
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
	DefaultVotePeriod       = 30   // vote every 1 minute
	DefaultSlashWindow      = 3600 // 2 hours
	DefaultMinVoters        = 4    // minimum of 4 voters for a pair to become valid
	DefaultExpirationBlocks = 900  // 30 minutes
)

// Default parameter values
var (
	DefaultVoteThreshold = math.LegacyOneDec().Quo(math.LegacyNewDec(3)) // 33.33%
	DefaultRewardBand    = math.LegacyNewDecWithPrec(2, 2)               // 2% (-1, 1)
	DefaultWhitelist     = []asset.Pair{
		// paired against the US fiat dollar
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
	DefaultSlashFraction      = math.LegacyNewDecWithPrec(5, 3)  // 0.5%
	DefaultMinValidPerWindow  = math.LegacyNewDecWithPrec(69, 2) // 69%
	DefaultTwapLookbackWindow = time.Duration(15 * time.Minute)  // 15 minutes
	DefaultValidatorFeeRatio  = math.LegacyNewDecWithPrec(5, 2)  // 0.05%
)

// DefaultParams creates default oracle module parameters
func DefaultParams() Params {
	return Params{
		VotePeriod:         DefaultVotePeriod,
		VoteThreshold:      DefaultVoteThreshold,
		MinVoters:          DefaultMinVoters,
		ExpirationBlocks:   DefaultExpirationBlocks,
		RewardBand:         DefaultRewardBand,
		Whitelist:          DefaultWhitelist,
		SlashFraction:      DefaultSlashFraction,
		SlashWindow:        DefaultSlashWindow,
		MinValidPerWindow:  DefaultMinValidPerWindow,
		TwapLookbackWindow: DefaultTwapLookbackWindow,
		ValidatorFeeRatio:  DefaultValidatorFeeRatio,
	}
}

// Validate performs basic validation on oracle parameters.
func (p Params) Validate() error {
	if p.VotePeriod == 0 {
		return fmt.Errorf("oracle parameter VotePeriod must be > 0, is %d", p.VotePeriod)
	}

	if p.VoteThreshold.LTE(math.LegacyNewDecWithPrec(33, 2)) {
		return fmt.Errorf("oracle parameter VoteThreshold must be greater than 33 percent")
	}

	if p.MinVoters <= 0 {
		return fmt.Errorf("oracle parameter MinVoters must be greater than 0")
	}

	if p.RewardBand.GT(math.LegacyOneDec()) || p.RewardBand.IsNegative() {
		return fmt.Errorf("oracle parameter RewardBand must be between [0, 1]")
	}

	if p.SlashFraction.GT(math.LegacyOneDec()) || p.SlashFraction.IsNegative() {
		return fmt.Errorf("oracle parameter SlashFraction must be between [0, 1]")
	}

	if p.SlashWindow < p.VotePeriod {
		return fmt.Errorf("oracle parameter SlashWindow must be greater than or equal with VotePeriod")
	}

	if p.MinValidPerWindow.GT(math.LegacyOneDec()) || p.MinValidPerWindow.IsNegative() {
		return fmt.Errorf("oracle parameter MinValidPerWindow must be between [0, 1]")
	}

	if p.ValidatorFeeRatio.GT(math.LegacyOneDec()) || p.ValidatorFeeRatio.IsNegative() {
		return fmt.Errorf("oracle parameter ValidatorFeeRatio must be between [0, 1]")
	}

	for _, pair := range p.Whitelist {
		if err := pair.Validate(); err != nil {
			return fmt.Errorf("oracle parameter Whitelist Pair invalid format: %w", err)
		}
	}
	return nil
}
