package stableset

import (
	"errors"
	"unsafe"
)

const (
	groupSize = 8

	slotEmpty   = 0x80
	slotDeleted = 0xFE

	bitsetLSB = 0x0101010101010101
	bitsetMSB = 0x8080808080808080
)

// StableSet is a set-like data structure, which uses swiss-tables under the hood.
// It's stable, because it's designed to never grow up - it retains the capacity
// it was initialized with. This is especially helpful for a large sets in memory.
// StableSet is not designed as a fully compatible set structure, it's just doesn't
// store values, only keys.
// Since we're going to use swiss table rehashing, it's not safe to iter over the set,
// and the iteration API is not provided.
type StableSet[K comparable] struct {
	// TODO: On a large set, we probably need buckets

	// ctrl holds the metadata (1 byte per slot)
	// We pad this by 8 or 16 bytes so SIMD/SWAR
	// always has a full group to read at the end.
	ctrls []uint8
	// slots holds the actual keys.
	slots []K

	capacity          uintptr
	capacityMask      uintptr
	capacityEffective uintptr
	size              uintptr

	hashFunc HashFunc[K]
}

type Option[K comparable] func(ss *StableSet[K])

func New[K comparable](capacity int, opts ...Option[K]) *StableSet[K] {
	normalizedCapacity := uintptr(NextPowerOf2(uint32(capacity)))
	capacityMask := uintptr(normalizedCapacity - 1)

	ss := &StableSet[K]{
		// we add groupSize for slice size for padding during SIMD/SWAR
		ctrls:             make([]uint8, normalizedCapacity+groupSize),
		slots:             make([]K, normalizedCapacity),
		capacity:          normalizedCapacity,
		capacityMask:      capacityMask,
		capacityEffective: normalizedCapacity * 7 / 8,
	}

	for _, opt := range opts {
		opt(ss)
	}

	if ss.hashFunc == nil {
		ss.hashFunc = MakeDefaultHashFunc[K]()
	}

	ss.ctrls[0] = slotEmpty
	for i := 1; i < len(ss.ctrls); i *= 2 {
		copy(ss.ctrls[i:], ss.ctrls[:i])
	}

	return ss
}

// Override default hash function.
func WithHashFunc[K comparable](f HashFunc[K]) Option[K] {
	return func(ss *StableSet[K]) {
		ss.hashFunc = f
	}
}

func (ss *StableSet[K]) EffectiveCapacity() int {
	return int(ss.capacityEffective)
}

func (ss *StableSet[K]) Has(key K) bool {
	ctrls := ss.ctrls
	slots := ss.slots
	mask := ss.capacityMask

	h1, h2 := HashSplit(ss.hashFunc(key))
	offset := h1 & mask

	for probe := uintptr(0); ; {
		group := *(*uint64)(unsafe.Pointer(&ctrls[offset]))
		matchMask := ss.matchH2(group, h2)
		for matchMask != 0 {
			idx := matchMask.first()
			if slots[(offset+idx)&mask] == key {
				return true
			}
			matchMask = matchMask.removeFirst()
		}

		if ss.matchEmpty(group) != 0 {
			return false
		}

		probe++
		offset = (offset + probe*groupSize) & mask
		if probe >= uintptr(ss.capacity)/groupSize {
			return false
		}
	}
}

// Puts a key in the set.
func (ss *StableSet[K]) Put(key K) (bool, bool) {
	// We reached the 87.5% of the capacity, table needs rehashing.
	if ss.size >= ss.capacityEffective {
		return false, true
	}

	var (
		h      = ss.hashFunc(key)
		h1, h2 = HashSplit(h)
		offset = h1 & ss.capacityMask

		slotAvailable    bool
		slotAvailableIdx uintptr
	)

	for probe := uintptr(0); ; {
		group := *(*uint64)(unsafe.Pointer(&ss.ctrls[offset]))
		matchMask := ss.matchH2(group, h2)
		for matchMask != 0 {
			idx := matchMask.first()
			if ss.slots[(offset+idx)&ss.capacityMask] == key {
				return false, false
			}

			matchMask = matchMask.removeFirst()
		}

		if !slotAvailable {
			matchMask = ss.matchEmptyOrDeleted(group)
			if matchMask != 0 {
				slotAvailable = true
				slotAvailableIdx = (offset + matchMask.first()) & ss.capacityMask
			}
		}

		if ss.matchEmpty(group) != 0 {
			break
		}

		probe++
		offset = (offset + probe*groupSize) & ss.capacityMask
	}

	if slotAvailable {
		ss.ctrls[slotAvailableIdx] = h2
		ss.slots[slotAvailableIdx] = key
		ss.size++

		if slotAvailableIdx < groupSize {
			ss.ctrls[uintptr(ss.capacity)+slotAvailableIdx] = h2
		}

		return true, false
	}

	// Needs rehashing
	return false, true
}

func (ss *StableSet[K]) Delete(key K) bool {
	h1, h2 := HashSplit(ss.hashFunc(key))
	offset := h1 & ss.capacityMask

	for probe := uintptr(0); ; {
		group := *(*uint64)(unsafe.Pointer(&ss.ctrls[offset]))
		// 1. Check current group for the key
		matchMask := ss.matchH2(group, h2)
		for matchMask != 0 {
			idx := matchMask.first()
			slotIdx := (offset + idx) & ss.capacityMask

			if ss.slots[slotIdx] == key {
				// Mark as Deleted (0xFE) to preserve the probe chain
				ss.setCtrl(slotIdx, slotDeleted)
				ss.size--

				return true
			}

			matchMask = matchMask.removeFirst()
		}

		// 2. Stop searching ONLY if we hit a truly Empty slot
		if ss.matchEmpty(group) != 0 {
			break
		}

		// 3. Keep probing
		probe++
		offset = (offset + probe*groupSize) & ss.capacityMask

		// Safety check to prevent infinite loop in a 100% full table
		if probe >= uintptr(ss.capacity)/groupSize {
			break
		}
	}

	return false
}

func (ss *StableSet[K]) Reset() {
	// Only the control bytes need to be reset to 'Empty'
	// to make the set logically empty.
	for i := range ss.ctrls {
		ss.ctrls[i] = slotEmpty
	}

	ss.size = 0
}

func (ss *StableSet[K]) Rehash() error {
	// We want to drop all of the deletes in place. We first walk over the
	// control bytes and mark every DELETED slot as EMPTY and every FULL slot
	// as DELETED. Marking the DELETED slots as EMPTY has effectively dropped
	// the tombstones, but we fouled up the probe invariant. Marking the FULL
	// slots as DELETED gives us a marker to locate the previously FULL slots.

	// Mark all DELETED slots as EMPTY and all FULL slots as DELETED.
	for idx := uintptr(0); idx < uintptr(len(ss.ctrls)); idx++ {
		c := ss.ctrls[idx]
		if c < 0x80 {
			ss.setCtrl(idx, slotDeleted)
		} else if c == slotDeleted {
			ss.setCtrl(idx, slotEmpty)
		}
	}

	for idx := uintptr(0); idx < ss.capacity; idx++ {
		// Only process slots we marked as Deleted (which were originally Full)
		if ss.ctrls[idx] != slotDeleted {
			continue
		}

		key := ss.slots[idx]
		h := ss.hashFunc(key)
		h1, h2 := HashSplit(h)

		// Logic: Where should this key go?
		desiredOffset := h1 & ss.capacityMask

		// Find the best available slot starting from its ideal position
		var (
			targetIdx uintptr
			found     bool
		)

		// We use a simplified probe sequence here
		probeIdx := desiredOffset
		for step := uintptr(0); ; step++ {
			// In a rehash, we only care about finding the first Empty or Deleted slot
			// matchEmptyOrDeleted detects both 0x80 and 0xFE
			group := *(*uint64)(unsafe.Pointer(&ss.ctrls[probeIdx]))
			mask := ss.matchEmptyOrDeleted(group)
			if mask != 0 {
				targetIdx = (probeIdx + mask.first()) & ss.capacityMask
				found = true
				break
			}

			probeIdx = (probeIdx + (step+1)*groupSize) & ss.capacityMask
		}

		if !found {
			return errors.New("no empty slots found, cannot rehash")
		}

		switch {
		case idx == targetIdx:
			// 1. Element is already in its best possible position.
			// Just restore its real h2.
			ss.setCtrl(idx, h2)

		case ss.ctrls[targetIdx] == slotEmpty:
			// 2. We found an empty spot. Move the key there and leave a hole behind.
			ss.setCtrl(targetIdx, h2)
			ss.slots[targetIdx] = key
			ss.setCtrl(idx, slotEmpty)

		case ss.ctrls[targetIdx] == slotDeleted:
			// 3. THE SWAP: The target is currently holding ANOTHER key that
			// hasn't been re-inserted yet.
			// We swap them, put the current key in its place, and re-process
			// the new key now sitting at index 'i'.
			ss.setCtrl(targetIdx, h2)
			ss.slots[idx], ss.slots[targetIdx] = ss.slots[targetIdx], ss.slots[idx]

			// Stay at current index 'i' to process the swapped key in the next iteration
			idx--

		default:
			return errors.New("rehash: invalid control state")
		}
	}

	return nil
}

// setCtrl ensures mirroring is maintained during rehash
func (ss *StableSet[K]) setCtrl(i uintptr, val uint8) {
	ss.ctrls[i] = val
	if i < groupSize {
		ss.ctrls[uintptr(ss.capacity)+i] = val
	}
}

func (ss *StableSet[K]) matchH2(group uint64, h2 uint8) bitset {
	v := group ^ (bitsetLSB * uint64(h2))
	return bitset(((v - bitsetLSB) &^ v) & bitsetMSB)
}

// matchEmpty: Check if MSB is 1 AND bit 1 is 0.
// (0x80 is 10000000, bit 1 is 0. 0xFE is 11111110, bit 1 is 1)
func (ss *StableSet[K]) matchEmpty(group uint64) bitset {
	return bitset((group &^ (group << 6)) & bitsetMSB)
}

// matchEmptyOrDeleted: Just check if the MSB is 1.
// (Both 0x80 and 0xFE have it, Full slots don't)
func (ss *StableSet[K]) matchEmptyOrDeleted(group uint64) bitset {
	return bitset(group & bitsetMSB)
}
