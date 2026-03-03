package keeper_test

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/NibiruChain/nibiru/v2/x/sudo"
)

func (s *Suite) TestQuerySudoers() {
	for _, tc := range []struct {
		name  string
		state sudo.Sudoers
	}{
		{
			name: "happy 1",
			state: sudo.Sudoers{
				Root:      "alice",
				Contracts: []string{"contractA", "contractB"},
			},
		},

		{
			name: "happy 2 (empty)",
			state: sudo.Sudoers{
				Root:      "",
				Contracts: []string(nil),
			},
		},

		{
			name: "happy 3",
			state: sudo.Sudoers{
				Root:      "",
				Contracts: []string{"boop", "blap"},
			},
		},
	} {
		s.Run(tc.name, func() {
			_, k, ctx := setup()

			k.Sudoers.Set(ctx, tc.state)

			req := new(sudo.QuerySudoersRequest)
			resp, err := k.QuerySudoers(
				sdk.WrapSDKContext(ctx), req,
			)
			s.Require().NoError(err)

			outSudoers := resp.Sudoers
			s.Require().EqualValues(tc.state, outSudoers)
		})
	}

	s.Run("nil request should not error", func() {
		_, k, ctx := setup()
		var req *sudo.QuerySudoersRequest = nil
		_, err := k.QuerySudoers(
			sdk.WrapSDKContext(ctx), req,
		)
		s.Require().NoError(err)
	})
}

func (s *Suite) TestGetZeroGasEvmContracts() {
	addr1 := gethcommon.BigToAddress(big.NewInt(1500))
	addr2 := gethcommon.BigToAddress(big.NewInt(2500))

	for _, tc := range []struct {
		name     string
		setState bool
		actors   sudo.ZeroGasActors
		expected []gethcommon.Address
	}{
		{
			name:     "default (unset) returns empty",
			setState: false,
			expected: nil,
		},
		{
			name:     "empty AlwaysZeroGasContracts",
			setState: true,
			actors:   sudo.DefaultZeroGasActors(),
			expected: nil,
		},
		{
			name:     "single valid EIP55",
			setState: true,
			actors: sudo.ZeroGasActors{
				AlwaysZeroGasContracts: []string{addr1.Hex()},
			},
			expected: []gethcommon.Address{addr1},
		},
		{
			name:     "multiple valid",
			setState: true,
			actors: sudo.ZeroGasActors{
				AlwaysZeroGasContracts: []string{addr1.Hex(), addr2.Hex()},
			},
			expected: []gethcommon.Address{addr1, addr2},
		},
		{
			name:     "duplicates in list produce unique map",
			setState: true,
			actors: sudo.ZeroGasActors{
				AlwaysZeroGasContracts: []string{addr1.Hex(), addr1.Hex(), addr2.Hex()},
			},
			expected: []gethcommon.Address{addr1, addr2},
		},
		{
			name:     "invalid hex skipped",
			setState: true,
			actors: sudo.ZeroGasActors{
				AlwaysZeroGasContracts: []string{"nibi1invalid", addr1.Hex()},
			},
			expected: []gethcommon.Address{addr1},
		},
		{
			name:     "mixed valid and invalid",
			setState: true,
			actors: sudo.ZeroGasActors{
				AlwaysZeroGasContracts: []string{"bad", addr1.Hex(), "0xNotHex", addr2.Hex()},
			},
			expected: []gethcommon.Address{addr1, addr2},
		},
		{
			name:     "all invalid returns empty",
			setState: true,
			actors: sudo.ZeroGasActors{
				AlwaysZeroGasContracts: []string{"nibi1xxx", "not-an-address"},
			},
			expected: nil,
		},
		{
			name:     "EIP55 normalization same address",
			setState: true,
			actors: sudo.ZeroGasActors{
				AlwaysZeroGasContracts: []string{addr1.Hex(), addr1.Hex()},
			},
			expected: []gethcommon.Address{addr1},
		},
		{
			name:     "ignores Senders and Contracts",
			setState: true,
			actors: sudo.ZeroGasActors{
				Senders:                []string{"nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl"},
				Contracts:              []string{"nibi1ah8gqrtjllhc5ld4rxgl4uglvwl93ag0sh6e6v"},
				AlwaysZeroGasContracts: []string{},
			},
			expected: nil,
		},
	} {
		s.Run(tc.name, func() {
			_, k, ctx := setup()
			if tc.setState {
				k.ZeroGasActors.Set(ctx, tc.actors)
			}
			got := k.GetZeroGasEvmContracts(ctx)
			s.Require().Len(got, len(tc.expected))
			for _, addr := range tc.expected {
				_, ok := got[addr]
				s.Require().True(ok, "expected %s in result", addr.Hex())
			}
		})
	}
}
