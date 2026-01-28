package stablemap

// StableMap is a map-like data structure, which uses swiss-tables under the hood.
// It's stable, because it's designed to never grow up - it retains the capacity
// it was initialized with. This is especially helpful for a large sets in memory.
// Since we're going to use swiss table rehashing, it's not safe to iter over the set,
// and the iteration API is not provided.
type StableMap[K comparable, V any] struct {
	table[K, V]
}

// Returns a new instance of the stable map.
func New[K comparable, V any](capacity int, opts ...Option[K, V]) *StableMap[K, V] {
	var sm StableMap[K, V]
	sm.init(capacity, opts...)

	return &sm
}

// Checks whether a key is in the map.
func (sm *StableMap[K, V]) Get(key K) (V, bool) {
	return sm.get(key)
}

// Sets a key in the map.
// If the key is already present, overwrites it.
// Returns an error if compaction is required.
func (sm *StableMap[K, V]) Set(key K, value V) error {
	return sm.set(key, value)
}

// Delets a key from the set.
func (sm *StableMap[K, V]) Delete(key K) bool {
	return sm.delete(key)
}
