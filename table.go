package stableset

import (
	"hash/maphash"
	"unsafe"
)

const (
	groupSize = 8

	slotEmpty   = 0x80
	slotDeleted = 0xFE

	bitsetLSB = 0x0101010101010101
	bitsetMSB = 0x8080808080808080
)

type group[K comparable, V any] struct {
	// 8 bytes of metadata (h2 or control states)
	// This fits perfectly in a single uint64 load
	ctrls [groupSize]uint8

	// 8 keys stored immediately after the metadata
	// In a 64-bit system, this group is (8 + 8*8) = 72 bytes.
	// That's just slightly over one 64-byte cache line.
	slots [groupSize]K

	// 8 values stored after the keys.
	// If V is a struct{} type, Go compiler will optimize it to 0 bytes.
	// It's very sensible, you need to be careful with the value type.
	// If it's too large, you'll end up missing CPU cache lines.
	// If it's too low, Go will add padding and you'll end up wasting
	// memory for nothing.
	values [groupSize]V
}

var (
	emptyCtrls = [groupSize]uint8{
		slotEmpty,
		slotEmpty,
		slotEmpty,
		slotEmpty,

		slotEmpty,
		slotEmpty,
		slotEmpty,
		slotEmpty,
	}
)

type table[K comparable, V any] struct {
	groups []group[K, V]

	capacity          uintptr
	numGroupsMask     uintptr
	capacityEffective uintptr
	size              uintptr

	hashFunc HashFunc[K]

	emptyV V
}

type Option[K comparable, V any] func(t *table[K, V])

// Override default hash function.
func WithHashFunc[K comparable, V any](f HashFunc[K]) Option[K, V] {
	return func(t *table[K, V]) {
		t.hashFunc = f
	}
}

func (t *table[K, V]) init(capacity int, opts ...Option[K, V]) {
	normalizedCapacity := uintptr(NextPowerOf2(uint32(capacity)))
	// Number of groups required
	numGroups := normalizedCapacity / groupSize
	numGroupsMask := uintptr(numGroups - 1)

	t.groups = make([]group[K, V], numGroups)
	t.capacity = normalizedCapacity
	t.numGroupsMask = numGroupsMask
	t.capacityEffective = normalizedCapacity * 7 / 8

	// Initialize all control bytes to Empty
	t.Reset()

	for _, opt := range opts {
		opt(t)
	}

	if t.hashFunc == nil {
		t.hashFunc = MakeDefaultHashFunc[K](maphash.MakeSeed())
	}
}

func (t *table[K, V]) EffectiveCapacity() int {
	return int(t.capacityEffective)
}

func (t *table[K, V]) get(key K) (V, bool) {
	h1, h2 := HashSplit(t.hashFunc(key))
	mask := t.numGroupsMask
	start := (h1 / groupSize) & mask

	for p, offset := uintptr(0), start; p <= mask; p++ {
		g := &t.groups[offset]
		ctrl := *(*uint64)(unsafe.Pointer(&g.ctrls))

		// SIMD-like match
		matches := matchH2(ctrl, h2)
		for matches != 0 {
			idx := matches.first()
			if g.slots[idx] == key {
				return g.values[idx], true
			}

			matches = matches.removeFirst()
		}

		// Termination
		if matchEmpty(ctrl) != 0 {
			return t.emptyV, false
		}

		// Quadratic probe math
		offset = (start + (p+1)*(p+2)/2) & mask
	}

	return t.emptyV, false
}

func (t *table[K, V]) put(key K, value V) (bool, bool) {
	// We reached the 87.5% of the capacity, table needs rehashing.
	if t.size >= t.capacityEffective {
		return false, true
	}

	var (
		h1, h2 = HashSplit(t.hashFunc(key))
		mask   = t.numGroupsMask
		start  = (h1 / groupSize) & mask

		targetGroup *group[K, V]
		targetSlot  uintptr
		foundSlot   bool
	)

	for p, offset := uintptr(0), start; p <= mask; p++ {
		g := &t.groups[offset]
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
				targetGroup.values[targetSlot] = value
				t.size++

				return true, false
			}

			return false, true
		}

		offset = (start + (p+1)*(p+2)/2) & mask
	}

	return false, true
}

func (t *table[K, V]) set(key K, value V) bool {
	// We reached the 87.5% of the capacity, table needs rehashing.
	if t.size >= t.capacityEffective {
		return true
	}

	var (
		h1, h2 = HashSplit(t.hashFunc(key))
		mask   = t.numGroupsMask
		start  = (h1 / groupSize) & mask

		targetGroup *group[K, V]
		targetSlot  uintptr
		foundSlot   bool
	)

	for p, offset := uintptr(0), start; p <= mask; p++ {
		g := &t.groups[offset]
		ctrl := *(*uint64)(unsafe.Pointer(&g.ctrls))

		// 1. Existing check
		matchMask := matchH2(ctrl, h2)
		for matchMask != 0 {
			idx := matchMask.first()
			if g.slots[idx] == key {
				g.values[idx] = value
				return false
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
				targetGroup.values[targetSlot] = value
				t.size++

				return false
			}

			return true
		}

		offset = (start + (p+1)*(p+2)/2) & mask
	}

	return true

}

func (t *table[K, V]) delete(key K) bool {
	h1, h2 := HashSplit(t.hashFunc(key))
	mask := t.numGroupsMask
	start := (h1 / groupSize) & mask

	for p, offset := uintptr(0), start; p <= mask; p++ {
		g := &t.groups[offset]
		ctrl := *(*uint64)(unsafe.Pointer(&g.ctrls))

		// 1. Check current group for the key
		matchMask := matchH2(ctrl, h2)
		for matchMask != 0 {
			idx := matchMask.first()
			if g.slots[idx] == key {
				// Mark as Deleted (0xFE) to preserve the probe chain
				g.ctrls[idx] = slotDeleted
				t.size--

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

func (t *table[K, V]) Reset() {
	for i := range t.groups {
		copy(t.groups[i].ctrls[:], emptyCtrls[:])
	}

	t.size = 0
}

func (t *table[K, V]) Compact() error {
	// We want to drop all of the deletes in place. We first walk over the
	// control bytes and mark every DELETED slot as EMPTY and every FULL slot
	// as DELETED. Marking the DELETED slots as EMPTY has effectively dropped
	// the tombstones, but we fouled up the probe invariant. Marking the FULL
	// slots as DELETED gives us a marker to locate the previously FULL slots.

	// Mark all DELETED slots as EMPTY and all FULL slots as DELETED.
	for i := range t.groups {
		g := &t.groups[i]
		for j := range groupSize {
			c := g.ctrls[j]
			if c < 0x80 {
				g.ctrls[j] = slotDeleted
			} else if c == slotDeleted {
				g.ctrls[j] = slotEmpty
			}
		}
	}

	for idx := 0; idx < len(t.groups); idx++ {
		g := &t.groups[idx]
		for j := uintptr(0); j < groupSize; j++ {
			// Only process slots we marked as Deleted (which were originally Full)
			if g.ctrls[j] != slotDeleted {
				continue
			}

			var (
				key          = g.slots[j]
				value        = g.values[j]
				h            = t.hashFunc(key)
				h1, h2       = HashSplit(h)
				destGroupIdx = (h1 / groupSize) & t.numGroupsMask

				targetGroup *group[K, V]
				targetSlot  uintptr

				p        = uintptr(0)
				currGIdx = destGroupIdx
			)

			for {
				tg := &t.groups[currGIdx]
				tc := *(*uint64)(unsafe.Pointer(&tg.ctrls))
				m := matchEmptyOrDeleted(tc)
				if m != 0 {
					targetGroup = tg
					targetSlot = m.first()
					break
				}
				p++
				currGIdx = (currGIdx + p) & t.numGroupsMask
			}

			// Swap / Move logic
			// Swapping within the same group
			if targetGroup == g && targetSlot == j {
				g.ctrls[j] = h2
			} else if targetGroup.ctrls[targetSlot] == slotEmpty {
				// Target group slot is empty
				targetGroup.ctrls[targetSlot] = h2
				targetGroup.slots[targetSlot] = key
				targetGroup.values[targetSlot] = value
				g.ctrls[j] = slotEmpty
			} else {
				// SWAP: targetG.ctrls[targetSlot] is slotDeleted
				targetGroup.ctrls[targetSlot] = h2

				// SWAP: Swapping values and keys as well
				g.slots[j], targetGroup.slots[targetSlot] = targetGroup.slots[targetSlot], g.slots[j]
				g.values[j], targetGroup.values[targetSlot] = targetGroup.values[targetSlot], g.values[j]

				// Repeat for swapped key
				j--
			}
		}
	}

	return nil
}
