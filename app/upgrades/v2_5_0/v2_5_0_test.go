package v2_5_0_test

import (
	"fmt"
	"math/big"
	"strconv"
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gogoproto "github.com/cosmos/gogoproto/proto"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/v2/app/upgrades/v2_5_0"
	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	evmkeeper "github.com/NibiruChain/nibiru/v2/x/evm/keeper"
	tf "github.com/NibiruChain/nibiru/v2/x/tokenfactory/types"
)

func (s *Suite) TestUpgrade() {
	s.T().Log("Set up a token with default metadata that mimics stNIBI on mainnet prior to v2.5.0")
	var (
		deps = evmtest.NewTestDeps()

		// Original creator of the Bank Coin version of the token
		erisAddr = testutil.AccAddress()

		// Metadata used for both faulty token formats of the stNIBI FunToken
		// mapping of
		originalbankMetadata = tf.TFDenom{
			Creator:  erisAddr.String(),
			Subdenom: "ampNIBI",
		}.DefaultBankMetadata()

		// FunToken mapping for stNIBI
		funtoken = evmtest.CreateFunTokenForBankCoin(
			deps, originalbankMetadata.Base, &s.Suite,
		)

		// ten holders for testing
		holders = []gethcommon.Address{
			gethcommon.BytesToAddress(testutil.AccAddress().Bytes()),
			gethcommon.BytesToAddress(testutil.AccAddress().Bytes()),
			gethcommon.BytesToAddress(testutil.AccAddress().Bytes()),
			gethcommon.BytesToAddress(testutil.AccAddress().Bytes()),
			gethcommon.BytesToAddress(testutil.AccAddress().Bytes()),
			gethcommon.BytesToAddress(testutil.AccAddress().Bytes()),
			gethcommon.BytesToAddress(testutil.AccAddress().Bytes()),
			gethcommon.BytesToAddress(testutil.AccAddress().Bytes()),
			gethcommon.BytesToAddress(testutil.AccAddress().Bytes()),
			gethcommon.BytesToAddress(testutil.AccAddress().Bytes()),
		}
	)

	s.T().Log("Adds some tokens in circulation. We'll use these later to create ERC20 holders")
	s.NoError(
		testapp.FundAccount(
			deps.App.BankKeeper,
			deps.Ctx,
			erisAddr,
			sdk.NewCoins(sdk.NewInt64Coin(originalbankMetadata.Base, 69_420)),
		),
	)

	s.T().Logf("evm.EVM_MODULE_ADDRESS %s", evm.EVM_MODULE_ADDRESS)
	{
		for idx, holderAddr := range holders {
			// Each holder gets balance as a multiple of 20
			balToFund := big.NewInt(20 * int64(idx+1))
			_, err := deps.EvmKeeper.ConvertCoinToEvm(deps.GoCtx(),
				&evm.MsgConvertCoinToEvm{
					ToEthAddr: eth.EIP55Addr{Address: holderAddr},
					Sender:    erisAddr.String(),
					BankCoin: sdk.NewCoin(
						funtoken.BankDenom, sdkmath.NewIntFromBigInt(balToFund),
					),
				},
			)
			s.Require().NoError(err)

			// Validate the ERC20 balance of the holder
			evmObj, _ := deps.NewEVMLessVerboseLogger()
			balErc20, err := deps.EvmKeeper.ERC20().BalanceOf(funtoken.Erc20Addr.Address, holderAddr, deps.Ctx, evmObj)
			s.Require().NoError(err)
			s.Require().Equal(strconv.Itoa(20*(idx+1)), balErc20.String())
		}

		evmObj, _ := deps.NewEVMLessVerboseLogger()
		totalSupplyErc20, err := deps.EvmKeeper.ERC20().TotalSupply(funtoken.Erc20Addr.Address, deps.Ctx, evmObj)
		s.Require().NoError(err)
		s.Require().Equal("1100", totalSupplyErc20.String())
	}

	s.T().Log("Confirm that stNIBI has faulty metadata prior to the upgrade")
	{
		compiledContract := embeds.SmartContract_ERC20MinterWithMetadataUpdates
		evmObj, _ := deps.NewEVMLessVerboseLogger()
		gotName, _ := deps.EvmKeeper.ERC20().LoadERC20Name(
			deps.Ctx, evmObj, compiledContract.ABI, funtoken.Erc20Addr.Address,
		)
		gotSymbol, _ := deps.EvmKeeper.ERC20().LoadERC20Symbol(
			deps.Ctx, evmObj, compiledContract.ABI, funtoken.Erc20Addr.Address,
		)
		gotDecimals, _ := deps.EvmKeeper.ERC20().LoadERC20Decimals(
			deps.Ctx, evmObj, compiledContract.ABI, funtoken.Erc20Addr.Address,
		)
		s.Equal(originalbankMetadata.Name, gotName)
		s.Equal(originalbankMetadata.Symbol, gotSymbol)
		s.Equal(uint8(0), gotDecimals)
	}

	s.Run(fmt.Sprintf("Perform upgrade on stNIBI ERC20 address: %s", funtoken.Erc20Addr.Address), func() {
		s.T().Log("IMPORTANT: Schedule the upgrade")
		deps.EvmKeeper.Bank.StateDB = nil // IMPORTANT: make sure to clear the StateDB before running the upgrade
		s.Require().True(deps.App.UpgradeKeeper.HasHandler(v2_5_0.Upgrade.UpgradeName))

		beforeEvents := deps.Ctx.EventManager().Events()
		err := v2_5_0.UpgradeStNibiContractOnMainnet(
			&deps.App.PublicKeepers, deps.Ctx, funtoken.Erc20Addr.Address,
		)
		s.Require().NoError(err)
		upgradeEvents := testutil.FilterNewEvents(beforeEvents, deps.Ctx.EventManager().Events())

		err = testutil.AssertEventPresent(upgradeEvents,
			gogoproto.MessageName(new(evm.EventContractDeployed)),
		)
		s.Require().NoError(err)

		err = testutil.AssertEventPresent(upgradeEvents,
			gogoproto.MessageName(new(tf.EventSetDenomMetadata)),
		)
		s.Require().NoError(err)

		err = testutil.AssertEventPresent(upgradeEvents,
			gogoproto.MessageName(new(evm.EventTxLog)),
		)
		s.Require().NoError(err)
	})

	compiledContract := embeds.SmartContract_ERC20MinterWithMetadataUpdates
	s.Run("Confirm that stNIBI has desired metadata after upgrade", func() {
		evmObj, _ := deps.NewEVMLessVerboseLogger()
		gotName, _ := deps.EvmKeeper.ERC20().LoadERC20Name(
			deps.Ctx, evmObj, compiledContract.ABI, funtoken.Erc20Addr.Address,
		)
		gotSymbol, _ := deps.EvmKeeper.ERC20().LoadERC20Symbol(
			deps.Ctx, evmObj, compiledContract.ABI, funtoken.Erc20Addr.Address,
		)
		gotDecimals, _ := deps.EvmKeeper.ERC20().LoadERC20Decimals(
			deps.Ctx, evmObj, compiledContract.ABI, funtoken.Erc20Addr.Address,
		)
		s.Equal("Liquid Staked NIBI", gotName)
		s.Equal("stNIBI", gotSymbol)
		s.Equal(uint8(6), gotDecimals)

		newBankMetadata, ok := deps.App.BankKeeper.GetDenomMetaData(deps.Ctx, funtoken.BankDenom)
		s.True(ok)
		s.Equal(2, len(newBankMetadata.DenomUnits))
		s.Equal("Liquid Staked NIBI", newBankMetadata.Name)
		s.Equal("stNIBI", newBankMetadata.Symbol)
		s.Equal(uint32(6), newBankMetadata.DenomUnits[1].Exponent)
	})

	s.Run("New ERC20 impl: owner should be the EVM module", func() {
		input, err := compiledContract.ABI.Pack("owner")
		s.Require().NoError(err)
		evmObj, _ := deps.NewEVMLessVerboseLogger()
		evmResp, err := deps.EvmKeeper.CallContractWithInput(
			deps.Ctx,
			evmObj,
			deps.Sender.EthAddr,
			&funtoken.Erc20Addr.Address,
			false, // commit
			input,
			evmkeeper.Erc20GasLimitQuery,
			nil,
		)
		s.Require().NoError(err)

		ownerVal := new(struct{ Value gethcommon.Address })
		err = compiledContract.ABI.UnpackIntoInterface(ownerVal, "owner", evmResp.Ret)
		s.Require().NoError(err)
		s.Require().Equal(evm.EVM_MODULE_ADDRESS.Hex(), ownerVal.Value.Hex())
	})

	s.Run("Confirm stNIBI ERC20 contract has new holder balances unharmed", func() {
		// It MUST still be a contract
		s.Require().True(deps.EvmKeeper.GetAccount(deps.Ctx, funtoken.Erc20Addr.Address).IsContract())

		evmObj, _ := deps.NewEVMLessVerboseLogger()
		for idx, holderAddr := range holders {
			balErc20, err := deps.EvmKeeper.ERC20().BalanceOf(funtoken.Erc20Addr.Address, holderAddr, deps.Ctx, evmObj)
			s.Require().NoError(err)
			s.Require().Equal(strconv.Itoa(20*(idx+1)), balErc20.String())
		}
	})
}

type Suite struct {
	suite.Suite
}

func TestV2_5_0(t *testing.T) {
	suite.Run(t, new(Suite))
}
