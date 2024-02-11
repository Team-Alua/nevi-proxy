package isync

type ReadWriter[T any] struct {
	closed *SetGetter[bool]
	writer chan interface{}
}

func NewReadWriter[T any]() *ReadWriter[T] {
	return &ReadWriter[T]{closed: NewSetGetter[bool](), writer: make(chan interface{})}
}

func (rw *ReadWriter[T]) Write(data T) {
	if rw == nil {
		return
	}

	if rw.closed.Get() {
		// ignore 
		return
	}

	rw.writer <- data
}

func (rw *ReadWriter[_]) Close() {
	if rw == nil {
		return
	}

	closed := rw.closed.Exchange(true)

	if !closed {
		close(rw.writer)
	}
}

func (rw *ReadWriter[_]) Closed() bool {
	if rw == nil {
		return true
	}

	return rw.closed.Get()
}

func (rw *ReadWriter[T]) Read() T {
	var zero T
	if rw == nil {
		return zero
	}

	if rw.closed.Get() {
		return zero
	}

	data, ok := <-rw.writer
	if !ok {
		return zero
	}

	return data.(T)
}

