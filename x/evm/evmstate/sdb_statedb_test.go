package evmstate_test

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"

	xcommon "github.com/NibiruChain/nibiru/v2/x/common"
	"github.com/NibiruChain/nibiru/v2/x/common/set"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmstate"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
)

// dummy variables for tests
var (
	taddr      common.Address = common.BigToAddress(big.NewInt(101))
	taddr2     common.Address = common.BigToAddress(big.NewInt(102))
	address3   common.Address = common.BigToAddress(big.NewInt(103))
	blockHash  common.Hash    = common.BigToHash(big.NewInt(9999))
	errAddress common.Address = common.BigToAddress(big.NewInt(100))
)

// CollectContractStorage is a helper function that collects all storage key-value pairs
// for a given contract address using the ForEachStorage method of the StateDB.
// It returns a map of storage slots to their values.
func CollectContractStorage(db vm.StateDB) evmstate.Storage {
	storage := make(evmstate.Storage)
	sdb := db.(*evmstate.SDB)
	err := sdb.ForEachStorage(
		taddr,
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
		malleate func(deps *evmtest.TestDeps, db *evmstate.SDB)
	}{
		{"non-exist account", func(deps *evmtest.TestDeps, sdb *evmstate.SDB) {
			s.Require().Equal(false, sdb.Exist(taddr))
			s.Require().Equal(true, sdb.Empty(taddr))
			s.Require().Equal(uint256.NewInt(0), sdb.GetBalance(taddr))
			s.Require().Equal([]byte(nil), sdb.GetCode(taddr))
			s.Require().Equal(evm.CodeHashForNilAccount, sdb.GetCodeHash(taddr))
			s.Require().Equal(uint64(0), sdb.GetNonce(taddr))
		}},
		{"empty account", func(deps *evmtest.TestDeps, sdb *evmstate.SDB) {
			sdb.CreateAccount(taddr)
			sdb.Commit()

			k := sdb.Keeper()
			acct := k.GetAccount(deps.Ctx(), taddr)
			s.Require().EqualValues(evmstate.NewEmptyAccount(), acct)
			s.Require().Empty(CollectContractStorage(sdb))

			sdb = deps.NewStateDB()
			s.Require().Equal(true, sdb.Exist(taddr))
			s.Require().Equal(true, sdb.Empty(taddr))
			s.Require().Equal(uint256.NewInt(0), sdb.GetBalance(taddr))
			s.Require().Equal([]byte(nil), sdb.GetCode(taddr))
			s.Require().Equal(evm.EmptyCodeHash, sdb.GetCodeHash(taddr))
			s.Require().Equal(uint64(0), sdb.GetNonce(taddr))
		}},
		{"suicide", func(deps *evmtest.TestDeps, sdb *evmstate.SDB) {
			// non-exist account.
			s.Require().False(sdb.HasSuicided(taddr))
			sdb.SelfDestruct(taddr)
			s.Require().True(sdb.HasSuicided(taddr))

			// create a contract account
			sdb.CreateAccount(taddr)
			sdb.SetCode(taddr, []byte("hello world"))
			AddBalanceSigned(sdb, taddr, big.NewInt(100))
			sdb.SetState(taddr, key1, value1)
			sdb.SetState(taddr, key2, value2)
			sdb.Commit()
			helpMsg := "created the account as a contract after self destruct"
			s.False(sdb.HasSuicided(taddr), helpMsg)
			s.True(sdb.IsCreatedThisBlock(taddr), helpMsg)

			// suicide
			deps.Commit() // Resets the sdb, so this new object doesn't have lingering state
			sdb = deps.NewStateDB()
			s.Require().False(sdb.HasSuicided(taddr))

			sdb.SelfDestruct(taddr)

			s.T().Log("after suicide, before commit -> soon to be empty account, where code and transient key-value state are still accessible (dirty)")
			s.Require().True(sdb.HasSuicided(taddr))
			s.Equal(uint256.NewInt(0), sdb.GetBalance(taddr))
			s.Equal(uint64(0), sdb.GetNonce(taddr))
			// Code and state are still accessible in dirty state
			s.Require().Equal(value1, sdb.GetState(taddr, key1))
			s.Equal([]byte("hello world"), sdb.GetCode(taddr))
			s.True(sdb.Exist(taddr), "expect suicided accounts to exist based on the vm.StateDB definition")
			s.Equal(false, sdb.Empty(taddr))

			sdb.Commit()
			deps.Commit()
			sdb = deps.NewStateDB()
			helpMsg = "account should not exist in state after commit if its self destructed"
			s.Equal("0", sdb.GetBalance(taddr).String(), helpMsg)
			s.False(sdb.Exist(taddr), helpMsg)
			s.Require().Equal(true, sdb.Empty(taddr), helpMsg)
			// TODO: UD-DEBUG: Need storage checks?
			// s.Require().Empty(CollectContractStorage(sdb))
		}},
	}
	for _, tc := range testCases {
		s.Run(tc.name, func() {
			deps := evmtest.NewTestDeps()
			db := deps.NewStateDB()
			tc.malleate(&deps, db)
		})
	}
}

func (s *Suite) TestAccountOverride() {
	deps := evmtest.NewTestDeps()
	sdb := deps.NewStateDB()
	// test balance carry over when overwritten
	amount := big.NewInt(1)

	// init an EOA account, account overridden only happens on EOA account.
	AddBalanceSigned(sdb, taddr, amount)
	sdb.SetNonce(taddr, 1)

	// override
	sdb.CreateAccount(taddr)

	// check balance is not lost
	s.Require().Equal(uint256.MustFromBig(amount), sdb.GetBalance(taddr))
	// but nonce is reset
	s.Require().Equal(uint64(0), sdb.GetNonce(taddr))
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
			db.SelfDestruct(errAddress)
			s.Require().True(db.HasSelfDestructed(errAddress))
		}},
	}
	for _, tc := range testCases {
		deps := evmtest.NewTestDeps()
		sdb := deps.NewStateDB()
		tc.malleate(sdb)
		sdb.Commit()
	}
}

func (s *Suite) TestBalance() {
	// NOTE: no need to test overflow/underflow, that is guaranteed by evm implementation.
	testCases := []struct {
		name       string
		do         func(*evmstate.SDB)
		expBalance *uint256.Int
	}{
		{
			name: "add balance",
			do: func(sdb *evmstate.SDB) {
				AddBalanceSigned(sdb, taddr, big.NewInt(10))
			},
			expBalance: uint256.NewInt(10),
		},
		{
			name: "sub balance",
			do: func(sdb *evmstate.SDB) {
				AddBalanceSigned(sdb, taddr, big.NewInt(10))
				s.Require().Equal(uint256.NewInt(10), sdb.GetBalance(taddr))
				AddBalanceSigned(sdb, taddr, big.NewInt(-2))
			},
			expBalance: uint256.NewInt(8),
		},
		{
			name: "add zero balance",
			do: func(sdb *evmstate.SDB) {
				AddBalanceSigned(sdb, taddr, big.NewInt(0))
			},
			expBalance: uint256.NewInt(0),
		},
		{
			name: "sub zero balance",
			do: func(sdb *evmstate.SDB) {
				AddBalanceSigned(sdb, taddr, big.NewInt(0))
			},
			expBalance: uint256.NewInt(0),
		},
		{
			name: "overflow on addition",
			do: func(sdb *evmstate.SDB) {
				AddBalanceSigned(sdb, taddr, big.NewInt(69))
				tooBig := new(big.Int).Exp(big.NewInt(2), big.NewInt(256), nil)
				maybeErr := xcommon.TryCatch(func() {
					AddBalanceSigned(sdb, taddr, tooBig)
				})()
				s.ErrorContains(maybeErr, "uint256 overflow occurred for big.Int")
			},
			expBalance: uint256.NewInt(69),
		},
		{
			name: "overflow on subtraction",
			do: func(sdb *evmstate.SDB) {
				AddBalanceSigned(sdb, taddr, big.NewInt(420))
				AddBalanceSigned(sdb, taddr, big.NewInt(-20)) // balance: 400

				// Construct -2^256
				tooBig := new(big.Int).Exp(big.NewInt(2), big.NewInt(256), nil)
				tooBig.Neg(tooBig)

				maybeErr := xcommon.TryCatch(func() {
					AddBalanceSigned(sdb, taddr, tooBig)
				})()

				s.ErrorContains(maybeErr, "uint256 overflow occurred for big.Int")
			},
			expBalance: uint256.NewInt(400),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			deps := evmtest.NewTestDeps()
			sdb := deps.NewStateDB()
			tc.do(sdb)

			// check dirty state
			s.Equal(tc.expBalance, sdb.GetBalance(taddr))
			sdb.Commit()

			// check committed balance too
			s.Require().Equal(tc.expBalance, sdb.GetBalance(taddr))
		})
	}
}

func (s *Suite) TestState() {
	key1 := common.BigToHash(big.NewInt(1))
	value1 := common.BigToHash(big.NewInt(1))
	testCases := []struct {
		name      string
		malleate  func(*evmstate.SDB)
		expStates evmstate.Storage
	}{
		{"empty state", func(db *evmstate.SDB) {
		}, nil},
		{"set empty value", func(db *evmstate.SDB) {
			db.SetState(taddr, key1, common.Hash{})
		}, evmstate.Storage{}},
		{"noop state change", func(db *evmstate.SDB) {
			db.SetState(taddr, key1, value1)
			db.SetState(taddr, key1, common.Hash{})
		}, evmstate.Storage{}},
		{"set state", func(db *evmstate.SDB) {
			// check empty initial state
			s.Require().Equal(common.Hash{}, db.GetState(taddr, key1))
			s.Require().Equal(common.Hash{}, db.GetCommittedState(taddr, key1))

			// set state
			db.SetState(taddr, key1, value1)
			// query dirty state
			s.Require().Equal(value1, db.GetState(taddr, key1))
			// check committed state is still not exist
			s.Require().Equal(common.Hash{}, db.GetCommittedState(taddr, key1))

			// set same value again, should be noop
			db.SetState(taddr, key1, value1)
			s.Require().Equal(value1, db.GetState(taddr, key1))
		}, evmstate.Storage{
			key1: value1,
		}},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			deps := evmtest.NewTestDeps()
			db := deps.NewStateDB()
			tc.malleate(db)
			db.Commit()

			// check committed states in keeper
			for k, v := range tc.expStates {
				s.Equal(v, db.GetState(taddr, k))
			}

			// check ForEachStorage
			db = deps.NewStateDB()
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
		{
			"non-exist account",
			func(vm.StateDB) {},
			nil,
			common.Hash{},
		},
		{
			"empty account",
			func(db vm.StateDB) {
				db.CreateAccount(taddr)
			},
			nil,
			common.BytesToHash(evm.EmptyCodeHashBz),
		},
		{"set code", func(db vm.StateDB) {
			db.SetCode(taddr, code)
		}, code, codeHash},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			deps := evmtest.NewTestDeps()
			db := deps.NewStateDB()
			tc.malleate(db)

			// check dirty state
			s.Require().Equal(tc.expCode, db.GetCode(taddr))
			s.Require().Equal(len(tc.expCode), db.GetCodeSize(taddr))
			s.Require().Equal(tc.expCodeHash, db.GetCodeHash(taddr))

			db.Commit()

			// check again
			db = deps.NewStateDB()
			s.Require().Equal(tc.expCode, db.GetCode(taddr))
			s.Require().Equal(len(tc.expCode), db.GetCodeSize(taddr))
			s.Require().Equal(tc.expCodeHash, db.GetCodeHash(taddr))
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
			db.SetState(taddr, v1, v3)
		}},
		{"set nonce", func(db vm.StateDB) {
			db.SetNonce(taddr, 10)
		}},
		{"change balance", func(db vm.StateDB) {
			db.AddBalance(taddr, uint256.NewInt(10), tracing.BalanceChangeUnspecified)
			db.SubBalance(taddr, uint256.NewInt(5), tracing.BalanceChangeUnspecified)
		}},
		{"override account", func(db vm.StateDB) {
			db.CreateAccount(taddr)
		}},
		{"set code", func(db vm.StateDB) {
			db.SetCode(taddr, []byte("hello world"))
		}},
		{"suicide", func(db vm.StateDB) {
			db.SetState(taddr, v1, v2)
			db.SetCode(taddr, []byte("hello world"))
			s.Require().False(db.HasSelfDestructed(taddr))
			db.SelfDestruct(taddr)
			s.Require().True(db.HasSelfDestructed(taddr))
		}},
		{"add log", func(db vm.StateDB) {
			db.AddLog(&gethcore.Log{
				Address: taddr,
			})
		}},
		{"add refund", func(db vm.StateDB) {
			db.AddRefund(10)
			db.SubRefund(5)
		}},
		{"access list", func(db vm.StateDB) {
			db.AddAddressToAccessList(taddr)
			db.AddSlotToAccessList(taddr, v1)
		}},
	}
	for _, tc := range testCases {
		s.Run(tc.name, func() {
			deps := evmtest.NewTestDeps()

			// do some arbitrary changes to the storage
			sdb := deps.NewStateDB()
			sdb.SetNonce(taddr, 1)
			AddBalanceSigned(sdb, taddr, big.NewInt(100))
			sdb.SetCode(taddr, []byte("hello world"))
			sdb.SetState(taddr, v1, v2)
			sdb.SetNonce(taddr2, 1)
			sdb.Commit()

			// Store original state values
			originalNonce := sdb.GetNonce(taddr)
			originalBalance := sdb.GetBalance(taddr)
			originalCode := sdb.GetCode(taddr)
			originalState := sdb.GetState(taddr, v1)
			originalNonce2 := sdb.GetNonce(taddr2)

			// run test
			rev := sdb.Snapshot()
			tc.malleate(sdb)
			sdb.RevertToSnapshot(rev)

			// check empty states after revert
			s.Require().Zero(sdb.GetRefund())
			s.Require().Empty(sdb.Logs())

			sdb.Commit()

			// Check again after commit to ensure persistence
			s.Require().Equal(originalNonce, sdb.GetNonce(taddr))
			s.Require().Equal(originalBalance, sdb.GetBalance(taddr))
			s.Require().Equal(originalCode, sdb.GetCode(taddr))
			s.Require().Equal(originalState, sdb.GetState(taddr, v1))
			s.Require().Equal(originalNonce2, sdb.GetNonce(taddr2))
		})
	}
}

func (s *Suite) TestNestedSnapshot() {
	key := common.BigToHash(big.NewInt(1))
	value1 := common.BigToHash(big.NewInt(1))
	value2 := common.BigToHash(big.NewInt(2))

	deps := evmtest.NewTestDeps()
	db := deps.NewStateDB()

	rev1 := db.Snapshot()
	db.SetState(taddr, key, value1)

	rev2 := db.Snapshot()
	db.SetState(taddr, key, value2)
	s.Require().Equal(value2, db.GetState(taddr, key))

	db.RevertToSnapshot(rev2)
	s.Require().Equal(value1, db.GetState(taddr, key))

	db.RevertToSnapshot(rev1)
	s.Require().Equal(common.Hash{}, db.GetState(taddr, key))
}

func (s *Suite) TestInvalidSnapshotId() {
	deps := evmtest.NewTestDeps()
	db := deps.NewStateDB()

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
			s.Require().False(db.AddressInAccessList(taddr))
			db.AddAddressToAccessList(taddr)
			s.Require().True(db.AddressInAccessList(taddr))

			addrPresent, slotPresent := db.SlotInAccessList(taddr, value1)
			s.Require().True(addrPresent)
			s.Require().False(slotPresent)

			// add again, should be no-op
			db.AddAddressToAccessList(taddr)
			s.Require().True(db.AddressInAccessList(taddr))
		}},
		{"add slot", func(db vm.StateDB) {
			addrPresent, slotPresent := db.SlotInAccessList(taddr, value1)
			s.Require().False(addrPresent)
			s.Require().False(slotPresent)
			db.AddSlotToAccessList(taddr, value1)
			addrPresent, slotPresent = db.SlotInAccessList(taddr, value1)
			s.Require().True(addrPresent)
			s.Require().True(slotPresent)

			// add another slot
			db.AddSlotToAccessList(taddr, value2)
			addrPresent, slotPresent = db.SlotInAccessList(taddr, value2)
			s.Require().True(addrPresent)
			s.Require().True(slotPresent)

			// add again, should be noop
			db.AddSlotToAccessList(taddr, value2)
			addrPresent, slotPresent = db.SlotInAccessList(taddr, value2)
			s.Require().True(addrPresent)
			s.Require().True(slotPresent)
		}},
		{"prepare access list", func(db vm.StateDB) {
			al := gethcore.AccessList{{
				Address:     address3,
				StorageKeys: []common.Hash{value1},
			}}

			sender, dest := taddr, &taddr2
			db.Prepare(params.Rules{}, sender, sender, dest, vm.PrecompiledAddressesBerlin, al)

			// check sender and dst
			s.Require().True(db.AddressInAccessList(taddr))
			s.Require().True(db.AddressInAccessList(taddr2))
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
		db := deps.NewStateDB()
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
	txConfig := evmstate.TxConfig{
		BlockHash: blockHash,
		TxHash:    txHash,
		TxIndex:   txIdx,
		LogIndex:  logIdx,
	}

	deps := evmtest.NewTestDeps()
	sdb := evmstate.NewSDB(deps.Ctx(), deps.App.EvmKeeper, txConfig)

	logData := []byte("hello world")
	log := &gethcore.Log{
		Address:     taddr,
		Topics:      []common.Hash{},
		Data:        logData,
		BlockNumber: blockNumber,
	}
	sdb.AddLog(log)
	s.Require().Equal(1, len(sdb.Logs()))

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
	s.Require().Equal(wantLog, sdb.Logs()[0])

	// Add a second log and assert values
	sdb.AddLog(log)
	wantLog.Index++
	s.Require().Equal(2, len(sdb.Logs()))
	gotLog := sdb.Logs()[1]
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
		db := deps.NewStateDB()
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
	sdb := deps.NewStateDB()
	sdb.SetState(taddr, key1, value1)
	sdb.SetState(taddr, key2, value2)

	// ForEachStorage only iterate committed state
	s.Require().Empty(CollectContractStorage(sdb))

	sdb.Commit()

	storage := CollectContractStorage(sdb)
	s.Require().Equal(2, len(storage))

	keySet := set.New[common.Hash](key1, key2)
	valSet := set.New[common.Hash](value1, value2)
	for _, stateKey := range storage.SortedKeys() {
		stateValue := deps.EvmKeeper.GetState(deps.Ctx(), taddr, stateKey)
		s.True(keySet.Has(stateKey))
		s.True(valSet.Has(stateValue))
	}

	// break early iteration
	storage = make(evmstate.Storage)
	err := sdb.ForEachStorage(taddr, func(k, v common.Hash) bool {
		storage[k] = v
		// return false to break early
		return false
	})
	s.Require().NoError(err)
	s.Require().Equal(1, len(storage))
}
