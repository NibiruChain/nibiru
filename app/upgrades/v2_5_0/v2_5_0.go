package v2_5_0

import (
	"encoding/json"
	"fmt"
	"math/big"

	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/NibiruChain/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	bank "github.com/cosmos/cosmos-sdk/x/bank/types"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	clientkeeper "github.com/cosmos/ibc-go/v7/modules/core/02-client/keeper"

	"github.com/NibiruChain/nibiru/v2/app/keepers"
	"github.com/NibiruChain/nibiru/v2/app/upgrades"
	"github.com/NibiruChain/nibiru/v2/x/common/set"
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
			err := UpgradeStNibiContractOnMainnet(nibiru, ctx, MAINNET_STNIBI_ADDR)
			if err != nil {
				panic(fmt.Errorf("v2.5.0 upgrade failure: %w", err))
			}

			return mm.RunMigrations(ctx, cfg, fromVM)
		}
	},
	StoreUpgrades: storetypes.StoreUpgrades{},
}

// Set of addresses that held stNIBI in ERC20 form prior to the v2.5.0 upgrade
func MAINNET_STNIBI_HOLDERS() []gethcommon.Address {
	holderStrs := set.New(
		"0x603871c2ddd41c26Ee77495E2E31e6De7f9957e0",
		"0x447fC0471f5d8150A790e19000930108aCCe0BC7",
		"0x2DD37531749e1AF248720fe8F9D1A51517D9748F",
		"0x07A1D54115D5B535F298231Cc6EaA360Bc4667f9",
		"0x5361EBC7CF6689565EC4C380b7793e238ad6Be45",
	)
	holders := make([]gethcommon.Address, len(holderStrs))
	for idx, h := range holderStrs.ToSlice() {
		holders[idx] = gethcommon.HexToAddress(h)
	}
	return holders
}

var (
	// MAINNET_STNIBI_ADDR is the (real) hex address of stNIBI on mainnet.
	MAINNET_STNIBI_ADDR = gethcommon.HexToAddress("0xcA0a9Fb5FBF692fa12fD13c0A900EC56Bb3f0a7b")

	// MAINNET_NIBIRU_SAFE_ADDR: Address of a Gnosis Safe managed by the Nibiru team
	// See https://nibiscan.io/address/0x22CBd7CbF3b33681abB3Ced4D64d71acB9a9dCd2/contract/6900/code
	MAINNET_NIBIRU_SAFE_ADDR = gethcommon.HexToAddress("0x22CBd7CbF3b33681abB3Ced4D64d71acB9a9dCd2")
)

func UpgradeStNibiContractOnMainnet(
	nibiru *keepers.PublicKeepers,
	ctx sdk.Context,
	// erc20Addr is the hex address of stNIBI on mainnet
	// The upgrade handler takes the address as an argument for testing purposes
	erc20Addr gethcommon.Address,
) error {
	// -------------------------------------------------------------------------
	// STEP 0: Early return if the FunToken mapping or ERC20 has an invalid state
	// -------------------------------------------------------------------------

	// Early return if the mapping for stNIBI doesn't exist
	erc20AddrIter := nibiru.EvmKeeper.FunTokens.Indexes.ERC20Addr.ExactMatch(ctx, erc20Addr)
	funTokenMappings := nibiru.EvmKeeper.FunTokens.Collect(ctx, erc20AddrIter)
	if len(funTokenMappings) != 1 {
		return nil
	}

	// Early return if the address is not a contract
	accOfContract := nibiru.EvmKeeper.GetAccount(ctx, erc20Addr)
	if accOfContract == nil || !accOfContract.IsContract() {
		return nil
	}

	// -------------------------------------------------------------------------
	// STEP 1: Record pre-upgrade token balances
	// -------------------------------------------------------------------------

	var (
		holderBalsBefore = make(map[gethcommon.Address]*big.Int) // Pre-upgrade token balances
		holders          = MAINNET_STNIBI_HOLDERS()              // Pre-upgrade set of holders
		evmLogs          = []evm.Log{}

		// excessBalance: Send excess ERC20 balance to Nibiru team, so it can be
		// sent out in case there are new holders before the upgrade.
		// Gnosis Safe: "cataclysm-1:0x22CBd7CbF3b33681abB3Ced4D64d71acB9a9dCd2"
		// This helps guarantee the total supply won't change.
		excessBalance *big.Int
	)

	// Blank EVM [core.Message] with no value to use as a placeholder for queries
	// Note that most of these fields are unused when we create EVM instances
	// outside of an EthereumTx.
	unusedBigInt := big.NewInt(0)
	evmMsg := core.Message{
		To:               &erc20Addr,
		From:             evm.EVM_MODULE_ADDRESS,
		Nonce:            nibiru.EvmKeeper.GetAccNonce(ctx, evm.EVM_MODULE_ADDRESS),
		Value:            unusedBigInt, // amount
		GasLimit:         0,
		GasPrice:         unusedBigInt,
		GasFeeCap:        unusedBigInt,
		GasTipCap:        unusedBigInt,
		Data:             []byte{},
		AccessList:       gethcore.AccessList{},
		SkipNonceChecks:  false,
		SkipFromEOACheck: false,
	}
	sdb := nibiru.EvmKeeper.NewStateDB(
		ctx,
		nibiru.EvmKeeper.TxConfig(ctx, gethcommon.Hash{}),
	)
	evmObj := nibiru.EvmKeeper.NewEVM(
		ctx,
		evmMsg,
		nibiru.EvmKeeper.GetEVMConfig(ctx),
		nil, // tracer: unused
		sdb,
	)

	totalSupplyErc20, err := nibiru.EvmKeeper.ERC20().TotalSupply(erc20Addr, ctx, evmObj)
	if err != nil {
		return err
	}
	excessBalance = new(big.Int).Set(totalSupplyErc20)
	for _, holder := range holders {
		bal, err := nibiru.EvmKeeper.ERC20().BalanceOf(erc20Addr, holder, ctx, evmObj)
		if err != nil {
			return err
		}
		holderBalsBefore[holder] = bal
		excessBalance = new(big.Int).Sub(excessBalance, bal)
	}

	// -------------------------------------------------------------------------
	// STEP 2: Erase existing ERC20 contract state
	// -------------------------------------------------------------------------

	// Now that we have the balances in memory, it's safe to erase the contract
	// state. The goal is to inject new bytecode and state in.
	{
		var erc20StateRange collections.Ranger[evmkeeper.AccStatePrimaryKey] = collections.PairRange[gethcommon.Address, gethcommon.Hash]{}.
			Prefix(erc20Addr)
		// ↑ The extra type hint above is meant to make the generics easier to
		// understand

		iter := nibiru.EvmKeeper.EvmState.AccState.Iterate(ctx, erc20StateRange)
		defer iter.Close()
		for ; iter.Valid(); iter.Next() {
			key := iter.Key()
			// Deleting through this iterator is safe because the underlying
			// `sdk.KVStore.Iterator` uses a snapshot. This means deleting the
			// current key does not affect the iterator unless you call `Set` on that
			// key again.
			_ = nibiru.EvmKeeper.EvmState.AccState.Delete(ctx, key)
		}
	}

	// -------------------------------------------------------------------------
	// STEP 3: Configure the bank.Metadata for the Bank Coin counterpart
	//   of the ERC20 if it exists.
	// -------------------------------------------------------------------------
	bankDenom := funTokenMappings[0].BankDenom
	name, symbol, decimals := "Liquid Staked NIBI", "stNIBI", uint8(6)
	bankMetadata := bank.Metadata{
		Description: "Liquid Staked NIBI is a fungible, liquid staked variant of NIBI produced by the Eris Protocol smart contracts",
		DenomUnits: []*bank.DenomUnit{
			{
				Denom:    bankDenom,
				Exponent: 0,
			},
			{
				Denom:    symbol,
				Exponent: uint32(decimals),
			},
		},
		Base:    bankDenom,
		Display: symbol,
		Name:    name,
		Symbol:  symbol,
	}
	nibiru.BankKeeper.SetDenomMetaData(ctx, bankMetadata)
	_ = ctx.EventManager().EmitTypedEvent(&tokenfactory.EventSetDenomMetadata{
		Denom:    bankDenom,
		Metadata: bankMetadata,
		Caller:   evm.EVM_MODULE_ADDRESS_NIBI.String(),
	})

	// -------------------------------------------------------------------------
	// STEP 4: Deploy the new bytecode for a configurable ERC20 at a new address.
	//
	// That will produce a valid state we can copy over to become the state for
	// stNIBI's ERC20 address.
	// -------------------------------------------------------------------------
	// Deploy to a new address with the standard "CREATE" pattern
	evmModuleNonce := nibiru.EvmKeeper.GetAccNonce(ctx, evm.EVM_MODULE_ADDRESS)
	erc20AddrForNewDeploment := crypto.CreateAddress(
		evm.EVM_MODULE_ADDRESS, evmModuleNonce,
	)
	compiledContract := embeds.SmartContract_ERC20MinterWithMetadataUpdates
	methodName := "" //  empty method name means deploy with the constructor
	args := []any{name, symbol, decimals}
	packedArgs, err := compiledContract.ABI.Pack(methodName, args...)
	if err != nil {
		return fmt.Errorf("failed to pack ABI args: %w", err)
	}
	input := append(compiledContract.Bytecode, packedArgs...)

	// Rebuild evmObj with new evmMsg for contract creation.
	// Note that most of these fields are unused when we create EVM instances
	// outside of an EthereumTx.
	evmMsg = core.Message{
		To:               nil,                    // To is blank -> deploy contract
		From:             evm.EVM_MODULE_ADDRESS, // From is the deployer
		Nonce:            evmModuleNonce,
		Value:            unusedBigInt, // amount
		GasLimit:         evmkeeper.Erc20GasLimitDeploy,
		GasPrice:         unusedBigInt,
		GasFeeCap:        unusedBigInt,
		GasTipCap:        unusedBigInt,
		Data:             input, // This manages the constructor args
		AccessList:       gethcore.AccessList{},
		SkipNonceChecks:  false,
		SkipFromEOACheck: false,
	}
	sdb = nibiru.EvmKeeper.NewStateDB(
		ctx,
		nibiru.EvmKeeper.TxConfig(ctx, gethcommon.Hash{}),
	)
	evmObj = nibiru.EvmKeeper.NewEVM(
		ctx,
		evmMsg,
		nibiru.EvmKeeper.GetEVMConfig(ctx),
		nil, // tracer: unused
		sdb,
	)

	evmResp, err := nibiru.EvmKeeper.CallContractWithInput(
		ctx, evmObj, evmMsg.From, nil, true /*commit*/, input,
		evmkeeper.Erc20GasLimitDeploy,
	)
	if err != nil {
		return fmt.Errorf("failed to deploy ERC20 contract: %w", err)
	} else if len(evmResp.VmError) > 0 {
		return fmt.Errorf("VM Error in deploy ERC20: %s", evmResp.VmError)
	}
	evmResp.Logs = append(evmLogs, evmResp.Logs...)
	_ = ctx.EventManager().EmitTypedEvents(
		&evm.EventContractDeployed{
			Sender:       evmMsg.From.Hex(),
			ContractAddr: erc20AddrForNewDeploment.Hex(),
		},
	)

	// -------------------------------------------------------------------------
	// STEP 5: Sanity check the new contract with address "erc20AddrForNewDeploment"
	// -------------------------------------------------------------------------
	{
		gotName, _ := nibiru.EvmKeeper.ERC20().LoadERC20Name(
			ctx, evmObj, compiledContract.ABI, erc20AddrForNewDeploment,
		)
		gotSymbol, _ := nibiru.EvmKeeper.ERC20().LoadERC20Symbol(
			ctx, evmObj, compiledContract.ABI, erc20AddrForNewDeploment,
		)
		gotDecimals, _ := nibiru.EvmKeeper.ERC20().LoadERC20Decimals(
			ctx, evmObj, compiledContract.ABI, erc20AddrForNewDeploment,
		)
		if name != gotName || symbol != gotSymbol || decimals != gotDecimals {
			type errOutput struct {
				Name     string `json:"name"`
				Symbol   string `json:"symbol"`
				Decimals uint8  `json:"decimals"`
			}
			wanted, _ := json.Marshal(errOutput{
				Name:     name,
				Symbol:   symbol,
				Decimals: decimals,
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

	// -------------------------------------------------------------------------
	// STEP 6: Copy over the new bytecode by overwriting the stNIBI ERC20
	// account's bytecode hash
	// -------------------------------------------------------------------------
	newDeploymentAcc := nibiru.EvmKeeper.GetAccount(ctx, erc20AddrForNewDeploment)
	accOfContract.CodeHash = newDeploymentAcc.CodeHash
	err = nibiru.EvmKeeper.SetAccount(ctx, erc20Addr, *accOfContract)
	if err != nil {
		return fmt.Errorf("overwrite of contract bytecode failed: %w", err)
	}

	// -------------------------------------------------------------------------
	// STEP 7: Copy over new contract state. This propagates the ABI and metadata
	// changes corresponding to the new deployment.
	// -------------------------------------------------------------------------
	{
		var erc20StateRange collections.Ranger[evmkeeper.AccStatePrimaryKey] = collections.PairRange[gethcommon.Address, gethcommon.Hash]{}.
			Prefix(erc20AddrForNewDeploment)
		iter := nibiru.EvmKeeper.EvmState.AccState.Iterate(ctx, erc20StateRange)
		defer iter.Close()
		for ; iter.Valid(); iter.Next() {
			// Insert state at the original address, "erc20Addr"
			stateKey := iter.Key().K2()
			// addrNewDeployment := iter.Key().K1()  // Don't need K1 since it's the contract
			stateValue := iter.Value()
			nibiru.EvmKeeper.EvmState.AccState.Insert(
				ctx,
				collections.Join(erc20Addr, stateKey),
				stateValue,
			)
		}
	}
	_ = ctx.EventManager().EmitTypedEvents(
		// This event is to show we've overwritten the bytecode. Think of this
		// like a redeployment.
		&evm.EventContractDeployed{
			Sender:       evmMsg.From.Hex(),
			ContractAddr: erc20Addr.Hex(),
		},
	)

	// -------------------------------------------------------------------------
	// STEP 8: Copy over old balance state to the new contract instance
	// -------------------------------------------------------------------------
	{
		if excessBalance.Cmp(big.NewInt(0)) > 0 {
			holderBalsBefore[MAINNET_NIBIRU_SAFE_ADDR] = excessBalance
		}
		for holder, bal := range holderBalsBefore {
			to := holder
			amount := bal
			evmResp, err := nibiru.EvmKeeper.ERC20().Mint(
				erc20Addr,              /*erc20Contract*/
				evm.EVM_MODULE_ADDRESS, /*from*/
				to,                     /*to*/
				amount,
				ctx,
				evmObj,
			)
			if err != nil {
				return fmt.Errorf("mint erc20 error: %w", err)
			}
			evmResp.Logs = append(evmLogs, evmResp.Logs...)
		}
	}

	_ = ctx.EventManager().EmitTypedEvent(&evm.EventTxLog{Logs: evmLogs})
	return nil
}
