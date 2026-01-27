package stablemap

import (
	"math/bits"
)

const (
	bitsetLSB = 0x0101010101010101
	bitsetMSB = 0x8080808080808080
)

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

//go:inline
func matchH2(group uint64, h2 uint8) bitset {
	v := group ^ (bitsetLSB * uint64(h2))
	return bitset(((v - bitsetLSB) &^ v) & bitsetMSB)
}

// matchEmpty: Check if MSB is 1 AND bit 1 is 0.
// (0x80 is 10000000, bit 1 is 0. 0xFE is 11111110, bit 1 is 1)
//
//go:inline
func matchEmpty(group uint64) bitset {
	return bitset((group &^ (group << 6)) & bitsetMSB)
}

// matchEmptyOrDeleted: Just check if the MSB is 1.
// (Both 0x80 and 0xFE have it, Full slots don't)
//
//go:inline
func matchEmptyOrDeleted(group uint64) bitset {
	return bitset(group & bitsetMSB)
}
