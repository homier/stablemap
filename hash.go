package stableset

import "hash/maphash"

type HashFunc[K comparable] func(K) uint64

// TODO: Use different hash functions
func MakeDefaultHashFunc[K comparable]() HashFunc[K] {
	seed := maphash.MakeSeed()

	return func(k K) uint64 {
		return maphash.Comparable(seed, k)
	}
}

func HashSplit(hash uint64) (uintptr, uint8) {
	h1 := uintptr(hash >> 7)
	h2 := uint8(hash & 0x7F)

	return h1, h2
}
