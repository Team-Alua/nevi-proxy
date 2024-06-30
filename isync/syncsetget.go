package isync

import (
    "sync"
)

type SetGetter[T any] struct {
    lock sync.RWMutex
    data T
}

func NewSetGetter[T any]() *SetGetter[T] {
    return &SetGetter[T]{}
}

func (sg *SetGetter[T]) Set(data T) {
    l := sg.lock
    l.Lock()
    sg.data = data
    l.Unlock()
}

func (sg *SetGetter[T]) Exchange(data T) T {
    l := sg.lock
    l.Lock()
    old := sg.data
    sg.data = data
    l.Unlock()
    return old
}

func (sg *SetGetter[T]) Get() T {
    l := sg.lock
    l.RLock()
    defer l.RUnlock()
    return sg.data
}

