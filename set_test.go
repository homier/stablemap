package stableset

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_New(t *testing.T) {
	ss := New[uint64](4096)

	require.Len(t, ss.groups, 4096/groupSize)
	require.Equal(t, uintptr((4096/groupSize)-1), ss.numGroupsMask)
}

func TestStableSet_EffectiveCapacity(t *testing.T) {
	ss := New[uint64](4096)

	require.Equal(t, 4096*7/8, ss.EffectiveCapacity())
}

func Test_Put(t *testing.T) {
	ss := New[uint64](4096)

	ok, rehash := ss.Put(1)
	require.True(t, ok)
	assert.False(t, rehash)

	ok, rehash = ss.Put(1)
	require.False(t, ok)
	assert.False(t, rehash)
}

func Test_Put_Fill(t *testing.T) {
	ss := New[uint64](4096)

	for i := range uint64(ss.EffectiveCapacity()) {
		ok, rehash := ss.Put(i)
		require.True(t, ok)
		require.False(t, rehash)
	}

	ok, rehash := ss.Put(uint64(ss.EffectiveCapacity()) + 1)
	require.False(t, ok)
	require.True(t, rehash)
}

func TestStableSet_Tombstones(t *testing.T) {
	// Use a custom hash function that forces collisions
	// by returning the same h1 for everything.
	collisionHash := func(k string) uint64 {
		return 0 // All keys start at index 0
	}

	ss := New(16, WithHashFunc(collisionHash))

	ok, r := ss.Put("A") // Slot 0
	require.True(t, ok)
	require.False(t, r)

	ok, r = ss.Put("B") // Slot 1 (via probe)
	require.True(t, ok)
	require.False(t, r)

	ok, r = ss.Put("C") // Slot 2 (via probe)
	require.True(t, ok)
	require.False(t, r)

	// Delete the "bridge" element
	require.True(t, ss.Delete("B"))

	// Verify we can still find "C" even though there's a hole at "B"
	require.True(t, ss.Has("C"), "Probe chain broken: could not find 'C' after deleting 'B'")
}

func TestStableSet_RehashInPlace(t *testing.T) {
	const cap = 32
	ss := New[int](cap)

	// 1. Fill it up to the effective capacity
	for i := 0; i < ss.EffectiveCapacity(); i++ {
		ss.Put(i)
	}

	// 2. Delete almost everything to create many tombstones
	for i := 0; i < ss.EffectiveCapacity()-1; i++ {
		ss.Delete(i)
	}

	// 3. Rehash
	require.NoError(t, ss.Rehash())

	// 4. Verify the one remaining element
	lastIdx := ss.EffectiveCapacity() - 1
	require.Truef(t, ss.Has(lastIdx), "Lost key %d after rehash: %b", lastIdx)

	// 5. Verify no tombstones (0xFE) remain in the ctrls
	for i := range ss.groups {
		for j := range groupSize {
			require.NotEqualf(t, slotDeleted, ss.groups[i].ctrls[j], "Found tombstone at index %d after rehash", i)
		}
	}
}

func TestStableSet_BoundaryMirror(t *testing.T) {
	// 16 slots / 8 per group = 2 groups
	ss := New[int](16)

	// The last valid group index is ss.numGroupsMask (which is 1)
	targetGroupIdx := ss.numGroupsMask

	lastIdxKey := 0
	for {
		h1, _ := HashSplit(ss.hashFunc(lastIdxKey))
		// h1/8 gives the group index. Mask it to find keys landing in the last group.
		if (h1 / 8 & ss.numGroupsMask) == targetGroupIdx {
			break
		}
		lastIdxKey++
	}

	ok, r := ss.Put(lastIdxKey)
	require.True(t, ok)
	require.False(t, r)

	require.True(t, ss.Has(lastIdxKey), "Failed to find key at the boundary of the capacity")
}
