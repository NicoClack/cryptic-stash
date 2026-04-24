package tempkeyvalue

import (
	"sync"
	"time"
)

type ValueWithTTL[T any] struct {
	Value     T
	ExpiresAt time.Time
}
type Store[T any] struct {
	values map[string]ValueWithTTL[T]
	mu     sync.RWMutex
}

func NewStore[T any]() *Store[T] {
	return &Store[T]{
		values: make(map[string]ValueWithTTL[T]),
	}
}

func (store *Store[T]) Get(key string, now time.Time) (T, bool) {
	store.mu.RLock()
	valueWithTTL, exists := store.values[key]
	store.mu.RUnlock()
	if !exists {
		var zeroValue T
		return zeroValue, false
	}
	if now.After(valueWithTTL.ExpiresAt) {
		store.Delete(key)
		var zeroValue T
		return zeroValue, false
	}
	return valueWithTTL.Value, true
}
func (store *Store[T]) Set(key string, value T, expiresAt time.Time) {
	store.mu.Lock()
	store.values[key] = ValueWithTTL[T]{
		Value:     value,
		ExpiresAt: expiresAt,
	}
	store.mu.Unlock()
}
func (store *Store[T]) Delete(key string) {
	store.mu.Lock()
	delete(store.values, key)
	store.mu.Unlock()
}

func (store *Store[T]) Prune(now time.Time) {
	store.mu.Lock()
	for key, valueWithTTL := range store.values {
		if now.After(valueWithTTL.ExpiresAt) {
			delete(store.values, key)
		}
	}
	store.mu.Unlock()
}
