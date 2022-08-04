package safemap

import (
	"sync"
)

type SafeMap[K comparable, V any] struct {
	m     map[K]V
	mutex sync.RWMutex
}

func New[K comparable, V any]() *SafeMap[K, V] {
	return &SafeMap[K, V]{
		mutex: sync.RWMutex{},
		m:     make(map[K]V),
	}
}

func (sm *SafeMap[K, V]) Get(key K) (V, bool) {
	sm.mutex.RLock()
	if v, ok := sm.m[key]; ok {
		sm.mutex.RUnlock()
		return v, true
	}
	sm.mutex.RUnlock()
	return *new(V), false
}

func (sm *SafeMap[K, V]) Set(key K, value V) {
	sm.mutex.Lock()
	sm.m[key] = value
	sm.mutex.Unlock()
}

func (sm *SafeMap[K, V]) Len() int {
	return len(sm.m)
}

func (sm *SafeMap[K, V]) Delete(key K) {
	sm.mutex.Lock()
	delete(sm.m, key)
	sm.mutex.Unlock()
}

func (sm *SafeMap[K, V]) Flush() {
	sm.mutex.Lock()
	sm.m = map[K]V{}
	sm.mutex.Unlock()
}
