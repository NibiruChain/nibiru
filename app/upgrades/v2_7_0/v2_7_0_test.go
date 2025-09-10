package v2_7_0_test

import (
	"math/big"
	"testing"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/app/upgrades/v2_7_0"
	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
)

// Prior to v2.7.0 on mainnet, WNIBI.sol is live as a contract, but the EVM
// module parameter for the canonical WNIBI address does not exist.
// This test shows that the upgrade edits only the EVM module parameters without
// deploying anything.
func (s *Suite) TestMainnet() {
	deps := evmtest.NewTestDeps()

	deps.Ctx = deps.Ctx.WithChainID("cataclysm-1") // Pretend to be mainnet
	s.Equal(
		big.NewInt(appconst.ETH_CHAIN_ID_MAINNET).String(),
		deps.EvmKeeper.EthChainID(deps.Ctx).String(),
	)

	// Initial condition - evm.Params.CanonicalWnibi not set
	{
		evmParams := evm.DefaultParams()
		evmParams.CanonicalWnibi = eth.EIP55Addr{Address: gethcommon.Address{}}
		err := deps.EvmKeeper.SetParams(deps.Ctx, evmParams)
		s.Require().NoError(err)
	}

	originalWnibiAcc := deps.EvmKeeper.GetAccount(deps.Ctx, appconst.MAINNET_WNIBI_ADDR)
	s.Nil(originalWnibiAcc)

	err := deps.RunUpgrade(v2_7_0.Upgrade)
	s.Require().NoError(err)

	evmParams := deps.EvmKeeper.GetParams(deps.Ctx)
	s.Equal(appconst.MAINNET_WNIBI_ADDR.Hex(), evmParams.CanonicalWnibi.Hex())
	newWnibiAcc := deps.EvmKeeper.GetAccount(deps.Ctx, appconst.MAINNET_WNIBI_ADDR)
	s.Nil(newWnibiAcc)
}

// This test shows that the v2.7.0 upgrade, when run on networks other than
// mainnet, adds WNIBI.sol at address
// "0x0CaCF669f8446BeCA826913a3c6B96aCD4b02a97". If an account already exists
// with that address, it gets overwritten to become WNIBI.sol.
func (s *Suite) TestOtherNibirus() {
	deps := evmtest.NewTestDeps()

	s.NotEqual(
		big.NewInt(appconst.ETH_CHAIN_ID_MAINNET).String(),
		deps.EvmKeeper.EthChainID(deps.Ctx).String(),
		"expect chain not to be Nibiru mainnet",
	)

	// Initial condition - evm.Params.CanonicalWnibi not set
	{
		evmParams := evm.DefaultParams()
		evmParams.CanonicalWnibi = eth.EIP55Addr{Address: gethcommon.Address{}}
		err := deps.EvmKeeper.SetParams(deps.Ctx, evmParams)
		s.Require().NoError(err)
	}

	originalWnibiAcc := deps.EvmKeeper.GetAccount(deps.Ctx, appconst.MAINNET_WNIBI_ADDR)
	s.Nil(originalWnibiAcc)

	err := deps.RunUpgrade(v2_7_0.Upgrade)
	s.Require().NoError(err)

	evmParams := deps.EvmKeeper.GetParams(deps.Ctx)
	s.Equal(appconst.MAINNET_WNIBI_ADDR.Hex(), evmParams.CanonicalWnibi.Hex())

	contract := appconst.MAINNET_WNIBI_ADDR
	newWnibiAcc := deps.EvmKeeper.GetAccount(deps.Ctx, contract)
	s.NotNil(newWnibiAcc)
	s.True(newWnibiAcc.IsContract())

	evmObj, _ := deps.NewEVM()
	erc20Info, err := deps.EvmKeeper.FindERC20Metadata(
		deps.Ctx, evmObj, contract, embeds.SmartContract_WNIBI.ABI,
	)
	s.NoError(err)
	s.Equal("Wrapped Nibiru", erc20Info.Name)
	s.Equal("WNIBI", erc20Info.Symbol)
	s.Equal(uint8(18), erc20Info.Decimals)
}

type Suite struct {
	suite.Suite
}

func TestV2_7_0(t *testing.T) {
	suite.Run(t, new(Suite))
}
