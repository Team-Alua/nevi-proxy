package main

import (
	"sync"
)

type SyncReadWriter struct {
	closed bool
	closedMutex sync.RWMutex
	writer chan *Message
}

func NewSyncReadWriter() *SyncReadWriter {
	return &SyncReadWriter{closed: false, writer: make(chan *Message)}
}

func (s *SyncReadWriter) Write(msg *Message) {
	if s == nil {
		return
	}

	s.closedMutex.RLock()
	closed := s.closed	
	s.closedMutex.RUnlock()
	if closed {
		// ignore 
		return
	}

	s.writer <- msg
}

func (s *SyncReadWriter) Close() {
	if s == nil {
		return
	}

	s.closedMutex.Lock()
	closed := s.closed
	s.closed = true
	s.closedMutex.Unlock()

	if !closed {
		close(s.writer)
	}

}

func (s *SyncReadWriter) Closed() bool {
	if s == nil {
		return true
	}

	s.closedMutex.RLock()
	closed := s.closed	
	s.closedMutex.RUnlock()
	return closed
}

func (s *SyncReadWriter) Read() *Message {
	if s == nil {
		return nil
	}

	s.closedMutex.RLock()
	closed := s.closed	
	s.closedMutex.RUnlock()
	if closed {
		return nil
	}
	data, ok := <-s.writer
	if !ok {
		return nil
	}

	return data
}
