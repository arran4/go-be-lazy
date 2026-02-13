# Gap Analysis

## Overview
The `go-be-lazy` library provides useful primitives for lazy evaluation (`Value[T]`) and a caching helper (`Map`). However, there are several gaps that limit its usability and robustness.

## Functional Gaps

1.  **Eviction Policy**: The `MaxSize` option implements random eviction (due to Go map iteration order). While simple, this is suboptimal compared to LRU (Least Recently Used) or LFU (Least Frequently Used) policies.
    *   **Recommendation**: Implement a more robust eviction policy infrastructure allowing users to select between "Random" (default, map iteration order) and potentially others in the future (e.g., specific key, oldest).

2.  **Concurrency in Refresh**: The `Refresh` option replaces the value in the map without coordination. If multiple goroutines call `Refresh` concurrently, they will all trigger a fetch, defeating the purpose of single-flight loading.
    *   **Recommendation**: Implement proper coordination for refreshes, perhaps using a `Reset` mechanism on `Value` or ensuring only one refresher wins.

3.  **`store` Method Visibility**: The `store` method on `Value` is unexported but useful for setting default values.
    *   **Recommendation**: Consider exporting it or providing a safe way to set a value without `once` semantics if needed (though `Set` exists, it respects `once`).

## Testing Gaps

1.  **Concurrency Coverage**: While there is a concurrency test, it focuses on `Load`. More comprehensive tests for `Map` concurrency (especially around `Refresh` and eviction) would be beneficial.
2.  **Key Type Coverage**: Tests currently only use `int` (which is compatible with `int32` in the current implementation).
    *   **Recommendation**: Add tests for `string` and struct keys once generic support is added.

## Documentation Gaps

1.  **Key Limitation**: The `int32` limitation is not clearly documented as a constraint, leading to potential confusion.
    *   **Recommendation**: Update documentation to reflect the new generic capabilities.
