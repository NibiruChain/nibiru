package evm_test

import (
	"testing"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/v2/eth/crypto/ethsecp256k1"
	"github.com/NibiruChain/nibiru/v2/x/evm"
)

type GenesisSuite struct {
	suite.Suite

	address string
	hash    gethcommon.Hash
	code    string
}

func (s *GenesisSuite) SetupTest() {
	priv, err := ethsecp256k1.GenerateKey()
	s.Require().NoError(err)

	s.address = gethcommon.BytesToAddress(priv.PubKey().Address().Bytes()).String()
	s.hash = gethcommon.BytesToHash([]byte("hash"))
	s.code = gethcommon.Bytes2Hex([]byte{1, 2, 3})
}

func TestGenesisSuite(t *testing.T) {
	suite.Run(t, new(GenesisSuite))
}

func (s *GenesisSuite) TestValidateGenesisAccount() {
	testCases := []struct {
		name    string
		genAcc  evm.GenesisAccount
		expPass bool
	}{
		{
			name: "valid genesis account",
			genAcc: evm.GenesisAccount{
				Address: s.address,
				Code:    s.code,
				Storage: evm.Storage{
					evm.NewStateFromEthHashes(s.hash, s.hash),
				},
			},
			expPass: true,
		},
		{
			name: "empty account address bytes",
			genAcc: evm.GenesisAccount{
				Address: "",
				Code:    s.code,
				Storage: evm.Storage{
					evm.NewStateFromEthHashes(s.hash, s.hash),
				},
			},
			expPass: false,
		},
		{
			name: "empty code bytes",
			genAcc: evm.GenesisAccount{
				Address: s.address,
				Code:    "",
				Storage: evm.Storage{
					evm.NewStateFromEthHashes(s.hash, s.hash),
				},
			},
			expPass: true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		err := tc.genAcc.Validate()
		if tc.expPass {
			s.Require().NoError(err, tc.name)
		} else {
			s.Require().Error(err, tc.name)
		}
	}
}

func (s *GenesisSuite) TestValidateGenesis() {
	testCases := []struct {
		name     string
		genState *evm.GenesisState
		expPass  bool
	}{
		{
			name:     "default",
			genState: evm.DefaultGenesisState(),
			expPass:  true,
		},
		{
			name: "valid genesis",
			genState: &evm.GenesisState{
				Accounts: []evm.GenesisAccount{
					{
						Address: s.address,

						Code: s.code,
						Storage: evm.Storage{
							{Key: s.hash.String()},
						},
					},
				},
				Params: evm.DefaultParams(),
			},
			expPass: true,
		},
		{
			name:     "empty genesis",
			genState: &evm.GenesisState{},
			expPass:  false,
		},
		{
			name: "copied genesis",
			genState: &evm.GenesisState{
				Accounts: evm.DefaultGenesisState().Accounts,
				Params:   evm.DefaultGenesisState().Params,
			},
			expPass: true,
		},
		{
			name: "invalid genesis",
			genState: &evm.GenesisState{
				Accounts: []evm.GenesisAccount{
					{
						Address: gethcommon.Address{}.String(),
					},
				},
			},
			expPass: false,
		},
		{
			name: "invalid genesis account",
			genState: &evm.GenesisState{
				Accounts: []evm.GenesisAccount{
					{
						Address: "123456",

						Code: s.code,
						Storage: evm.Storage{
							{Key: s.hash.String()},
						},
					},
				},
				Params: evm.DefaultParams(),
			},
			expPass: false,
		},
		{
			name: "duplicated genesis account",
			genState: &evm.GenesisState{
				Accounts: []evm.GenesisAccount{
					{
						Address: s.address,

						Code: s.code,
						Storage: evm.Storage{
							evm.NewStateFromEthHashes(s.hash, s.hash),
						},
					},
					{
						Address: s.address,

						Code: s.code,
						Storage: evm.Storage{
							evm.NewStateFromEthHashes(s.hash, s.hash),
						},
					},
				},
			},
			expPass: false,
		},
		{
			name: "duplicated tx log",
			genState: &evm.GenesisState{
				Accounts: []evm.GenesisAccount{
					{
						Address: s.address,

						Code: s.code,
						Storage: evm.Storage{
							{Key: s.hash.String()},
						},
					},
				},
			},
			expPass: false,
		},
		{
			name: "invalid tx log",
			genState: &evm.GenesisState{
				Accounts: []evm.GenesisAccount{
					{
						Address: s.address,

						Code: s.code,
						Storage: evm.Storage{
							{Key: s.hash.String()},
						},
					},
				},
			},
			expPass: false,
		},
		{
			name: "invalid params",
			genState: &evm.GenesisState{
				Params: evm.Params{},
			},
			expPass: false,
		},
	}

	for _, tc := range testCases {
		err := tc.genState.Validate()
		if tc.expPass {
			s.Require().NoError(err, tc.name)
			continue
		}
		s.Require().Error(err, tc.name)
	}
}
