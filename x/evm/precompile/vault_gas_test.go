package precompile_test

import (
	"bytes"
	"encoding/json"
	"math/big"
	"testing"

	sdkmath "cosmossdk.io/math"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bank "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/evm/precompile"
	"github.com/NibiruChain/nibiru/v2/x/evm/precompile/test"
)

type VaultGasTestSuite struct {
	suite.Suite
	deps              evmtest.TestDeps
	vaultContract     sdk.AccAddress
	minterContract    sdk.AccAddress
	collateralDenom   string
	collateralErc20   common.Address
	vaultInterfaceABI abi.ABI
}

func TestVaultGasConsumption(t *testing.T) {
	suite.Run(t, new(VaultGasTestSuite))
}

func (s *VaultGasTestSuite) SetupTest() {
	s.deps = evmtest.NewTestDeps()
	
	// Setup collateral token
	s.setupCollateralTokenWithFunToken()
	
	// Deploy vault contracts
	contracts := test.SetupVaultContracts(&s.deps, &s.Suite)
	s.vaultContract = contracts["vault"]
	s.minterContract = contracts["vault_token_minter"]
	
	// Setup simplified ABI for testing
	s.setupSimpleVaultABI()
}

func (s *VaultGasTestSuite) setupCollateralTokenWithFunToken() {
	// Create test token as collateral
	s.collateralDenom = "utest"
	
	// Setup denom metadata
	s.deps.App.BankKeeper.SetDenomMetaData(s.deps.Ctx, bank.Metadata{
		DenomUnits: []*bank.DenomUnit{
			{
				Denom:    s.collateralDenom,
				Exponent: 0,
				Aliases:  []string{},
			},
		},
		Base:        s.collateralDenom,
		Display:     s.collateralDenom,
		Name:        "Test Token",
		Symbol:      "TEST",
		Description: "Test token for gas measurements",
	})
	
	// Mint tokens to sender
	initialSupply := sdk.NewCoins(sdk.NewCoin(s.collateralDenom, sdkmath.NewInt(10_000_000_000)))
	err := s.deps.App.BankKeeper.MintCoins(s.deps.Ctx, evm.ModuleName, initialSupply)
	s.Require().NoError(err)
	
	err = s.deps.App.BankKeeper.SendCoinsFromModuleToAccount(
		s.deps.Ctx, evm.ModuleName, s.deps.Sender.NibiruAddr, initialSupply,
	)
	s.Require().NoError(err)
	
	// Fund sender with fee for creating FunToken
	s.Require().NoError(testapp.FundAccount(
		s.deps.App.BankKeeper,
		s.deps.Ctx,
		s.deps.Sender.NibiruAddr,
		s.deps.EvmKeeper.FeeForCreateFunToken(s.deps.Ctx),
	))
	
	// Create FunToken mapping for collateral
	evmObj, _ := s.deps.NewEVM()
	
	// Deploy ERC20 representation
	createFunTokenResp, err := s.deps.EvmKeeper.CreateFunToken(
		sdk.WrapSDKContext(s.deps.Ctx),
		&evm.MsgCreateFunToken{
			FromBankDenom: s.collateralDenom,
			Sender:        s.deps.Sender.NibiruAddr.String(),
		},
	)
	s.Require().NoError(err)
	s.collateralErc20 = createFunTokenResp.FuntokenMapping.Erc20Addr.Address
	
	// Send some tokens to EVM for testing
	_, err = s.deps.EvmKeeper.CallContractWithInput(
		s.deps.Ctx,
		evmObj,
		s.deps.Sender.EthAddr,
		&precompile.PrecompileAddr_FunToken,
		true,
		s.packFunTokenMethod("sendToEvm", s.collateralDenom, big.NewInt(1_000_000_000), s.deps.Sender.EthAddr.String()),
		1_000_000,
	)
	s.Require().NoError(err)
}

func (s *VaultGasTestSuite) setupSimpleVaultABI() {
	// Create a minimal ABI for testing vault operations
	abiJSON := `[
		{
			"name": "deposit",
			"type": "function",
			"inputs": [{"name": "amount", "type": "uint256"}],
			"outputs": []
		},
		{
			"name": "measureSendToBank",
			"type": "function", 
			"inputs": [{"name": "amount", "type": "uint256"}],
			"outputs": [{"name": "gasUsed", "type": "uint256"}]
		},
		{
			"name": "measureVaultDeposit",
			"type": "function",
			"inputs": [{"name": "amount", "type": "uint256"}],
			"outputs": [{"name": "gasUsed", "type": "uint256"}]
		}
	]`
	
	parsedABI, err := abi.JSON(bytes.NewReader([]byte(abiJSON)))
	s.Require().NoError(err)
	s.vaultInterfaceABI = parsedABI
}

func (s *VaultGasTestSuite) packFunTokenMethod(method string, args ...interface{}) []byte {
	input, err := embeds.SmartContract_FunToken.ABI.Pack(method, args...)
	s.Require().NoError(err)
	return input
}

// Test Cases

func (s *VaultGasTestSuite) TestDirectWasmVaultDeposit() {
	if s.vaultContract == nil {
		s.T().Skip("Vault contract not deployed")
		return
	}
	
	s.Run("Direct WASM Deposit", func() {
		depositMsg := []byte(`{"deposit":{}}`)
		funds := sdk.NewCoins(sdk.NewCoin(s.collateralDenom, sdkmath.NewInt(100_000)))
		
		startGas := s.deps.Ctx.GasMeter().GasConsumed()
		
		wasmPermissionedKeeper := wasmkeeper.NewDefaultPermissionKeeper(s.deps.App.WasmKeeper)
		_, err := wasmPermissionedKeeper.Execute(
			s.deps.Ctx,
			s.vaultContract,
			s.deps.Sender.NibiruAddr,
			depositMsg,
			funds,
		)
		s.Require().NoError(err)
		
		gasUsed := s.deps.Ctx.GasMeter().GasConsumed() - startGas
		s.T().Logf("Direct WASM Deposit Gas: %d", gasUsed)
	})
}

func (s *VaultGasTestSuite) TestPrecompileWasmDeposit() {
	if s.vaultContract == nil {
		s.T().Skip("Vault contract not deployed")
		return
	}
	
	evmObj, _ := s.deps.NewEVM()
	
	s.Run("WASM Deposit via Precompile", func() {
		depositMsg := []byte(`{"deposit":{}}`)
		funds := []precompile.WasmBankCoin{
			{
				Denom:  s.collateralDenom,
				Amount: big.NewInt(100_000),
			},
		}
		
		contractInput, err := embeds.SmartContract_Wasm.ABI.Pack(
			string(precompile.WasmMethod_execute),
			s.vaultContract.String(),
			depositMsg,
			funds,
		)
		s.Require().NoError(err)
		
		gasLimit := uint64(2_000_000)
		startGas := s.deps.Ctx.GasMeter().GasConsumed()
		
		resp, err := s.deps.EvmKeeper.CallContractWithInput(
			s.deps.Ctx,
			evmObj,
			s.deps.Sender.EthAddr,
			&precompile.PrecompileAddr_Wasm,
			true,
			contractInput,
			gasLimit,
		)
		s.Require().NoError(err)
		
		gasUsed := s.deps.Ctx.GasMeter().GasConsumed() - startGas
		s.T().Logf("WASM Deposit via Precompile - Gas Used: %d", gasUsed)
		s.T().Logf("Return data length: %d bytes", len(resp.Ret))
	})
}

func (s *VaultGasTestSuite) TestFunTokenConversion() {
	evmObj, _ := s.deps.NewEVM()
	
	s.Run("ERC20 to Bank Token Conversion", func() {
		amount := big.NewInt(50_000)
		
		// Measure sendToBank gas consumption
		contractInput := s.packFunTokenMethod(
			"sendToBank",
			s.collateralErc20,
			amount,
			s.deps.Sender.NibiruAddr.String(),
		)
		
		gasLimit := uint64(500_000)
		startGas := s.deps.Ctx.GasMeter().GasConsumed()
		
		_, err := s.deps.EvmKeeper.CallContractWithInput(
			s.deps.Ctx,
			evmObj,
			s.deps.Sender.EthAddr,
			&precompile.PrecompileAddr_FunToken,
			true,
			contractInput,
			gasLimit,
		)
		s.Require().NoError(err)
		
		gasUsed := s.deps.Ctx.GasMeter().GasConsumed() - startGas
		s.T().Logf("FunToken sendToBank Gas: %d", gasUsed)
	})
	
	s.Run("Bank Token to ERC20 Conversion", func() {
		amount := big.NewInt(25_000)
		
		// Measure sendToEvm gas consumption
		contractInput := s.packFunTokenMethod(
			"sendToEvm",
			s.collateralDenom,
			amount,
			s.deps.Sender.EthAddr.String(),
		)
		
		gasLimit := uint64(500_000)
		startGas := s.deps.Ctx.GasMeter().GasConsumed()
		
		_, err := s.deps.EvmKeeper.CallContractWithInput(
			s.deps.Ctx,
			evmObj,
			s.deps.Sender.EthAddr,
			&precompile.PrecompileAddr_FunToken,
			true,
			contractInput,
			gasLimit,
		)
		s.Require().NoError(err)
		
		gasUsed := s.deps.Ctx.GasMeter().GasConsumed() - startGas
		s.T().Logf("FunToken sendToEvm Gas: %d", gasUsed)
	})
}

func (s *VaultGasTestSuite) TestCompleteDepositFlow() {
	if s.vaultContract == nil {
		s.T().Skip("Vault contract not deployed")
		return
	}
	
	evmObj, _ := s.deps.NewEVM()
	
	s.Run("Complete Deposit Flow (ERC20 -> Bank -> Vault)", func() {
		// Step 1: Convert ERC20 to Bank tokens
		amount := big.NewInt(200_000)
		
		sendToBankInput := s.packFunTokenMethod(
			"sendToBank",
			s.collateralErc20,
			amount,
			s.deps.Sender.NibiruAddr.String(),
		)
		
		gasLimit1 := uint64(500_000)
		startGas1 := s.deps.Ctx.GasMeter().GasConsumed()
		
		_, err := s.deps.EvmKeeper.CallContractWithInput(
			s.deps.Ctx,
			evmObj,
			s.deps.Sender.EthAddr,
			&precompile.PrecompileAddr_FunToken,
			true,
			sendToBankInput,
			gasLimit1,
		)
		s.Require().NoError(err)
		
		gas1 := s.deps.Ctx.GasMeter().GasConsumed() - startGas1
		s.T().Logf("Step 1 - ERC20 to Bank: %d gas", gas1)
		
		// Step 2: Deposit Bank tokens to Vault
		depositMsg := []byte(`{"deposit":{}}`)
		funds := []precompile.WasmBankCoin{
			{
				Denom:  s.collateralDenom,
				Amount: amount,
			},
		}
		
		wasmExecuteInput, err := embeds.SmartContract_Wasm.ABI.Pack(
			string(precompile.WasmMethod_execute),
			s.vaultContract.String(),
			depositMsg,
			funds,
		)
		s.Require().NoError(err)
		
		gasLimit2 := uint64(2_000_000)
		startGas2 := s.deps.Ctx.GasMeter().GasConsumed()
		
		_, err = s.deps.EvmKeeper.CallContractWithInput(
			s.deps.Ctx,
			evmObj,
			s.deps.Sender.EthAddr,
			&precompile.PrecompileAddr_Wasm,
			true,
			wasmExecuteInput,
			gasLimit2,
		)
		s.Require().NoError(err)
		
		gas2 := s.deps.Ctx.GasMeter().GasConsumed() - startGas2
		s.T().Logf("Step 2 - Bank to Vault: %d gas", gas2)
		s.T().Logf("Total Gas for Complete Flow: %d", gas1+gas2)
	})
}

func (s *VaultGasTestSuite) TestVaultQueries() {
	if s.vaultContract == nil {
		s.T().Skip("Vault contract not deployed")
		return
	}
	
	evmObj, _ := s.deps.NewEVM()
	
	queries := []struct {
		name  string
		query string
	}{
		{"Get Collateral Denom", `{"get_collateral_denom":{}}`},
		{"Get Vault Share Denom", `{"get_vault_share_denom":{}}`},
	}
	
	for _, q := range queries {
		s.Run(q.name, func() {
			queryInput, err := embeds.SmartContract_Wasm.ABI.Pack(
				string(precompile.WasmMethod_query),
				s.vaultContract.String(),
				[]byte(q.query),
			)
			s.Require().NoError(err)
			
			gasLimit := uint64(200_000)
			startGas := s.deps.Ctx.GasMeter().GasConsumed()
			
			resp, err := s.deps.EvmKeeper.CallContractWithInput(
				s.deps.Ctx,
				evmObj,
				s.deps.Sender.EthAddr,
				&precompile.PrecompileAddr_Wasm,
				false, // readonly
				queryInput,
				gasLimit,
			)
			s.Require().NoError(err)
			
			// Unpack the response
			var queryResp []byte
			err = embeds.SmartContract_Wasm.ABI.UnpackIntoInterface(
				&queryResp,
				string(precompile.WasmMethod_query),
				resp.Ret,
			)
			s.Require().NoError(err)
			
			// Parse JSON response
			var result string
			err = json.Unmarshal(queryResp, &result)
			s.Require().NoError(err)
			
			gasUsed := s.deps.Ctx.GasMeter().GasConsumed() - startGas
			s.T().Logf("%s Gas: %d, Result: %s", q.name, gasUsed, result)
		})
	}
}

func (s *VaultGasTestSuite) TestGasSummary() {
	s.T().Log("\n=== Vault Gas Consumption Summary ===")
	s.T().Log("This test suite measures gas consumption for:")
	s.T().Log("1. Direct WASM vault operations")
	s.T().Log("2. WASM operations through EVM precompiles")
	s.T().Log("3. FunToken conversions (ERC20 <-> Bank)")
	s.T().Log("4. Complete deposit flow (ERC20 -> Bank -> Vault)")
	s.T().Log("")
	s.T().Log("Key metrics to analyze:")
	s.T().Log("- Precompile overhead vs direct WASM calls")
	s.T().Log("- Cost of token conversions")
	s.T().Log("- Total cost for user deposit flow")
	s.T().Log("- Query operation costs")
}