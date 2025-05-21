package v2_5_0_test

import (
	"fmt"
	"math/big"
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
	tokenfactory "github.com/NibiruChain/nibiru/v2/x/tokenfactory/types"
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
	mainnetHolderBals := make(map[gethcommon.Address]*big.Int)
	{
		holderAddrs := v2_5_0.MAINNET_STNIBI_HOLDERS()
		for idx, holderAddr := range holderAddrs {
			// Each holder gets balance as a multiple of 20
			balToFund := big.NewInt(20 * int64(idx+1))
			mainnetHolderBals[holderAddr] = balToFund
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
			balErc20, err := deps.EvmKeeper.ERC20().BalanceOf(
				funtoken.Erc20Addr.Address, holderAddr, deps.Ctx, evmObj)
			s.Require().NoError(err)
			s.Require().Equalf(balToFund.String(), balErc20.String(),
				"holderAddr %s", holderAddr)
		}

		s.T().Log("Send the remainder of Eris's balance to ERC20")
		holderAddr := v2_5_0.MAINNET_NIBIRU_SAFE_ADDR
		balToFund := deps.App.BankKeeper.GetBalance(deps.Ctx, erisAddr, funtoken.BankDenom)
		_, err := deps.EvmKeeper.ConvertCoinToEvm(deps.GoCtx(),
			&evm.MsgConvertCoinToEvm{
				ToEthAddr: eth.EIP55Addr{Address: holderAddr},
				Sender:    erisAddr.String(),
				BankCoin:  balToFund,
			},
		)
		s.Require().NoError(err)

		evmObj, _ := deps.NewEVMLessVerboseLogger()
		totalSupplyErc20, err := deps.EvmKeeper.ERC20().TotalSupply(
			funtoken.Erc20Addr.Address, deps.Ctx, evmObj)
		s.Require().NoError(err)
		s.Require().Equal("69420", totalSupplyErc20.String())
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
		s.Len(originalbankMetadata.DenomUnits, 1)
		s.Equal("0", fmt.Sprintf("%d", gotDecimals))
	}

	s.Run(fmt.Sprintf("Perform upgrade on stNIBI ERC20 address: %s", funtoken.Erc20Addr.Address), func() {
		s.T().Log("IMPORATNT: Schedule the ugprade")
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
			gogoproto.MessageName(new(tokenfactory.EventSetDenomMetadata)),
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
		s.NotEqual(originalbankMetadata.Name, gotName)
		s.NotEqual(originalbankMetadata.Symbol, gotSymbol)

		newBankMetadata, ok := deps.App.BankKeeper.GetDenomMetaData(
			deps.Ctx, funtoken.BankDenom)
		s.Require().True(ok)
		s.Len(newBankMetadata.DenomUnits, 2)
		s.Require().Equal("6", fmt.Sprintf("%d", gotDecimals))
		s.Require().Equal("stNIBI", gotSymbol)
		s.Require().Equal("Liquid Staked NIBI", gotName)
	})

	s.Run("New ERC20 impl: owner should be the EVM module", func() {
		methodName := "owner"
		input, err := compiledContract.ABI.Pack(methodName)
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
		)
		s.Require().NoError(err)

		ownerVal := new(struct{ Value gethcommon.Address })
		err = compiledContract.ABI.UnpackIntoInterface(
			ownerVal, methodName, evmResp.Ret,
		)
		s.Require().NoError(err)
		s.Require().Equal(evm.EVM_MODULE_ADDRESS.Hex(), ownerVal.Value.Hex())
	})

	s.Run("Confirm stNIBI ERC20 contract has new holder balances unharmed", func() {
		// It MUST still be a contract
		sdbAccForERC20 := deps.EvmKeeper.GetAccount(deps.Ctx, funtoken.Erc20Addr.Address)
		s.Require().True(sdbAccForERC20.IsContract())

		evmObj, _ := deps.NewEVMLessVerboseLogger()
		for holderAddr, wantBal := range mainnetHolderBals {
			balErc20, err := deps.EvmKeeper.ERC20().BalanceOf(
				funtoken.Erc20Addr.Address, holderAddr, deps.Ctx, evmObj)
			s.Require().NoError(err)
			s.Require().Equalf(wantBal.String(), balErc20.String(),
				"holderAddr %s", holderAddr)
		}
	})

	s.Run("Potential excess supply is sent to Nibiru team for redistribution", func() {
		// Each holder got a balance as an incrementing multiple of 20
		// The total supply was 69,420.
		// Thus, the excess balance is 69_420 - ( 20 * SumFrom0To(numHolders) )
		numHolders := big.NewInt(int64(len(mainnetHolderBals)))
		wantExcessBal := new(big.Int).Sub(
			big.NewInt(69_420),
			new(big.Int).Mul(SumFrom0To(numHolders), big.NewInt(20)),
		)
		holderAddr := v2_5_0.MAINNET_NIBIRU_SAFE_ADDR
		evmObj, _ := deps.NewEVMLessVerboseLogger()
		balErc20, err := deps.EvmKeeper.ERC20().BalanceOf(
			funtoken.Erc20Addr.Address, holderAddr, deps.Ctx, evmObj)
		s.Require().NoError(err)
		s.Require().Equalf(wantExcessBal.String(), balErc20.String(),
			"holderAddr %s", holderAddr)
	})
}

func SumFrom0To(n *big.Int) *big.Int {
	if n.Cmp(big.NewInt(0)) == 0 {
		return big.NewInt(0)
	} else if n.Cmp(big.NewInt(1)) == 0 {
		return big.NewInt(1)
	}
	return new(big.Int).Add(
		n, SumFrom0To(new(big.Int).Sub(n, big.NewInt(1))),
	)
}

type Suite struct {
	suite.Suite
}

func TestV2_5_0(t *testing.T) {
	suite.Run(t, new(Suite))
}
