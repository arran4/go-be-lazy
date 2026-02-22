package lazy

import (
	"context"
	"time"
)

// Expiry defines a policy for determining if a value has expired.
type Expiry[V any] interface {
	IsExpired(v *Value[V]) bool
}

// ExpireAt returns an Expiry policy that expires the value at the given time.
func ExpireAt[V any](t time.Time) Expiry[V] {
	return &expireAt[V]{t: t}
}

type expireAt[V any] struct {
	t time.Time
}

func (e *expireAt[V]) IsExpired(v *Value[V]) bool {
	if v.IsReleased() {
		return true
	}
	return time.Now().After(e.t)
}

// ExpireAfter returns an Expiry policy that expires the value after the given duration.
func ExpireAfter[V any](d time.Duration) Expiry[V] {
	return &expireAfter[V]{d: d}
}

type expireAfter[V any] struct {
	d time.Duration
}

func (e *expireAfter[V]) IsExpired(v *Value[V]) bool {
	if v.IsReleased() {
		return true
	}
	createdAt := v.CreatedAt()
	if createdAt.IsZero() {
		return false
	}
	return time.Since(createdAt) > e.d
}

// ExpireAfterUses returns an Expiry policy that expires the value after the given number of uses.
func ExpireAfterUses[V any](n int64) Expiry[V] {
	return &expireAfterUses[V]{n: n}
}

type expireAfterUses[V any] struct {
	n int64
}

func (e *expireAfterUses[V]) IsExpired(v *Value[V]) bool {
	if v.IsReleased() {
		return true
	}
	return v.Uses() >= e.n
}

// ExpireAll returns an Expiry policy that expires if ALL of the given policies expire.
func ExpireAll[V any](policies ...Expiry[V]) Expiry[V] {
	return &expireAll[V]{policies: policies}
}

type expireAll[V any] struct {
	policies []Expiry[V]
}

func (e *expireAll[V]) IsExpired(v *Value[V]) bool {
	if v.IsReleased() {
		return true
	}
	if len(e.policies) == 0 {
		return false
	}
	for _, p := range e.policies {
		if !p.IsExpired(v) {
			return false
		}
	}
	return true
}

// ExpireAny returns an Expiry policy that expires if ANY of the given policies expire.
func ExpireAny[V any](policies ...Expiry[V]) Expiry[V] {
	return &expireAny[V]{policies: policies}
}

type expireAny[V any] struct {
	policies []Expiry[V]
}

func (e *expireAny[V]) IsExpired(v *Value[V]) bool {
	if v.IsReleased() {
		return true
	}
	for _, p := range e.policies {
		if p.IsExpired(v) {
			return true
		}
	}
	return false
}

// NeverExpires returns an Expiry policy that never expires.
func NeverExpires[V any]() Expiry[V] {
	return &neverExpires[V]{}
}

type neverExpires[V any] struct{}

func (e *neverExpires[V]) IsExpired(v *Value[V]) bool {
	if v.IsReleased() {
		return true
	}
	return false
}

// ExpireCustom returns an Expiry policy that uses a custom function.
func ExpireCustom[V any](f func(v *Value[V]) bool) Expiry[V] {
	return &expireCustom[V]{f: f}
}

type expireCustom[V any] struct {
	f func(v *Value[V]) bool
}

func (e *expireCustom[V]) IsExpired(v *Value[V]) bool {
	if v.IsReleased() {
		return true
	}
	if e.f == nil {
		return false
	}
	return e.f(v)
}

// ExpireContext returns an Expiry policy that expires when the given context is cancelled or times out.
func ExpireContext[V any](ctx context.Context) Expiry[V] {
	return &expireContext[V]{ctx: ctx}
}

type expireContext[V any] struct {
	ctx context.Context
}

func (e *expireContext[V]) IsExpired(v *Value[V]) bool {
	if v.IsReleased() {
		return true
	}
	return e.ctx.Err() != nil
}
