package stablemap

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStableSet_Basic(t *testing.T) {
	ss := NewSet[string](16)

	// Put and Has
	isNew, err := ss.Put("foo")
	require.NoError(t, err)
	assert.True(t, isNew)

	ok := ss.Has("foo")
	assert.True(t, ok)

	// Put existing key
	isNew, err = ss.Put("foo")
	require.NoError(t, err)
	assert.False(t, isNew)

	// Has non-existent key
	ok = ss.Has("bar")
	assert.False(t, ok)

	// Delete
	deleted := ss.Delete("foo")
	assert.True(t, deleted)

	ok = ss.Has("foo")
	assert.False(t, ok)

	// Delete non-existent key
	deleted = ss.Delete("foo")
	assert.False(t, deleted)
}

func TestStableSet_Stats(t *testing.T) {
	ss := NewSet[int](16)

	stats := ss.Stats()
	assert.Equal(t, 0, stats.Size)
	assert.Equal(t, 14, stats.EffectiveCapacity) // 16 * 7/8 = 14

	for i := range 5 {
		ss.Put(i)
	}

	stats = ss.Stats()
	assert.Equal(t, 5, stats.Size)
}

func TestStableSet_Compact(t *testing.T) {
	ss := NewSet[int](16)

	for i := range 10 {
		ss.Put(i)
	}

	for i := range 5 {
		ss.Delete(i)
	}

	stats := ss.Stats()
	assert.Equal(t, 5, stats.Tombstones)

	ss.Compact()

	stats = ss.Stats()
	assert.Equal(t, 0, stats.Tombstones)
	assert.Equal(t, 5, stats.Size)

	// Verify remaining keys
	for i := 5; i < 10; i++ {
		assert.True(t, ss.Has(i))
	}
}

func TestStableSet_Reset(t *testing.T) {
	ss := NewSet[int](16)

	for i := range 5 {
		ss.Put(i)
	}

	assert.Equal(t, 5, ss.Stats().Size)

	ss.Reset()

	assert.Equal(t, 0, ss.Stats().Size)
	assert.False(t, ss.Has(0))
}

func TestStableSet_ErrTableFull(t *testing.T) {
	ss := NewSet[int](8)
	capacity := ss.Stats().EffectiveCapacity

	for i := range capacity {
		_, err := ss.Put(i)
		require.NoError(t, err)
	}

	_, err := ss.Put(capacity + 1)
	assert.ErrorIs(t, err, ErrTableFull)
}

func TestStableSet_WithHashFunc(t *testing.T) {
	customHash := func(k int) uint64 {
		return uint64(k * 31)
	}

	ss := NewSet(16, WithHashFunc[int, struct{}](customHash))

	ss.Put(1)
	assert.True(t, ss.Has(1))
}
