package types_test

import (
	"testing"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/eth/crypto/ethsecp256k1"
	"github.com/NibiruChain/nibiru/x/evm/types"
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
		genAcc  types.GenesisAccount
		expPass bool
	}{
		{
			name: "valid genesis account",
			genAcc: types.GenesisAccount{
				Address: s.address,
				Code:    s.code,
				Storage: types.Storage{
					types.NewStateFromEthHashes(s.hash, s.hash),
				},
			},
			expPass: true,
		},
		{
			name: "empty account address bytes",
			genAcc: types.GenesisAccount{
				Address: "",
				Code:    s.code,
				Storage: types.Storage{
					types.NewStateFromEthHashes(s.hash, s.hash),
				},
			},
			expPass: false,
		},
		{
			name: "empty code bytes",
			genAcc: types.GenesisAccount{
				Address: s.address,
				Code:    "",
				Storage: types.Storage{
					types.NewStateFromEthHashes(s.hash, s.hash),
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
		genState *types.GenesisState
		expPass  bool
	}{
		{
			name:     "default",
			genState: types.DefaultGenesisState(),
			expPass:  true,
		},
		{
			name: "valid genesis",
			genState: &types.GenesisState{
				Accounts: []types.GenesisAccount{
					{
						Address: s.address,

						Code: s.code,
						Storage: types.Storage{
							{Key: s.hash.String()},
						},
					},
				},
				Params: types.DefaultParams(),
			},
			expPass: true,
		},
		{
			name:     "empty genesis",
			genState: &types.GenesisState{},
			expPass:  false,
		},
		{
			name: "copied genesis",
			genState: &types.GenesisState{
				Accounts: types.DefaultGenesisState().Accounts,
				Params:   types.DefaultGenesisState().Params,
			},
			expPass: true,
		},
		{
			name: "invalid genesis",
			genState: &types.GenesisState{
				Accounts: []types.GenesisAccount{
					{
						Address: gethcommon.Address{}.String(),
					},
				},
			},
			expPass: false,
		},
		{
			name: "invalid genesis account",
			genState: &types.GenesisState{
				Accounts: []types.GenesisAccount{
					{
						Address: "123456",

						Code: s.code,
						Storage: types.Storage{
							{Key: s.hash.String()},
						},
					},
				},
				Params: types.DefaultParams(),
			},
			expPass: false,
		},
		{
			name: "duplicated genesis account",
			genState: &types.GenesisState{
				Accounts: []types.GenesisAccount{
					{
						Address: s.address,

						Code: s.code,
						Storage: types.Storage{
							types.NewStateFromEthHashes(s.hash, s.hash),
						},
					},
					{
						Address: s.address,

						Code: s.code,
						Storage: types.Storage{
							types.NewStateFromEthHashes(s.hash, s.hash),
						},
					},
				},
			},
			expPass: false,
		},
		{
			name: "duplicated tx log",
			genState: &types.GenesisState{
				Accounts: []types.GenesisAccount{
					{
						Address: s.address,

						Code: s.code,
						Storage: types.Storage{
							{Key: s.hash.String()},
						},
					},
				},
			},
			expPass: false,
		},
		{
			name: "invalid tx log",
			genState: &types.GenesisState{
				Accounts: []types.GenesisAccount{
					{
						Address: s.address,

						Code: s.code,
						Storage: types.Storage{
							{Key: s.hash.String()},
						},
					},
				},
			},
			expPass: false,
		},
		{
			name: "invalid params",
			genState: &types.GenesisState{
				Params: types.Params{},
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
