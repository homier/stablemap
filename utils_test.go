package stablemap

import (
	"testing"
	"unsafe"

	"github.com/stretchr/testify/require"
)

func TestCapacityFromSize(t *testing.T) {
	t.Run("int,int", func(t *testing.T) {
		sizeOfGroup := unsafe.Sizeof(group[int, int]{})

		tests := []struct {
			name string
			size uintptr
			want int
		}{
			{"zero", 0, 0},
			{"less than one group", sizeOfGroup - 1, 0},
			{"exactly one group", sizeOfGroup, 8},
			{"one and a half groups", sizeOfGroup + sizeOfGroup/2, 8},
			{"two groups", sizeOfGroup * 2, 16},
			{"ten groups", sizeOfGroup * 10, 80},
			{"1KB", 1024, int(1024/sizeOfGroup) * 8},
			{"1MB", 1024 * 1024, int(1024*1024/sizeOfGroup) * 8},
			{"1GB", 1024 * 1024 * 1024, int(1024*1024*1024/sizeOfGroup) * 8},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := CapacityFromSize[int, int](tt.size)
				require.Equal(t, tt.want, got)
			})
		}
	})

	t.Run("string,string", func(t *testing.T) {
		sizeOfGroup := unsafe.Sizeof(group[string, string]{})

		got := CapacityFromSize[string, string](sizeOfGroup * 5)
		require.Equal(t, 40, got)
	})

	t.Run("int,struct{}", func(t *testing.T) {
		sizeOfGroup := unsafe.Sizeof(group[int, struct{}]{})

		got := CapacityFromSize[int, struct{}](sizeOfGroup * 3)
		require.Equal(t, 24, got)
	})

	t.Run("usage with New", func(t *testing.T) {
		// CapacityFromSize returns capacity (slots) that fit in given memory.
		sizeOfGroup := unsafe.Sizeof(group[int, int]{})

		capacity := CapacityFromSize[int, int](sizeOfGroup * 4)
		require.Equal(t, 32, capacity)

		// Can pass directly to New
		sm := New[int, int](capacity)
		stats := sm.Stats()
		// 4 groups * 7 effective slots per group (87.5% load factor) = 28
		require.Equal(t, 4*7, stats.EffectiveCapacity)
	})
}
