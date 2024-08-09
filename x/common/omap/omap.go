// Package omap defines a generic-based type for creating ordered maps. It
// exports a "Sorter" interface, allowing the creation of sorted maps with
// custom key and value types.
//
// See impl.go for examples.
//
// ## Motivation
//
// Ensuring deterministic behavior is crucial in blockchain systems, as all
// nodes must reach a consensus on the state of the blockchain. Every action,
// given the same input, should consistently yield the same result. A
// divergence in state could impede the ability of nodes to validate a block,
// prohibiting the addition of the block to the chain, which could lead to
// chain halts.
package omap

import (
	"fmt"
	"sort"
)

// SortedMap is a wrapper struct around the built-in map that has guarantees
// about order because it sorts its keys with a custom sorter. It has a public
// API that mirrors that functionality of `map`. SortedMap is built with
// generics, so it can hold various combinations of key-value types.
type SortedMap[K comparable, V any] struct {
	data        map[K]V
	orderedKeys []K
	isOrdered   bool
	sorter      Sorter[K]
}

// Sorter is an interface used for ordering the keys in the OrderedMap.
type Sorter[K any] interface {
	// Returns true if 'a' is less than 'b' Less needs to be defined for the
	// key type, K, to provide a comparison operation.
	Less(a K, b K) bool
}

// SorterLeq is true if a <= b implements "less than or equal" using "Less"
func SorterLeq[K comparable](sorter Sorter[K], a, b K) bool {
	return sorter.Less(a, b) || a == b
}

// Data returns a copy of the underlying map (unordered, unsorted)
func (om *SortedMap[K, V]) Data() map[K]V {
	dataCopy := make(map[K]V, len(om.data))
	for k, v := range om.InternalData() {
		dataCopy[k] = v
	}
	return dataCopy
}

// InternalData returns the SortedMap's private map.
func (om *SortedMap[K, V]) InternalData() map[K]V {
	return om.data
}

func (om *SortedMap[K, V]) Get(key K) (val V, exists bool) {
	val, exists = om.data[key]
	return val, exists
}

// ensureOrder is a method on the OrderedMap that sorts the keys in the map
// and rebuilds the index map.
func (om *SortedMap[K, V]) ensureOrder() {
	if om.isOrdered {
		return
	}

	keys := make([]K, 0, len(om.data))
	for key := range om.data {
		keys = append(keys, key)
	}

	// Sort the keys using the Sort function
	lessFunc := func(i, j int) bool {
		return om.sorter.Less(keys[i], keys[j])
	}
	sort.Slice(keys, lessFunc)

	om.orderedKeys = keys
	om.isOrdered = true
}

// BuildFrom is a method that builds an OrderedMap from a given map and a
// sorter for the keys. This function is useful for creating new OrderedMap
// types with typed keys.
func (om *SortedMap[K, V]) BuildFrom(
	data map[K]V, sorter Sorter[K],
) *SortedMap[K, V] {
	om.data = data
	om.sorter = sorter
	om.ensureOrder()
	return om
}

// Range returns a channel of keys in their sorted order. This allows you
// to iterate over the map in a deterministic order. Using a channel here
// makes it so that the iteration is done lazily rather loading the entire
// map (OrderedMap.data) into memory and then iterating.
func (om *SortedMap[K, V]) Range() <-chan (K) {
	iterChan := make(chan K)
	go func() {
		defer close(iterChan)
		// Generate or compute values on-demand
		for _, key := range om.orderedKeys {
			iterChan <- key
		}
	}()
	return iterChan
}

// Has checks whether a key exists in the map.
func (om *SortedMap[K, V]) Has(key K) bool {
	_, exists := om.data[key]
	return exists
}

// Len returns the number of items in the map.
func (om *SortedMap[K, V]) Len() int {
	return len(om.data)
}

// Keys returns a slice of the keys in their sorted order.
func (om *SortedMap[K, V]) Keys() []K {
	if !om.isOrdered {
		om.ensureOrder()
	}
	return om.orderedKeys
}

// Set adds a key-value pair to the map, or updates the value if the key
// already exists. It ensures the keys are ordered after the operation.
func (om *SortedMap[K, V]) Set(key K, val V) {
	_, exists := om.data[key]
	om.data[key] = val

	if !exists {
		lenBefore := len(om.orderedKeys)

		if lenBefore == 0 {
			// If the map is empty, create it. There's no need to search.
			om.orderedKeys = []K{key}
			return
		}

		// If the key is new, insert it to the correctly sorted position
		// Binary search works here and is in the standard library.
		idx := sort.Search(lenBefore, func(i int) bool {
			return om.sorter.Less(key, om.orderedKeys[i])
		})

		fmt.Printf("idx: %d\n", idx)
		fmt.Printf("lenBefore: %d\n", lenBefore)

		// Update om.orderedKeys
		newSortedKeys := make([]K, lenBefore+1)
		front, back := om.orderedKeys[:idx], om.orderedKeys[idx:]
		copy(newSortedKeys[:idx], front)  // front
		newSortedKeys[idx] = key          // middle
		copy(newSortedKeys[idx+1:], back) // back
		om.orderedKeys = newSortedKeys
	}
}

// Union combines new key-value pairs into the ordered map.
func (om *SortedMap[K, V]) Union(kvMap map[K]V) {
	for key, val := range kvMap {
		om.data[key] = val
	}
	om.isOrdered = false
	om.ensureOrder() // TODO perf: make this more efficient with a clever insert.
}

// Delete removes a key-value pair from the map if the key exists.
func (om *SortedMap[K, V]) Delete(key K) {
	if _, keyExists := om.data[key]; keyExists {
		lenBeforeDelete := om.Len()
		delete(om.data, key)

		// Remove the key from orderedKeys while preserving the order
		idx := sort.Search(lenBeforeDelete, func(i int) bool {
			return SorterLeq(om.sorter, key, om.orderedKeys[i])
		})

		// Update om.orderedKeys, skipping the deleted key (om.orderedKeys[idx])
		newSortedKeys := make([]K, lenBeforeDelete-1)
		copy(newSortedKeys[:idx], om.orderedKeys[:idx])   // front
		copy(newSortedKeys[idx:], om.orderedKeys[idx+1:]) // middle + back
		om.orderedKeys = newSortedKeys
	}
}
