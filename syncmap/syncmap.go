// VERY slim wrapper around sync.Map with Go generics for that sweet sweet type safety
package syncmap

import (
	"iter"
	"sync"
)

type Map[K comparable, V any] struct {
	InnerMap     sync.Map
	DefaultValue V
}

func FromRegularMap[K comparable, V any](m map[K]V) *Map[K, V] {
	syncMap := Map[K, V]{}
	for key, value := range m {
		syncMap.Store(key, value)
	}
	return &syncMap
}

func (m *Map[K, V]) Clear() {
	m.InnerMap.Clear()
}

func (m *Map[K, V]) Delete(key K) {
	m.InnerMap.Delete(key)
}

func (m *Map[K, V]) Load(key K) (value V, loaded bool) {
	v, loaded := m.InnerMap.Load(key)
	value, ok := v.(V)
	if !ok {
		return m.DefaultValue, loaded
	}
	return
}

func (m *Map[K, V]) LoadAndDelete(key K) (value V, loaded bool) {
	v, loaded := m.InnerMap.LoadAndDelete(key)
	value, ok := v.(V)
	if !ok {
		return m.DefaultValue, loaded
	}
	return
}

func (m *Map[K, V]) LoadOrStore(key K, value V) (actual V, loaded bool) {
	v, loaded := m.InnerMap.LoadOrStore(key, value)
	actual, ok := v.(V)
	if !ok {
		return m.DefaultValue, loaded
	}
	return
}

func (m *Map[K, V]) Range(f func(key K, value V) bool) {
	m.InnerMap.Range(func(key, value any) bool {
		return f(key.(K), value.(V))
	})
}

func (m *Map[K, V]) Items() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		m.Range(func(key K, value V) bool {
			return yield(key, value)
		})
	}
}

func (m *Map[K, V]) Store(key K, value V) {
	m.InnerMap.Store(key, value)
}

func (m *Map[K, V]) Swap(key K, value V) (previous V, loaded bool) {
	v, loaded := m.InnerMap.Swap(key, value)
	previous, ok := v.(V)
	if !ok {
		return m.DefaultValue, loaded
	}
	return
}

func (m *Map[K, V]) ToRegularMap() map[K]V {
	newMap := map[K]V{}
	m.Range(func(key K, value V) bool {
		newMap[key] = value
		return true
	})
	return newMap
}

func (m *Map[K, V]) Length() int {
	l := 0
	m.Range(func(key K, value V) bool {
		l += 1
		return true
	})
	return l
}

type ComparableMap[K comparable, V comparable] struct {
	Map[K, V]
}

func ComparableFromRegularMap[K comparable, V comparable](m map[K]V) *ComparableMap[K, V] {
	syncMap := ComparableMap[K, V]{}
	for key, value := range m {
		syncMap.Store(key, value)
	}
	return &syncMap
}

func (m *ComparableMap[K, V]) CompareAndDelete(key K, old V) bool {
	return m.InnerMap.CompareAndDelete(key, old)
}

func (m *ComparableMap[K, V]) CompareAndSwap(key K, old V, new V) bool {
	return m.InnerMap.CompareAndSwap(key, old, new)
}
