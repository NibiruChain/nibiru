package types

import (
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/NibiruChain/nibiru/x/common"
)

const (
	ProposalTypeCreatePool = "CreatePool"
)

var _ govtypes.Content = &CreatePoolProposal{}

// NEVER MUTATE THESE!
// they exist for comparisons only and to avoid constant allocations of a new big int
// which is used only for reading.
var (
	oneDec  = sdk.OneDec()
	zeroDec = sdk.ZeroDec()
)

func init() {
	govtypes.RegisterProposalType(ProposalTypeCreatePool)
	govtypes.RegisterProposalTypeCodec(&CreatePoolProposal{}, "nibiru/CreatePoolProposal")
}

func (m *CreatePoolProposal) ProposalRoute() string {
	return RouterKey
}

func (m *CreatePoolProposal) ProposalType() string {
	return ProposalTypeCreatePool
}

func (m *CreatePoolProposal) ValidateBasic() error {
	if err := govtypes.ValidateAbstract(m); err != nil {
		return err
	}

	if _, err := common.NewAssetPair(m.Pair); err != nil {
		return err
	}

	// trade limit ratio always between 0 and 1
	// TODO(mercilex): does it really make sense for this to be equal to zero?
	if m.TradeLimitRatio.LT(zeroDec) || m.TradeLimitRatio.GT(oneDec) {
		return fmt.Errorf("trade limit ratio must be 0 <= ratio <= 1")
	}

	// quote asset reserve always > 0
	if m.QuoteAssetReserve.LTE(zeroDec) {
		return fmt.Errorf("quote asset reserve must be > 0")
	}

	// base asset reserve always > 0
	if m.BaseAssetReserve.LTE(zeroDec) {
		return fmt.Errorf("base asset reserve must be > 0")
	}

	// fluctuation limit ratio between 0 and 1
	// TODO(mercilex): does it really make sense for this to be equal to zero?
	if m.FluctuationLimitRatio.LT(zeroDec) || m.FluctuationLimitRatio.GT(oneDec) {
		return fmt.Errorf("fluctuation limit ratio must be 0 <= ratio <= 1")
	}

	// max oracle spread ratio between 0 and 1
	// TODO(mercilex): does it really make sense for this to be equal to zero?
	if m.MaxOracleSpreadRatio.LT(zeroDec) || m.MaxOracleSpreadRatio.GT(oneDec) {
		return fmt.Errorf("max oracle spread ratio must be 0 <= ratio <= 1")
	}

	return nil
}
