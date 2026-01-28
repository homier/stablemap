package stablemap

import (
	"runtime"
	"testing"
	"unsafe"
)

// Generate some data for testing
func setupBenchData(n int) []uint64 {
	data := make([]uint64, n)
	for i := range n {
		data[i] = uint64(i * 1234567) // Distributed keys
	}
	return data
}

func BenchmarkStableSet_Contains(b *testing.B) {
	const capacity = 8192
	keys := setupBenchData(capacity / 2)
	ss := NewSet[uint64](capacity)
	for _, k := range keys {
		_, _ = ss.Put(k)
	}

	for i := 0; b.Loop(); i++ {
		// We use bitwise AND to stay within the slice range
		// and test both hits and misses
		ss.Has(uint64(i))
	}
}

func BenchmarkStdMap_Contains(b *testing.B) {
	const capacity = 8192
	keys := setupBenchData(capacity / 2)
	m := make(map[uint64]struct{}, capacity)
	for _, k := range keys {
		m[k] = struct{}{}
	}

	for i := 0; b.Loop(); i++ {
		_ = m[uint64(i)]
	}
}

func BenchmarkStableSet_Put(b *testing.B) {
	const capacity = 8192
	keys := setupBenchData(capacity)
	ss := NewSet[uint64](capacity)

	for i := 0; b.Loop(); i++ {
		// Reset when nearly full to measure steady-state Put
		if ss.size >= ss.capacityEffective {
			b.StopTimer()
			ss.Reset()
			b.StartTimer()
		}
		_, _ = ss.Put(keys[i%len(keys)])
	}
}

func BenchmarkStdMap_Put(b *testing.B) {
	const capacity = 8192
	keys := setupBenchData(capacity)
	// We initialize with capacity to prevent resizing during the benchmark
	m := make(map[uint64]struct{}, capacity)

	for i := 0; b.Loop(); i++ {
		if len(m) >= capacity*7/8 {
			b.StopTimer()
			// Clearing a map is O(N). We do this to stay in a steady state.
			for k := range m {
				delete(m, k)
			}
			b.StartTimer()
		}
		m[keys[i%len(keys)]] = struct{}{}
	}
}

func BenchmarkStableSet_Delete(b *testing.B) {
	const size = 1000
	ss := NewSet[int](size)
	for i := range size {
		_, _ = ss.Put(i)
	}

	for i := 0; b.Loop(); i++ {
		ss.Delete(i % size)
	}
}

func BenchmarkStdMap_Delete(b *testing.B) {
	const size = 1000
	m := make(map[int]struct{}, size)
	for i := range size {
		m[i] = struct{}{}
	}

	for i := 0; b.Loop(); i++ {
		delete(m, i%size)
	}
}

func BenchmarkLargeScale_StableSet_Delete(b *testing.B) {
	const capacity = 1 << 20
	ss := NewSet[int](capacity)
	for i := range capacity / 2 {
		_, _ = ss.Put(i)
	}

	for i := 0; b.Loop(); i++ {
		ss.Delete(i % (capacity / 2))
	}
}

func BenchmarkLargeScale_StdMap_Delete(b *testing.B) {
	const capacity = 1 << 20
	m := make(map[int]struct{}, capacity)
	for i := range capacity / 2 {
		m[i] = struct{}{}
	}

	for i := 0; b.Loop(); i++ {
		delete(m, i%(capacity/2))
	}
}

func BenchmarkLargeScale_StableSet(b *testing.B) {
	const capacity = 4194304 // 2^22
	// Pre-generate keys to avoid hashing/gen time in the loop
	keys := make([]uint64, capacity/2)
	for i := range keys {
		keys[i] = uint64(i * 9876543210123) // High entropy distribution
	}

	ss := NewSet[uint64](capacity)
	for _, k := range keys {
		_, _ = ss.Put(k)
	}

	for i := 0; b.Loop(); i++ {
		// Use a large prime to jump around the set and force cache misses
		_ = ss.Has(keys[(uintptr(i)*1337)%(capacity/2)])
	}
}

func BenchmarkLargeScale_StdMap(b *testing.B) {
	const capacity = 4194304
	keys := make([]uint64, capacity/2)
	for i := range keys {
		keys[i] = uint64(i * 9876543210123)
	}

	m := make(map[uint64]struct{}, capacity)
	for _, k := range keys {
		m[k] = struct{}{}
	}

	for i := 0; b.Loop(); i++ {
		_ = m[keys[(uintptr(i)*1337)%(capacity/2)]]
	}
}

func BenchmarkLargeScale_StableSet_HighLoad(b *testing.B) {
	const capacity = 4194304
	// 0.875 is 7/8 load—the theoretical limit for many Swiss Tables
	const loadFactor = 0.875
	fillCount := int(float64(capacity) * loadFactor)

	keys := make([]uint64, fillCount)
	for i := range keys {
		keys[i] = uint64(i * 9876543210123)
	}

	ss := NewSet[uint64](capacity)
	for _, k := range keys {
		_, _ = ss.Put(k)
	}

	for i := 0; b.Loop(); i++ {
		// Use a subset of the keys to ensure we are hitting existing values
		_ = ss.Has(keys[i%len(keys)])
	}
}

func BenchmarkLargeScale_StdMap_HighLoad(b *testing.B) {
	const capacity = 4194304
	// 0.875 is 7/8 load—the theoretical limit for many Swiss Tables
	const loadFactor = 0.875
	fillCount := int(float64(capacity) * loadFactor)

	keys := make([]uint64, fillCount)
	for i := range keys {
		keys[i] = uint64(i * 9876543210123)
	}

	m := make(map[uint64]struct{}, capacity)
	for _, k := range keys {
		m[k] = struct{}{}
	}

	for i := 0; b.Loop(); i++ {
		_ = m[keys[(uintptr(i)*1337)%(capacity/2)]]
	}
}

func BenchmarkMemoryUsage_StableSet(b *testing.B) {
	var m1, m2 runtime.MemStats

	g := group[uint64, struct{}]{}
	for idx := range groupSize {
		g.ctrls[idx] = 0x7b
		g.slots[idx] = uint64(idx)
	}

	b.Logf("size of table: %v B\n", unsafe.Sizeof(g))

	runtime.GC()
	runtime.ReadMemStats(&m1)

	ss := NewSet[uint64](16777216)
	_ = ss

	runtime.ReadMemStats(&m2)
	b.Logf("Actual Memory: %v MB\n", (m2.Alloc-m1.Alloc)/1024/1024)
}

func BenchmarkMemoryUsage_StdMap(b *testing.B) {
	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	ss := make(map[uint64]struct{}, 16777216)
	_ = ss

	runtime.ReadMemStats(&m2)
	b.Logf("Actual Memory: %v MB\n", (m2.Alloc-m1.Alloc)/1024/1024)
}
