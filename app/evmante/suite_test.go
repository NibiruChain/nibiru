package evmante_test

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/app/evmante"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmstate"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
)

type TestSuite struct {
	testutil.LogRoutingSuite

	encCfg app.EncodingConfig
}

func TestAppTestSuite(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

func (s *TestSuite) SetupSuite() {
	s.LogRoutingSuite.SetupSuite()
	s.encCfg = app.MakeEncodingConfig()
}

type AnteTC struct {
	Name           string
	EvmAnteHandler evmante.EvmAnteHandler
	TxSetup        func(deps *evmtest.TestDeps, sdb *evmstate.SDB) evm.Tx
	MaxGasWanted   uint64
	WantErr        string
}

func RunAnteTCs(s *suite.Suite, tcs []AnteTC) {
	for _, tc := range tcs {
		s.Run(tc.Name, func() {
			deps := evmtest.NewTestDeps()
			sdb := deps.NewStateDB()
			tx := tc.TxSetup(&deps, sdb)
			err := tc.EvmAnteHandler(
				sdb,
				sdb.Keeper(),
				tx,
				false,
				AnteOptionsForTests{MaxTxGasWanted: tc.MaxGasWanted},
			)
			if tc.WantErr != "" {
				s.Require().ErrorContains(err, tc.WantErr)
				return
			}
			s.Require().NoError(err)
		})
	}
}

var _ evmante.AnteOptionsEVM = (*AnteOptionsForTests)(nil)

type AnteOptionsForTests struct {
	MaxTxGasWanted uint64
}

func (opts AnteOptionsForTests) GetMaxTxGasWanted() uint64 {
	return opts.MaxTxGasWanted
}

func (s *TestSuite) TestGenesis() {
	getDefaultStakingGenesis := func() *stakingtypes.GenesisState {
		genStaking := new(stakingtypes.GenesisState)
		s.encCfg.Codec.MustUnmarshalJSON(
			app.StakingModule{}.DefaultGenesis(s.encCfg.Codec),
			genStaking,
		)
		return genStaking
	}

	gens := []*stakingtypes.GenesisState{}
	gens = append(gens, getDefaultStakingGenesis())

	genStaking := getDefaultStakingGenesis()
	genStaking.Params.MinCommissionRate = sdkmath.LegacyZeroDec()
	gens = append(gens, genStaking)

	for _, tc := range []struct {
		name    string
		gen     *stakingtypes.GenesisState
		wantErr string
	}{
		{
			name: "default should work fine",
			gen:  gens[0],
		},
		{
			name:    "zero commission should fail",
			gen:     gens[1],
			wantErr: "min_commission must be positive",
		},
	} {
		s.Run(tc.name, func() {
			genStakingJson := s.encCfg.Codec.MustMarshalJSON(tc.gen)
			err := app.StakingModule{}.ValidateGenesis(
				s.encCfg.Codec,
				s.encCfg.TxConfig,
				genStakingJson,
			)
			if tc.wantErr != "" {
				s.ErrorContains(err, tc.wantErr)
				return
			}
			s.NoError(err)
		})
	}
}
