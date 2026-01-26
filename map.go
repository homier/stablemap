package stableset

// StableMap is a map-like data structure, which uses swiss-tables under the hood.
// It's stable, because it's designed to never grow up - it retains the capacity
// it was initialized with. This is especially helpful for a large sets in memory.
// Since we're going to use swiss table rehashing, it's not safe to iter over the set,
// and the iteration API is not provided.
type StableMap[K comparable, V any] struct {
	table[K, V]
}

// Returns a new instance of the stable map.
func NewMap[K comparable, V any](capacity int, opts ...Option[K, V]) *StableMap[K, V] {
	var sm StableMap[K, V]
	sm.init(capacity, opts...)

	return &sm
}

// Checks whether a key is in the set.
func (sm *StableMap[K, V]) Get(key K) (V, bool) {
	return sm.get(key)
}

// Puts a key in the set.
// Returns whether a key is new and if compaction is required to be done first.
func (sm *StableMap[K, V]) Put(key K, value V) (bool, bool) {
	return sm.put(key, value)
}

// Delets a key from the set.
func (sm *StableMap[K, V]) Delete(key K) bool {
	return sm.delete(key)
}
