package lazy_test

import (
	"context"
	"fmt"
	"time"

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

func ExampleLazyMap_withExpiry() {
	// Create a map where items expire after 1 use
	cache := lazy.NewLazyMap[string, int](
		lazy.WithExpiry[string, int](lazy.ExpireAfterUses[int](1)),
	)

	callCount := 0
	fetch := func(name string) (int, error) {
		callCount++
		return len(name), nil
	}

	// First fetch (Fetch #1)
	val, _ := cache.Get("foo", fetch)
	fmt.Printf("Val: %d, Calls: %d\n", val, callCount)

	// Second fetch (Used once, so returns cached value but marks as expired for next time if we strictly followed "AfterUses".
	// However, ExpireAfterUses(1) means: IsExpired if Uses >= 1.
	// So immediately after load, Uses=1.
	// Next Get calls Map -> checks IsExpired -> Uses(1) >= 1 -> True.
	// So it should refresh immediately.
	val, _ = cache.Get("foo", fetch)
	fmt.Printf("Val: %d, Calls: %d\n", val, callCount)

	// Output:
	// Val: 3, Calls: 1
	// Val: 3, Calls: 2
}

func ExampleLazyMap_withTimeExpiry() {
	// Create a map where items expire quickly
	cache := lazy.NewLazyMap[string, int](
		lazy.WithExpiry[string, int](lazy.ExpireAfter[int](10 * time.Millisecond)),
	)

	callCount := 0
	fetch := func(name string) (int, error) {
		callCount++
		return len(name), nil
	}

	// Fetch 1
	cache.Get("bar", fetch)
	fmt.Printf("Calls: %d\n", callCount)

	// Immediate fetch (cached)
	cache.Get("bar", fetch)
	fmt.Printf("Calls: %d\n", callCount)

	// Wait for expiry
	time.Sleep(20 * time.Millisecond)

	// Fetch 2 (refreshed)
	cache.Get("bar", fetch)
	fmt.Printf("Calls: %d\n", callCount)

	// Output:
	// Calls: 1
	// Calls: 1
	// Calls: 2
}

func ExampleLazyMap_withContextExpiry() {
	ctx, cancel := context.WithCancel(context.Background())

	// Create a map where items expire when context is cancelled
	cache := lazy.NewLazyMap[string, int](
		lazy.WithExpiry[string, int](lazy.ExpireContext[int](ctx)),
	)

	callCount := 0
	fetch := func(name string) (int, error) {
		callCount++
		return len(name), nil
	}

	// Fetch 1
	cache.Get("baz", fetch)
	fmt.Printf("Calls: %d\n", callCount)

	// Cancel context
	cancel()

	// Fetch 2 (refreshed because context is done)
	cache.Get("baz", fetch)
	fmt.Printf("Calls: %d\n", callCount)

	// Output:
	// Calls: 1
	// Calls: 2
}
