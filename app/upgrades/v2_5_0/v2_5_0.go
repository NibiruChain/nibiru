package v2_5_0

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/NibiruChain/collections"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	bank "github.com/cosmos/cosmos-sdk/x/bank/types"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	clientkeeper "github.com/cosmos/ibc-go/v7/modules/core/02-client/keeper"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/app/keepers"
	"github.com/NibiruChain/nibiru/v2/app/upgrades"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
	evmkeeper "github.com/NibiruChain/nibiru/v2/x/evm/keeper"
	tokenfactory "github.com/NibiruChain/nibiru/v2/x/tokenfactory/types"
)

const UpgradeName = "v2.5.0"

var Upgrade = upgrades.Upgrade{
	UpgradeName: UpgradeName,
	CreateUpgradeHandler: func(
		mm *module.Manager,
		cfg module.Configurator,
		nibiru *keepers.PublicKeepers,
		clientKeeper clientkeeper.Keeper,
	) upgradetypes.UpgradeHandler {
		return func(
			ctx sdk.Context,
			plan upgradetypes.Plan,
			fromVM module.VersionMap,
		) (module.VersionMap, error) {
			err := UpgradeStNibiContractOnMainnet(nibiru, ctx, appconst.MAINNET_STNIBI_ADDR)
			if err != nil {
				panic(fmt.Errorf("v2.5.0 upgrade failure: %w", err))
			}

			return mm.RunMigrations(ctx, cfg, fromVM)
		}
	},
	StoreUpgrades: storetypes.StoreUpgrades{},
}

func UpgradeStNibiContractOnMainnet(
	keepers *keepers.PublicKeepers,
	ctx sdk.Context,
	// erc20Addr is the hex address of stNIBI on mainnet
	// The upgrade handler takes the address as an argument for testing purposes
	originalErc20Addr gethcommon.Address,
) error {
	// -------------------------------------------------------------------------
	// STEP 0: Early return if the FunToken mapping or ERC20 has an invalid state
	// -------------------------------------------------------------------------

	// Early return if the mapping for stNIBI doesn't exist
	erc20AddrIter := keepers.EvmKeeper.FunTokens.Indexes.ERC20Addr.ExactMatch(ctx, originalErc20Addr)
	funTokenMappings := keepers.EvmKeeper.FunTokens.Collect(ctx, erc20AddrIter)
	if len(funTokenMappings) != 1 {
		return nil
	}

	// Early return if the address is not a contract
	originalErc20Account := keepers.EvmKeeper.GetAccount(ctx, originalErc20Addr)
	if originalErc20Account == nil || !originalErc20Account.IsContract() {
		return nil
	}

	var (
		evmLogs  = []evm.Log{}
		accState = keepers.EvmKeeper.EvmState.AccState
	)

	// -------------------------------------------------------------------------
	// STEP 1: Configure the bank.Metadata for the Bank Coin counterpart
	//   of the ERC20 if it exists.
	// -------------------------------------------------------------------------
	bankDenom := funTokenMappings[0].BankDenom
	desiredName, desiredSymbol, desiredDecimals := "Liquid Staked NIBI", "stNIBI", uint8(6)
	bankMetadata := bank.Metadata{
		Description: "Liquid Staked NIBI is a fungible, liquid staked variant of NIBI produced by the Eris Protocol smart contracts",
		DenomUnits: []*bank.DenomUnit{
			{
				Denom:    bankDenom,
				Exponent: 0,
			},
			{
				Denom:    desiredSymbol,
				Exponent: uint32(desiredDecimals),
			},
		},
		Base:    bankDenom,
		Display: desiredSymbol,
		Name:    desiredName,
		Symbol:  desiredSymbol,
	}
	keepers.BankKeeper.SetDenomMetaData(ctx, bankMetadata)
	_ = ctx.EventManager().EmitTypedEvent(&tokenfactory.EventSetDenomMetadata{
		Denom:    bankDenom,
		Metadata: bankMetadata,
		Caller:   evm.EVM_MODULE_ADDRESS_NIBI.String(),
	})

	// -------------------------------------------------------------------------
	// STEP 2: Deploy the new bytecode for a configurable ERC20 at a new address.
	// That will produce a valid state we can copy over to become the state for
	// stNIBI's ERC20 address.
	//
	// This is the "new" ERC20 contract that will replace the old one.
	// -------------------------------------------------------------------------
	// Deploy to a new address with the standard "CREATE" pattern
	evmModuleNonce := keepers.EvmKeeper.GetAccNonce(ctx, evm.EVM_MODULE_ADDRESS)
	newErc20Addr := crypto.CreateAddress(evm.EVM_MODULE_ADDRESS, evmModuleNonce)
	newCompiledContract := embeds.SmartContract_ERC20MinterWithMetadataUpdates
	// empty method name means deploy with the constructor
	packedArgs, err := newCompiledContract.ABI.Pack("", desiredName, desiredSymbol, desiredDecimals)
	if err != nil {
		return fmt.Errorf("failed to pack ABI args: %w", err)
	}
	contractInput := append(newCompiledContract.Bytecode, packedArgs...)

	// Rebuild evmObj with new evmMsg for contract creation.
	// Note that most of these fields are unused when we create EVM instances
	// outside of an EthereumTx.
	unusedBigInt := big.NewInt(0)
	evmMsg := core.Message{
		To:               nil,                    // To is blank -> deploy contract
		From:             evm.EVM_MODULE_ADDRESS, // From is the deployer
		Nonce:            evmModuleNonce,
		Value:            unusedBigInt, // amount
		GasLimit:         evmkeeper.Erc20GasLimitDeploy,
		GasPrice:         unusedBigInt,
		GasFeeCap:        unusedBigInt,
		GasTipCap:        unusedBigInt,
		Data:             contractInput, // This manages the constructor args
		AccessList:       gethcore.AccessList{},
		SkipNonceChecks:  false,
		SkipFromEOACheck: false,
	}
	stateDB := keepers.EvmKeeper.Bank.StateDB
	if stateDB == nil {
		stateDB = keepers.EvmKeeper.NewStateDB(ctx, keepers.EvmKeeper.TxConfig(ctx, gethcommon.Hash{}))
	}
	defer func() {
		keepers.EvmKeeper.Bank.StateDB = nil
	}()
	evmObj := keepers.EvmKeeper.NewEVM(ctx, evmMsg, keepers.EvmKeeper.GetEVMConfig(ctx), nil, stateDB)

	evmResp, err := keepers.EvmKeeper.CallContract(
		ctx, evmObj, evmMsg.From, nil, contractInput,
		evmkeeper.Erc20GasLimitDeploy,
		evm.COMMIT_ETH_TX, /*commit*/
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to deploy ERC20 contract: %w", err)
	} else if len(evmResp.VmError) > 0 {
		return fmt.Errorf("VM Error in deploy ERC20: %s", evmResp.VmError)
	} else if err := stateDB.Commit(); err != nil {
		return fmt.Errorf("%s: %w", evm.ErrStateDBCommit, err)
	}

	evmLogs = append(evmLogs, evmResp.Logs...)
	_ = ctx.EventManager().EmitTypedEvents(
		&evm.EventContractDeployed{
			Sender:       evmMsg.From.Hex(),
			ContractAddr: newErc20Addr.Hex(),
		},
	)
	fmt.Println("deployed ERC20 contract at", newErc20Addr.Hex())

	// -------------------------------------------------------------------------
	// STEP 3: Copy over the new bytecode by overwriting the stNIBI ERC20
	// account's bytecode hash
	// -------------------------------------------------------------------------
	fmt.Println("old originalErc20Account", originalErc20Account)
	fmt.Println("old code hash", string(originalErc20Account.CodeHash))
	fmt.Println("overwriting bytecode hash")
	newErc20Acc := keepers.EvmKeeper.GetAccount(ctx, newErc20Addr)
	originalErc20Account.CodeHash = newErc20Acc.CodeHash
	err = keepers.EvmKeeper.SetAccount(ctx, originalErc20Addr, *originalErc20Account)
	if err != nil {
		return fmt.Errorf("overwrite of contract bytecode failed: %w", err)
	}
	fmt.Println("new code hash", string(newErc20Acc.CodeHash))
	fmt.Println("new originalErc20Account", originalErc20Account)
	fmt.Println("overwrote bytecode hash")

	// -------------------------------------------------------------------------
	// STEP 4: Copy over new contract state. This propagates the ABI and metadata
	// changes corresponding to the new deployment.
	// -------------------------------------------------------------------------
	{
		fmt.Println("copying over new contract's state", newErc20Addr.Hex())
		iter := accState.Iterate(ctx, collections.PairRange[gethcommon.Address, gethcommon.Hash]{}.Prefix(newErc20Addr))
		defer iter.Close()
		for ; iter.Valid(); iter.Next() {
			fmt.Println("copying over", iter.Key().K1().Hex(), iter.Key().K2().Hex())

			accState.Insert(
				ctx,
				collections.Join(originalErc20Addr, iter.Key().K2()),
				iter.Value(),
			)
		}
		fmt.Println("done copying over new contract state")
	}
	_ = ctx.EventManager().EmitTypedEvents(
		// This event is to show we've overwritten the bytecode. Think of this
		// like a redeployment.
		&evm.EventContractDeployed{
			Sender:       evmMsg.From.Hex(),
			ContractAddr: originalErc20Addr.Hex(),
		},
	)

	// -------------------------------------------------------------------------
	// STEP 5: Sanity check the new contract with address "erc20AddrForNewDeploment"
	// -------------------------------------------------------------------------
	{
		gotName, _ := keepers.EvmKeeper.ERC20().LoadERC20Name(
			ctx, evmObj, newCompiledContract.ABI, originalErc20Addr,
		)
		gotSymbol, _ := keepers.EvmKeeper.ERC20().LoadERC20Symbol(
			ctx, evmObj, newCompiledContract.ABI, originalErc20Addr,
		)
		gotDecimals, _ := keepers.EvmKeeper.ERC20().LoadERC20Decimals(
			ctx, evmObj, newCompiledContract.ABI, originalErc20Addr,
		)
		if desiredName != gotName || desiredSymbol != gotSymbol || desiredDecimals != gotDecimals {
			type errOutput struct {
				Name     string `json:"name"`
				Symbol   string `json:"symbol"`
				Decimals uint8  `json:"decimals"`
			}
			wanted, _ := json.Marshal(errOutput{
				Name:     desiredName,
				Symbol:   desiredSymbol,
				Decimals: desiredDecimals,
			})
			got, _ := json.Marshal(errOutput{
				Name:     gotName,
				Symbol:   gotSymbol,
				Decimals: gotDecimals,
			})
			return fmt.Errorf(
				"mismatch in deployed contract: wanted %s, got %s", wanted, got,
			)
		}
	}

	_ = ctx.EventManager().EmitTypedEvent(&evm.EventTxLog{Logs: evmLogs})
	return nil
}
