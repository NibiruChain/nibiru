package evmante_test

import (
	"math/big"

	gethcommon "github.com/ethereum/go-ethereum/common"
	gethparams "github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/sha3"

	"github.com/NibiruChain/nibiru/v2/app/evmante"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/evm/statedb"
)

func (s *TestSuite) TestAnteDecoratorVerifyEthAcc_CheckTx() {
	testCases := []struct {
		name          string
		beforeTxSetup func(deps *evmtest.TestDeps, sdb *statedb.StateDB, wnibi gethcommon.Address)
		txSetup       func(deps *evmtest.TestDeps) *evm.MsgEthereumTx
		wantErr       string
	}{
		{
			name: "happy: sender with funds",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *statedb.StateDB, _ gethcommon.Address) {
				sdb.AddBalanceSigned(deps.Sender.EthAddr, evm.NativeToWei(happyGasLimit()))
			},
			txSetup: evmtest.HappyCreateContractTx,
			wantErr: "",
		},
		{
			name: "happy: sender with wnibi",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *statedb.StateDB, wnibi gethcommon.Address) {
				// Fund the contract wnibi with unibi and set the mapping slot of the sender to be the equivalent
				// of happyGasLimit in unibi, so that the sender has enough wnibi to pay for gas
				sdb.AddBalanceSigned(wnibi, evm.NativeToWei(happyGasLimit()))
				slot := CalcMappingSlot(deps.Sender.EthAddr, 3)
				value := gethcommon.BigToHash(evm.NativeToWei(happyGasLimit()))
				sdb.SetState(wnibi, slot, value)
			},
			txSetup: evmtest.HappyCreateContractTx,
			wantErr: "",
		},
		{
			name:          "sad: sender has insufficient gas balance",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *statedb.StateDB, _ gethcommon.Address) {},
			txSetup:       evmtest.HappyCreateContractTx,
			wantErr:       "sender balance < tx cost",
		},
		{
			name: "sad: sender cannot be a contract -> no contract bytecode",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *statedb.StateDB, _ gethcommon.Address) {
				// Force account to be a smart contract
				sdb.SetCode(deps.Sender.EthAddr, []byte("evm bytecode stuff"))
			},
			txSetup: evmtest.HappyCreateContractTx,
			wantErr: "sender is not EOA",
		},
		{
			name:          "sad: invalid tx",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *statedb.StateDB, _ gethcommon.Address) {},
			txSetup: func(deps *evmtest.TestDeps) *evm.MsgEthereumTx {
				return new(evm.MsgEthereumTx)
			},
			wantErr: "failed to unpack tx data",
		},
		{
			name:          "sad: empty from addr",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *statedb.StateDB, _ gethcommon.Address) {},
			txSetup: func(deps *evmtest.TestDeps) *evm.MsgEthereumTx {
				tx := evmtest.HappyCreateContractTx(deps)
				tx.From = ""
				return tx
			},
			wantErr: "from address cannot be empty",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			deps := evmtest.NewTestDeps()
			contract, err := deployWNIBI(&deps)
			require.NoError(s.T(), err)
			stateDB := deps.NewStateDB()
			anteDec := evmante.NewAnteDecVerifyEthAcc(deps.App.AppKeepers.EvmKeeper, &deps.App.AppKeepers.AccountKeeper, &deps.App.GasTokenKeeper)

			tc.beforeTxSetup(&deps, stateDB, contract)
			tx := tc.txSetup(&deps)
			s.Require().NoError(stateDB.Commit())

			deps.Ctx = deps.Ctx.WithIsCheckTx(true)
			_, err = anteDec.AnteHandle(
				deps.Ctx, tx, false, evmtest.NextNoOpAnteHandler,
			)
			if tc.wantErr != "" {
				s.Require().ErrorContains(err, tc.wantErr)
				return
			}
			s.Require().NoError(err)
		})
	}
}

func happyGasLimit() *big.Int {
	return new(big.Int).SetUint64(
		gethparams.TxGasContractCreation + 888,
		// 888 is a cushion to account for KV store reads and writes
	)
}

func deployWNIBI(deps *evmtest.TestDeps) (gethcommon.Address, error) {
	deployResp, err := evmtest.DeployContract(
		deps,
		embeds.SmartContract_WNIBI,
	)
	if err != nil {
		return gethcommon.Address{}, err
	}
	gasTokenParams, err := deps.App.GasTokenKeeper.Params.Get(deps.Ctx)
	if err != nil {
		return gethcommon.Address{}, err
	}
	gasTokenParams.WnibiAddress = deployResp.ContractAddr.String()
	deps.App.GasTokenKeeper.Params.Set(deps.Ctx, gasTokenParams)
	return deployResp.ContractAddr, nil
}

func CalcMappingSlot(key gethcommon.Address, baseSlot uint64) gethcommon.Hash {
	// pad base slot to 32 bytes
	baseSlotHash := gethcommon.BigToHash(new(big.Int).SetUint64(baseSlot))

	// concatenate key + slot
	data := append(make([]byte, 0, 64), key.Hash().Bytes()...)
	data = append(data, baseSlotHash.Bytes()...)

	// keccak256
	hash := sha3.NewLegacyKeccak256()
	hash.Write(data)
	return gethcommon.BytesToHash(hash.Sum(nil))
}
