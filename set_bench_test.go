package stableset

import (
	"testing"
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
	ss := New[uint64](capacity)
	for _, k := range keys {
		ss.Put(k)
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
	ss := New[uint64](capacity)

	for i := 0; b.Loop(); i++ {
		// Reset when nearly full to measure steady-state Put
		if ss.size >= ss.capacityEffective {
			b.StopTimer()
			ss.Reset()
			b.StartTimer()
		}
		ss.Put(keys[i%len(keys)])
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

func BenchmarkLargeScale_StableSet(b *testing.B) {
	const capacity = 4194304 // 2^22
	// Pre-generate keys to avoid hashing/gen time in the loop
	keys := make([]uint64, capacity/2)
	for i := range keys {
		keys[i] = uint64(i * 9876543210123) // High entropy distribution
	}

	ss := New[uint64](capacity)
	for _, k := range keys {
		ss.Put(k)
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
