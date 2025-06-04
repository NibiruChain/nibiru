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
		"0x7525dE1549A9DdDEfa0Ffd8E69523Ea9e895b280", // alice
	)
	holders := make([]gethcommon.Address, len(holderStrs))
	for idx, h := range holderStrs.ToSlice() {
		holders[idx] = gethcommon.HexToAddress(h)
	}
	return holders
}

var (
	// MAINNET_STNIBI_ADDR is the (real) hex address of stNIBI on mainnet.
	MAINNET_STNIBI_ADDR = gethcommon.HexToAddress("0x7D4B7B8CA7E1a24928Bb96D59249c7a5bd1DfBe6")

	// MAINNET_NIBIRU_SAFE_ADDR: Address of a Gnosis Safe managed by the Nibiru team
	// See https://nibiscan.io/address/0x22CBd7CbF3b33681abB3Ced4D64d71acB9a9dCd2/contract/6900/code
	MAINNET_NIBIRU_SAFE_ADDR = gethcommon.HexToAddress("0x22CBd7CbF3b33681abB3Ced4D64d71acB9a9dCd2")
)

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
		To:               &originalErc20Addr,
		From:             evm.EVM_MODULE_ADDRESS,
		Nonce:            keepers.EvmKeeper.GetAccNonce(ctx, evm.EVM_MODULE_ADDRESS),
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
	sdb := keepers.EvmKeeper.NewStateDB(
		ctx,
		keepers.EvmKeeper.TxConfig(ctx, gethcommon.Hash{}),
	)
	evmObj := keepers.EvmKeeper.NewEVM(
		ctx,
		evmMsg,
		keepers.EvmKeeper.GetEVMConfig(ctx),
		nil, // tracer: unused
		sdb,
	)

	totalSupplyErc20, err := keepers.EvmKeeper.ERC20().TotalSupply(originalErc20Addr, ctx, evmObj)
	if err != nil {
		return err
	}
	excessBalance = new(big.Int).Set(totalSupplyErc20)
	for _, holder := range holders {
		fmt.Println("holder", holder.Hex())
		bal, err := keepers.EvmKeeper.ERC20().BalanceOf(originalErc20Addr, holder, ctx, evmObj)
		if err != nil {
			return err
		}
		fmt.Println("bal", bal.String())
		holderBalsBefore[holder] = bal
		excessBalance = new(big.Int).Sub(excessBalance, bal)
	}
	fmt.Println("excessBalance", excessBalance.String())

	// -------------------------------------------------------------------------
	// STEP 2: Erase existing ERC20 contract state
	// -------------------------------------------------------------------------

	// Now that we have the balances in memory, it's safe to erase the contract
	// state. The goal is to inject new bytecode and state in.
	accState := keepers.EvmKeeper.EvmState.AccState
	{
		fmt.Println("iterating over original erc20 state", originalErc20Addr.Hex())
		iter := accState.Iterate(ctx, collections.PairRange[gethcommon.Address, gethcommon.Hash]{}.Prefix(originalErc20Addr))
		defer iter.Close()
		for ; iter.Valid(); iter.Next() {
			key := iter.Key()
			k1 := key.K1()
			k2 := key.K2()
			value := iter.Value()
			// Deleting through this iterator is safe because the underlying
			// `sdk.KVStore.Iterator` uses a snapshot. This means deleting the
			// current key does not affect the iterator unless you call `Set` on that
			// key again.
			err := keepers.EvmKeeper.EvmState.AccState.Delete(ctx, key)
			if err != nil {
				return fmt.Errorf("failed to delete state: %w", err)
			}
			fmt.Println("deleted", k1.Hex(), k2.Hex(), value)
		}
		fmt.Println("done iterating over original erc20 state")
	}

	// -------------------------------------------------------------------------
	// STEP 3: Configure the bank.Metadata for the Bank Coin counterpart
	//   of the ERC20 if it exists.
	// -------------------------------------------------------------------------
	bankDenom := funTokenMappings[0].BankDenom
	newName, newSymbol, newDecimals := "Liquid Staked NIBI", "stNIBI", uint8(6)
	bankMetadata := bank.Metadata{
		Description: "Liquid Staked NIBI is a fungible, liquid staked variant of NIBI produced by the Eris Protocol smart contracts",
		DenomUnits: []*bank.DenomUnit{
			{
				Denom:    bankDenom,
				Exponent: 0,
			},
			{
				Denom:    newSymbol,
				Exponent: uint32(newDecimals),
			},
		},
		Base:    bankDenom,
		Display: newSymbol,
		Name:    newName,
		Symbol:  newSymbol,
	}
	keepers.BankKeeper.SetDenomMetaData(ctx, bankMetadata)
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
	evmModuleNonce := keepers.EvmKeeper.GetAccNonce(ctx, evm.EVM_MODULE_ADDRESS)
	newErc20Addr := crypto.CreateAddress(evm.EVM_MODULE_ADDRESS, evmModuleNonce)
	newCompiledContract := embeds.SmartContract_ERC20MinterWithMetadataUpdates
	// empty method name means deploy with the constructor
	packedArgs, err := newCompiledContract.ABI.Pack("", newName, newSymbol, newDecimals)
	if err != nil {
		return fmt.Errorf("failed to pack ABI args: %w", err)
	}
	input := append(newCompiledContract.Bytecode, packedArgs...)

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
	sdb = keepers.EvmKeeper.NewStateDB(
		ctx,
		keepers.EvmKeeper.TxConfig(ctx, gethcommon.Hash{}),
	)
	evmObj = keepers.EvmKeeper.NewEVM(
		ctx,
		evmMsg,
		keepers.EvmKeeper.GetEVMConfig(ctx),
		nil, // tracer: unused
		sdb,
	)

	evmResp, err := keepers.EvmKeeper.CallContractWithInput(
		ctx, evmObj, evmMsg.From, nil, true /*commit*/, input,
		evmkeeper.Erc20GasLimitDeploy,
	)
	if err != nil {
		return fmt.Errorf("failed to deploy ERC20 contract: %w", err)
	} else if len(evmResp.VmError) > 0 {
		return fmt.Errorf("VM Error in deploy ERC20: %s", evmResp.VmError)
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
	// STEP 5: Sanity check the new contract with address "erc20AddrForNewDeploment"
	// -------------------------------------------------------------------------
	{
		gotName, _ := keepers.EvmKeeper.ERC20().LoadERC20Name(
			ctx, evmObj, newCompiledContract.ABI, newErc20Addr,
		)
		gotSymbol, _ := keepers.EvmKeeper.ERC20().LoadERC20Symbol(
			ctx, evmObj, newCompiledContract.ABI, newErc20Addr,
		)
		gotDecimals, _ := keepers.EvmKeeper.ERC20().LoadERC20Decimals(
			ctx, evmObj, newCompiledContract.ABI, newErc20Addr,
		)
		if newName != gotName || newSymbol != gotSymbol || newDecimals != gotDecimals {
			type errOutput struct {
				Name     string `json:"name"`
				Symbol   string `json:"symbol"`
				Decimals uint8  `json:"decimals"`
			}
			wanted, _ := json.Marshal(errOutput{
				Name:     newName,
				Symbol:   newSymbol,
				Decimals: newDecimals,
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
	fmt.Println("overwriting bytecode hash")
	newErc20Acc := keepers.EvmKeeper.GetAccount(ctx, newErc20Addr)
	originalErc20Account.CodeHash = newErc20Acc.CodeHash
	fmt.Println("old code hash", originalErc20Account.CodeHash)
	fmt.Println("new code hash", newErc20Acc.CodeHash)
	err = keepers.EvmKeeper.SetAccount(ctx, originalErc20Addr, *originalErc20Account)
	if err != nil {
		return fmt.Errorf("overwrite of contract bytecode failed: %w", err)
	}
	fmt.Println("overwrote bytecode hash")
	fmt.Println("originalErc20Account", originalErc20Account)

	// -------------------------------------------------------------------------
	// STEP 7: Copy over new contract state. This propagates the ABI and metadata
	// changes corresponding to the new deployment.
	// -------------------------------------------------------------------------
	{
		fmt.Println("copying over new contract state", newErc20Addr.Hex())
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
	// STEP 8: Copy over old balance state to the new contract instance
	// -------------------------------------------------------------------------
	{
		evmObj = keepers.EvmKeeper.NewEVM(
			ctx,
			evmMsg,
			keepers.EvmKeeper.GetEVMConfig(ctx),
			nil, // tracer: unused
			sdb,
		)

		if excessBalance.Cmp(big.NewInt(0)) > 0 {
			holderBalsBefore[MAINNET_NIBIRU_SAFE_ADDR] = excessBalance
		}

		for holder, bal := range holderBalsBefore {
			contractInput, err := newCompiledContract.ABI.Pack("mint", holder, bal)
			if err != nil {
				return fmt.Errorf("failed to pack ABI args: %w", err)
			}

			evmResp, err := keepers.EvmKeeper.CallContractWithInput(
				ctx, evmObj, evmMsg.From, &originalErc20Addr, true /*commit*/, contractInput,
				evmkeeper.Erc20GasLimitDeploy,
			)
			if err != nil {
				return fmt.Errorf("failed to call contract: %w", err)
			}

			if len(evmResp.VmError) > 0 {
				return fmt.Errorf("VM Error in mint: %s", evmResp.VmError)
			}

			evmLogs = append(evmLogs, evmResp.Logs...)
			fmt.Println("minted", bal.String(), "to", holder.Hex())
		}
	}

	_ = ctx.EventManager().EmitTypedEvent(&evm.EventTxLog{Logs: evmLogs})
	return nil
}
