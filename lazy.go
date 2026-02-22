package lazy

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// result holds the value and error for a lazy Value.
type result[T any] struct {
	value     T
	err       error
	createdAt time.Time
}

var (
	ErrMapPointerNil  = errors.New("lazy map pointer nil")
	ErrMapMutexNil    = errors.New("lazy map mutex nil")
	ErrValueNotCached = errors.New("value not cached")
)

// Value manages a value that is loaded on demand.
// It guarantees that the initialization function is called only once,
// even if accessed concurrently.
// It uses atomic.Value and sync.Mutex for synchronization.
type Value[T any] struct {
	val      atomic.Value
	mu       sync.Mutex
	uses     atomic.Int64
	canceled atomic.Bool
}

// Load ensures the value is loaded by executing fn if it hasn't been loaded yet.
// Subsequent calls return the cached value and error.
// Safe for concurrent use.
func (l *Value[T]) Load(fn func() (T, error)) (T, error) {
	if v := l.val.Load(); v != nil {
		l.uses.Add(1)
		r := v.(*result[T])
		return r.value, r.err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if v := l.val.Load(); v != nil {
		l.uses.Add(1)
		r := v.(*result[T])
		return r.value, r.err
	}
	val, err := fn()
	l.val.Store(&result[T]{value: val, err: err, createdAt: time.Now()})
	l.uses.Add(1)
	return val, err
}

// Set manually sets the value if it hasn't been loaded yet.
// If the value is already loaded (via Load or Set), this operation is a no-op.
// Safe for concurrent use.
func (l *Value[T]) Set(v T) {
	if l.val.Load() != nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.val.Load() != nil {
		return
	}
	l.val.Store(&result[T]{value: v, err: nil, createdAt: time.Now()})
}

// Store forcibly sets the value, bypassing the "once" check.
// This is used internally to overwrite an error state with a default value.
func (l *Value[T]) Store(v T) {
	l.val.Store(&result[T]{value: v, err: nil, createdAt: time.Now()})
}

// Peek returns the cached value and true if it has been loaded.
// If not loaded, it returns the zero value of T and false.
// Safe for concurrent use.
func (l *Value[T]) Peek() (T, bool) {
	if v := l.val.Load(); v != nil {
		l.uses.Add(1)
		r := v.(*result[T])
		return r.value, true
	}
	var zero T
	return zero, false
}

// CreatedAt returns the time when the value was loaded.
// Returns zero time if not loaded.
func (l *Value[T]) CreatedAt() time.Time {
	if v := l.val.Load(); v != nil {
		r := v.(*result[T])
		return r.createdAt
	}
	return time.Time{}
}

// Uses returns the number of times the value has been accessed.
func (l *Value[T]) Uses() int64 {
	return l.uses.Load()
}

// Value returns the cached value, true if loaded, and error if any.
// Unlike Peek or Load, this method does not increment the usage count.
func (l *Value[T]) Value() (T, bool, error) {
	if v := l.val.Load(); v != nil {
		r := v.(*result[T])
		return r.value, true, r.err
	}
	var zero T
	return zero, false, nil
}

// IsLoaded returns true if the value has been loaded.
func (l *Value[T]) IsLoaded() bool {
	return l.val.Load() != nil
}

// Cancel marks the value as canceled.
func (l *Value[T]) Cancel() {
	l.canceled.Store(true)
}

// IsCanceled returns true if the value has been canceled.
func (l *Value[T]) IsCanceled() bool {
	return l.canceled.Load()
}

// args holds the configuration for Map operations.
type args[K comparable, V any] struct {
	dontFetch      bool
	refresh        bool
	clear          bool
	must           bool
	mustCached     bool
	setID          *K
	setValue       *V
	defaultValue   *V
	maxSize        int
	evictionPolicy EvictionPolicy[K, V]
	expiry         Expiry[V]
	cancelDest     *func()
}

// Option configures the behavior of the Map function.
type Option[K comparable, V any] func(*args[K, V])

// DontFetch returns an Option that prevents fetching the value if it's not in the cache.
// If the value is missing, Map will return the zero value (or DefaultValue if set) and no error.
func DontFetch[K comparable, V any]() Option[K, V] { return func(a *args[K, V]) { a.dontFetch = true } }

// Set returns an Option that manually sets the value for the given ID in the map.
// This bypasses the fetch function.
func Set[K comparable, V any](v V) Option[K, V] { return func(a *args[K, V]) { a.setValue = &v } }

// SetID returns an Option that overrides the ID used for the map lookup.
func SetID[K comparable, V any](id K) Option[K, V] { return func(a *args[K, V]) { a.setID = &id } }

// Refresh returns an Option that forces a reload of the value, discarding any cached entry.
func Refresh[K comparable, V any]() Option[K, V] { return func(a *args[K, V]) { a.refresh = true } }

// Clear returns an Option that removes the value associated with the ID from the map.
func Clear[K comparable, V any]() Option[K, V] { return func(a *args[K, V]) { a.clear = true } }

// MustBeCached returns an Option that causes Map to return an error if the value is not already cached.
// Typically used with DontFetch.
func MustBeCached[K comparable, V any]() Option[K, V] {
	return func(a *args[K, V]) { a.mustCached = true }
}

// Must returns an Option that wraps any error returned by the fetch function.
func Must[K comparable, V any]() Option[K, V] { return func(a *args[K, V]) { a.must = true } }

// DefaultValue returns an Option that specifies a fallback value to return if the value is not found
// (when DontFetch is used) or if fetching fails (unless Must is also used).
func DefaultValue[K comparable, V any](v V) Option[K, V] {
	return func(a *args[K, V]) { a.defaultValue = &v }
}

// MaxSize returns an Option that limits the size of the map.
// If the map reaches the specified size, adding a new item will cause an existing item to be evicted.
// The default eviction policy is RandomEvictionPolicy.
func MaxSize[K comparable, V any](size int) Option[K, V] {
	return func(a *args[K, V]) { a.maxSize = size }
}

// WithEvictionPolicy returns an Option that specifies the eviction policy to use when MaxSize is reached.
func WithEvictionPolicy[K comparable, V any](policy EvictionPolicy[K, V]) Option[K, V] {
	return func(a *args[K, V]) { a.evictionPolicy = policy }
}

// WithExpiry returns an Option that specifies the expiration policy for the value.
func WithExpiry[K comparable, V any](policy Expiry[V]) Option[K, V] {
	return func(a *args[K, V]) { a.expiry = policy }
}

// WithCancel returns an Option that provides a function to cancel the value.
// Calling the returned function will purge the entry from the map and mark the value as canceled.
func WithCancel[K comparable, V any](dest *func()) Option[K, V] {
	return func(a *args[K, V]) { a.cancelDest = dest }
}

// Map retrieves or creates a lazy Value in the provided map.
// It handles locking the map using the provided mutex.
//
// Parameters:
//   - m: Pointer to the map caching the values.
//   - mu: Mutex protecting the map.
//   - id: The key to look up in the map.
//   - fetch: Function to generate the value if not found.
//   - opts: Optional modifiers.
//
// Returns the value and any error encountered.
func Map[K comparable, V any](m *map[K]*Value[V], mu *sync.RWMutex, id K, fetch func(K) (V, error), opts ...Option[K, V]) (V, error) {
	var zero V
	args := &args[K, V]{}
	for _, opt := range opts {
		opt(args)
	}
	if args.setID != nil {
		id = *args.setID
	}
	if m == nil {
		return zero, ErrMapPointerNil
	}
	if mu == nil {
		return zero, ErrMapMutexNil
	}

	var lv *Value[V]

	mu.RLock()
	if args.clear {
		mu.RUnlock()
		goto WriteLock
	}
	if *m != nil {
		if val, ok := (*m)[id]; ok && !args.refresh {
			if args.expiry != nil && val.IsLoaded() && args.expiry.IsExpired(val) {
				mu.RUnlock()
				goto WriteLock
			}
			lv = val
			mu.RUnlock()
			goto ProcessValue
		}
	}
	mu.RUnlock()

WriteLock:
	mu.Lock()
	if *m == nil {
		*m = make(map[K]*Value[V])
	}
	if args.clear {
		delete(*m, id)
		mu.Unlock()
		return zero, nil
	}
	if val, ok := (*m)[id]; ok && !args.refresh {
		expired := false
		if args.expiry != nil && val.IsLoaded() && args.expiry.IsExpired(val) {
			expired = true
		}
		if expired {
			delete(*m, id)
			lv = &Value[V]{}
			(*m)[id] = lv
		} else {
			lv = val
		}
	} else {
		if !ok && args.maxSize > 0 && len(*m) >= args.maxSize {
			if args.evictionPolicy != nil {
				victim, found := args.evictionPolicy.SelectVictim(*m)
				if found {
					delete(*m, victim)
				}
			} else {
				// Fallback to random/range if policy is unknown/nil
				for k := range *m {
					delete(*m, k)
					break
				}
			}
		}
		lv = &Value[V]{}
		(*m)[id] = lv
	}
	mu.Unlock()

ProcessValue:
	if args.cancelDest != nil {
		*args.cancelDest = func() {
			if lv != nil {
				lv.Cancel()
			}
			mu.Lock()
			defer mu.Unlock()
			if *m == nil {
				return
			}
			if val, ok := (*m)[id]; ok {
				if val == lv {
					delete(*m, id)
				}
			}
		}
	}

	if args.setValue != nil {
		lv.Set(*args.setValue)
		if args.evictionPolicy != nil {
			args.evictionPolicy.Access(id)
		}
		return *args.setValue, nil
	}

	v, loaded := lv.Peek()
	if loaded {
		if args.evictionPolicy != nil {
			args.evictionPolicy.Access(id)
		}
		return v, nil
	}

	if args.dontFetch {
		if args.mustCached && !loaded {
			return zero, ErrValueNotCached
		}
		if args.defaultValue != nil {
			return *args.defaultValue, nil
		}
		return v, nil
	}

	if fetch == nil {
		return zero, nil
	}

	v, err := lv.Load(func() (V, error) { return fetch(id) })
	if err != nil {
		if args.defaultValue != nil && !args.must {
			lv.Store(*args.defaultValue)
			// Should we consider default value access? Yes.
			if args.evictionPolicy != nil {
				args.evictionPolicy.Access(id)
			}
			return *args.defaultValue, nil
		}
		if args.must {
			return v, fmt.Errorf("fetch error: %w", err)
		}
		return v, err
	}
	// Successful load
	if args.evictionPolicy != nil {
		args.evictionPolicy.Access(id)
	}
	return v, nil
}

// LazyMap manages a collection of lazy values with a built-in mutex.
type LazyMap[K comparable, V any] struct {
	mu   sync.RWMutex
	m    map[K]*Value[V]
	opts []Option[K, V]
}

// NewLazyMap creates a new LazyMap with optional default settings.
func NewLazyMap[K comparable, V any](opts ...Option[K, V]) *LazyMap[K, V] {
	return &LazyMap[K, V]{
		m:    make(map[K]*Value[V]),
		opts: opts,
	}
}

// Get retrieves or creates a value for the given key.
// It wraps the Map function, handling the map and mutex automatically.
// Options passed here are merged with the default options provided to NewLazyMap.
func (lm *LazyMap[K, V]) Get(key K, fetch func(K) (V, error), opts ...Option[K, V]) (V, error) {
	// Combine default options with call-specific options.
	// Call-specific options come last to override defaults.
	combinedOpts := make([]Option[K, V], 0, len(lm.opts)+len(opts))
	combinedOpts = append(combinedOpts, lm.opts...)
	combinedOpts = append(combinedOpts, opts...)
	return Map(&lm.m, &lm.mu, key, fetch, combinedOpts...)
}

// Set manually sets the value for the given key.
func (lm *LazyMap[K, V]) Set(key K, value V) {
	// We use Map with Set option. We also pass global options so policies (like eviction) are respected if Access is triggered.
	// Note: Set option bypasses fetch but triggers policy access if updated in Map logic.
	combinedOpts := make([]Option[K, V], 0, len(lm.opts)+1)
	combinedOpts = append(combinedOpts, lm.opts...)
	combinedOpts = append(combinedOpts, Set[K, V](value))
	_, _ = Map(&lm.m, &lm.mu, key, nil, combinedOpts...)
}

// Remove removes the value associated with the key.
func (lm *LazyMap[K, V]) Remove(key K) {
	combinedOpts := make([]Option[K, V], 0, len(lm.opts)+1)
	combinedOpts = append(combinedOpts, lm.opts...)
	combinedOpts = append(combinedOpts, Clear[K, V]())
	_, _ = Map(&lm.m, &lm.mu, key, nil, combinedOpts...)
}
