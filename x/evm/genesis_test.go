package evm_test

import (
	"encoding/json"
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
		wantErr  string
	}{
		{
			name:     "default",
			genState: evm.DefaultGenesisState(),
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
		},
		{
			name:     "empty genesis",
			genState: &evm.GenesisState{},
		},
		{
			name: "copied genesis",
			genState: &evm.GenesisState{
				Accounts: evm.DefaultGenesisState().Accounts,
				Params:   evm.DefaultGenesisState().Params,
			},
		},
		{
			name: "happy genesis with account",
			genState: &evm.GenesisState{
				Accounts: []evm.GenesisAccount{
					{
						Address: gethcommon.Address{}.String(), // zero address
					},
				},
			},
		},
		{
			name: "invalid genesis account",
			genState: &evm.GenesisState{
				Accounts: []evm.GenesisAccount{
					{
						Address: "123456", // not a valid ethereum hex address

						Code: s.code,
						Storage: evm.Storage{
							{Key: s.hash.String()},
						},
					},
				},
				Params: evm.DefaultParams(),
			},
			wantErr: "not a valid ethereum hex address",
		},
		{
			name:    "duplicate account",
			wantErr: "duplicate genesis account",
			genState: &evm.GenesisState{
				Accounts: []evm.GenesisAccount{
					{
						Address: s.address,

						Code: s.code,
						Storage: evm.Storage{
							{Key: s.hash.String()},
						},
					},
					{
						Address: s.address,

						Code: s.code,
						Storage: evm.Storage{
							{Key: s.hash.String()},
						},
					},
				},
			},
		},
		{
			name: "happy: empty params",
			genState: &evm.GenesisState{
				Params: evm.Params{},
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			err := tc.genState.Validate()
			jsonBz, _ := json.Marshal(tc.genState)
			jsonStr := string(jsonBz)
			if tc.wantErr != "" {
				s.Require().ErrorContains(err, tc.wantErr, jsonStr)
				return
			}
			s.Require().NoError(err, jsonStr)
		})
	}
}
