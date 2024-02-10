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

	s.closedMutex.Lock()
	closed := s.closed
	s.closed = true
	s.closedMutex.Unlock()

	if !closed {
		close(s.writer)
	}

}

func (s *SyncReadWriter) Read() *Message {
	s.closedMutex.RLock()
	closed := s.closed	
	s.closedMutex.RUnlock()
	if closed {
		return nil
	}
	return <-s.writer
}
