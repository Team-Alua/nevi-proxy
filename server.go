package main

import (
	"encoding/binary"
	"math"
	"sync"

    "github.com/gorilla/websocket"
)

type Server struct {
	Conn *websocket.Conn
	Writer chan *Message
	Filter string
	Limit uint32
	Clients []*Client
	clientLock sync.RWMutex
}

func (s *Server) DecodeBinaryMessage(data []byte) (id uint32, payload []byte) {
	if len(data) < 4 {
		id = math.MaxUint32
		payload = data
		return
	}

	id = binary.BigEndian.Uint32(data[0:4])
	payload = data[4:]
	return
}

// Listen for server messages
func (s *Server) Listen() {
	for {
		msgType, data, err := s.Conn.ReadMessage()
		if err != nil {
			break
		}

		if msgType == websocket.BinaryMessage {
			clientId, payload := s.DecodeBinaryMessage(data)
			var client *Client
			cl := s.clientLock
			cl.RLock()
			if clientId < uint32(len(s.Clients)) {
				client = s.Clients[clientId]	
			}
			cl.RUnlock()
			if client != nil {
				var msg Message
				msg.Type = msgType
				msg.Data = payload
				client.Writer <- &msg
			}
		} else {
			// Ignore for now
		}
	}
}

// Repeat all messages sent by the client
func (s *Server) Repeat() {
	for {
		msg, ok := <-s.Writer
		if !ok {
			break
		}
		s.Conn.WriteMessage(msg.Type, msg.Data)
	}
}


func (s *Server) CompatibleWith(c *Client) bool {
	if s == nil {
		return false
	}

	if s.Filter != c.Filter {
		return false
	}

	if s.Limit < uint32(len(s.Clients)) {
		return false
	}

	for _, client := range s.Clients {
		if client == nil {
			return true
		}
	}

	return true
}


func (s *Server) AddClient(c *Client) (id uint32) {
	cl := s.clientLock
	cl.Lock()
	defer cl.Unlock()
	for idx, client := range s.Clients {
		if client == nil {
			id = uint32(idx)
			s.Clients[idx] = c
			return
		}
	}

	id = uint32(len(s.Clients))
	s.Clients = append(s.Clients, c)
	return
}

func (s *Server) RemoveClient(c *Client) {
	cl := s.clientLock
	cl.Lock()
	defer cl.Unlock()
	for idx, client := range s.Clients {
		if client == c {
			s.Clients[idx] = nil
			return
		}
	}
}

