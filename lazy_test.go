package lazy_test

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	lazy "github.com/arran4/go-be-lazy"
)

func Must[T any](v T, err error) T {
	if err != nil {
		panic(fmt.Sprintf("Must failed: %v", err))
	}
	return v
}

func TestValueLoadOnce(t *testing.T) {
	var v lazy.Value[int]
	calls := 0
	got, err := v.Load(func() (int, error) {
		calls++
		return 42, nil
	})
	if err != nil || got != 42 {
		t.Fatalf("first load got %v %v", got, err)
	}
	got, err = v.Load(func() (int, error) {
		calls++
		return 99, nil
	})
	if err != nil || got != 42 {
		t.Fatalf("second load got %v %v", got, err)
	}
	if calls != 1 {
		t.Fatalf("calls=%d", calls)
	}
}

func TestValueLoadError(t *testing.T) {
	var v lazy.Value[int]
	firstErr := errors.New("bad")
	if _, err := v.Load(func() (int, error) { return 0, firstErr }); err != firstErr {
		t.Fatalf("err=%v", err)
	}
	if v, err := v.Load(func() (int, error) { return 1, nil }); err != firstErr || v != 0 {
		t.Fatalf("second load v=%d err=%v", v, err)
	}
}

func TestValueSetPeek(t *testing.T) {
	var v lazy.Value[string]
	v.Set("hello")
	if val, ok := v.Peek(); !ok || val != "hello" {
		t.Fatalf("peek got %v %v", val, ok)
	}
	v.Set("world")
	if val, _ := v.Peek(); val != "hello" {
		t.Fatalf("overwrite val=%s", val)
	}
}

func TestMapNilMap(t *testing.T) {
	var mu sync.RWMutex
	_, err := lazy.Map[int, int](nil, &mu, 1, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMapFetchCaching(t *testing.T) {
	m := make(map[int32]*lazy.Value[int])
	calls := 0
	fetch := func(id int32) (int, error) {
		calls++
		return int(id * 2), nil
	}
	var mu sync.RWMutex
	v := Must(lazy.Map(&m, &mu, 1, fetch))
	if v != 2 {
		t.Fatalf("got %v", v)
	}
	v = Must(lazy.Map(&m, &mu, 1, fetch))
	if v != 2 || calls != 1 {
		t.Fatalf("cached %v calls=%d", v, calls)
	}
}

func TestMapDontFetchMustCached(t *testing.T) {
	m := make(map[int32]*lazy.Value[int])
	var mu sync.RWMutex
	_, err := lazy.Map(&m, &mu, 1, nil, lazy.DontFetch[int32, int](), lazy.MustBeCached[int32, int]())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMapDontFetchDefaultValue(t *testing.T) {
	m := make(map[int32]*lazy.Value[int])
	var mu sync.RWMutex
	v, err := lazy.Map(&m, &mu, 5, nil, lazy.DontFetch[int32, int](), lazy.DefaultValue[int32, int](42))
	if err != nil || v != 42 {
		t.Fatalf("got %v %v", v, err)
	}
	if got, ok := m[5].Peek(); ok {
		t.Fatalf("unexpectedly cached %v %v", got, ok)
	}
}

func TestMapMustWrapError(t *testing.T) {
	m := make(map[int32]*lazy.Value[int])
	var mu sync.RWMutex
	_, err := lazy.Map(&m, &mu, 1, func(int32) (int, error) { return 0, errors.New("bad") }, lazy.Must[int32, int]())
	if err == nil || err.Error() != "fetch error: bad" {
		t.Fatalf("err=%v", err)
	}
}

func TestMapDefaultValueOnError(t *testing.T) {
	m := make(map[int32]*lazy.Value[int])
	var mu sync.RWMutex
	v, err := lazy.Map(&m, &mu, 1, func(int32) (int, error) { return 0, errors.New("bad") }, lazy.DefaultValue[int32, int](5))
	if err != nil || v != 5 {
		t.Fatalf("got %v %v", v, err)
	}
}

func TestMapClear(t *testing.T) {
	m := make(map[int32]*lazy.Value[int])
	var mu sync.RWMutex
	if _, err := lazy.Map(&m, &mu, 1, func(int32) (int, error) { return 1, nil }); err != nil {
		t.Fatal(err)
	}
	if _, err := lazy.Map(&m, &mu, 1, nil, lazy.Clear[int32, int]()); err != nil {
		t.Fatal(err)
	}
	if _, ok := m[1]; ok {
		t.Fatal("value not cleared")
	}
}

func TestMapRefresh(t *testing.T) {
	m := make(map[int32]*lazy.Value[int])
	calls := 0
	fetch := func(int32) (int, error) { calls++; return calls, nil }
	var mu sync.RWMutex
	v := Must(lazy.Map(&m, &mu, 1, fetch))
	if v != 1 {
		t.Fatalf("first=%d", v)
	}
	v = Must(lazy.Map(&m, &mu, 1, fetch, lazy.Refresh[int32, int]()))
	if v != 2 {
		t.Fatalf("refresh=%d", v)
	}
	if calls != 2 {
		t.Fatalf("calls=%d", calls)
	}
}

func TestMapSet(t *testing.T) {
	m := make(map[int32]*lazy.Value[int])
	var mu sync.RWMutex
	v, err := lazy.Map(&m, &mu, 1, nil, lazy.Set[int32, int](7))
	if err != nil || v != 7 {
		t.Fatalf("set %v %v", v, err)
	}
	v, err = lazy.Map(&m, &mu, 1, nil, lazy.DontFetch[int32, int]())
	if err != nil || v != 7 {
		t.Fatalf("cached %v %v", v, err)
	}
}

func TestMapSetID(t *testing.T) {
	m := make(map[int32]*lazy.Value[int])
	var calls int
	fetch := func(id int32) (int, error) { calls++; return int(id), nil }
	var mu sync.RWMutex
	v, err := lazy.Map(&m, &mu, 1, fetch, lazy.SetID[int32, int](2))
	if err != nil || v != 2 {
		t.Fatalf("got %v %v", v, err)
	}
	if _, ok := m[2]; !ok {
		t.Fatal("missing id 2")
	}
	if _, ok := m[1]; ok {
		t.Fatal("unexpected id 1")
	}
	if calls != 1 {
		t.Fatalf("calls=%d", calls)
	}
}

func TestMapConcurrent(t *testing.T) {
	m := make(map[int32]*lazy.Value[int])
	var mu sync.RWMutex
	calls := 0
	fetch := func(id int32) (int, error) {
		mu.Lock()
		defer mu.Unlock()
		calls++
		return int(id), nil
	}
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if v, err := lazy.Map(&m, &mu, 1, fetch); err != nil || v != 1 {
				t.Errorf("%v %v", v, err)
			}
		}()
	}
	wg.Wait()
	if calls != 1 {
		t.Fatalf("calls=%d", calls)
	}
}

func TestMapBoundedGrowth(t *testing.T) {
	m := make(map[int32]*lazy.Value[int])
	var mu sync.RWMutex
	fetch := func(id int32) (int, error) { return int(id), nil }

	// Simulate adding many items with MaxSize
	limit := 1000
	maxSize := 100
	for i := 0; i < limit; i++ {
		_, err := lazy.Map(&m, &mu, int32(i), fetch, lazy.MaxSize[int32, int](maxSize))
		if err != nil {
			t.Fatalf("Map failed: %v", err)
		}
	}

	if len(m) != maxSize {
		t.Fatalf("Expected map size %d, got %d", maxSize, len(m))
	}
}

func TestMapEvictionPolicy(t *testing.T) {
	// Use RandomEvictionPolicy explicitly
	m := make(map[int32]*lazy.Value[int])
	var mu sync.RWMutex
	fetch := func(id int32) (int, error) { return int(id), nil }

	limit := 50
	maxSize := 10
	policy := &lazy.RandomEvictionPolicy[int32, int]{}
	for i := 0; i < limit; i++ {
		_, err := lazy.Map(&m, &mu, int32(i), fetch,
			lazy.MaxSize[int32, int](maxSize),
			lazy.WithEvictionPolicy[int32, int](policy))
		if err != nil {
			t.Fatalf("Map failed: %v", err)
		}
	}

	if len(m) != maxSize {
		t.Fatalf("Expected map size %d, got %d", maxSize, len(m))
	}
}

// MockEvictionPolicy for testing stateful policy hooks
type MockEvictionPolicy struct {
	accessCount int
	evictCount  int
}

func (p *MockEvictionPolicy) Access(key int32) {
	p.accessCount++
}

func (p *MockEvictionPolicy) SelectVictim(m map[int32]*lazy.Value[int]) (int32, bool) {
	p.evictCount++
	// Evict the first one found
	for k := range m {
		return k, true
	}
	return 0, false
}

func TestMapStatefulEvictionPolicy(t *testing.T) {
	m := make(map[int32]*lazy.Value[int])
	var mu sync.RWMutex
	fetch := func(id int32) (int, error) { return int(id), nil }
	policy := &MockEvictionPolicy{}

	// 1. Add item 1
	if _, err := lazy.Map(&m, &mu, 1, fetch,
		lazy.MaxSize[int32, int](2),
		lazy.WithEvictionPolicy[int32, int](policy)); err != nil {
		t.Fatal(err)
	}

	if policy.accessCount != 1 {
		t.Fatalf("Access count expected 1, got %d", policy.accessCount)
	}

	// 2. Access item 1 again (cached)
	if _, err := lazy.Map(&m, &mu, 1, fetch,
		lazy.MaxSize[int32, int](2),
		lazy.WithEvictionPolicy[int32, int](policy)); err != nil {
		t.Fatal(err)
	}

	if policy.accessCount != 2 {
		t.Fatalf("Access count expected 2, got %d", policy.accessCount)
	}

	// 3. Add item 2
	if _, err := lazy.Map(&m, &mu, 2, fetch,
		lazy.MaxSize[int32, int](2),
		lazy.WithEvictionPolicy[int32, int](policy)); err != nil {
		t.Fatal(err)
	}

	if policy.accessCount != 3 {
		t.Fatalf("Access count expected 3, got %d", policy.accessCount)
	}
	if policy.evictCount != 0 {
		t.Fatalf("Evict count expected 0, got %d", policy.evictCount)
	}

	// 4. Add item 3 (trigger eviction)
	if _, err := lazy.Map(&m, &mu, 3, fetch,
		lazy.MaxSize[int32, int](2),
		lazy.WithEvictionPolicy[int32, int](policy)); err != nil {
		t.Fatal(err)
	}

	if policy.evictCount != 1 {
		t.Fatalf("Evict count expected 1, got %d", policy.evictCount)
	}
	if len(m) != 2 {
		t.Fatalf("Map size expected 2, got %d", len(m))
	}
}

func TestMapDefaultValueCachingBug(t *testing.T) {
	m := make(map[int32]*lazy.Value[int])
	var mu sync.RWMutex

	// First call: fetch fails, should return default value and cache it
	v, err := lazy.Map(&m, &mu, 1, func(int32) (int, error) {
		return 0, errors.New("fail")
	}, lazy.DefaultValue[int32, int](100))

	if err != nil {
		t.Fatalf("first call: unexpected error: %v", err)
	}
	if v != 100 {
		t.Fatalf("first call: expected 100, got %v", v)
	}

	// Second call: should return the cached default value
	v2, err2 := lazy.Map(&m, &mu, 1, func(int32) (int, error) {
		return 0, errors.New("fail again")
	})

	if err2 != nil {
		t.Fatalf("second call: unexpected error: %v", err2)
	}
	if v2 != 100 {
		t.Fatalf("second call: expected 100, got %v", v2)
	}
}

func TestMapStringKeys(t *testing.T) {
	m := make(map[string]*lazy.Value[string])
	var mu sync.RWMutex
	fetch := func(key string) (string, error) {
		return "val:" + key, nil
	}

	v, err := lazy.Map(&m, &mu, "foo", fetch)
	if err != nil || v != "val:foo" {
		t.Fatalf("got %v %v", v, err)
	}

	// Cache hit
	v, err = lazy.Map(&m, &mu, "foo", nil, lazy.DontFetch[string, string]())
	if err != nil || v != "val:foo" {
		t.Fatalf("cached got %v %v", v, err)
	}
}

func TestLazyMap(t *testing.T) {
	lm := lazy.NewLazyMap[string, int]()

	// Get with fetch
	v, err := lm.Get("one", func(string) (int, error) { return 1, nil })
	if err != nil || v != 1 {
		t.Fatalf("Get failed: %v %v", v, err)
	}

	// Get cached
	v, err = lm.Get("one", nil)
	if err != nil || v != 1 {
		t.Fatalf("Get cached failed: %v %v", v, err)
	}

	// Set
	lm.Set("two", 2)
	v, err = lm.Get("two", nil)
	if err != nil || v != 2 {
		t.Fatalf("Set failed: %v %v", v, err)
	}

	// Remove
	lm.Remove("two")
	v, err = lm.Get("two", nil, lazy.DontFetch[string, int]())
	if err != nil || v != 0 {
		t.Fatalf("Remove failed: %v %v", v, err)
	}
}

func TestWithRelease(t *testing.T) {
	m := make(map[string]*lazy.Value[int])
	var mu sync.RWMutex
	fetch := func(key string) (int, error) {
		return 123, nil
	}

	var release func()
	v, err := lazy.Map(&m, &mu, "key1", fetch, lazy.WithRelease[string, int](&release))
	if err != nil {
		t.Fatalf("Map failed: %v", err)
	}
	if v != 123 {
		t.Fatalf("Expected 123, got %v", v)
	}
	if release == nil {
		t.Fatal("release function was not populated")
	}

	// Verify entry exists
	if _, ok := m["key1"]; !ok {
		t.Fatal("Entry should exist before release")
	}

	// Release
	release()

	// Verify entry is gone (purged)
	if _, ok := m["key1"]; ok {
		t.Fatal("Entry should be purged after release")
	}

	// Verify refetch works
	v, err = lazy.Map(&m, &mu, "key1", fetch)
	if err != nil {
		t.Fatalf("Refetch failed: %v", err)
	}
	if v != 123 {
		t.Fatalf("Expected 123 on refetch, got %v", v)
	}
}

func TestWithReleaseExpiry(t *testing.T) {
	// This test verifies that releasing makes the value "expired" for policies
	m := make(map[string]*lazy.Value[int])
	var mu sync.RWMutex

	// Use a long expiration time
	policy := lazy.ExpireAfter[int](time.Hour)

	var release func()
	fetch := func(key string) (int, error) { return 456, nil }

	_, _ = lazy.Map(&m, &mu, "key1", fetch, lazy.WithExpiry[string, int](policy), lazy.WithRelease[string, int](&release))

	val := m["key1"]
	if val == nil {
		t.Fatal("Value should be in map")
	}

	if policy.IsExpired(val) {
		t.Fatal("Should not be expired initially")
	}

	release()

	// Check if the purged value is now considered expired
	if !policy.IsExpired(val) {
		t.Fatal("Value should be considered expired after release")
	}
}

func TestWithReleaseReplaced(t *testing.T) {
	m := make(map[string]*lazy.Value[int])
	var mu sync.RWMutex
	fetch := func(key string) (int, error) { return 1, nil }

	var release func()
	v1, _ := lazy.Map(&m, &mu, "key", fetch, lazy.WithRelease[string, int](&release))
	if v1 != 1 {
		t.Fatal("expected 1")
	}

	// Simulate replacement
	// We use Set to replace it, but Set only works if not loaded or new.
	// So we clear first to force a new Value instance.
	lazy.Map(&m, &mu, "key", nil, lazy.Clear[string, int]())
	lazy.Map(&m, &mu, "key", nil, lazy.Set[string, int](2))

	// Now m["key"] has value 2 (different Value instance)
	v2, _ := lazy.Map(&m, &mu, "key", nil) // Peek
	if v2 != 2 {
		t.Fatal("expected 2")
	}

	// Call release from the first operation
	release()

	// Ensure m["key"] is still 2 and present
	v3, err := lazy.Map(&m, &mu, "key", nil, lazy.DontFetch[string, int]())
	if err != nil {
		t.Fatalf("Value 2 should still be cached: %v", err)
	}
	if v3 != 2 {
		t.Fatalf("Expected 2, got %v", v3)
	}
}
