package sudo_test

import (
	"bytes"
	"math/big"
	"testing"

	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/v2/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil"
	"github.com/NibiruChain/nibiru/v2/x/sudo"
	wasmtypes "github.com/NibiruChain/nibiru/v2/x/wasm/types"
)

var _ suite.SetupAllSuite = (*Suite)(nil) // LogRoutingSuite has a setup fn

type Suite struct {
	testutil.LogRoutingSuite
}

func TestSudo(t *testing.T) {
	evmtest.EnsureNibiruPrefix()
	suite.Run(t, new(Suite))
}

func (s *Suite) TestMsgEditSudoers_ValidateBasic_EditWasmBlockHooksContract() {
	goodSender := testutil.NewAccAddress().String()
	goodContract := sdk.AccAddress(bytes.Repeat([]byte{1}, wasmtypes.ContractAddrLen)).String()
	sdkLenAddr := testutil.NewAccAddress().String()

	cases := []struct {
		name    string
		msg     sudo.MsgEditSudoers
		wantErr string
	}{
		{
			name: "ok: set contract",
			msg: sudo.MsgEditSudoers{
				Action:    string(sudo.EditWasmBlockHooksContract),
				Sender:    goodSender,
				Contracts: []string{goodContract},
			},
		},
		{
			name: "ok: clear contract",
			msg: sudo.MsgEditSudoers{
				Action:    string(sudo.EditWasmBlockHooksContract),
				Sender:    goodSender,
				Contracts: []string{""},
			},
		},
		{
			name: "err: zero contracts",
			msg: sudo.MsgEditSudoers{
				Action: string(sudo.EditWasmBlockHooksContract),
				Sender: goodSender,
			},
			wantErr: "expects exactly one contract argument",
		},
		{
			name: "err: multiple contracts",
			msg: sudo.MsgEditSudoers{
				Action:    string(sudo.EditWasmBlockHooksContract),
				Sender:    goodSender,
				Contracts: []string{goodContract, goodContract},
			},
			wantErr: "expects exactly one contract argument",
		},
		{
			name: "err: invalid bech32",
			msg: sudo.MsgEditSudoers{
				Action:    string(sudo.EditWasmBlockHooksContract),
				Sender:    goodSender,
				Contracts: []string{"not-an-address"},
			},
			wantErr: "decoding bech32 failed",
		},
		{
			name: "err: valid account address with sdk length",
			msg: sudo.MsgEditSudoers{
				Action:    string(sudo.EditWasmBlockHooksContract),
				Sender:    goodSender,
				Contracts: []string{sdkLenAddr},
			},
			wantErr: "wasm block hooks contract address must be 32 bytes",
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			err := tc.msg.ValidateBasic()
			if tc.wantErr == "" {
				s.Require().NoError(err)
				return
			}
			s.Require().ErrorContains(err, tc.wantErr)
		})
	}
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
			name: "ok: always_zero_gas_contracts with valid EVM addresses",
			build: func() *sudo.MsgEditZeroGasActors {
				return &sudo.MsgEditZeroGasActors{
					Sender: goodSender,
					Actors: sudo.ZeroGasActors{
						AlwaysZeroGasContracts: goodContracts,
					},
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
			name: "err: invalid address in always_zero_gas_contracts",
			build: func() *sudo.MsgEditZeroGasActors {
				return &sudo.MsgEditZeroGasActors{
					Sender: goodSender,
					Actors: sudo.ZeroGasActors{
						AlwaysZeroGasContracts: []string{"0xBAD"},
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
