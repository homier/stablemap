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

func BenchmarkStableMap_Get(b *testing.B) {
	const capacity = 8192
	keys := setupBenchData(capacity / 2)
	sm := New[uint64, uint64](capacity)
	for _, k := range keys {
		_ = sm.Set(k, k)
	}

	for i := 0; b.Loop(); i++ {
		sm.Get(uint64(i))
	}
}

func BenchmarkStdMap_Get(b *testing.B) {
	const capacity = 8192
	keys := setupBenchData(capacity / 2)
	m := make(map[uint64]uint64, capacity)
	for _, k := range keys {
		m[k] = k
	}

	for i := 0; b.Loop(); i++ {
		_ = m[uint64(i)]
	}
}

func BenchmarkStableMap_Set(b *testing.B) {
	const capacity = 8192
	keys := setupBenchData(capacity)
	sm := New[uint64, uint64](capacity)

	for i := 0; b.Loop(); i++ {
		// Reset when nearly full to measure steady-state Set
		if sm.size >= sm.capacityEffective {
			b.StopTimer()
			sm.Reset()
			b.StartTimer()
		}
		_ = sm.Set(keys[i%len(keys)], keys[i%len(keys)])
	}
}

func BenchmarkStdMap_Set(b *testing.B) {
	const capacity = 8192
	keys := setupBenchData(capacity)
	m := make(map[uint64]uint64, capacity)

	for i := 0; b.Loop(); i++ {
		if len(m) >= capacity*7/8 {
			b.StopTimer()
			for k := range m {
				delete(m, k)
			}
			b.StartTimer()
		}
		m[keys[i%len(keys)]] = keys[i%len(keys)]
	}
}

func BenchmarkStableMap_Delete(b *testing.B) {
	const size = 1000
	sm := New[int, int](size)
	for i := range size {
		_ = sm.Set(i, i)
	}

	for i := 0; b.Loop(); i++ {
		sm.Delete(i % size)
	}
}

func BenchmarkStdMap_Delete(b *testing.B) {
	const size = 1000
	m := make(map[int]int, size)
	for i := range size {
		m[i] = i
	}

	for i := 0; b.Loop(); i++ {
		delete(m, i%size)
	}
}

func BenchmarkLargeScale_StableMap_Delete(b *testing.B) {
	const capacity = 1 << 20
	sm := New[int, int](capacity)
	for i := range capacity / 2 {
		_ = sm.Set(i, i)
	}

	for i := 0; b.Loop(); i++ {
		sm.Delete(i % (capacity / 2))
	}
}

func BenchmarkLargeScale_StdMap_Delete(b *testing.B) {
	const capacity = 1 << 20
	m := make(map[int]int, capacity)
	for i := range capacity / 2 {
		m[i] = i
	}

	for i := 0; b.Loop(); i++ {
		delete(m, i%(capacity/2))
	}
}

func BenchmarkLargeScale_StableMap(b *testing.B) {
	const capacity = 4194304 // 2^22
	keys := make([]uint64, capacity/2)
	for i := range keys {
		keys[i] = uint64(i * 9876543210123)
	}

	sm := New[uint64, uint64](capacity)
	for _, k := range keys {
		_ = sm.Set(k, k)
	}

	for i := 0; b.Loop(); i++ {
		sm.Get(keys[(uintptr(i)*1337)%(capacity/2)])
	}
}

func BenchmarkLargeScale_StdMap(b *testing.B) {
	const capacity = 4194304
	keys := make([]uint64, capacity/2)
	for i := range keys {
		keys[i] = uint64(i * 9876543210123)
	}

	m := make(map[uint64]uint64, capacity)
	for _, k := range keys {
		m[k] = k
	}

	for i := 0; b.Loop(); i++ {
		_ = m[keys[(uintptr(i)*1337)%(capacity/2)]]
	}
}

func BenchmarkLargeScale_StableMap_HighLoad(b *testing.B) {
	const capacity = 4194304
	const loadFactor = 0.875
	fillCount := int(float64(capacity) * loadFactor)

	keys := make([]uint64, fillCount)
	for i := range keys {
		keys[i] = uint64(i * 9876543210123)
	}

	sm := New[uint64, uint64](capacity)
	for _, k := range keys {
		_ = sm.Set(k, k)
	}

	for i := 0; b.Loop(); i++ {
		sm.Get(keys[i%len(keys)])
	}
}

func BenchmarkLargeScale_StdMap_HighLoad(b *testing.B) {
	const capacity = 4194304
	const loadFactor = 0.875
	fillCount := int(float64(capacity) * loadFactor)

	keys := make([]uint64, fillCount)
	for i := range keys {
		keys[i] = uint64(i * 9876543210123)
	}

	m := make(map[uint64]uint64, capacity)
	for _, k := range keys {
		m[k] = k
	}

	for i := 0; b.Loop(); i++ {
		_ = m[keys[(uintptr(i)*1337)%(capacity/2)]]
	}
}

func BenchmarkMemoryUsage_StableMap(b *testing.B) {
	var m1, m2 runtime.MemStats

	g := group[uint64, uint64]{}
	for idx := range groupSize {
		g.ctrls[idx] = 0x7b
		g.slots[idx] = uint64(idx)
	}

	b.Logf("size of group: %v B\n", unsafe.Sizeof(g))

	runtime.GC()
	runtime.ReadMemStats(&m1)

	sm := New[uint64, uint64](16777216)
	_ = sm

	runtime.ReadMemStats(&m2)
	b.Logf("Actual Memory: %v MB\n", (m2.Alloc-m1.Alloc)/1024/1024)
}

func BenchmarkMemoryUsage_StdMap(b *testing.B) {
	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	m := make(map[uint64]uint64, 16777216)
	_ = m

	runtime.ReadMemStats(&m2)
	b.Logf("Actual Memory: %v MB\n", (m2.Alloc-m1.Alloc)/1024/1024)
}
