package stableset

import "unsafe"

const (
	groupSize = 8

	slotEmpty   = 0x80
	slotDeleted = 0xFE

	bitsetLSB = 0x0101010101010101
	bitsetMSB = 0x8080808080808080
)

type group[K comparable] struct {
	// 8 bytes of metadata (h2 or control states)
	// This fits perfectly in a single uint64 load
	ctrls [groupSize]uint8

	// 8 keys stored immediately after the metadata
	// In a 64-bit system, this group is (8 + 8*8) = 72 bytes.
	// That's just slightly over one 64-byte cache line.
	slots [groupSize]K
}

// StableSet is a set-like data structure, which uses swiss-tables under the hood.
// It's stable, because it's designed to never grow up - it retains the capacity
// it was initialized with. This is especially helpful for a large sets in memory.
// StableSet is not designed as a fully compatible set structure, it's just doesn't
// store values, only keys.
// Since we're going to use swiss table rehashing, it's not safe to iter over the set,
// and the iteration API is not provided.
type StableSet[K comparable] struct {
	groups []group[K]

	capacity          uintptr
	numGroupsMask     uintptr
	capacityEffective uintptr
	size              uintptr

	hashFunc HashFunc[K]
}

type Option[K comparable] func(ss *StableSet[K])

func New[K comparable](capacity int, opts ...Option[K]) *StableSet[K] {
	normalizedCapacity := uintptr(NextPowerOf2(uint32(capacity)))
	// Number of groups required
	numGroups := normalizedCapacity / groupSize
	numGroupsMask := uintptr(numGroups - 1)

	ss := &StableSet[K]{
		groups:            make([]group[K], numGroups),
		capacity:          normalizedCapacity,
		numGroupsMask:     numGroupsMask,
		capacityEffective: normalizedCapacity * 7 / 8,
	}

	for _, opt := range opts {
		opt(ss)
	}

	if ss.hashFunc == nil {
		ss.hashFunc = MakeDefaultHashFunc[K]()
	}

	// Initialize all control bytes to Empty
	for i := range ss.groups {
		for j := range ss.groups[i].ctrls {
			ss.groups[i].ctrls[j] = slotEmpty
		}
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
	h1, h2 := HashSplit(ss.hashFunc(key))
	mask := ss.numGroupsMask
	start := (h1 / groupSize) & mask

	for p, offset := uintptr(0), start; p <= mask; p++ {
		g := &ss.groups[offset]
		ctrl := *(*uint64)(unsafe.Pointer(&g.ctrls))

		// SIMD-like match
		matches := matchH2(ctrl, h2)
		for matches != 0 {
			if g.slots[matches.first()] == key {
				return true
			}
			matches = matches.removeFirst()
		}

		// Termination
		if matchEmpty(ctrl) != 0 {
			return false
		}

		// Quadratic probe math
		offset = (start + (p+1)*(p+2)/2) & mask
	}

	return false
}

// Puts a key in the set.
func (ss *StableSet[K]) Put(key K) (bool, bool) {
	// We reached the 87.5% of the capacity, table needs rehashing.
	if ss.size >= ss.capacityEffective {
		return false, true
	}

	var (
		h1, h2 = HashSplit(ss.hashFunc(key))
		mask   = ss.numGroupsMask
		start  = (h1 / groupSize) & mask

		targetGroup *group[K]
		targetSlot  uintptr
		foundSlot   bool
	)

	for p, offset := uintptr(0), start; p <= mask; p++ {
		g := &ss.groups[offset]
		ctrl := *(*uint64)(unsafe.Pointer(&g.ctrls))

		// 1. Existing check
		matchMask := matchH2(ctrl, h2)
		for matchMask != 0 {
			if g.slots[matchMask.first()] == key {
				return false, false
			}

			matchMask = matchMask.removeFirst()
		}

		// 2. Cache first available slot
		if !foundSlot {
			matchMask = matchEmptyOrDeleted(ctrl)
			if matchMask != 0 {
				targetGroup = g
				targetSlot = matchMask.first()
				foundSlot = true
			}
		}

		// 3. Termination condition
		matchMask = matchEmpty(ctrl)
		if matchMask != 0 {
			if foundSlot {
				targetGroup.ctrls[targetSlot] = h2
				targetGroup.slots[targetSlot] = key
				ss.size++

				return true, false
			}

			return false, true
		}

		offset = (start + (p+1)*(p+2)/2) & mask
	}

	return false, true
}

func (ss *StableSet[K]) Delete(key K) bool {
	h1, h2 := HashSplit(ss.hashFunc(key))
	mask := ss.numGroupsMask
	start := (h1 / groupSize) & mask

	for p, offset := uintptr(0), start; p <= mask; p++ {
		g := &ss.groups[offset]
		ctrl := *(*uint64)(unsafe.Pointer(&g.ctrls))

		// 1. Check current group for the key
		matchMask := matchH2(ctrl, h2)
		for matchMask != 0 {
			idx := matchMask.first()
			if g.slots[idx] == key {
				// Mark as Deleted (0xFE) to preserve the probe chain
				g.ctrls[idx] = slotDeleted
				ss.size--

				return true
			}

			matchMask = matchMask.removeFirst()
		}

		if matchEmpty(ctrl) != 0 {
			return false
		}

		offset = (start + (p+1)*(p+2)/2) & mask
	}

	return false
}

func (ss *StableSet[K]) Reset() {
	for i := range ss.groups {
		for j := range groupSize {
			ss.groups[i].ctrls[j] = slotEmpty
		}
	}

	ss.size = 0
}

func (ss *StableSet[K]) Compact() error {
	// We want to drop all of the deletes in place. We first walk over the
	// control bytes and mark every DELETED slot as EMPTY and every FULL slot
	// as DELETED. Marking the DELETED slots as EMPTY has effectively dropped
	// the tombstones, but we fouled up the probe invariant. Marking the FULL
	// slots as DELETED gives us a marker to locate the previously FULL slots.

	// Mark all DELETED slots as EMPTY and all FULL slots as DELETED.
	for i := range ss.groups {
		g := &ss.groups[i]
		for j := range groupSize {
			c := g.ctrls[j]
			if c < 0x80 {
				g.ctrls[j] = slotDeleted
			} else if c == slotDeleted {
				g.ctrls[j] = slotEmpty
			}
		}
	}

	for idx := 0; idx < len(ss.groups); idx++ {
		g := &ss.groups[idx]
		for j := uintptr(0); j < groupSize; j++ {
			// Only process slots we marked as Deleted (which were originally Full)
			if g.ctrls[j] != slotDeleted {
				continue
			}

			var (
				key          = g.slots[j]
				h            = ss.hashFunc(key)
				h1, h2       = HashSplit(h)
				destGroupIdx = (h1 / groupSize) & ss.numGroupsMask

				targetGroup *group[K]
				targetSlot  uintptr

				p        = uintptr(0)
				currGIdx = destGroupIdx
			)

			for {
				tg := &ss.groups[currGIdx]
				tc := *(*uint64)(unsafe.Pointer(&tg.ctrls))
				m := matchEmptyOrDeleted(tc)
				if m != 0 {
					targetGroup = tg
					targetSlot = m.first()
					break
				}
				p++
				currGIdx = (currGIdx + p) & ss.numGroupsMask
			}

			// Swap / Move logic
			if targetGroup == g && targetSlot == j {
				g.ctrls[j] = h2
			} else if targetGroup.ctrls[targetSlot] == slotEmpty {
				targetGroup.ctrls[targetSlot] = h2
				targetGroup.slots[targetSlot] = key
				g.ctrls[j] = slotEmpty
			} else {
				// SWAP: targetG.ctrls[targetSlot] is slotDeleted
				// 1. Move our current key to its new home
				// 2. Take the key that was there and put it in our current slot
				// 3. Keep our current slot marked as slotDeleted so it gets processed next
				g.slots[j] = targetGroup.slots[targetSlot]
				targetGroup.slots[targetSlot] = key
				targetGroup.ctrls[targetSlot] = h2
				j-- // Repeat for swapped key
			}
		}
	}

	return nil
}
