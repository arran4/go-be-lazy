package lazy

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestExpireAfter(t *testing.T) {
	var mu sync.RWMutex
	m := make(map[string]*Value[int])

	// Create a value with 100ms expiration
	opts := []Option[string, int]{
		WithExpiry[string, int](ExpireAfter[int](100 * time.Millisecond)),
	}

	fetchCount := 0
	fetch := func(k string) (int, error) {
		fetchCount++
		return fetchCount, nil
	}

	// First access
	v, err := Map(&m, &mu, "key", fetch, opts...)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != 1 {
		t.Errorf("expected 1, got %d", v)
	}

	// Immediate access should be cached
	v, err = Map(&m, &mu, "key", fetch, opts...)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != 1 {
		t.Errorf("expected 1, got %d", v)
	}

	// Wait for expiration
	time.Sleep(200 * time.Millisecond)

	// Access after expiration should reload
	v, err = Map(&m, &mu, "key", fetch, opts...)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != 2 {
		t.Errorf("expected 2, got %d", v)
	}
}

func TestExpireAfterUses(t *testing.T) {
	var mu sync.RWMutex
	m := make(map[string]*Value[int])

	// Expire after 2 uses
	opts := []Option[string, int]{
		WithExpiry[string, int](ExpireAfterUses[int](2)),
	}

	fetchCount := 0
	fetch := func(k string) (int, error) {
		fetchCount++
		return fetchCount, nil
	}

	// Use 1 (Fetch)
	v, err := Map(&m, &mu, "key", fetch, opts...) // fetchCount=1, uses=1
	if v != 1 || err != nil {
		t.Fatalf("Use 1 failed: %v, %v", v, err)
	}

	// Use 2 (Cached)
	v, err = Map(&m, &mu, "key", fetch, opts...) // uses=2
	if v != 1 || err != nil {
		t.Fatalf("Use 2 failed: %v, %v", v, err)
	}

	// Use 3 (Expired -> Fetch)
	// At start of Map: Uses=2. Limit=2. IsExpired -> true.
	// Map removes item. Creates new.
	// Fetch called. fetchCount=2.
	// New item uses=1.
	v, err = Map(&m, &mu, "key", fetch, opts...)
	if v != 2 || err != nil {
		t.Fatalf("Use 3 failed: %v, %v", v, err)
	}
}

func TestExpireAt(t *testing.T) {
	var mu sync.RWMutex
	m := make(map[string]*Value[int])

	expireTime := time.Now().Add(100 * time.Millisecond)
	opts := []Option[string, int]{
		WithExpiry[string, int](ExpireAt[int](expireTime)),
	}

	fetchCount := 0
	fetch := func(k string) (int, error) {
		fetchCount++
		return fetchCount, nil
	}

	if _, err := Map(&m, &mu, "key", fetch, opts...); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := Map(&m, &mu, "key", fetch, opts...); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fetchCount != 1 {
		t.Errorf("expected 1 fetch, got %d", fetchCount)
	}

	time.Sleep(200 * time.Millisecond)
	if _, err := Map(&m, &mu, "key", fetch, opts...); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fetchCount != 2 {
		t.Errorf("expected 2 fetches, got %d", fetchCount)
	}
}

func TestExpireAny(t *testing.T) {
	var mu sync.RWMutex
	m := make(map[string]*Value[int])

	// Expire if uses > 2 OR time > 100ms
	// We will trigger uses first
	opts := []Option[string, int]{
		WithExpiry[string, int](ExpireAny(
			ExpireAfterUses[int](2),
			ExpireAfter[int](1*time.Hour),
		)),
	}

	fetchCount := 0
	fetch := func(k string) (int, error) {
		fetchCount++
		return fetchCount, nil
	}

	if _, err := Map(&m, &mu, "key", fetch, opts...); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := Map(&m, &mu, "key", fetch, opts...); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fetchCount != 1 {
		t.Errorf("expected 1 fetch, got %d", fetchCount)
	}

	if _, err := Map(&m, &mu, "key", fetch, opts...); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fetchCount != 2 {
		t.Errorf("expected 2 fetches, got %d", fetchCount)
	}
}

func TestExpireAll(t *testing.T) {
	var mu sync.RWMutex
	m := make(map[string]*Value[int])

	// Expire if uses >= 2 AND time > 100ms
	opts := []Option[string, int]{
		WithExpiry[string, int](ExpireAll(
			ExpireAfterUses[int](2),
			ExpireAfter[int](100*time.Millisecond),
		)),
	}

	fetchCount := 0
	fetch := func(k string) (int, error) {
		fetchCount++
		return fetchCount, nil
	}

	if _, err := Map(&m, &mu, "key", fetch, opts...); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := Map(&m, &mu, "key", fetch, opts...); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fetchCount != 1 {
		t.Errorf("expected 1 fetch, got %d", fetchCount)
	}

	// Uses condition met, but time not met. Should not expire.
	if _, err := Map(&m, &mu, "key", fetch, opts...); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fetchCount != 1 {
		t.Errorf("expected 1 fetch (not expired), got %d", fetchCount)
	}

	// Wait for time
	time.Sleep(200 * time.Millisecond)

	// Now both met
	if _, err := Map(&m, &mu, "key", fetch, opts...); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fetchCount != 2 {
		t.Errorf("expected 2 fetches, got %d", fetchCount)
	}
}

func TestLazyMapWithExpiry(t *testing.T) {
	lm := NewLazyMap[string, int](
		WithExpiry[string, int](ExpireAfterUses[int](1)),
	)

	count := 0
	fetch := func(k string) (int, error) {
		count++
		return count, nil
	}

	if _, err := lm.Get("a", fetch); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := lm.Get("a", fetch); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 fetches for LazyMap with ExpireAfterUses(1), got %d", count)
	}
}

func TestExpireContext(t *testing.T) {
	var mu sync.RWMutex
	m := make(map[string]*Value[int])

	ctx, cancel := context.WithCancel(context.Background())

	opts := []Option[string, int]{
		WithExpiry[string, int](ExpireContext[int](ctx)),
	}

	fetchCount := 0
	fetch := func(k string) (int, error) {
		fetchCount++
		return fetchCount, nil
	}

	// First access
	v, err := Map(&m, &mu, "key", fetch, opts...)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != 1 {
		t.Errorf("expected 1, got %d", v)
	}

	// Subsequent access (ctx active)
	v, err = Map(&m, &mu, "key", fetch, opts...)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != 1 {
		t.Errorf("expected 1, got %d", v)
	}
	if fetchCount != 1 {
		t.Errorf("expected 1 fetch, got %d", fetchCount)
	}

	// Cancel context
	cancel()

	// Subsequent access (ctx cancelled) -> Should refresh
	v, err = Map(&m, &mu, "key", fetch, opts...)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != 2 {
		t.Errorf("expected 2, got %d", v)
	}
	if fetchCount != 2 {
		t.Errorf("expected 2 fetches, got %d", fetchCount)
	}
}
