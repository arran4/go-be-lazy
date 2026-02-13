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
	cache := lazy.NewLazyMap[string, int]()
	lru := lazy.NewLRUEvictionPolicy[string, int]()

	fetch := func(key string) (int, error) { return len(key), nil }

	// Add items with MaxSize 2
	// A: [A]
	cache.Get("A", fetch, lazy.MaxSize[string, int](2), lazy.WithEvictionPolicy[string, int](lru))
	// B: [B, A]
	cache.Get("B", fetch, lazy.MaxSize[string, int](2), lazy.WithEvictionPolicy[string, int](lru))

	// Access A: [A, B]
	cache.Get("A", fetch, lazy.WithEvictionPolicy[string, int](lru))

	// Add C: [C, A]. B is evicted.
	cache.Get("C", fetch, lazy.MaxSize[string, int](2), lazy.WithEvictionPolicy[string, int](lru))

	_, errA := cache.Get("A", nil, lazy.DontFetch[string, int]())
	// B should be evicted, so DontFetch returns error/not found
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
