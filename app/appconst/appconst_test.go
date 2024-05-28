package appconst_test

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/app/appconst"
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
			big.NewInt(appconst.ETH_CHAIN_ID_LOCAL),
			appconst.GetEthChainID("nibiru-localnet-0"),
		)
	})
	s.Run("devnet", func() {
		want := big.NewInt(appconst.ETH_CHAIN_ID_DEVNET)
		given := "nibiru-testnet-1"
		s.EqualValues(want, appconst.GetEthChainID(given))

		given = "nibiru-devnet-2"
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
