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
