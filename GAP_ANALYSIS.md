# Gap Analysis

## Overview
The `go-be-lazy` library provides useful primitives for lazy evaluation (`Value[T]`) and a caching helper (`Map`). However, there are several gaps that limit its usability and robustness.

## Functional Gaps

1.  **Concurrency in Refresh**: The `Refresh` option replaces the value in the map without coordination. If multiple goroutines call `Refresh` concurrently, they will all trigger a fetch, defeating the purpose of single-flight loading.
    *   **Recommendation**: Implement proper coordination for refreshes. Options include:
        *   **Reset Mechanism**: Invalidate the current `once` state, allowing the next fetch to proceed safely.
        *   **Single-flight**: Use a single-flight group to ensure only one refresh fetch happens at a time.

## Testing Gaps

1.  **Concurrency Coverage**: While there is a concurrency test, it focuses on `Load`. More comprehensive tests for `Map` concurrency (especially around `Refresh` and eviction) would be beneficial.
