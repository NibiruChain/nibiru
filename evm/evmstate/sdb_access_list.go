package evmstate

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
	"fmt"

	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/NibiruChain/nibiru/v2/x/nutil/set"
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

var (
	_ json.Marshaler   = (*accessList)(nil)
	_ json.Unmarshaler = (*accessList)(nil)
)

func (al accessList) MarshalJSON() (bz []byte, err error) {
	accessTupleJson := make(map[gethcommon.Address][]gethcommon.Hash)
	if len(al) == 0 {
		return json.Marshal(accessTupleJson)
	}
	for addr, slotset := range al {
		accessTupleJson[addr] = slotset.ToSlice()
	}
	return json.Marshal(accessTupleJson)
}

func (al *accessList) UnmarshalJSON(bz []byte) (err error) {
	var accessTupleJson map[gethcommon.Address][]gethcommon.Hash
	if err := json.Unmarshal(bz, &accessTupleJson); err != nil {
		return fmt.Errorf("accessList error: %w", err)
	}
	if *al == nil {
		*al = make(accessList) // Safe: initializes if nil
	}
	for addr, slots := range accessTupleJson {
		(*al)[addr] = set.New(slots...) // Safe: set.New handles empty slices
	}
	return nil
}

// AddAddressToAccessList adds the given address to the access list. This
// operation is safe to perform even if the ccess list fork is not active yet.
// This function implements the [vm.StateDB] interface.
func (s *SDB) AddAddressToAccessList(addr gethcommon.Address) {
	al := s.getAccessList()
	defer s.setAccessList(al)
	if _, addrOk := al[addr]; addrOk {
		return
	}
	al[addr] = set.New[gethcommon.Hash]()
}

// AddSlotToAccessList adds the given (address, slot)-tuple to the access list.
// This operation is safe to perform even if the ccess list fork is not active
// yet.
// This function implements the [vm.StateDB] interface.
func (s *SDB) AddSlotToAccessList(addr gethcommon.Address, slot gethcommon.Hash) {
	al := s.getAccessList()
	_, _ = al.AddSlot(addr, slot)
	s.setAccessList(al)
}

func (s *SDB) getAccessList() accessList {
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
		// Safe: Since [accessList] has only private access, we guard against
		// malformatted data and verified that's the case with tests. This panic
		// should be logically impossible.
		panic(err)
	}
	return al
}

func (s *SDB) setAccessList(al accessList) {
	accessListBz, err := al.MarshalJSON()
	if err != nil {
		// Safe: Since [accessList] has only private access, we guard against
		// malformatted data and verified that's the case with tests. This panic
		// should be logically impossible.
		panic(err)
	}
	s.localState.accessList = accessListBz
}

// AddressInAccessList returns true if the given address is in the access list.
func (s *SDB) AddressInAccessList(addr gethcommon.Address) bool {
	al := s.getAccessList()
	_, ok := al[addr]
	return ok
}

// SlotInAccessList returns true if the given (address, slot)-tuple is in the
// access list. Checks if a slot for some account is present in the access list,
// returning separate flags for the presence of the account and the slot
// respectively.
func (s *SDB) SlotInAccessList(
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

// Equals returns true if the two access lists are equal. This
// function is for testing purposes.
func (al accessList) Equals(other accessList) (
	isEqual bool, inequalityReason string,
) {
	if len(al) != len(other) {
		return false, "mismatch in number of keys"
	}
	for addr, list := range al {
		listOther, ok := other[addr]
		if !ok {
			return false, fmt.Sprintf("other has missing key { addr: %s }", addr)
		}
		if !listOther.Equals(list) {
			return false, fmt.Sprintf("slots mismatch for key { addr: %s }", addr)
		}
	}
	return true, ""
}
