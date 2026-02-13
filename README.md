# Lazy Evaluation Package

The `lazy` package provides generic, thread-safe primitives for lazy evaluation and caching of values. It is designed to handle expensive initialization operations that should only be performed once, or to manage caches of items loaded on demand.

## Features

- **Thread-Safe**: Uses `sync.Once` and atomic operations to ensure values are initialized exactly once, even under concurrent access.
- **Generics**: Fully supports Go generics for type safety (`Map[K, V]`).
- **Flexible Mapping**: Includes a helper for managing lazily loaded values in a map.
- **Configurable**: extensive options for controlling fetch behavior (timeouts, defaults, forced refreshes, etc.).
- **Eviction Policies**: Built-in support for multiple eviction policies (Random, LRU, LFU, FIFO) and custom policy implementation.

## Usage

### Single Value

The `Value[T]` struct allows you to lazily load a single value.

```go
package main

import (
	"fmt"
	"github.com/arran4/go-be-lazy"
)

func main() {
	var config lazy.Value[map[string]string]

	// The initialization function is only called once.
	val, err := config.Load(func() (map[string]string, error) {
		fmt.Println("Loading config...")
		return map[string]string{"key": "value"}, nil
	})

	if err != nil {
		panic(err)
	}

	fmt.Println(val["key"]) // Output: value

	// Subsequent calls return the cached value immediately.
	cachedVal, _ := config.Load(nil)
	fmt.Println(cachedVal["key"]) // Output: value
}
```

### Lazy Map

The `LazyMap` struct provides a convenient way to manage a collection of lazy values, keyed by any comparable type (e.g., strings, integers). It handles map locking and value initialization.

```go
package main

import (
	"fmt"
	"github.com/arran4/go-be-lazy"
)

func main() {
	// Create a new LazyMap with string keys and int values.
	cache := lazy.NewLazyMap[string, int]()

	fetchUserAge := func(name string) (int, error) {
		fmt.Printf("Fetching age for %s\n", name)
		return len(name) * 10, nil // Dummy logic
	}

	// Fetch age for "Alice" (will trigger fetch)
	age1, err := cache.Get("Alice", fetchUserAge)
	if err != nil {
		panic(err)
	}
	fmt.Println(age1) // Output: 50

	// Fetch age for "Alice" again (will use cache)
	ageCached, _ := cache.Get("Alice", fetchUserAge)
	fmt.Println(ageCached) // Output: 50

	// Options can modify behavior, e.g., force refresh
	ageRefreshed, _ := cache.Get("Alice", fetchUserAge, lazy.Refresh[string, int]())
	fmt.Println(ageRefreshed)
}
```

### Eviction Policies

You can specify an eviction policy using `WithEvictionPolicy`. The library provides several implementations:

*   `RandomEvictionPolicy`: Uses Go's map iteration order (default).
*   `LRUEvictionPolicy`: Least Recently Used eviction.
*   `LFUEvictionPolicy`: Least Frequently Used eviction.
*   `FIFOEvictionPolicy`: First-In-First-Out eviction.
*   `NoEvictionPolicy`: No eviction (MaxSize is effectively ignored).

```go
// Create an LRU policy
lru := lazy.NewLRUEvictionPolicy[string, int]()

// Use it with MaxSize
cache.Get("Bob", fetchUserAge,
    lazy.MaxSize[string, int](100),
    lazy.WithEvictionPolicy[string, int](lru),
)
```

## API Overview

### Types

- `Value[T]`: The core struct for lazy loading. Zero value is ready to use.
- `LazyMap[K, V]`: A thread-safe map wrapper for lazy values.
- `Option[K, V]`: Functional options for `Map` and `LazyMap`.
- `EvictionPolicy[K, V]`: Interface for custom eviction strategies.

### Functions

- `Map`: Lower-level function for managing lazy values in a raw map.
- `NewLazyMap`: Creates a `LazyMap` instance.

### Options for Map

- `DontFetch`: Returns the cached value if present, otherwise zero/default (does not trigger fetch).
- `Set`: Manually sets the value for the key.
- `SetID`: Overrides the ID used for lookup.
- `Refresh`: Forces a reload of the value.
- `Clear`: Removes the value from the map.
- `Must`: Wraps errors from the fetch function.
- `MustBeCached`: Returns an error if the value is not already cached.
- `DefaultValue`: Returns this value if lookup fails or (optionally) if fetch fails.
- `MaxSize`: Limits the size of the map, triggering eviction based on the policy.
- `WithEvictionPolicy`: Sets the eviction strategy.

## Thread Safety

- **Value[T]**: `Load`, `Set`, and `Peek` are safe for concurrent use. `Load` guarantees the initialization function runs exactly once.
- **LazyMap**: Wraps `Map` and handles mutex locking internally.
- **Map**: Requires the caller to provide a `sync.Mutex` which it uses to protect map operations (insertion/deletion). The value loading itself happens outside the map lock to avoid blocking other lookups.
- **EvictionPolicy**: Implementations provided (`LRU`, `LFU`, `FIFO`, `Random`) are thread-safe for concurrent access.

## License

This project is licensed under the 3-Clause BSD License - see the [LICENSE](LICENSE) file for details.
