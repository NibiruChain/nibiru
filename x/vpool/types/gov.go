package types

import (
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

	assetPair, err := common.NewAssetPair(m.Pair)
	if err != nil {
		return err
	}
	pool := &VPool{
		Pair:                   assetPair,
		BaseAssetReserve:       m.BaseAssetReserve,
		QuoteAssetReserve:      m.QuoteAssetReserve,
		TradeLimitRatio:        m.TradeLimitRatio,
		FluctuationLimitRatio:  m.FluctuationLimitRatio,
		MaxOracleSpreadRatio:   m.MaxOracleSpreadRatio,
		MaintenanceMarginRatio: m.MaintenanceMarginRatio,
		MaxLeverage:            m.MaxLeverage,
	}

	return pool.Validate()
}
