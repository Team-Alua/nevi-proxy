package isync

import (
    "sync"
    "golang.org/x/exp/constraints"
)

type Incrementer[T constraints.Integer] struct {
    lock sync.RWMutex
    data T
}

func NewIncrementer[T constraints.Integer](data T) *Incrementer[T] {
    ret := &Incrementer[T]{}
    ret.data = data
    return ret
}

func (i *Incrementer[T]) Increment() T {
	l := i.lock
	l.Lock()
    data := i.data
    i.data += 1
	l.Unlock()
    return data
}

