package stablemap

import (
	"strconv"
	"testing"
)

var sizes = []int{
	// 6,
	// 8192,
	// 1 << 16,
	// 1 << 20,
	1 << 22,
	1 << 26,
}

func BenchmarkSetHas_Miss(b *testing.B) {
	b.Run("variant=stdSet", func(b *testing.B) {
		// b.Run("K=string", benchSimulateLoadSet(benchmarkStdSetHasMiss[string], genKeys[string]))
		// b.Run("K=uint32", benchSimulateLoadSet(benchmarkStdSetHasMiss[uint32], genKeys[uint32]))
		b.Run("K=uint64", benchSimulateLoadSet(benchmarkStdSetHasMiss[uint64], genKeys[uint64]))
	})

	b.Run("variant=stableSet", func(b *testing.B) {
		// b.Run("K=string", benchSimulateLoadSet(benchmarkStableSetHasMiss[string], genKeys[string]))
		// b.Run("K=uint32", benchSimulateLoadSet(benchmarkStableSetHasMiss[uint32], genKeys[uint32]))
		b.Run("K=uint64", benchSimulateLoadSet(benchmarkStableSetHasMiss[uint64], genKeys[uint64]))
	})
}

func BenchmarkSetHas_Hit(b *testing.B) {
	b.Run("variant=stdSet", func(b *testing.B) {
		// b.Run("K=string", benchSimulateLoadSet(benchmarkStdSetHasHit[string], genKeys[string]))
		// b.Run("K=uint32", benchSimulateLoadSet(benchmarkStdSetHasHit[uint32], genKeys[uint32]))
		b.Run("K=uint64", benchSimulateLoadSet(benchmarkStdSetHasHit[uint64], genKeys[uint64]))
	})

	b.Run("variant=stableSet", func(b *testing.B) {
		// b.Run("K=string", benchSimulateLoadSet(benchmarkStableSetHasHit[string], genKeys[string]))
		// b.Run("K=uint32", benchSimulateLoadSet(benchmarkStableSetHasHit[uint32], genKeys[uint32]))
		b.Run("K=uint64", benchSimulateLoadSet(benchmarkStableSetHasHit[uint64], genKeys[uint64]))
	})
}

// TODO: Need to fix bench
//
//	func BenchmarkSetPut_Miss(b *testing.B) {
//		b.Run("variant=stdSet", func(b *testing.B) {
//			b.Run("K=string", benchSimulateLoadSet(benchmarkStdSetPutMiss[string], genKeys[string]))
//			b.Run("K=uint32", benchSimulateLoadSet(benchmarkStdSetPutMiss[uint32], genKeys[uint32]))
//			b.Run("K=uint64", benchSimulateLoadSet(benchmarkStdSetPutMiss[uint64], genKeys[uint64]))
//		})
//
//		b.Run("variant=stableSet", func(b *testing.B) {
//			b.Run("K=string", benchSimulateLoadSet(benchmarkStableSetPutMiss[string], genKeys[string]))
//			b.Run("K=uint32", benchSimulateLoadSet(benchmarkStableSetPutMiss[uint32], genKeys[uint32]))
//			b.Run("K=uint64", benchSimulateLoadSet(benchmarkStableSetPutMiss[uint64], genKeys[uint64]))
//		})
//	}

// func BenchmarkSetPut_Hit(b *testing.B) {
// 	b.Run("variant=stdSet", func(b *testing.B) {
// 		b.Run("K=string", benchSimulateLoadSet(benchmarkStdSetPutHit[string], genKeys[string]))
// 		b.Run("K=uint32", benchSimulateLoadSet(benchmarkStdSetPutHit[uint32], genKeys[uint32]))
// 		b.Run("K=uint64", benchSimulateLoadSet(benchmarkStdSetPutHit[uint64], genKeys[uint64]))
// 	})
//
// 	b.Run("variant=stableSet", func(b *testing.B) {
// 		b.Run("K=string", benchSimulateLoadSet(benchmarkStableSetPutHit[string], genKeys[string]))
// 		b.Run("K=uint32", benchSimulateLoadSet(benchmarkStableSetPutHit[uint32], genKeys[uint32]))
// 		b.Run("K=uint64", benchSimulateLoadSet(benchmarkStableSetPutHit[uint64], genKeys[uint64]))
// 	})
// }

//	func BenchmarkSetDelete_Miss(b *testing.B) {
//		b.Run("variant=stdSet", func(b *testing.B) {
//			b.Run("K=string", benchSimulateLoadSet(benchmarkStdSetDeleteMiss[string], genKeys[string]))
//			b.Run("K=uint32", benchSimulateLoadSet(benchmarkStdSetDeleteMiss[uint32], genKeys[uint32]))
//			b.Run("K=uint64", benchSimulateLoadSet(benchmarkStdSetDeleteMiss[uint64], genKeys[uint64]))
//		})
//
//		b.Run("variant=stableSet", func(b *testing.B) {
//			b.Run("K=string", benchSimulateLoadSet(benchmarkStableSetDeleteMiss[string], genKeys[string]))
//			b.Run("K=uint32", benchSimulateLoadSet(benchmarkStableSetDeleteMiss[uint32], genKeys[uint32]))
//			b.Run("K=uint64", benchSimulateLoadSet(benchmarkStableSetDeleteMiss[uint64], genKeys[uint64]))
//		})
//	}
func benchmarkStdSetHasMiss[K comparable](
	b *testing.B,
	capacity int,
	genKeys func(start, end int) []K,
) {
	m := make(map[K]struct{}, capacity)
	keys := genKeys(0, capacity*8/7)
	misses := genKeys(-capacity, 0)

	for _, k := range keys {
		m[k] = struct{}{}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m[misses[i%len(misses)]]
	}
}

func benchmarkStableSetHasMiss[K comparable](
	b *testing.B,
	capacity int,
	genKeys func(start, end int) []K,
) {
	ss := NewSet[K](capacity)
	keys := genKeys(0, capacity*8/7)
	misses := genKeys(-capacity, 0)

	for _, k := range keys {
		ss.Put(k)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ss.Has(misses[i%len(misses)])
	}
}

func benchmarkStdSetHasHit[K comparable](
	b *testing.B,
	capacity int,
	genKeys func(start, end int) []K,
) {
	m := make(map[K]struct{}, capacity)
	keys := genKeys(0, capacity*8/7)
	for _, k := range keys {
		m[k] = struct{}{}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m[keys[i%len(keys)]]
	}
}

func benchmarkStableSetHasHit[K comparable](
	b *testing.B,
	capacity int,
	genKeys func(start, end int) []K,
) {
	ss := NewSet[K](capacity)
	keys := genKeys(0, capacity*8/7)

	for _, k := range keys {
		ss.Put(k)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ss.Has(keys[i%len(keys)])
	}
}

func benchmarkStdSetPutMiss[K comparable](
	b *testing.B,
	capacity int,
	genKeys func(start, end int) []K,
) {
	keys := genKeys(0, capacity*8/7)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StartTimer()
		m := make(map[K]struct{}, capacity)
		b.StopTimer()

		for _, key := range keys {
			m[key] = struct{}{}
		}
	}
}

func benchmarkStableSetPutMiss[K comparable](
	b *testing.B,
	capacity int,
	genKeys func(start, end int) []K,
) {
	keys := genKeys(0, capacity*8/7)
	s := NewSet[K](capacity)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StartTimer()
		s.Reset()
		b.StopTimer()

		for _, key := range keys {
			_, _ = s.Put(key)
		}
	}
}

func benchmarkStdSetPutHit[K comparable](
	b *testing.B,
	capacity int,
	genKeys func(start, end int) []K,
) {
	keys := genKeys(0, capacity*8/7)
	m := make(map[K]struct{}, capacity)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m[keys[i%len(keys)]] = struct{}{}
	}
}

func benchmarkStableSetPutHit[K comparable](
	b *testing.B,
	capacity int,
	genKeys func(start, end int) []K,
) {
	keys := genKeys(0, capacity*8/7-1)
	s := NewSet[K](capacity)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = s.Put(keys[i%len(keys)])
	}
}

func benchmarkStdSetDeleteMiss[K comparable](
	b *testing.B,
	capacity int,
	genKeys func(start, end int) []K,
) {
	keys := genKeys(0, capacity*8/7)
	m := make(map[K]struct{}, capacity)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		delete(m, keys[i%len(keys)])
	}
}

func benchmarkStableSetDeleteMiss[K comparable](
	b *testing.B,
	capacity int,
	genKeys func(start, end int) []K,
) {
	keys := genKeys(0, capacity*8/7)
	s := NewSet[K](capacity)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.Delete(keys[i%len(keys)])
	}
}

func genKeys[K comparable](start, end int) []K {
	var k K
	switch any(k).(type) {
	case uint32:
		keys := make([]uint32, end-start)
		for i := range keys {
			keys[i] = uint32(start + i)
		}
		return unsafeConvertSlice[K](keys)
	case uint64:
		keys := make([]uint64, end-start)
		for i := range keys {
			keys[i] = uint64(start + i)
		}
		return unsafeConvertSlice[K](keys)
	case string:
		keys := make([]string, end-start)
		for i := range keys {
			keys[i] = strconv.Itoa(start + i)
		}
		return unsafeConvertSlice[K](keys)
	default:
		panic("not reached")
	}
}

func benchSimulateLoadSet[K comparable](
	benchFunc func(b *testing.B, capacity int, keysFunc func(start, end int) []K),
	keysFunc func(start, end int) []K,
) func(b *testing.B) {
	return func(b *testing.B) {
		for _, size := range sizes {
			b.Run("capacity="+strconv.Itoa(size), func(b *testing.B) {
				benchFunc(b, size, keysFunc)
			})
		}
	}
}
