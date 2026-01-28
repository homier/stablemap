package stablemap

import (
	"math/bits"
	"unsafe"
)

// Returns the next power of 2 for the given value `v`.
func NextPowerOf2(v uint32) uint32 {
	return uint32(1) << min(bits.Len32(v-1), 31)
}

// Estimates capacity (number of slots) from the given memory size in bytes.
func CapacityFromSize[K comparable, V any](size uintptr) int {
	sizeOfGroup := unsafe.Sizeof(group[K, V]{})
	numGroups := size / sizeOfGroup

	return int(numGroups * groupSize)
}
