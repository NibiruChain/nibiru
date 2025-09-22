package evmante_test

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/app/evmante"
	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/evm/statedb"

	gethcommon "github.com/ethereum/go-ethereum/common"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
)

func (s *TestSuite) TestAnteDecEthGasConsume() {
	testCases := []struct {
		name          string
		beforeTxSetup func(deps *evmtest.TestDeps, sdb *statedb.StateDB, wnibi gethcommon.Address)
		txSetup       func(deps *evmtest.TestDeps) *evm.MsgEthereumTx
		wantErr       string
		maxGasWanted  uint64
		gasMeter      sdk.GasMeter
	}{
		{
			name: "happy: sender with funds (native)",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *statedb.StateDB, _ gethcommon.Address) {
				gasLimit := happyGasLimit()
				balance := evm.NativeToWei(new(big.Int).Add(gasLimit, big.NewInt(100)))
				sdb.AddBalanceSigned(deps.Sender.EthAddr, balance)
			},
			txSetup:      evmtest.HappyCreateContractTx,
			wantErr:      "",
			gasMeter:     eth.NewInfiniteGasMeterWithLimit(happyGasLimit().Uint64()),
			maxGasWanted: 0,
		},
		{
			name: "happy: sender with funds (wnibi)",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *statedb.StateDB, wnibi gethcommon.Address) {
				acc := deps.App.AccountKeeper.NewAccountWithAddress(deps.Ctx, eth.EthAddrToNibiruAddr(deps.Sender.EthAddr))
				deps.App.AccountKeeper.SetAccount(deps.Ctx, acc)
				// Fund the contract wnibi with unibi and set the mapping slot of the sender to be the equivalent
				// of happyGasLimit in unibi, so that the sender has enough wnibi to pay for gas
				sdb.AddBalanceSigned(wnibi, evm.NativeToWei(happyGasLimit()))
				slot := CalcMappingSlot(deps.Sender.EthAddr, 3)
				value := gethcommon.BigToHash(evm.NativeToWei(happyGasLimit()))
				sdb.SetState(wnibi, slot, value)
			},
			txSetup:      evmtest.HappyCreateContractTx,
			wantErr:      "",
			gasMeter:     eth.NewInfiniteGasMeterWithLimit(happyGasLimit().Uint64()),
			maxGasWanted: 0,
		},
		{
			name: "happy: is recheck tx",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *statedb.StateDB, _ gethcommon.Address) {
				deps.Ctx = deps.Ctx.WithIsReCheckTx(true)
			},
			txSetup:  evmtest.HappyCreateContractTx,
			gasMeter: eth.NewInfiniteGasMeterWithLimit(0),
			wantErr:  "",
		},
		{
			name: "sad: out of gas",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *statedb.StateDB, _ gethcommon.Address) {
				gasLimit := happyGasLimit()
				balance := evm.NativeToWei(new(big.Int).Add(gasLimit, big.NewInt(100)))
				sdb.AddBalanceSigned(deps.Sender.EthAddr, balance)
			},
			txSetup:      evmtest.HappyCreateContractTx,
			wantErr:      "exceeds block gas limit (0)",
			gasMeter:     eth.NewInfiniteGasMeterWithLimit(0),
			maxGasWanted: 0,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			deps := evmtest.NewTestDeps()
			deps.DeployWNIBI(&s.Suite)
			stateDB := deps.NewStateDB()
			anteDec := evmante.NewAnteDecEthGasConsume(
				deps.App.EvmKeeper, deps.App.AccountKeeper, tc.maxGasWanted,
			)

			wnibi := deps.EvmKeeper.GetParams(deps.Ctx).CanonicalWnibi.Address
			tc.beforeTxSetup(&deps, stateDB, wnibi)
			tx := tc.txSetup(&deps)
			s.Require().NoError(stateDB.Commit())

			deps.Ctx = deps.Ctx.WithIsCheckTx(true)
			deps.Ctx = deps.Ctx.WithBlockGasMeter(tc.gasMeter)
			_, err := anteDec.AnteHandle(
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

func CalcMappingSlot(key gethcommon.Address, baseSlot uint64) gethcommon.Hash {
	addrPadded := gethcommon.LeftPadBytes(key.Bytes(), 32)
	slotPadded := gethcommon.LeftPadBytes(new(big.Int).SetUint64(baseSlot).Bytes(), 32)
	return gethcrypto.Keccak256Hash(addrPadded, slotPadded)
}
