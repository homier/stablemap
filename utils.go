package stableset

import "math/bits"

// Returns the next power of 2 for the given value `v`.
func NextPowerOf2(v uint32) uint32 {
	return uint32(1) << min(bits.Len32(v-1), 31)
}

// TODO: Estimate capacity from the given memory in bytes
func CapacityFromSize[K comparable](size uint) int {
	return 0
}
