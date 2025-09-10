package v2_7_0_test

import (
	"math/big"
	"strconv"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	auth "github.com/cosmos/cosmos-sdk/x/auth/types"
	bank "github.com/cosmos/cosmos-sdk/x/bank/types"
	gogoproto "github.com/cosmos/gogoproto/proto"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/app/upgrades/v2_7_0"
	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	evmkeeper "github.com/NibiruChain/nibiru/v2/x/evm/keeper"
	tf "github.com/NibiruChain/nibiru/v2/x/tokenfactory/types"
)

// The v2.7.0 upgrade on "nibiru-testnet-2"
//
// Test Procedure
//  1. Edits the EVM params for WNIBI.
//  2. adds WNIBI.sol at address "0x0CaCF669f8446BeCA826913a3c6B96aCD4b02a97". If
//     an account already exists with that address, it gets overwritten to become
//     WNIBI.sol.
//  3. ERC20 metatadata is overwritten for the FunToken mapping for stNIBI on
//     testnet.
func (s *Suite) TestTestnet() {
	var (
		deps = evmtest.NewTestDeps()

		// Original creator of the Bank Coin version of the token
		erisAddr sdk.AccAddress

		// FunToken mapping for stNIBI
		funtoken evm.FunToken

		// Metadata used for both faulty token formats of the stNIBI FunToken
		// mapping of
		originalbankMetadata = v2_7_0.OldTestnetStnibi()

		// some ERC20 holders of stNIBI to make sure the upgrade doesn't corrupt
		// any EVM state
		holders = []gethcommon.Address{
			gethcommon.BytesToAddress(testutil.AccAddress().Bytes()), // bal 20
			gethcommon.BytesToAddress(testutil.AccAddress().Bytes()), // bal 40
			gethcommon.BytesToAddress(testutil.AccAddress().Bytes()), // bal 60
		}
	)

	s.T().Log(`Set chain to nibiru-testnet-2 (EVM 6911)`)
	deps.Ctx = deps.Ctx.WithChainID("nibiru-testnet-2")
	gotEthChainId := deps.EvmKeeper.EthChainID(deps.Ctx).String()
	s.NotEqual(
		big.NewInt(appconst.ETH_CHAIN_ID_MAINNET).String(),
		gotEthChainId,
		"expect chain not to be Nibiru mainnet",
	)
	s.Equal(
		big.NewInt(appconst.ETH_CHAIN_ID_TESTNET_2).String(),
		gotEthChainId,
		"expect chain to be Nibiru testnet 2",
	)

	s.T().Log("Create the old stNIBI from before the upgrade")
	{
		bankDenomTf, err := tf.DenomStr(originalbankMetadata.Base).ToStruct()
		s.Require().NoError(err)
		erisAddr = sdk.MustAccAddressFromBech32(bankDenomTf.Creator)
	}

	s.T().Log("Initial condition - evm.Params.CanonicalWnibi not set")
	{
		evmParams := evm.DefaultParams()
		evmParams.CanonicalWnibi = eth.EIP55Addr{Address: gethcommon.Address{}}
		err := deps.EvmKeeper.SetParams(deps.Ctx, evmParams)
		s.Require().NoError(err)
	}

	s.T().Log("Copy paste assertions form the v2_5_0 upgrade handler")

	bankDenom := originalbankMetadata.Base
	{
		if deps.App.BankKeeper.HasDenomMetaData(deps.Ctx, bankDenom) {
			s.Failf("setting bank.DenomMetadata would overwrite existing denom \"%s\"", bankDenom)
		}

		s.T().Log("Setup: Create a coin in the bank state")
		bankMetadata := bank.Metadata{
			DenomUnits: []*bank.DenomUnit{
				{
					Denom:    bankDenom,
					Exponent: 0,
				},
			},
			Base:    bankDenom,
			Display: bankDenom,
			Name:    bankDenom,
			Symbol:  bankDenom,
		}

		deps.App.BankKeeper.SetDenomMetaData(deps.Ctx, bankMetadata)

		// Give the sender funds for the fee
		s.Require().NoError(testapp.FundAccount(
			deps.App.BankKeeper,
			deps.Ctx,
			deps.Sender.NibiruAddr,
			deps.EvmKeeper.FeeForCreateFunToken(deps.Ctx),
		))

		s.T().Log("happy: CreateFunToken for the bank coin")
		createFuntokenResp, err := deps.EvmKeeper.CreateFunToken(
			sdk.WrapSDKContext(deps.Ctx),
			&evm.MsgCreateFunToken{
				FromBankDenom:     bankDenom,
				Sender:            deps.Sender.NibiruAddr.String(),
				AllowZeroDecimals: true,
			},
		)
		s.Require().NoError(err, "bankDenom %s", bankDenom)

		erc20 := createFuntokenResp.FuntokenMapping.Erc20Addr
		tempFuntoken := evm.FunToken{
			Erc20Addr:      erc20,
			BankDenom:      bankDenom,
			IsMadeFromCoin: true,
		}
		s.Equal(tempFuntoken, createFuntokenResp.FuntokenMapping)

		s.T().Log("Expect ERC20 to be deployed")
		_, err = deps.EvmKeeper.Code(deps.Ctx,
			&evm.QueryCodeRequest{
				Address: erc20.String(),
			},
		)
		s.NoError(err)

		s.Require().NotEqual(erc20.Hex(), v2_7_0.TESTNET_STNIBI_ADDR.Hex(), "current temporary FunToken shouldn't yet mirror the Testnet state")

		erc20AuthAcc := deps.App.AccountKeeper.GetAccount(
			deps.Ctx,
			eth.EthAddrToNibiruAddr(erc20.Address),
		)
		s.Require().NotNil(erc20AuthAcc)

		s.T().Log("Commandeer that account number and bytecode")
		accNum := erc20AuthAcc.GetAccountNumber()
		sequence := erc20AuthAcc.GetSequence()
		pubkey := erc20AuthAcc.GetPubKey()
		s.Require().Nil(pubkey, "Contracts don't have public keys")

		tempErc20GenAcc, isOk := deps.EvmKeeper.ExportGenesisContractAccount(deps.Ctx, erc20AuthAcc)
		s.Require().True(isOk, "expect contract defined in auth module")

		s.T().Log("Inject in auth - testnet stNIBI EVM contract")
		stnibiEvmTestnetGenAcc := evm.GenesisAccount{
			Address: v2_7_0.TESTNET_STNIBI_ADDR.Hex(),
			Code:    tempErc20GenAcc.Code,
			Storage: tempErc20GenAcc.Storage,
		}

		stnibiEvmTestnetAuthAccI := erc20AuthAcc.(eth.EthAccountI)
		stnibiEvmTestnetAuthAcc := eth.EthAccount{
			BaseAccount: &auth.BaseAccount{
				Address:       eth.EthAddrToNibiruAddr(v2_7_0.TESTNET_STNIBI_ADDR).String(),
				PubKey:        nil,
				AccountNumber: accNum,
				Sequence:      sequence,
			},
			CodeHash: stnibiEvmTestnetAuthAccI.GetCodeHash().Hex(),
		}
		deps.App.AccountKeeper.SetAccount(deps.Ctx, &stnibiEvmTestnetAuthAcc)

		s.T().Log("Inject in EVM - testnet stNIBI EVM contract")
		authAcc := stnibiEvmTestnetAuthAcc
		evmGenAcc := stnibiEvmTestnetGenAcc

		codeHashAuth := authAcc.GetCodeHash()
		codeHashEvm := crypto.Keccak256Hash(
			gethcommon.Hex2Bytes(evmGenAcc.Code),
		)
		s.Require().Equal(codeHashAuth, codeHashEvm, "code hash mismatch between auth and evm modules")

		err = deps.EvmKeeper.ImportGenesisAccount(deps.Ctx, evmGenAcc)
		s.Require().NoError(err)

		contract := deps.EvmKeeper.GetAccount(deps.Ctx, v2_7_0.TESTNET_STNIBI_ADDR)
		s.Require().NotNil(contract)
		s.Require().True(contract.IsContract(), "expect testnet stNIBI to be an EVM contract ")

		s.T().Logf(`Clean up and write this to be the FunToken mapping: bank coin "%s", erc20 "%s"`, bankDenom, v2_7_0.TESTNET_STNIBI_ADDR)

		// This mapping mimics testnet stNIBI exactly.
		funtoken = evm.FunToken{
			Erc20Addr:      eth.EIP55Addr{Address: v2_7_0.TESTNET_STNIBI_ADDR},
			BankDenom:      tempFuntoken.BankDenom,
			IsMadeFromCoin: tempFuntoken.IsMadeFromCoin,
		}

		// The reason tempFuntoken has prefix "temp"
		err = deps.EvmKeeper.FunTokens.Delete(deps.Ctx, tempFuntoken.ID())
		s.Require().NoError(err)
		err = deps.EvmKeeper.FunTokens.SafeInsert(
			deps.Ctx,
			funtoken.Erc20Addr.Address,
			funtoken.BankDenom,
			funtoken.IsMadeFromCoin,
		)
		s.Require().NoError(err)
	}

	s.T().Log("Adds some tokens in circulation. We'll use these later to create ERC20 holders")
	s.NoError(
		testapp.FundAccount(
			deps.App.BankKeeper,
			deps.Ctx,
			erisAddr,
			sdk.NewCoins(sdk.NewInt64Coin(originalbankMetadata.Base, 69_420)),
		),
	)

	s.T().Log("sanity check FunToken mapping for testnet stNIBI")
	s.Equal(funtoken.BankDenom, originalbankMetadata.Base)
	s.Equal(funtoken.Erc20Addr.Hex(), v2_7_0.TESTNET_STNIBI_ADDR.Hex())
	s.Equal(funtoken.IsMadeFromCoin, true)

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
		s.Require().Equal("120", totalSupplyErc20.String())
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

	s.Run("Perform upgrade to stNIBI ERC20 address inside v2.7.0 on testnet", func() {
		s.Require().True(
			deps.App.UpgradeKeeper.HasHandler(v2_7_0.Upgrade.UpgradeName),
		)

		originalWnibiAcc := deps.EvmKeeper.GetAccount(deps.Ctx, appconst.MAINNET_WNIBI_ADDR)
		s.Nil(originalWnibiAcc)

		eventsBeforeUpgrade := deps.Ctx.EventManager().Events()

		err := deps.RunUpgrade(v2_7_0.Upgrade)
		s.Require().NoError(err)

		eventsInUpgrade := testutil.FilterNewEvents(eventsBeforeUpgrade, deps.Ctx.EventManager().Events())

		s.T().Log("assertions for WNIBI")

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

		s.T().Log("assertions for stNIBI events")

		err = testutil.AssertEventPresent(eventsInUpgrade,
			gogoproto.MessageName(new(evm.EventContractDeployed)),
		)
		s.Require().NoError(err)

		err = testutil.AssertEventPresent(eventsInUpgrade,
			gogoproto.MessageName(new(tf.EventSetDenomMetadata)),
		)
		s.Require().NoError(err)

		err = testutil.AssertEventPresent(eventsInUpgrade,
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
		evmResp, err := deps.EvmKeeper.CallContract(
			deps.Ctx,
			evmObj,
			deps.Sender.EthAddr,
			&funtoken.Erc20Addr.Address,
			input,
			evmkeeper.Erc20GasLimitQuery,
			evm.COMMIT_READONLY, /*commit*/
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
