package rpcapi_test

import (
	"math/big"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"golang.org/x/crypto/sha3"

	rpc "github.com/NibiruChain/nibiru/v2/eth/rpc"
)

func (s *BackendSuite) TestGetCode() {
	testCases := []struct {
		name         string
		contractAddr gethcommon.Address
		blockNumber  rpc.BlockNumber
		codeFound    bool
	}{
		{
			name:         "happy: valid contract address",
			contractAddr: *s.SuccessfulTxDeployContract().Receipt.ContractAddress,
			blockNumber:  *s.SuccessfulTxDeployContract().BlockNumberRpc,
			codeFound:    true,
		},
		{
			name:         "sad: not a contract address",
			contractAddr: s.evmSenderEthAddr,
			blockNumber:  *s.SuccessfulTxDeployContract().BlockNumberRpc,
			codeFound:    false,
		},
	}
	for _, tc := range testCases {
		s.Run(tc.name, func() {
			code, err := s.backend.GetCode(
				tc.contractAddr,
				rpc.BlockNumberOrHash{
					BlockNumber: &tc.blockNumber,
				},
			)
			s.Require().NoError(err)
			if !tc.codeFound {
				s.Require().Empty(code)
				return
			}
			s.Require().NotNil(code)
		})
	}
}

func (s *BackendSuite) TestGetProof() {
	testCases := []struct {
		name         string
		contractAddr gethcommon.Address
		blockNumber  rpc.BlockNumber
		address      gethcommon.Address
		slot         uint64
		wantValue    string
	}{
		{
			name:         "happy: balance of the contract deployer",
			contractAddr: *s.SuccessfulTxDeployContract().Receipt.ContractAddress,
			address:      s.evmSenderEthAddr,
			blockNumber:  *s.SuccessfulTxDeployContract().BlockNumberRpc,
			slot:         0, // _balances is the first slot in ERC20
			wantValue:    proofValueHex(s.accInfo.ExpectedERC20InitialSupplyWei),
		},
		{
			name:         "happy: unused address has zero contract storage",
			contractAddr: *s.SuccessfulTxDeployContract().Receipt.ContractAddress,
			address:      s.accInfo.UnusedAddress,
			blockNumber:  *s.SuccessfulTxDeployContract().BlockNumberRpc,
			slot:         0,
			wantValue:    "0x0",
		},
		{
			name:         "sad: EOA has no contract storage",
			contractAddr: s.evmSenderEthAddr,
			address:      s.accInfo.UnusedAddress,
			blockNumber:  *s.SuccessfulTxDeployContract().BlockNumberRpc,
			slot:         0,
			wantValue:    "0x0",
		},
	}
	for _, tc := range testCases {
		s.Run(tc.name, func() {
			proof, err := s.backend.GetProof(
				tc.contractAddr,
				[]string{generateStorageKey(tc.address, tc.slot)},
				rpc.BlockNumberOrHash{
					BlockNumber: &tc.blockNumber,
				},
			)
			s.Require().NoError(err)
			s.Require().NotNil(proof)
			s.Require().Equal(tc.wantValue, proof.StorageProof[0].Value.String())
		})
	}
}

func (s *BackendSuite) TestGetStorageAt() {
	testCases := []struct {
		name         string
		contractAddr gethcommon.Address
		blockNumber  rpc.BlockNumber
		address      gethcommon.Address
		slot         uint64
		wantValue    string
	}{
		{
			name:         "happy: balance of the contract deployer",
			contractAddr: *s.SuccessfulTxDeployContract().Receipt.ContractAddress,
			address:      s.evmSenderEthAddr,
			blockNumber:  *s.SuccessfulTxDeployContract().BlockNumberRpc,
			// _balances is the first slot in ERC20
			slot: 0,
			// The TestERC20 constructor mints the initial supply to the deployer.
			wantValue: storageWordHex(s.accInfo.ExpectedERC20InitialSupplyWei),
		},
		{
			name:         "happy: unused address has zero contract storage",
			contractAddr: *s.SuccessfulTxDeployContract().Receipt.ContractAddress,
			address:      s.accInfo.UnusedAddress,
			blockNumber:  *s.SuccessfulTxDeployContract().BlockNumberRpc,
			slot:         0,
			wantValue:    zeroStorageWordHex(),
		},
		{
			name:         "sad: EOA has no contract storage",
			contractAddr: s.evmSenderEthAddr,
			address:      s.accInfo.UnusedAddress,
			blockNumber:  *s.SuccessfulTxDeployContract().BlockNumberRpc,
			slot:         0,
			wantValue:    zeroStorageWordHex(),
		},
	}
	for _, tc := range testCases {
		s.Run(tc.name, func() {
			value, err := s.backend.GetStorageAt(
				tc.contractAddr,
				generateStorageKey(tc.address, tc.slot),
				rpc.BlockNumberOrHash{
					BlockNumber: &tc.blockNumber,
				},
			)
			s.Require().NoError(err)
			s.Require().NotNil(value)
			s.Require().Equal(tc.wantValue, value.String())
		})
	}
}

func (s *BackendSuite) TestGetBalance() {
	testCases := []struct {
		name                string
		blockNumber         rpc.BlockNumber
		address             gethcommon.Address
		wantPositiveBalance bool
	}{
		{
			name:                "happy: funded account balance",
			address:             s.evmSenderEthAddr,
			blockNumber:         *s.SuccessfulTxTransfer().BlockNumberRpc,
			wantPositiveBalance: true,
		},
		{
			name:                "happy: recipient balance before transfer",
			address:             s.accInfo.Recipient,
			blockNumber:         s.accInfo.RecipientBalanceBeforeBlock,
			wantPositiveBalance: false,
		},
		{
			name:                "happy: recipient balance after transfer",
			address:             s.accInfo.Recipient,
			blockNumber:         *s.SuccessfulTxTransfer().BlockNumberRpc,
			wantPositiveBalance: true,
		},
		{
			name:                "sad: not existing account",
			address:             s.accInfo.UnusedAddress,
			blockNumber:         *s.SuccessfulTxTransfer().BlockNumberRpc,
			wantPositiveBalance: false,
		},
	}
	for _, tc := range testCases {
		s.Run(tc.name, func() {
			balance, err := s.backend.GetBalance(
				tc.address,
				rpc.BlockNumberOrHash{
					BlockNumber: &tc.blockNumber,
				},
			)
			s.Require().NoError(err)
			s.Require().NotNil(balance)
			if tc.wantPositiveBalance {
				s.Require().Positive(balance.ToInt().Sign())
			} else {
				s.Require().Zero(balance.ToInt().Sign())
			}
		})
	}
}

func proofValueHex(value *big.Int) string {
	if value.Sign() == 0 {
		return "0x0"
	}
	return "0x" + value.Text(16)
}

func storageWordHex(value *big.Int) string {
	return gethcommon.BytesToHash(value.Bytes()).Hex()
}

func zeroStorageWordHex() string {
	return gethcommon.Hash{}.Hex()
}

// generateStorageKey produces the storage key from address and slot (order of the variable in solidity declaration)
func generateStorageKey(key gethcommon.Address, slot uint64) string {
	// Prepare the key and slot as 32-byte values
	keyBytes := gethcommon.LeftPadBytes(key.Bytes(), 32)
	slotBytes := gethcommon.LeftPadBytes(new(big.Int).SetUint64(slot).Bytes(), 32)

	// Concatenate key and slot
	data := append(keyBytes, slotBytes...)

	// Compute the data hash using Keccak256
	hash := sha3.NewLegacyKeccak256()
	hash.Write(data)
	return gethcommon.BytesToHash(hash.Sum(nil)).Hex()
}
