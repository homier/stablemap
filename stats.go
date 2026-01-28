package stablemap

type Stats struct {
	Size                    int
	Tombstones              int
	TombstonesCapacityRatio float32
	TombstonesSizeRatio     float32
}
