package stablemap

import (
	"math/rand"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTable[K comparable, V any](capacity int, opts ...Option[K, V]) *table[K, V] {
	var tt table[K, V]
	tt.init(capacity, opts...)

	return &tt
}

func TestTable_init(t *testing.T) {
	var tt table[uint64, struct{}]

	tt.init(4096)

	require.Len(t, tt.groups, 4096/groupSize)
	require.Equal(t, uintptr((4096/groupSize)-1), tt.numGroupsMask)
}

func TestTable_EffectiveCapacity(t *testing.T) {
	tt := newTable[uint64, struct{}](4096)

	require.Equal(t, 4096*7/8, tt.EffectiveCapacity())
}

func TestTable_put(t *testing.T) {
	tt := newTable[string, string](4096)

	ok, rehash := tt.put("foo", "bar")
	require.True(t, ok)
	assert.False(t, rehash)

	ok, rehash = tt.put("foo", "bar2")
	require.False(t, ok)
	assert.False(t, rehash)
}

func TestTable_put_Fill(t *testing.T) {
	tt := newTable[uint64, uint64](4096)

	for i := range uint64(tt.EffectiveCapacity()) {
		ok, rehash := tt.put(i, i)
		require.True(t, ok)
		require.False(t, rehash)
	}

	ok, rehash := tt.put(uint64(tt.EffectiveCapacity())+1, uint64(tt.EffectiveCapacity())+1)
	require.False(t, ok)
	require.True(t, rehash)
}

func TestTable_put_Tombstones(t *testing.T) {
	// Use a custom hash function that forces collisions
	// by returning the same h1 for everything.
	collisionHash := func(k string) uint64 {
		return 0 // All keys start at index 0
	}

	tt := newTable(16, WithHashFunc[string, string](collisionHash))

	ok, r := tt.put("A", "foo") // Slot 0
	require.True(t, ok)
	require.False(t, r)

	ok, r = tt.put("B", "bar") // Slot 1 (via probe)
	require.True(t, ok)
	require.False(t, r)

	ok, r = tt.put("C", "lol") // Slot 2 (via probe)
	require.True(t, ok)
	require.False(t, r)

	// Delete the "bridge" element
	require.True(t, tt.delete("B"))

	// Verify we can still find "C" even though there's a hole at "B"
	v, ok := tt.get("C")
	require.True(t, ok, "Probe chain broken: could not find 'C' after deleting 'B'")
	require.Equal(t, "lol", v)
}

func TestTable_set(t *testing.T) {
	tt := newTable[string, string](16)

	compact := tt.set("foo", "foo")
	assert.False(t, compact)

	v, ok := tt.get("foo")
	require.True(t, ok)
	require.Equal(t, "foo", v)

	compact = tt.set("foo", "bar")
	assert.False(t, compact)

	v, ok = tt.get("foo")
	require.True(t, ok)
	require.Equal(t, "bar", v)
}

func TestTable_Compact(t *testing.T) {
	const capacity = 32
	tt := newTable[int, int](capacity)

	// 1. Fill it up to the effective capacity
	for i := 0; i < tt.EffectiveCapacity(); i++ {
		ok, rehash := tt.put(i, i)
		require.True(t, ok)
		require.False(t, rehash)
	}

	// 2. Delete almost everything to create many tombstones
	for i := 0; i < tt.EffectiveCapacity()-1; i++ {
		require.True(t, tt.delete(i))
	}

	// 3. Compact
	tt.Compact()

	// 4. Verify the one remaining element
	lastIdx := tt.EffectiveCapacity() - 1
	v, ok := tt.get(lastIdx)
	require.Truef(t, ok, "Lost key %d after compaction: %b", lastIdx)
	require.Equal(t, lastIdx, v)

	// 5. Verify no tombstones (0xFE) remain in the ctrls
	for i := range tt.groups {
		for j := range groupSize {
			require.NotEqualf(t, slotDeleted, tt.groups[i].ctrls[j], "Found tombstone at index %d after rehash", i)
		}
	}
}

func TestTable_Compact_Sync(t *testing.T) {
	tt := newTable[int, int](16)

	// 1. Fill it up to trigger many tombstones
	for i := range 10 {
		ok, compact := tt.put(i, i*100)
		require.True(t, ok)
		require.False(t, compact)
	}

	keys := make([]int, 0, 5)

	// 2. Delete half to create holes (tombstones)
	for i := 0; len(keys) < 5; i++ {
		idx := rand.Intn(10)

		if tt.delete(idx) {
			keys = append(keys, idx)
		}
	}

	// 3. Compact in-place
	tt.Compact()

	// 4. Verify remaining keys still have their correct values
	for idx := range 10 {
		if slices.Contains(keys, idx) {
			continue
		}

		val, ok := tt.get(idx)
		require.True(t, ok)
		require.Equal(t, idx*100, val)
	}

	// 5. Verify deleted keys are not present
	for _, key := range keys {
		_, ok := tt.get(key)

		require.False(t, ok)
	}
}

func TestTable_put_BoundaryMirror(t *testing.T) {
	// 16 slots / 8 per group = 2 groups
	tt := newTable[int, int](16)

	// The last valid group index is ss.numGroupsMask (which is 1)
	targetGroupIdx := tt.numGroupsMask

	lastIdxKey := 0
	for {
		h1, _ := HashSplit(tt.hashFunc(lastIdxKey))
		// h1/8 gives the group index. Mask it to find keys landing in the last group.
		if (h1 / 8 & tt.numGroupsMask) == targetGroupIdx {
			break
		}
		lastIdxKey++
	}

	ok, r := tt.put(lastIdxKey, lastIdxKey)
	require.True(t, ok)
	require.False(t, r)

	v, ok := tt.get(lastIdxKey)
	require.True(t, ok, "Failed to find key at the boundary of the capacity")
	require.Equal(t, lastIdxKey, v)
}
