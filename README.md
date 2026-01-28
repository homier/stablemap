# stablemap
Go (S)wiss (T)able Map

## Intro
StableMap is a high-performance, contiguous-memory hash map for Go, inspired by the Swiss Table (Abseil) design. It is engineered for scenarios requiring ultra-low latency, zero heap allocations during operation, and mechanical sympathy for modern CPU caches.

By leveraging SWAR (SIMD-within-a-register) techniques and a group-based metadata layout, StableSet achieves lookup and insertion speeds that rival the Go standard map, while maintaining a significantly smaller garbage collection (GC) footprint.

It is designed with a **fixed-size memory model**. It does not grow automatically, providing predictable memory estimation for large datasets and preventing unexpected OOMs in production.

## Inspired by
* [CockroachDB Swiss](https://github.com/cockroachdb/swiss) map implementation
* [Abseil Swiss Tables](https://abseil.io/about/design/swisstables)
* Go internal map

## Key features
* **Zero allocation hot path**: after initial initialization, `Set`, `Put`, `Has` and `Delete` methods do not allocate additional memory.
* **In-place rehash**: rehashing happens in-place to remove tombstones without additional allocations or doubling memory.
* **Contiguous memory**: data stored in a single slice of groups.
* **Custom hash function**: you can provide your own hash function instead of default `hash/maphash`.
* **Set-like variant**: use `StableSet` for set-like datastructure instead of `StableMap`.

## Implementation details
StableSet uses Swiss table design, organizing data into groups of 8 slots. Each group contains a 64-bit control word (8 bytes of metadata) and 8 data slots.
1. H1 Hashing: Determines the starting group index.
2. H2 Fingerprinting: A 7-bit hash stored in the control byte for rapid SIMD-style filtering.
3. Quadratic Probing: Uses $\frac{p^2 + p}{2}$ to resolve collisions, preventing the "primary clustering" common in linear probing.
4. Tombstones: Uses a special `0xFE` marker for deleted slots to maintain the probe invariant without moving keys immediately.

## Usage
### StableMap
```go
import "github.com/homier/stablemap"

// Initialize with a capacity hint
sm := stablemap.New[int, string](1024)

// Add elements - Set returns error if compaction is needed
err := sm.Set(42, "foo")
if errors.Is(err, stablemap.ErrTableFull) {
    sm.Compact()
    sm.Set(42, "foo") // retry
}

// Get value
v, ok := sm.Get(42)
if ok {
    fmt.Println("Found it: ", v)
}

// Put returns (isNew, error) - fails if key exists
ok, err = sm.Put(43, "bar")
if !ok {
    fmt.Println("Can't put a value, key exists")
}
if errors.Is(err, stablemap.ErrTableFull) {
    sm.Compact()
}

// Set overwrites existing values
err = sm.Set(42, "bar")
if errors.Is(err, stablemap.ErrTableFull) {
    sm.Compact()
    sm.Set(42, "bar")
}

v, ok = sm.Get(42)
if ok {
    fmt.Println("Found it! Now it should be `bar`: ", v)
}

// Delete element
if sm.Delete(42) {
    fmt.Println("Deleted it")
}

_, ok = sm.Get(42)
if !ok {
    fmt.Println("Does not exist anymore!")
}
```

### StableSet
```go
import "github.com/homier/stablemap"

// Initialize with a capacity hint
ss := stablemap.NewSet[int](1024)

// Add elements - Put returns (isNew, error)
ok, err := ss.Put(42)
if errors.Is(err, stablemap.ErrTableFull) {
    ss.Compact()
    ss.Put(42) // retry
}

// Check existence
if ss.Has(42) {
    fmt.Println("Found it!")
}

// Remove elements
if ss.Delete(42) {
    fmt.Println("Deleted it")
}

if !ss.Has(42) {
    fmt.Println("Does not exist anymore!")
}
```

### Stats
Both `StableMap` and `StableSet` provide a `Stats()` method for monitoring table health:
```go
stats := sm.Stats()
fmt.Printf("Size: %d\n", stats.Size)
fmt.Printf("Tombstones: %d\n", stats.Tombstones)
fmt.Printf("Tombstones/Capacity: %.2f\n", stats.TombstonesCapacityRatio)
fmt.Printf("Tombstones/Size: %.2f\n", stats.TombstonesSizeRatio)

// Use stats to decide when to compact
if stats.TombstonesCapacityRatio > 0.25 {
    sm.Compact()
}
```

## When to use StableMap
Use Go map first. But, while the standard Go map is the right choice for most cases, StableMap excels when:
1. You are handling large datasets (GBs of data) where GC scan times for standard maps become a bottleneck.
2. You need predictable memory usage and want to avoid the "latency spikes" caused by map growth/evacuation.
3. You have a high-churn workload (constant Puts/Deletes) and want to manage tombstone cleanup manually via Rehash().

## TODO list
1. Expand unit tests for edge cases (maximum capacity, hash collisions, rehashing).
2. More proper benchmarks across different CPU architectures.
3. Explore platform-specific SIMD (SSE/AVX) as an alternative to the current SWAR implementation.
4. Beautiful table for benchmarks and memory consumption in README.

