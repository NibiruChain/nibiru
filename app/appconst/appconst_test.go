package appconst_test

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
)

type TestSuite struct {
	suite.Suite
}

func TestSuite_RunAll(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

func (s *TestSuite) TestGetEthChainID() {
	s.Run("mainnet", func() {
		s.EqualValues(
			big.NewInt(appconst.ETH_CHAIN_ID_MAINNET),
			appconst.GetEthChainID("cataclysm-1"),
		)
	})
	s.Run("localnet", func() {
		s.EqualValues(
			big.NewInt(appconst.ETH_CHAIN_ID_LOCALNET_0),
			appconst.GetEthChainID("nibiru-localnet0"),
		)
	})
	s.Run("devnet", func() {
		want := big.NewInt(appconst.ETH_CHAIN_ID_TESTNET_1)
		given := "nibiru-testnet1"
		s.EqualValues(want, appconst.GetEthChainID(given))

		want = big.NewInt(appconst.ETH_CHAIN_ID_DEVNET_2)
		given = "nibiru-devnet2"
		s.EqualValues(want, appconst.GetEthChainID(given))
	})
	s.Run("else", func() {
		want := big.NewInt(appconst.ETH_CHAIN_ID_DEFAULT)
		for _, given := range []string{
			"foo", "bloop-blap", "not a chain ID", "", "0x12345",
		} {
			s.EqualValues(want, appconst.GetEthChainID(given))
		}
	})
}
