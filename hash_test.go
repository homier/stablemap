package stableset

import (
	"hash/maphash"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMakeDefaultHash(t *testing.T) {
	v := "foo"
	s := maphash.MakeSeed()

	h1 := MakeDefaultHashFunc[string](s)(v)
	h2 := maphash.Comparable(s, v)

	require.Equal(t, h2, h1)
}

func TestHashSplit(t *testing.T) {
	tests := []struct {
		name   string
		input  uint64
		wantH1 uintptr
		wantH2 uint8
	}{
		{
			name:   "Zero value",
			input:  0,
			wantH1: 0,
			wantH2: 0,
		},
		{
			name:   "Max H2 (7 bits)",
			input:  0x7F, // 0111 1111
			wantH1: 0,
			wantH2: 0x7F,
		},
		{
			name:   "First bit of H1",
			input:  1 << 7, // 1000 0000
			wantH1: 1,
			wantH2: 0,
		},
		{
			name:   "Max uint64",
			input:  0xFFFFFFFFFFFFFFFF,
			wantH1: uintptr(0xFFFFFFFFFFFFFFFF >> 7),
			wantH2: 0x7F,
		},
		{
			name:   "Random pattern",
			input:  0xABCD1234567890EF,
			wantH1: uintptr(0xABCD1234567890EF >> 7),
			wantH2: 0xEF & 0x7F, // 0x6F
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h1, h2 := HashSplit(tt.input)

			require.Equal(t, tt.wantH1, h1)
			require.Equal(t, tt.wantH2, h2)
		})
	}
}
