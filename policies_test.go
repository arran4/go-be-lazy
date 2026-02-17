package lazy_test

import (
	"math/rand"
	"sync"
	"testing"

	lazy "github.com/arran4/go-be-lazy"
)

func TestLRUEvictionPolicy(t *testing.T) {
	m := make(map[int]*lazy.Value[int])
	var mu sync.RWMutex
	fetch := func(id int) (int, error) { return id, nil }
	policy := lazy.NewLRUEvictionPolicy[int, int]()

	// 1: Add 1. LRU: [1]
	Must(lazy.Map(&m, &mu, 1, fetch, lazy.MaxSize[int, int](2), lazy.WithEvictionPolicy[int, int](policy)))
	// 2: Add 2. LRU: [2, 1]
	Must(lazy.Map(&m, &mu, 2, fetch, lazy.MaxSize[int, int](2), lazy.WithEvictionPolicy[int, int](policy)))

	// Access 1. LRU: [1, 2]
	Must(lazy.Map(&m, &mu, 1, fetch, lazy.MaxSize[int, int](2), lazy.WithEvictionPolicy[int, int](policy)))

	// Add 3. Should evict 2. LRU: [3, 1]
	Must(lazy.Map(&m, &mu, 3, fetch, lazy.MaxSize[int, int](2), lazy.WithEvictionPolicy[int, int](policy)))

	if _, ok := m[2]; ok {
		t.Fatal("Expected 2 to be evicted")
	}
	if _, ok := m[1]; !ok {
		t.Fatal("Expected 1 to be present")
	}
	if _, ok := m[3]; !ok {
		t.Fatal("Expected 3 to be present")
	}
}

func TestFIFOEvictionPolicy(t *testing.T) {
	m := make(map[int]*lazy.Value[int])
	var mu sync.RWMutex
	fetch := func(id int) (int, error) { return id, nil }
	policy := lazy.NewFIFOEvictionPolicy[int, int]()

	// Add 1. FIFO: [1]
	Must(lazy.Map(&m, &mu, 1, fetch, lazy.MaxSize[int, int](2), lazy.WithEvictionPolicy[int, int](policy)))
	// Add 2. FIFO: [1, 2]
	Must(lazy.Map(&m, &mu, 2, fetch, lazy.MaxSize[int, int](2), lazy.WithEvictionPolicy[int, int](policy)))

	// Access 1. FIFO order shouldn't change on access.
	Must(lazy.Map(&m, &mu, 1, fetch, lazy.MaxSize[int, int](2), lazy.WithEvictionPolicy[int, int](policy)))

	// Add 3. Should evict 1 (First In).
	Must(lazy.Map(&m, &mu, 3, fetch, lazy.MaxSize[int, int](2), lazy.WithEvictionPolicy[int, int](policy)))

	if _, ok := m[1]; ok {
		t.Fatal("Expected 1 to be evicted")
	}
	if _, ok := m[2]; !ok {
		t.Fatal("Expected 2 to be present")
	}
	if _, ok := m[3]; !ok {
		t.Fatal("Expected 3 to be present")
	}
}

func TestLFUEvictionPolicy(t *testing.T) {
	m := make(map[int]*lazy.Value[int])
	var mu sync.RWMutex
	fetch := func(id int) (int, error) { return id, nil }
	policy := lazy.NewLFUEvictionPolicy[int, int]()

	// Add 1. Freqs: {1:1}
	Must(lazy.Map(&m, &mu, 1, fetch, lazy.MaxSize[int, int](2), lazy.WithEvictionPolicy[int, int](policy)))
	// Add 2. Freqs: {1:1, 2:1}
	Must(lazy.Map(&m, &mu, 2, fetch, lazy.MaxSize[int, int](2), lazy.WithEvictionPolicy[int, int](policy)))

	// Access 1 twice more. Freqs: {1:3, 2:1}
	Must(lazy.Map(&m, &mu, 1, nil, lazy.DontFetch[int, int](), lazy.WithEvictionPolicy[int, int](policy)))
	Must(lazy.Map(&m, &mu, 1, nil, lazy.DontFetch[int, int](), lazy.WithEvictionPolicy[int, int](policy)))

	// Access 2 once more. Freqs: {1:3, 2:2}
	Must(lazy.Map(&m, &mu, 2, nil, lazy.DontFetch[int, int](), lazy.WithEvictionPolicy[int, int](policy)))

	// Add 3. Should evict 2 (Freq 2 < Freq 3).
	// Currently map has {1, 2}. MaxSize is 2. We need to evict.
	// 1 has freq 3. 2 has freq 2.
	// Victim is 2.
	Must(lazy.Map(&m, &mu, 3, fetch, lazy.MaxSize[int, int](2), lazy.WithEvictionPolicy[int, int](policy)))

	if _, ok := m[2]; ok {
		t.Fatalf("Expected 2 to be evicted. Map: %v", m)
	}
	if _, ok := m[1]; !ok {
		t.Fatal("Expected 1 to be present")
	}
	if _, ok := m[3]; !ok {
		t.Fatal("Expected 3 to be present")
	}
}

func TestEvictionPolicyConcurrency(t *testing.T) {
	m := make(map[int]*lazy.Value[int])
	var mu sync.RWMutex
	fetch := func(id int) (int, error) { return id, nil }

	// Test LRU concurrency
	policy := lazy.NewLRUEvictionPolicy[int, int]()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			// Randomly access keys 0-9
			key := rand.Intn(10)
			Must(lazy.Map(&m, &mu, key, fetch, lazy.MaxSize[int, int](5), lazy.WithEvictionPolicy[int, int](policy)))
		}(i)
	}
	wg.Wait()

	if len(m) > 5 {
		t.Fatalf("Map size exceeded max size: %d", len(m))
	}
}

func TestNoEvictionPolicy(t *testing.T) {
	m := make(map[int]*lazy.Value[int])
	var mu sync.RWMutex
	fetch := func(id int) (int, error) { return id, nil }
	policy := &lazy.NoEvictionPolicy[int, int]{}

	// Add 3 items with MaxSize 2.
	Must(lazy.Map(&m, &mu, 1, fetch, lazy.MaxSize[int, int](2), lazy.WithEvictionPolicy[int, int](policy)))
	Must(lazy.Map(&m, &mu, 2, fetch, lazy.MaxSize[int, int](2), lazy.WithEvictionPolicy[int, int](policy)))
	Must(lazy.Map(&m, &mu, 3, fetch, lazy.MaxSize[int, int](2), lazy.WithEvictionPolicy[int, int](policy)))

	if len(m) != 3 {
		t.Fatalf("Expected map size 3 (no eviction), got %d", len(m))
	}
}
