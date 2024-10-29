package statedb_test

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	s "github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/v2/x/common/set"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/evm/statedb"
)

// emptyCodeHash: The hash for empty contract bytecode, or a blank byte
// array. This is the code hash for a non-existent or empty account.
var emptyCodeHash []byte = crypto.Keccak256(nil)

// dummy variables for tests
var (
	address    common.Address = common.BigToAddress(big.NewInt(101))
	address2   common.Address = common.BigToAddress(big.NewInt(102))
	address3   common.Address = common.BigToAddress(big.NewInt(103))
	blockHash  common.Hash    = common.BigToHash(big.NewInt(9999))
	errAddress common.Address = common.BigToAddress(big.NewInt(100))
)

// TestSuite runs the entire test suite.
func TestSuite(t *testing.T) {
	s.Run(t, new(Suite))
}

type Suite struct {
	s.Suite
}

// CollectContractStorage is a helper function that collects all storage key-value pairs
// for a given contract address using the ForEachStorage method of the StateDB.
// It returns a map of storage slots to their values.
func CollectContractStorage(db vm.StateDB) statedb.Storage {
	storage := make(statedb.Storage)
	err := db.ForEachStorage(
		address,
		func(k, v common.Hash) bool {
			storage[k] = v
			return true
		},
	)
	if err != nil {
		return nil
	}

	return storage
}

func (s *Suite) TestAccount() {
	key1 := common.BigToHash(big.NewInt(1))
	value1 := common.BigToHash(big.NewInt(2))
	key2 := common.BigToHash(big.NewInt(3))
	value2 := common.BigToHash(big.NewInt(4))
	testCases := []struct {
		name     string
		malleate func(deps *evmtest.TestDeps, db *statedb.StateDB)
	}{
		{"non-exist account", func(deps *evmtest.TestDeps, db *statedb.StateDB) {
			s.Require().Equal(false, db.Exist(address))
			s.Require().Equal(true, db.Empty(address))
			s.Require().Equal(big.NewInt(0), db.GetBalance(address))
			s.Require().Equal([]byte(nil), db.GetCode(address))
			s.Require().Equal(common.Hash{}, db.GetCodeHash(address))
			s.Require().Equal(uint64(0), db.GetNonce(address))
		}},
		{"empty account", func(deps *evmtest.TestDeps, db *statedb.StateDB) {
			db.CreateAccount(address)
			s.Require().NoError(db.Commit())

			k := db.Keeper()
			acct := k.GetAccount(deps.Ctx, address)
			s.Require().EqualValues(statedb.NewEmptyAccount(), acct)
			s.Require().Empty(CollectContractStorage(db))

			db = deps.StateDB()
			s.Require().Equal(true, db.Exist(address))
			s.Require().Equal(true, db.Empty(address))
			s.Require().Equal(big.NewInt(0), db.GetBalance(address))
			s.Require().Equal([]byte(nil), db.GetCode(address))
			s.Require().Equal(common.BytesToHash(emptyCodeHash), db.GetCodeHash(address))
			s.Require().Equal(uint64(0), db.GetNonce(address))
		}},
		{"suicide", func(deps *evmtest.TestDeps, db *statedb.StateDB) {
			// non-exist account.
			s.Require().False(db.Suicide(address))
			s.Require().False(db.HasSuicided(address))

			// create a contract account
			db.CreateAccount(address)
			db.SetCode(address, []byte("hello world"))
			db.AddBalance(address, big.NewInt(100))
			db.SetState(address, key1, value1)
			db.SetState(address, key2, value2)
			s.Require().NoError(db.Commit())

			// suicide
			db = deps.StateDB()
			s.Require().False(db.HasSuicided(address))
			s.Require().True(db.Suicide(address))

			// check dirty state
			s.Require().True(db.HasSuicided(address))
			// balance is cleared
			s.Require().Equal(big.NewInt(0), db.GetBalance(address))
			// but code and state are still accessible in dirty state
			s.Require().Equal(value1, db.GetState(address, key1))
			s.Require().Equal([]byte("hello world"), db.GetCode(address))

			s.Require().NoError(db.Commit())

			// not accessible from StateDB anymore
			db = deps.StateDB()
			s.Require().False(db.Exist(address))
			s.Require().Empty(CollectContractStorage(db))
		}},
	}
	for _, tc := range testCases {
		s.Run(tc.name, func() {
			deps := evmtest.NewTestDeps()
			db := deps.StateDB()
			tc.malleate(&deps, db)
		})
	}
}

func (s *Suite) TestAccountOverride() {
	deps := evmtest.NewTestDeps()
	db := deps.StateDB()
	// test balance carry over when overwritten
	amount := big.NewInt(1)

	// init an EOA account, account overridden only happens on EOA account.
	db.AddBalance(address, amount)
	db.SetNonce(address, 1)

	// override
	db.CreateAccount(address)

	// check balance is not lost
	s.Require().Equal(amount, db.GetBalance(address))
	// but nonce is reset
	s.Require().Equal(uint64(0), db.GetNonce(address))
}

func (s *Suite) TestDBError() {
	testCases := []struct {
		name     string
		malleate func(vm.StateDB)
	}{
		{"set account", func(db vm.StateDB) {
			db.SetNonce(errAddress, 1)
		}},
		{"delete account", func(db vm.StateDB) {
			db.SetNonce(errAddress, 1)
			s.Require().True(db.Suicide(errAddress))
			s.True(db.HasSuicided(errAddress))
		}},
	}
	for _, tc := range testCases {
		deps := evmtest.NewTestDeps()
		db := deps.StateDB()
		tc.malleate(db)
		s.Require().NoError(db.Commit())
	}
}

func (s *Suite) TestBalance() {
	// NOTE: no need to test overflow/underflow, that is guaranteed by evm implementation.
	testCases := []struct {
		name       string
		malleate   func(*statedb.StateDB)
		expBalance *big.Int
	}{
		{"add balance", func(db *statedb.StateDB) {
			db.AddBalance(address, big.NewInt(10))
		}, big.NewInt(10)},
		{"sub balance", func(db *statedb.StateDB) {
			db.AddBalance(address, big.NewInt(10))
			// get dirty balance
			s.Require().Equal(big.NewInt(10), db.GetBalance(address))
			db.SubBalance(address, big.NewInt(2))
		}, big.NewInt(8)},
		{"add zero balance", func(db *statedb.StateDB) {
			db.AddBalance(address, big.NewInt(0))
		}, big.NewInt(0)},
		{"sub zero balance", func(db *statedb.StateDB) {
			db.SubBalance(address, big.NewInt(0))
		}, big.NewInt(0)},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			deps := evmtest.NewTestDeps()
			db := deps.StateDB()
			tc.malleate(db)

			// check dirty state
			s.Require().Equal(tc.expBalance, db.GetBalance(address))
			s.Require().NoError(db.Commit())

			// check committed balance too
			s.Require().Equal(tc.expBalance, db.GetBalance(address))
		})
	}
}

func (s *Suite) TestState() {
	key1 := common.BigToHash(big.NewInt(1))
	value1 := common.BigToHash(big.NewInt(1))
	testCases := []struct {
		name      string
		malleate  func(*statedb.StateDB)
		expStates statedb.Storage
	}{
		{"empty state", func(db *statedb.StateDB) {
		}, nil},
		{"set empty value", func(db *statedb.StateDB) {
			db.SetState(address, key1, common.Hash{})
		}, statedb.Storage{}},
		{"noop state change", func(db *statedb.StateDB) {
			db.SetState(address, key1, value1)
			db.SetState(address, key1, common.Hash{})
		}, statedb.Storage{}},
		{"set state", func(db *statedb.StateDB) {
			// check empty initial state
			s.Require().Equal(common.Hash{}, db.GetState(address, key1))
			s.Require().Equal(common.Hash{}, db.GetCommittedState(address, key1))

			// set state
			db.SetState(address, key1, value1)
			// query dirty state
			s.Require().Equal(value1, db.GetState(address, key1))
			// check committed state is still not exist
			s.Require().Equal(common.Hash{}, db.GetCommittedState(address, key1))

			// set same value again, should be noop
			db.SetState(address, key1, value1)
			s.Require().Equal(value1, db.GetState(address, key1))
		}, statedb.Storage{
			key1: value1,
		}},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			deps := evmtest.NewTestDeps()
			db := deps.StateDB()
			tc.malleate(db)
			s.Require().NoError(db.Commit())

			// check committed states in keeper
			for k, v := range tc.expStates {
				s.Equal(v, db.GetState(address, k))
			}

			// check ForEachStorage
			db = deps.StateDB()
			collected := CollectContractStorage(db)
			if len(tc.expStates) > 0 {
				s.Require().Equal(tc.expStates, collected)
			} else {
				s.Require().Empty(collected)
			}
		})
	}
}

func (s *Suite) TestCode() {
	code := []byte("hello world")
	codeHash := crypto.Keccak256Hash(code)

	testCases := []struct {
		name        string
		malleate    func(vm.StateDB)
		expCode     []byte
		expCodeHash common.Hash
	}{
		{"non-exist account", func(vm.StateDB) {}, nil, common.Hash{}},
		{"empty account", func(db vm.StateDB) {
			db.CreateAccount(address)
		}, nil, common.BytesToHash(emptyCodeHash)},
		{"set code", func(db vm.StateDB) {
			db.SetCode(address, code)
		}, code, codeHash},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			deps := evmtest.NewTestDeps()
			db := deps.StateDB()
			tc.malleate(db)

			// check dirty state
			s.Require().Equal(tc.expCode, db.GetCode(address))
			s.Require().Equal(len(tc.expCode), db.GetCodeSize(address))
			s.Require().Equal(tc.expCodeHash, db.GetCodeHash(address))

			s.Require().NoError(db.Commit())

			// check again
			db = deps.StateDB()
			s.Require().Equal(tc.expCode, db.GetCode(address))
			s.Require().Equal(len(tc.expCode), db.GetCodeSize(address))
			s.Require().Equal(tc.expCodeHash, db.GetCodeHash(address))
		})
	}
}

func (s *Suite) TestRevertSnapshot() {
	v1 := common.BigToHash(big.NewInt(1))
	v2 := common.BigToHash(big.NewInt(2))
	v3 := common.BigToHash(big.NewInt(3))
	testCases := []struct {
		name     string
		malleate func(vm.StateDB)
	}{
		{"set state", func(db vm.StateDB) {
			db.SetState(address, v1, v3)
		}},
		{"set nonce", func(db vm.StateDB) {
			db.SetNonce(address, 10)
		}},
		{"change balance", func(db vm.StateDB) {
			db.AddBalance(address, big.NewInt(10))
			db.SubBalance(address, big.NewInt(5))
		}},
		{"override account", func(db vm.StateDB) {
			db.CreateAccount(address)
		}},
		{"set code", func(db vm.StateDB) {
			db.SetCode(address, []byte("hello world"))
		}},
		{"suicide", func(db vm.StateDB) {
			db.SetState(address, v1, v2)
			db.SetCode(address, []byte("hello world"))
			s.Require().True(db.Suicide(address))
		}},
		{"add log", func(db vm.StateDB) {
			db.AddLog(&gethcore.Log{
				Address: address,
			})
		}},
		{"add refund", func(db vm.StateDB) {
			db.AddRefund(10)
			db.SubRefund(5)
		}},
		{"access list", func(db vm.StateDB) {
			db.AddAddressToAccessList(address)
			db.AddSlotToAccessList(address, v1)
		}},
	}
	for _, tc := range testCases {
		s.Run(tc.name, func() {
			deps := evmtest.NewTestDeps()

			// do some arbitrary changes to the storage
			db := deps.StateDB()
			db.SetNonce(address, 1)
			db.AddBalance(address, big.NewInt(100))
			db.SetCode(address, []byte("hello world"))
			db.SetState(address, v1, v2)
			db.SetNonce(address2, 1)
			s.Require().NoError(db.Commit())

			// Store original state values
			originalNonce := db.GetNonce(address)
			originalBalance := db.GetBalance(address)
			originalCode := db.GetCode(address)
			originalState := db.GetState(address, v1)
			originalNonce2 := db.GetNonce(address2)

			// run test
			rev := db.Snapshot()
			tc.malleate(db)
			db.RevertToSnapshot(rev)

			// check empty states after revert
			s.Require().Zero(db.GetRefund())
			s.Require().Empty(db.Logs())

			s.Require().NoError(db.Commit())

			// Check again after commit to ensure persistence
			s.Require().Equal(originalNonce, db.GetNonce(address))
			s.Require().Equal(originalBalance, db.GetBalance(address))
			s.Require().Equal(originalCode, db.GetCode(address))
			s.Require().Equal(originalState, db.GetState(address, v1))
			s.Require().Equal(originalNonce2, db.GetNonce(address2))
		})
	}
}

func (s *Suite) TestNestedSnapshot() {
	key := common.BigToHash(big.NewInt(1))
	value1 := common.BigToHash(big.NewInt(1))
	value2 := common.BigToHash(big.NewInt(2))

	deps := evmtest.NewTestDeps()
	db := deps.StateDB()

	rev1 := db.Snapshot()
	db.SetState(address, key, value1)

	rev2 := db.Snapshot()
	db.SetState(address, key, value2)
	s.Require().Equal(value2, db.GetState(address, key))

	db.RevertToSnapshot(rev2)
	s.Require().Equal(value1, db.GetState(address, key))

	db.RevertToSnapshot(rev1)
	s.Require().Equal(common.Hash{}, db.GetState(address, key))
}

func (s *Suite) TestInvalidSnapshotId() {
	deps := evmtest.NewTestDeps()
	db := deps.StateDB()

	s.Require().Panics(func() {
		db.RevertToSnapshot(1)
	})
}

func (s *Suite) TestAccessList() {
	value1 := common.BigToHash(big.NewInt(1))
	value2 := common.BigToHash(big.NewInt(2))

	testCases := []struct {
		name     string
		malleate func(vm.StateDB)
	}{
		{"add address", func(db vm.StateDB) {
			s.Require().False(db.AddressInAccessList(address))
			db.AddAddressToAccessList(address)
			s.Require().True(db.AddressInAccessList(address))

			addrPresent, slotPresent := db.SlotInAccessList(address, value1)
			s.Require().True(addrPresent)
			s.Require().False(slotPresent)

			// add again, should be no-op
			db.AddAddressToAccessList(address)
			s.Require().True(db.AddressInAccessList(address))
		}},
		{"add slot", func(db vm.StateDB) {
			addrPresent, slotPresent := db.SlotInAccessList(address, value1)
			s.Require().False(addrPresent)
			s.Require().False(slotPresent)
			db.AddSlotToAccessList(address, value1)
			addrPresent, slotPresent = db.SlotInAccessList(address, value1)
			s.Require().True(addrPresent)
			s.Require().True(slotPresent)

			// add another slot
			db.AddSlotToAccessList(address, value2)
			addrPresent, slotPresent = db.SlotInAccessList(address, value2)
			s.Require().True(addrPresent)
			s.Require().True(slotPresent)

			// add again, should be noop
			db.AddSlotToAccessList(address, value2)
			addrPresent, slotPresent = db.SlotInAccessList(address, value2)
			s.Require().True(addrPresent)
			s.Require().True(slotPresent)
		}},
		{"prepare access list", func(db vm.StateDB) {
			al := gethcore.AccessList{{
				Address:     address3,
				StorageKeys: []common.Hash{value1},
			}}

			db.PrepareAccessList(address, &address2, vm.PrecompiledAddressesBerlin, al)

			// check sender and dst
			s.Require().True(db.AddressInAccessList(address))
			s.Require().True(db.AddressInAccessList(address2))
			// check precompiles
			s.Require().True(db.AddressInAccessList(common.BytesToAddress([]byte{1})))
			// check AccessList
			s.Require().True(db.AddressInAccessList(address3))
			addrPresent, slotPresent := db.SlotInAccessList(address3, value1)
			s.Require().True(addrPresent)
			s.Require().True(slotPresent)
			addrPresent, slotPresent = db.SlotInAccessList(address3, value2)
			s.Require().True(addrPresent)
			s.Require().False(slotPresent)
		}},
	}

	for _, tc := range testCases {
		deps := evmtest.NewTestDeps()
		db := deps.StateDB()
		tc.malleate(db)
	}
}

func (s *Suite) TestLog() {
	txHash := common.BytesToHash([]byte("tx"))

	// use a non-default tx config
	const (
		blockNumber = uint64(1)
		txIdx       = uint(1)
		logIdx      = uint(1)
	)
	txConfig := statedb.TxConfig{
		BlockHash: blockHash,
		TxHash:    txHash,
		TxIndex:   txIdx,
		LogIndex:  logIdx,
	}

	deps := evmtest.NewTestDeps()
	db := statedb.New(deps.Ctx, &deps.App.EvmKeeper, txConfig)

	logData := []byte("hello world")
	log := &gethcore.Log{
		Address:     address,
		Topics:      []common.Hash{},
		Data:        logData,
		BlockNumber: blockNumber,
	}
	db.AddLog(log)
	s.Require().Equal(1, len(db.Logs()))

	wantLog := &gethcore.Log{
		Address:     log.Address,
		Topics:      log.Topics,
		Data:        log.Data,
		BlockNumber: log.BlockNumber,

		// New fields
		BlockHash: blockHash,
		TxHash:    txHash,
		TxIndex:   txIdx,
		Index:     logIdx,
	}
	s.Require().Equal(wantLog, db.Logs()[0])

	// Add a second log and assert values
	db.AddLog(log)
	wantLog.Index++
	s.Require().Equal(2, len(db.Logs()))
	gotLog := db.Logs()[1]
	s.Require().Equal(wantLog, gotLog)
}

func (s *Suite) TestRefund() {
	testCases := []struct {
		name      string
		malleate  func(vm.StateDB)
		expRefund uint64
		expPanic  bool
	}{
		{"add refund", func(db vm.StateDB) {
			db.AddRefund(uint64(10))
		}, 10, false},
		{"sub refund", func(db vm.StateDB) {
			db.AddRefund(uint64(10))
			db.SubRefund(uint64(5))
		}, 5, false},
		{"negative refund counter", func(db vm.StateDB) {
			db.AddRefund(uint64(5))
			db.SubRefund(uint64(10))
		}, 0, true},
	}
	for _, tc := range testCases {
		deps := evmtest.NewTestDeps()
		db := deps.StateDB()
		if !tc.expPanic {
			tc.malleate(db)
			s.Require().Equal(tc.expRefund, db.GetRefund())
		} else {
			s.Require().Panics(func() {
				tc.malleate(db)
			})
		}
	}
}

func (s *Suite) TestIterateStorage() {
	key1 := common.BigToHash(big.NewInt(1))
	value1 := common.BigToHash(big.NewInt(2))
	key2 := common.BigToHash(big.NewInt(3))
	value2 := common.BigToHash(big.NewInt(4))

	deps := evmtest.NewTestDeps()
	db := deps.StateDB()
	db.SetState(address, key1, value1)
	db.SetState(address, key2, value2)

	// ForEachStorage only iterate committed state
	s.Require().Empty(CollectContractStorage(db))

	s.Require().NoError(db.Commit())

	storage := CollectContractStorage(db)
	s.Require().Equal(2, len(storage))

	keySet := set.New[common.Hash](key1, key2)
	valSet := set.New[common.Hash](value1, value2)
	for _, stateKey := range storage.SortedKeys() {
		stateValue := deps.EvmKeeper.GetState(deps.Ctx, address, stateKey)
		s.True(keySet.Has(stateKey))
		s.True(valSet.Has(stateValue))
	}

	// break early iteration
	storage = make(statedb.Storage)
	err := db.ForEachStorage(address, func(k, v common.Hash) bool {
		storage[k] = v
		// return false to break early
		return false
	})
	s.Require().NoError(err)
	s.Require().Equal(1, len(storage))
}
