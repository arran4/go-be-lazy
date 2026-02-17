package lazy_test

import (
	"errors"
	"sync"
	"testing"

	lazy "github.com/arran4/go-be-lazy"
)

func TestMapErrorsIs(t *testing.T) {
	t.Run("MapPointerNil", func(t *testing.T) {
		var mu sync.RWMutex
		_, err := lazy.Map[int, int](nil, &mu, 1, nil)
		if !errors.Is(err, lazy.ErrMapPointerNil) {
			t.Errorf("expected ErrMapPointerNil, got %v", err)
		}
	})

	t.Run("MapMutexNil", func(t *testing.T) {
		m := make(map[int32]*lazy.Value[int])
		_, err := lazy.Map[int32, int](&m, nil, 1, nil)
		if !errors.Is(err, lazy.ErrMapMutexNil) {
			t.Errorf("expected ErrMapMutexNil, got %v", err)
		}
	})

	t.Run("ValueNotCached", func(t *testing.T) {
		m := make(map[int32]*lazy.Value[int])
		var mu sync.RWMutex
		_, err := lazy.Map(&m, &mu, 1, nil, lazy.DontFetch[int32, int](), lazy.MustBeCached[int32, int]())
		if !errors.Is(err, lazy.ErrValueNotCached) {
			t.Errorf("expected ErrValueNotCached, got %v", err)
		}
	})
}
