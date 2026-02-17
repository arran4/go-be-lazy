package lazy_test

import (
	"fmt"

	lazy "github.com/arran4/go-be-lazy"
)

func ExampleLazyMap() {
	cache := lazy.NewLazyMap[string, int]()

	fetchUserAge := func(name string) (int, error) {
		// Simulate database lookup
		return len(name) * 10, nil
	}

	// First fetch
	age, err := cache.Get("Alice", fetchUserAge)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Printf("Alice: %d\n", age)

	// Cached fetch
	age2, _ := cache.Get("Alice", fetchUserAge)
	fmt.Printf("Alice (Cached): %d\n", age2)

	// Output:
	// Alice: 50
	// Alice (Cached): 50
}

func ExampleLazyMap_withEviction() {
	// Use LRU policy for deterministic eviction
	lru := lazy.NewLRUEvictionPolicy[string, int]()

	// Apply MaxSize and EvictionPolicy globally
	cache := lazy.NewLazyMap[string, int](
		lazy.MaxSize[string, int](2),
		lazy.WithEvictionPolicy[string, int](lru),
	)

	fetch := func(key string) (int, error) { return len(key), nil }

	// Add items. MaxSize is 2.
	// A: [A]
	_, _ = cache.Get("A", fetch)
	// B: [B, A]
	_, _ = cache.Get("B", fetch)

	// Access A: [A, B]
	_, _ = cache.Get("A", fetch)

	// Add C: [C, A]. B is evicted.
	_, _ = cache.Get("C", fetch)

	_, errA := cache.Get("A", nil, lazy.DontFetch[string, int]())
	_, errB := cache.Get("B", nil, lazy.DontFetch[string, int](), lazy.MustBeCached[string, int]())
	_, errC := cache.Get("C", nil, lazy.DontFetch[string, int]())

	fmt.Println("Has A:", errA == nil)
	fmt.Println("Has B:", errB == nil)
	fmt.Println("Has C:", errC == nil)

	// Output:
	// Has A: true
	// Has B: false
	// Has C: true
}
