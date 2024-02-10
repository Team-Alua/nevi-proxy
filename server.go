package main

import (
	"encoding/binary"
	"math"
	"sync"

    "github.com/gorilla/websocket"
)

type Server struct {
	Writer *SyncReadWriter
	Filter string
	Limit uint32

	conn *websocket.Conn
	clientLock sync.RWMutex
	clients []*Client
}

func NewServer(conn *websocket.Conn, filter string, limit uint32) *Server {
	var s Server
	s.Writer = NewSyncReadWriter()
	s.conn = conn
	s.Filter = filter
	s.Limit = limit
	s.clients = make([]*Client, 0)
	return &s
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
		msgType, data, err := s.conn.ReadMessage()
		if err != nil {
			break
		}

		if msgType == websocket.BinaryMessage {
			clientId, payload := s.DecodeBinaryMessage(data)
			var client *Client
			cl := s.clientLock
			cl.RLock()
			if clientId < uint32(len(s.clients)) {
				client = s.clients[clientId]	
			}
			cl.RUnlock()
			if client != nil {
				var msg Message
				msg.Type = msgType
				msg.Data = payload
				client.Writer.Write(&msg)
			}
		} else {
			// Ignore for now
		}
	}
}

// Repeat all messages sent by the client
func (s *Server) Repeat() {
	for {
		msg := s.Writer.Read()
		if msg == nil {
			break
		}
		s.conn.WriteMessage(msg.Type, msg.Data)
	}
}


func (s *Server) CompatibleWith(c *Client) bool {
	if s == nil {
		return false
	}

	if s.Filter != c.Filter {
		return false
	}

	if s.Limit < uint32(len(s.clients)) {
		return false
	}

	for _, client := range s.clients {
		if client == nil {
			return true
		}
	}

	return true
}


func (s *Server) AddClient(c *Client) {
	var id uint32 = math.MaxUint32
	cl := s.clientLock
	cl.Lock()
	for idx, client := range s.clients {
		if client == nil {
			id = uint32(idx)
			break
		}
	}
	if id == math.MaxUint32 {
		s.clients = append(s.clients, c)
	} else {
		s.clients[id] = c
	}
	cl.Unlock()
}

func (s *Server) ConnectClient(c *Client) {
	c.ServerWriter = s.Writer
	go c.Listen()
	go c.Repeat()
}

func (s *Server) RemoveClient(c *Client) {
	cl := s.clientLock
	cl.Lock()
	defer cl.Unlock()
	for idx, client := range s.clients {
		if client == c {
			s.clients[idx] = nil
			return
		}
	}
}

