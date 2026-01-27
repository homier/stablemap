package stablemap

const groupSize = 8

type group[K comparable, V any] struct {
	// 8 bytes of metadata (h2 or control states)
	// This fits perfectly in a single uint64 load
	ctrls [groupSize]uint8

	// 8 keys stored immediately after the metadata
	// In a 64-bit system, this group is (8 + 8*8) = 72 bytes.
	// That's just slightly over one 64-byte cache line.
	slots [groupSize]K

	// 8 values stored after the keys.
	// Even If V is a struct{} type, Go compiler will add padding.
	// It's very sensible, you need to be careful with the value type.
	// If it's too large, you'll end up missing CPU cache lines.
	// If it's too low, Go will add padding and you'll end up wasting
	// memory for nothing.
	values [groupSize]V
}
