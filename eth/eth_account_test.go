package eth_test

import (
	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
)

func (s *Suite) TestEthAddrToNibiruAddr() {
	accInfo := evmtest.NewEthPrivAcc()
	s.Equal(
		accInfo.EthAddr,
		eth.NibiruAddrToEthAddr(accInfo.NibiruAddr),
	)
	s.Equal(
		accInfo.NibiruAddr,
		eth.EthAddrToNibiruAddr(accInfo.EthAddr),
	)

	s.T().Log("unit operation - hex -> nibi -> hex")
	{
		addr := evmtest.NewEthPrivAcc().NibiruAddr
		s.Equal(
			addr,
			eth.EthAddrToNibiruAddr(
				eth.NibiruAddrToEthAddr(addr),
			),
		)
	}

	s.T().Log("unit operation - nibi -> hex -> nibi")
	{
		addr := evmtest.NewEthPrivAcc().EthAddr
		s.Equal(
			addr,
			eth.NibiruAddrToEthAddr(
				eth.EthAddrToNibiruAddr(addr),
			),
		)
	}
}
