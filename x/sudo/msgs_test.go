package sudo_test

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil"
	"github.com/NibiruChain/nibiru/v2/x/sudo"
)

var _ suite.SetupAllSuite = (*Suite)(nil) // LogRoutingSuite has a setup fn

type Suite struct {
	testutil.LogRoutingSuite
}

func TestSudo(t *testing.T) {
	evmtest.EnsureNibiruPrefix()
	suite.Run(t, new(Suite))
}

func (s *Suite) TestMsgEditZeroGasActors_ValidateBasic() {
	goodSender := "nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl"
	goodContracts := []string{
		common.BigToAddress(big.NewInt(5)).Hex(),
		common.BigToAddress(big.NewInt(12)).Hex(),
	}

	cases := []struct {
		name    string
		build   func() *sudo.MsgEditZeroGasActors
		wantErr string
	}{
		{
			name: "ok: one contract and one sender",
			build: func() *sudo.MsgEditZeroGasActors {
				return &sudo.MsgEditZeroGasActors{
					Sender: goodSender,
					Actors: sudo.ZeroGasActors{
						Contracts: goodContracts,
						Senders:   []string{goodSender},
					},
				}
			},
		},
		{
			name: "ok: empty lists allowed",
			build: func() *sudo.MsgEditZeroGasActors {
				return &sudo.MsgEditZeroGasActors{
					Sender: goodSender,
					Actors: sudo.ZeroGasActors{},
				}
			},
		},
		{
			name: "err: invalid contract address",
			build: func() *sudo.MsgEditZeroGasActors {
				return &sudo.MsgEditZeroGasActors{
					Sender: goodSender,
					Actors: sudo.ZeroGasActors{
						Contracts: []string{"0xBAD"},
					},
				}
			},
			wantErr: "ZeroGasActors stateless validation error",
		},
		{
			name: "err: invalid sender bech32 in actors",
			build: func() *sudo.MsgEditZeroGasActors {
				return &sudo.MsgEditZeroGasActors{
					Sender: goodSender,
					Actors: sudo.ZeroGasActors{
						Senders: []string{"invalid"},
					},
				}
			},
			wantErr: "ZeroGasActors stateless validation error",
		},
		{
			name: "err: invalid top-level sender",
			build: func() *sudo.MsgEditZeroGasActors {
				return &sudo.MsgEditZeroGasActors{
					Sender: "invalid",
					Actors: sudo.ZeroGasActors{},
				}
			},
			wantErr: "decoding bech32 failed",
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			err := tc.build().ValidateBasic()
			if tc.wantErr == "" {
				s.Require().NoError(err)
			} else {
				s.Require().ErrorContains(err, tc.wantErr)
			}
		})
	}
}
