// Copyright (c) 2023-2024 Nibi, Inc.
package evmmodule_test

import (
	"fmt"
	"testing"

	"github.com/NibiruChain/nibiru/x/evm/embeds"
	"github.com/NibiruChain/nibiru/x/evm/evmmodule"
	"github.com/NibiruChain/nibiru/x/evm/evmtest"
	"github.com/stretchr/testify/require"
)

func TestExportGenesis(t *testing.T) {
	deps := evmtest.NewTestDeps()

	fmt.Println("Sender address: ", deps.Sender.EthAddr.String())

	//deps.K.EvmState.GetContractBytecode()

	resp, err := evmtest.DeployContract(&deps, embeds.SmartContract_TestERC20, t)
	require.NoError(t, err)
	fmt.Println("Contract address: ", resp.ContractAddr.String())
	resp, err = evmtest.DeployContract(&deps, embeds.SmartContract_FunToken, t)
	require.NoError(t, err)
	fmt.Println("Contract address: ", resp.ContractAddr.String())

	msg, predecessors := evmtest.DeployAndExecuteERC20Transfer(&deps, t)
	fmt.Println(msg, predecessors)

	genState := evmmodule.ExportGenesis(deps.Ctx, &deps.K, deps.Chain.AccountKeeper)
	fmt.Println("===============")
	fmt.Println(genState)

}
