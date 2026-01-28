package stablemap

// StableSet is a set-like data structure, which uses swiss-tables under the hood.
// It's stable, because it's designed to never grow up - it retains the capacity
// it was initialized with. This is especially helpful for a large sets in memory.
// StableSet is not designed as a fully compatible set structure, it's just doesn't
// store values, only keys.
// Since we're going to use swiss table rehashing, it's not safe to iter over the set,
// and the iteration API is not provided.
type StableSet[K comparable] struct {
	table[K, struct{}]
}

// Returns a new instance of the stable set.
func NewSet[K comparable](capacity int, opts ...Option[K, struct{}]) *StableSet[K] {
	var ss StableSet[K]
	ss.init(capacity, opts...)

	return &ss
}

// Checks whether a key is in the set.
func (ss *StableSet[K]) Has(key K) bool {
	_, ok := ss.get(key)

	return ok
}

// Puts a key in the set.
// Returns whether a key is new and an error if compaction is required.
func (ss *StableSet[K]) Put(key K) (bool, error) {
	return ss.put(key, struct{}{})
}

// Delets a key from the set.
func (ss *StableSet[K]) Delete(key K) bool {
	return ss.delete(key)
}
