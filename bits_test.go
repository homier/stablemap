package stablemap

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInvertCtrls(t *testing.T) {
	tests := []struct {
		name  string
		input uint64
		want  uint64
	}{
		{
			name:  "All empty",
			input: 0x8080808080808080,
			want:  0x8080808080808080,
		},
		{
			name:  "All deleted",
			input: 0xFEFEFEFEFEFEFEFE,
			want:  0x8080808080808080,
		},
		{
			name:  "All full (H2=0)",
			input: 0x0000000000000000,
			want:  0xFEFEFEFEFEFEFEFE,
		},
		{
			name:  "All full (H2=0x7F)",
			input: 0x7F7F7F7F7F7F7F7F,
			want:  0xFEFEFEFEFEFEFEFE,
		},
		{
			name:  "Mixed H2 values",
			input: 0x0102030405060708,
			want:  0xFEFEFEFEFEFEFEFE,
		},
		{
			name:  "Mixed: full, empty, deleted",
			input: 0x00_80_FE_42_80_FE_7F_01,
			want:  0xFE_80_80_FE_80_80_FE_FE,
		},
		{
			name:  "Alternating full and empty",
			input: 0x80_00_80_00_80_00_80_00,
			want:  0x80_FE_80_FE_80_FE_80_FE,
		},
		{
			name:  "Single full slot (first byte)",
			input: 0x8080808080808000,
			want:  0x80808080808080FE,
		},
		{
			name:  "Single deleted slot (last byte)",
			input: 0xFE80808080808080,
			want:  0x8080808080808080,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := invertCtrls(tt.input)
			require.Equal(t, tt.want, got, "invertCtrls(0x%016X) = 0x%016X, want 0x%016X", tt.input, got, tt.want)
		})
	}
}
