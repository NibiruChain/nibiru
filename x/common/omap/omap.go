// Package omap defines a generic-based type for creating ordered maps. The
// package publicly exports the "Sorter" interface to enable the creation of
// new types of ordered maps with arbitrary key and value types.
//
// For instance, the omap package implements for ordered maps that can accept
// string or fmt.Stringer types as keys and arbitrary types as values. See
// OrderedMap_String and OrderedMap_Pair in impl.go.
//
// ## Motivation
//
// A blockchain needs to maintain a deterministic system, where the same action
// with the same input should always produce the same result. This is vital
// because all nodes need to agree on the state of the blockchain, and if
// different nodes get different results for the same action, they won't be
// able to reach consensus.
//
// Disagreements in state can prevent nodes from being able to discern the
// validity of a block. This prevents them from adding a block to the chain and
// can result in chain halts.
package omap

import (
	"sort"
)

// OrderedMap is a wrapper struct around the built-in map that has guarantees
// about order because it sorts its keys with a custom sorter. It has a public
// API that mirrors that functionality of `map`. OrderedMap is built with
// generics, so it can hold various combinations of key-value types.
type OrderedMap[K comparable, V any] struct {
	data        map[K]V
	orderedKeys []K
	keyIndexMap map[K]int // useful for delete operation
	isOrdered   bool
	sorter      Sorter[K]
}

// Sorter is an interface used for ordering the keys in the OrderedMap.
type Sorter[K any] interface {
	// Returns true if 'a' is less than 'b' Less needs to be defined for the
	// key type, K, to provide a comparison operation.
	Less(a K, b K) bool
}

// ensureOrder is a method on the OrderedMap that sorts the keys in the map
// and rebuilds the index map.
func (om *OrderedMap[K, V]) ensureOrder() {
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
	om.keyIndexMap = make(map[K]int)
	for idx, key := range om.orderedKeys {
		om.keyIndexMap[key] = idx
	}
	om.isOrdered = true
}

// BuildFrom is a method that builds an OrderedMap from a given map and a
// sorter for the keys. This function is useful for creating new OrderedMap
// types with typed keys.
func (om OrderedMap[K, V]) BuildFrom(
	data map[K]V, sorter Sorter[K],
) OrderedMap[K, V] {
	om.data = data
	om.sorter = sorter
	om.ensureOrder()
	return om
}

// Range returns a channel of keys in their sorted order. This allows you
// to iterate over the map in a deterministic order. Using a channel here
// makes it so that the iteration is done lazily rather loading the entire
// map (OrderedMap.data) into memory and then iterating.
func (om OrderedMap[K, V]) Range() <-chan (K) {
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
func (om OrderedMap[K, V]) Has(key K) bool {
	_, exists := om.data[key]
	return exists
}

// Len returns the number of items in the map.
func (om OrderedMap[K, V]) Len() int {
	return len(om.data)
}

// Keys returns a slice of the keys in their sorted order.
func (om *OrderedMap[K, V]) Keys() []K {
	if !om.isOrdered {
		om.ensureOrder()
	}
	return om.orderedKeys
}

// Get returns the value associated with a key in the map. If the key is not
// in the map, it returns nil.
func (om *OrderedMap[K, V]) Get(key K) (out *V) {
	v, exists := om.data[key]
	if exists {
		*out = v
	} else {
		out = nil
	}
	return out
}

// Set adds a key-value pair to the map, or updates the value if the key
// already exists. It ensures the keys are ordered after the operation.
func (om *OrderedMap[K, V]) Set(key K, val V) {
	om.data[key] = val
	om.ensureOrder() // TODO perf: make this more efficient with a clever insert.
}

// Delete removes a key-value pair from the map if the key exists.
func (om *OrderedMap[K, V]) Delete(key K) {
	idx, keyExists := om.keyIndexMap[key]
	if keyExists {
		delete(om.data, key)

		orderedKeys := om.orderedKeys
		orderedKeys = append(orderedKeys[:idx], orderedKeys[idx+1:]...)
		om.orderedKeys = orderedKeys
	}
}
