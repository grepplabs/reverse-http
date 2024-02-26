package proxy

import (
	"sync"
)

func NewSyncedMap[K comparable, V any]() *SyncedMap[K, V] {
	return &SyncedMap[K, V]{
		mp: new(sync.Map),
	}
}

type SyncedMap[K comparable, V any] struct {
	mp *sync.Map
}

func (m *SyncedMap[K, V]) Get(key K) (V, bool) {
	v, ok := m.mp.Load(key)
	if ok {
		return v.(V), ok
	}
	var zeroV V
	return zeroV, false
}

func (m *SyncedMap[K, V]) Set(key K, value V) {
	m.mp.Store(key, value)
}

func (m *SyncedMap[K, V]) Swap(key K, value V) (V, bool) {
	v, loaded := m.mp.Swap(key, value)
	if loaded {
		return v.(V), loaded
	}
	var zeroV V
	return zeroV, false
}

func (m *SyncedMap[K, V]) Delete(key K) {
	m.mp.Delete(key)
}

func (m *SyncedMap[K, V]) GetAndDelete(key K) (V, bool) {
	v, ok := m.mp.LoadAndDelete(key)
	if ok {
		return v.(V), ok
	}
	var zeroV V
	return zeroV, false
}

func (m *SyncedMap[K, V]) CompareAndDelete(key K, old V) bool {
	return m.mp.CompareAndDelete(key, old)
}

func (m *SyncedMap[K, V]) CompareAndSwap(key K, old V, new V) bool {
	return m.mp.CompareAndSwap(key, old, new)
}

func (m *SyncedMap[K, V]) Values() []V {
	var vs []V
	m.mp.Range(func(key, value any) bool {
		v, ok := value.(V)
		if ok {
			vs = append(vs, v)
		}
		return true
	})
	return vs
}

func (m *SyncedMap[K, V]) Keys() []K {
	var ks []K
	m.mp.Range(func(key, value any) bool {
		k, ok := key.(K)
		if ok {
			ks = append(ks, k)
		}
		return true
	})
	return ks
}

func (m *SyncedMap[K, V]) Entries() ([]K, []V) {
	var ks []K
	var vs []V
	m.mp.Range(func(key, value any) bool {
		k, ok := key.(K)
		if ok {
			ks = append(ks, k)
		}
		v, ok := value.(V)
		if ok {
			vs = append(vs, v)
		}
		return true
	})
	return ks, vs
}
