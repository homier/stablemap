package stableset

import (
	"math/bits"
	"strings"
)

// Returns the next power of 2 for the given value `v`.
func NextPowerOf2(v uint32) uint32 {
	return uint32(1) << min(bits.Len32(v-1), 31)
}

// TODO: Estimate capacity from the given memory in bytes
func CapacityFromSize[K comparable](size uint) int {
	return 0
}

// bitset represents a set of slots within a group.
//
// The underlying representation uses one byte per slot, where each byte is
// either 0x80 if the slot is part of the set or 0x00 otherwise. This makes it
// convenient to calculate for an entire group at once (e.g. see matchEmpty).
type bitset uint64

// first assumes that only the MSB of each control byte can be set (e.g. bitset
// is the result of matchEmpty or similar) and returns the relative index of the
// first control byte in the group that has the MSB set.
//
// Returns 8 if the bitset is 0.
// Returns groupSize if the bitset is empty.
func (b bitset) first() uintptr {
	return uintptr(bits.TrailingZeros64(uint64(b)) >> 3)
}

// removeFirst removes the first set bit (that is, resets the least significant set bit to 0).
func (b bitset) removeFirst() bitset {
	return b & ^(bitset(slotEmpty) << (bits.TrailingZeros64(uint64(b)) & ^7))
}

func (b bitset) String() string {
	var buf strings.Builder
	buf.Grow(groupSize)
	for i := range groupSize {
		if (b & (bitset(0x80) << (i << 3))) != 0 {
			buf.WriteString("1")
		} else {
			buf.WriteString("0")
		}
	}
	return buf.String()
}
