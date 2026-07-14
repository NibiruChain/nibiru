package ibctesting

import (
	"time"

	connectiontypes "github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/core/03-connection/types"
	channeltypes "github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/core/04-channel/types"
	"github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/core/exported"
	ibctm "github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/light-clients/07-tendermint"
	"github.com/NibiruChain/nibiru/v2/lib/ibc-go/testing/mock"
)

type ClientConfig interface {
	GetClientType() string
}

type TendermintConfig struct {
	TrustLevel      ibctm.Fraction
	TrustingPeriod  time.Duration
	UnbondingPeriod time.Duration
	MaxClockDrift   time.Duration
}

func NewTendermintConfig() *TendermintConfig {
	return &TendermintConfig{
		TrustLevel:      DefaultTrustLevel,
		TrustingPeriod:  TrustingPeriod,
		UnbondingPeriod: UnbondingPeriod,
		MaxClockDrift:   MaxClockDrift,
	}
}

func (tmcfg *TendermintConfig) GetClientType() string {
	return exported.Tendermint
}

type ConnectionConfig struct {
	DelayPeriod uint64
	Version     *connectiontypes.Version
}

func NewConnectionConfig() *ConnectionConfig {
	return &ConnectionConfig{
		DelayPeriod: DefaultDelayPeriod,
		Version:     ConnectionVersion,
	}
}

type ChannelConfig struct {
	PortID  string
	Version string
	Order   channeltypes.Order
}

func NewChannelConfig() *ChannelConfig {
	return &ChannelConfig{
		PortID:  mock.PortID,
		Version: DefaultChannelVersion,
		Order:   channeltypes.UNORDERED,
	}
}
