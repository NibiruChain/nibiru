package statedb

// Copyright 2020 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

import (
	"encoding/json"

	"github.com/NibiruChain/nibiru/v2/x/common/set"

	gethcommon "github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"
)

// accessList is an EIP-2930 access list. The specification requires unique slots
// per address, fast membership testing with O(1) lookups, and order
// independence.
//
// Invariant:
//   - addr ∉ map               => not present in accessList
//   - map[addr] -> nil | empty => address present, zero slots
//   - map[addr] -> slots       => address + N slots
type accessList map[gethcommon.Address]set.Set[gethcommon.Hash]

var _ json.Marshaler = (*accessList)(nil)
var _ json.Unmarshaler = (*accessList)(nil)

func (al *accessList) MarshalJSON() (bz []byte, err error) {
	accessTupleJson := make(map[gethcommon.Address][]gethcommon.Hash)
	if al == nil {
		return json.Marshal(accessTupleJson)
	}

	for addr, slotset := range *al {
		accessTupleJson[addr] = slotset.ToSlice()
	}
	return json.Marshal(accessTupleJson)
}

// TODO: UD-DEBUG: test
func (al *accessList) UnmarshalJSON(bz []byte) (err error) {
	var accessTupleJson map[gethcommon.Address][]gethcommon.Hash
	if err := json.Unmarshal(bz, &accessTupleJson); err != nil {
		return err // TODO: UD-DEBUG: err msg
	}
	if *al == nil {
		*al = make(accessList)
	}
	for addr, slots := range accessTupleJson {
		(*al)[addr] = set.New(slots...)
	}
	return nil
}

// type AccessList struct {
// 	// Addrs is a map from address to slot index, the slice index of
// 	// `AccessList.Slots`. The index "-1" is a sentinel value meaning the address
// 	// is tracked but does not yet have any slots.

// 	Addrs map[gethcommon.Address]int `json:"addrs"`
// 	Slots []set.Set[gethcommon.Hash] `json:"slots"`
// }

// AddAddressToAccessList adds the given address to the access list
func (s *StateDB) AddAddressToAccessList(addr gethcommon.Address) {
	al := s.getAccessList()
	defer s.setAccessList(al)
	if _, addrOk := al[addr]; addrOk {
		return
	}
	al[addr] = set.New[gethcommon.Hash]()
}

// AddSlotToAccessList adds the given (address, slot)-tuple to the access list
func (s *StateDB) AddSlotToAccessList(addr gethcommon.Address, slot gethcommon.Hash) {
	al := s.getAccessList()
	_, _ = al.AddSlot(addr, slot)
	s.setAccessList(al)
}

func (s *StateDB) getAccessList() accessList {
	accessListBz := func() []byte {
		if len(s.localState.accessList) > 0 {
			return s.localState.accessList
		}
		for i := len(s.savedStates) - 1; i >= 0; i-- {
			bz := s.savedStates[i].accessList
			if len(bz) > 0 {
				return bz
			}
		}
		return nil
	}()
	if len(accessListBz) == 0 {
		return make(accessList)
	}
	var al accessList
	if err := json.Unmarshal(accessListBz, &al); err != nil {
		panic(err) // TODO: UD-DEBUG: err mesg
	}
	return al
}

func (s *StateDB) setAccessList(al accessList) {
	accessListBz, err := json.Marshal(al)
	if err != nil {
		panic(err) // TODO: UD-DEBUG: err mesg
	}
	s.localState.accessList = accessListBz
}

// AddressInAccessList returns true if the given address is in the access list.
func (s *StateDB) AddressInAccessList(addr gethcommon.Address) bool {
	al := s.getAccessList()
	_, ok := al[addr]
	return ok
}

// SlotInAccessList returns true if the given (address, slot)-tuple is in the
// access list. Checks if a slot for some account is present in the access list,
// returning separate flags for the presence of the account and the slot
// respectively.
func (s *StateDB) SlotInAccessList(
	addr gethcommon.Address, slot gethcommon.Hash,
) (addrPresent bool, slotPresent bool) {
	al := s.getAccessList()
	slotSet, ok := al[addr]
	if !ok {
		// no such address (and hence zero slots)
		return false, false
	}
	if slotSet.Len() == 0 {
		// address yes, but no slots
		return true, false
	}
	return true, slotSet.Has(slot)
}

// AddAddress adds an address to the access list, and returns 'true' if the operation
// caused a change (addr was not previously in the list).
func (al accessList) AddAddress(addr gethcommon.Address) bool {
	if _, present := al[addr]; present {
		return false
	}
	al[addr] = make(set.Set[gethcommon.Hash])
	return true
}

// AddSlot adds the specified (addr, slot) combo to the access list.
// Return values are:
// - address added
// - slot added
// For any 'true' value returned, a corresponding journal entry must be made.
func (al accessList) AddSlot(
	addr gethcommon.Address, slot gethcommon.Hash,
) (addrChange bool, slotChange bool) {
	slotset, addrPresent := al[addr]
	if !addrPresent || slotset.Len() == 0 {
		// Address not present, or addr present but no slots there
		addrChange = !addrPresent
		slotChange = true
		slotSet := set.New(slot)
		al[addr] = slotSet
		return addrChange, slotChange
	}
	// There is already an (address,slot) mapping
	if !slotset.Has(slot) {
		slotset.Add(slot) // Add slot change
		return false, true
	}
	// No changes required
	return false, false
}

// PrepareAccessList handles the preparatory steps for executing a state
// transition with regards to both EIP-2929 and EIP-2930:
//
// - Add sender to access list (2929)
// - Add destination to access list (2929)
// - Add precompiles to access list (2929)
// - Add the contents of the optional tx access list (2930)
//
// This method should only be called if Yolov3/Berlin/2929+2930 is applicable at the current number.
func (s *StateDB) PrepareAccessList(
	sender gethcommon.Address,
	dst *gethcommon.Address,
	precompiles []gethcommon.Address,
	txAccesses gethcore.AccessList,
) {
	s.AddAddressToAccessList(sender)
	if dst != nil {
		s.AddAddressToAccessList(*dst)
		// If it's a create-tx, the destination will be added inside evm.create
	}
	for _, addr := range precompiles {
		s.AddAddressToAccessList(addr)
	}
	for _, el := range txAccesses {
		s.AddAddressToAccessList(el.Address)
		for _, key := range el.StorageKeys {
			s.AddSlotToAccessList(el.Address, key)
		}
	}
}
