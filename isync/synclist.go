package isync

import (
    "sync"
)

type List[T comparable] struct {
    lock  sync.RWMutex
    data []T
}

func NewList[T comparable]() *List[T] {
    var l List[T]
    l.data = make([]T, 0)
    return &l
}

func (l *List[T]) Set(data []T) {
    l.lock.Lock()
    l.data = data
    l.lock.Unlock()
}

func (l *List[T]) Add(element T) uint {
    l.lock.Lock()
    count := uint(len(l.data))
    l.data = append(l.data, element)
    l.lock.Unlock()
    return count
}

func (l *List[T]) Remove(element T) uint {
    l.lock.Lock()
    data := make([]T, 0)
    for _, item := range l.data {
        if item != element {
            data = append(data, item)
        }
    }
    l.data = data
    l.lock.Unlock()
    return 0
}

func (l *List[T]) RemoveByIndex(index uint) T {
    var target T
    l.lock.Lock()
    if index < uint(len(l.data)) {
        // Get the item at the index
        target = l.data[index]

        // Exclude it
        data := make([]T, 0)
        for i, item := range l.data {
            if uint(i) == index {
                continue
            }
            data = append(data, item)
        }
        l.data = data
    }
    l.lock.Unlock()
    return target
}

func (l *List[T]) Get(index uint) T {
    var target T
    l.lock.RLock()
    target = l.data[index]
    l.lock.RUnlock()
    return target
}

func (l *List[T]) Clone() []T {
    clone := make([]T, 0)
    l.lock.RLock()
    for _, item := range l.data {
        clone = append(clone, item)
    }
    l.lock.RUnlock()
    return clone
}

func (l *List[T]) Clear() {
    l.lock.Lock()
    l.data = make([]T, 0)
    l.lock.Unlock()
}

