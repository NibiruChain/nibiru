package backend_test

import (
	"math/big"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"golang.org/x/crypto/sha3"

	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"

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
			contractAddr: testContractAddress,
			blockNumber:  deployContractBlockNumber,
			codeFound:    true,
		},
		{
			name:         "sad: not a contract address",
			contractAddr: s.fundedAccEthAddr,
			blockNumber:  deployContractBlockNumber,
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
			if !tc.codeFound {
				s.Require().Nil(code)
				return
			}
			s.Require().NoError(err)
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
			contractAddr: testContractAddress,
			address:      s.fundedAccEthAddr,
			blockNumber:  deployContractBlockNumber,
			slot:         0,                        // _balances is the first slot in ERC20
			wantValue:    "0xd3c21bcecceda1000000", // = 1000000 * (10**18), initial supply
		},
		{
			name:         "sad: address which is not in contract storage",
			contractAddr: s.fundedAccEthAddr,
			address:      recipient,
			blockNumber:  deployContractBlockNumber,
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
			contractAddr: testContractAddress,
			address:      s.fundedAccEthAddr,
			blockNumber:  deployContractBlockNumber,
			// _balances is the first slot in ERC20
			slot: 0,
			// = 1000000 * (10**18), initial supply
			wantValue: "0x00000000000000000000000000000000000000000000d3c21bcecceda1000000",
		},
		{
			name:         "sad: address which is not in contract storage",
			contractAddr: s.fundedAccEthAddr,
			address:      recipient,
			blockNumber:  deployContractBlockNumber,
			slot:         0,
			wantValue:    "0x0000000000000000000000000000000000000000000000000000000000000000",
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
			address:             s.fundedAccEthAddr,
			blockNumber:         transferTxBlockNumber,
			wantPositiveBalance: true,
		},
		{
			name:                "happy: recipient balance at block 1",
			address:             recipient,
			blockNumber:         rpc.NewBlockNumber(big.NewInt(1)),
			wantPositiveBalance: false,
		},
		{
			name:                "happy: recipient balance after transfer",
			address:             recipient,
			blockNumber:         transferTxBlockNumber,
			wantPositiveBalance: true,
		},
		{
			name:                "sad: not existing account",
			address:             evmtest.NewEthPrivAcc().EthAddr,
			blockNumber:         transferTxBlockNumber,
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
				s.Require().Greater(balance.ToInt().Int64(), int64(0))
			} else {
				s.Require().Equal(balance.ToInt().Int64(), int64(0))
			}
		})
	}
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
