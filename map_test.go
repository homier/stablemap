package stablemap

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStableMap_Basic(t *testing.T) {
	sm := New[string, int](16)

	// Set and Get
	err := sm.Set("foo", 42)
	require.NoError(t, err)

	v, ok := sm.Get("foo")
	require.True(t, ok)
	assert.Equal(t, 42, v)

	// Update existing key
	err = sm.Set("foo", 100)
	require.NoError(t, err)

	v, ok = sm.Get("foo")
	require.True(t, ok)
	assert.Equal(t, 100, v)

	// Get non-existent key
	_, ok = sm.Get("bar")
	assert.False(t, ok)

	// Delete
	deleted := sm.Delete("foo")
	assert.True(t, deleted)

	_, ok = sm.Get("foo")
	assert.False(t, ok)

	// Delete non-existent key
	deleted = sm.Delete("foo")
	assert.False(t, deleted)
}

func TestStableMap_Stats(t *testing.T) {
	sm := New[int, int](16)

	stats := sm.Stats()
	assert.Equal(t, 0, stats.Size)
	assert.Equal(t, 14, stats.EffectiveCapacity) // 16 * 7/8 = 14

	for i := range 5 {
		_ = sm.Set(i, i)
	}

	stats = sm.Stats()
	assert.Equal(t, 5, stats.Size)
}

func TestStableMap_AutoCompaction(t *testing.T) {
	sm := New[int, int](32)
	effectiveCapacity := sm.Stats().EffectiveCapacity
	threshold := effectiveCapacity / 3

	// Fill the table
	for i := range effectiveCapacity {
		_ = sm.Set(i, i*10)
	}

	// Delete up to the compaction threshold — compaction triggers automatically
	for i := range threshold {
		sm.Delete(i)
	}

	stats := sm.Stats()
	assert.Equal(t, 0, stats.Tombstones, "tombstones should be cleared after auto-compaction")
	assert.Equal(t, effectiveCapacity-threshold, stats.Size)

	// Verify remaining values are intact
	for i := threshold; i < effectiveCapacity; i++ {
		v, ok := sm.Get(i)
		require.True(t, ok)
		assert.Equal(t, i*10, v)
	}
}

func TestStableMap_Reset(t *testing.T) {
	sm := New[int, int](16)

	for i := range 5 {
		_ = sm.Set(i, i)
	}

	assert.Equal(t, 5, sm.Stats().Size)

	sm.Reset()

	assert.Equal(t, 0, sm.Stats().Size)

	_, ok := sm.Get(0)
	assert.False(t, ok)
}

func TestStableMap_ErrTableFull(t *testing.T) {
	sm := New[int, int](8)
	capacity := sm.Stats().EffectiveCapacity

	for i := range capacity {
		err := sm.Set(i, i)
		require.NoError(t, err)
	}

	err := sm.Set(capacity+1, 999)
	assert.ErrorIs(t, err, ErrTableFull)
}

func TestStableMap_WithHashFunc(t *testing.T) {
	customHash := func(k int) uint64 {
		return uint64(k * 31)
	}

	sm := New(16, WithHashFunc[int, int](customHash))

	_ = sm.Set(1, 100)
	v, ok := sm.Get(1)
	require.True(t, ok)
	assert.Equal(t, 100, v)
}
