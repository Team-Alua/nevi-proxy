package isync

import (
    "sync"
    "maps"
)

type Map[K comparable, V any] struct {
    lock sync.RWMutex
    data map[K]V
}

func NewMap[K comparable, V any]() *Map[K,V] {
    var ret Map[K,V]
    ret.data = make(map[K]V)
    return &ret
}

func (m *Map[K,V]) Clone() map[K]V {
    l := m.lock

    l.Lock()
    ret := maps.Clone(m.data)
    l.Unlock()
    return ret
}

func (m *Map[K,V]) Set(k K, v V) {
    l := m.lock

    l.Lock()
    m.data[k] = v
    l.Unlock()
}

func (m *Map[K, V]) Get(k K) V {
    l := m.lock

    l.RLock()
    ret := m.data[k]
    l.RUnlock()
    return ret
}

func (m *Map[K, V]) Delete(k K) {
    l := m.lock

    l.Lock()
    delete(m.data, k)
    l.Unlock()
}

func (m *Map[K, V]) Clear() {
    l := m.lock
    l.Lock()
    clear(m.data)
    l.Unlock()
}

func (m *Map[K, V]) Count() int {
    l := m.lock

    l.RLock()
    ret := len(m.data)
    l.RUnlock()
    return ret

}

