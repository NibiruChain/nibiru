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
	if m.TradeLimitRatio.LT(sdk.ZeroDec()) || m.TradeLimitRatio.GT(sdk.OneDec()) {
		return fmt.Errorf("trade limit ratio must be 0 <= ratio <= 1")
	}

	// quote asset reserve always > 0
	if !m.QuoteAssetReserve.IsPositive() {
		return fmt.Errorf("quote asset reserve must be > 0")
	}

	// base asset reserve always > 0
	if !m.BaseAssetReserve.IsPositive() {
		return fmt.Errorf("base asset reserve must be > 0")
	}

	// fluctuation limit ratio between 0 and 1
	if m.FluctuationLimitRatio.LT(sdk.ZeroDec()) || m.FluctuationLimitRatio.GT(sdk.OneDec()) {
		return fmt.Errorf("fluctuation limit ratio must be 0 <= ratio <= 1")
	}

	// max oracle spread ratio between 0 and 1
	if m.MaxOracleSpreadRatio.LT(sdk.ZeroDec()) || m.MaxOracleSpreadRatio.GT(sdk.OneDec()) {
		return fmt.Errorf("max oracle spread ratio must be 0 <= ratio <= 1")
	}

	if m.MaintenanceMarginRatio.LT(sdk.ZeroDec()) || m.MaintenanceMarginRatio.GT(sdk.OneDec()) {
		return fmt.Errorf("maintenance margin ratio ratio must be 0 <= ratio <= 1")
	}

	return nil
}
