package lazy

import (
	"container/list"
	"sync"
)

// EvictionPolicy defines the strategy for removing items when the map reaches MaxSize.
// Implementations must be thread-safe for Access if they maintain state and are used concurrently.
type EvictionPolicy[K comparable, V any] interface {
	// Access is called when a key is accessed (read or written).
	// This is called outside the map mutex, so implementations must handle concurrency.
	Access(key K)
	// SelectVictim returns the key that should be evicted.
	// It is passed the map in case it needs to inspect it.
	// This is called while the map mutex is held.
	SelectVictim(m map[K]*Value[V]) (K, bool)
}

// RandomEvictionPolicy implements EvictionPolicy using Go's map iteration order.
type RandomEvictionPolicy[K comparable, V any] struct{}

func (p *RandomEvictionPolicy[K, V]) Access(key K) {}

func (p *RandomEvictionPolicy[K, V]) SelectVictim(m map[K]*Value[V]) (K, bool) {
	for k := range m {
		return k, true
	}
	var zero K
	return zero, false
}

// NoEvictionPolicy is a no-op policy.
type NoEvictionPolicy[K comparable, V any] struct{}

func (p *NoEvictionPolicy[K, V]) Access(key K) {}

func (p *NoEvictionPolicy[K, V]) SelectVictim(m map[K]*Value[V]) (K, bool) {
	var zero K
	return zero, false
}

// LRUEvictionPolicy implements Least Recently Used eviction.
type LRUEvictionPolicy[K comparable, V any] struct {
	mu    sync.Mutex
	queue *list.List
	items map[K]*list.Element
}

func NewLRUEvictionPolicy[K comparable, V any]() *LRUEvictionPolicy[K, V] {
	return &LRUEvictionPolicy[K, V]{
		queue: list.New(),
		items: make(map[K]*list.Element),
	}
}

func (p *LRUEvictionPolicy[K, V]) Access(key K) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if elem, ok := p.items[key]; ok {
		p.queue.MoveToFront(elem)
		return
	}
	elem := p.queue.PushFront(key)
	p.items[key] = elem
}

func (p *LRUEvictionPolicy[K, V]) SelectVictim(m map[K]*Value[V]) (K, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for p.queue.Len() > 0 {
		elem := p.queue.Back()
		key := elem.Value.(K)

		// If the key is not in the map (e.g. was deleted externally), skip and remove from our tracking
		if _, ok := m[key]; !ok {
			p.queue.Remove(elem)
			delete(p.items, key)
			continue
		}

		p.queue.Remove(elem)
		delete(p.items, key)
		return key, true
	}

	// Fallback if tracking is empty but map is not (e.g. created without policy initially)
	for k := range m {
		return k, true
	}

	var zero K
	return zero, false
}

// FIFOEvictionPolicy implements First-In-First-Out eviction.
type FIFOEvictionPolicy[K comparable, V any] struct {
	mu    sync.Mutex
	queue *list.List
	items map[K]*list.Element
}

func NewFIFOEvictionPolicy[K comparable, V any]() *FIFOEvictionPolicy[K, V] {
	return &FIFOEvictionPolicy[K, V]{
		queue: list.New(),
		items: make(map[K]*list.Element),
	}
}

func (p *FIFOEvictionPolicy[K, V]) Access(key K) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.items[key]; ok {
		return
	}
	elem := p.queue.PushBack(key)
	p.items[key] = elem
}

func (p *FIFOEvictionPolicy[K, V]) SelectVictim(m map[K]*Value[V]) (K, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for p.queue.Len() > 0 {
		elem := p.queue.Front()
		key := elem.Value.(K)

		if _, ok := m[key]; !ok {
			p.queue.Remove(elem)
			delete(p.items, key)
			continue
		}

		p.queue.Remove(elem)
		delete(p.items, key)
		return key, true
	}

	for k := range m {
		return k, true
	}
	var zero K
	return zero, false
}

// LFUEvictionPolicy implements Least Frequently Used eviction.
type LFUEvictionPolicy[K comparable, V any] struct {
	mu    sync.Mutex
	freqs map[K]int
}

func NewLFUEvictionPolicy[K comparable, V any]() *LFUEvictionPolicy[K, V] {
	return &LFUEvictionPolicy[K, V]{
		freqs: make(map[K]int),
	}
}

func (p *LFUEvictionPolicy[K, V]) Access(key K) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.freqs[key]++
}

func (p *LFUEvictionPolicy[K, V]) SelectVictim(m map[K]*Value[V]) (K, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	var victim K
	minFreq := -1
	found := false

	// Clean up stale entries in freqs while searching
	// Note: Iterating m is O(N).
	for k := range m {
		freq := p.freqs[k]
		if !found || minFreq == -1 || freq < minFreq {
			minFreq = freq
			victim = k
			found = true
		}
	}

	if found {
		delete(p.freqs, victim)
		return victim, true
	}

	// Fallback
	for k := range m {
		return k, true
	}

	var zero K
	return zero, false
}
