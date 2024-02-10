package main

import (
	"encoding/binary"
	"encoding/json"
	"math"
	"sync"
	"time"

    "github.com/gorilla/websocket"
)

type Server struct {
	Filter string
	Limit uint32
	Remover chan *Server
	Writer *SyncReadWriter

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

func (s *Server) Close() {
	if s == nil {
		return
	}

	for i, client := range s.clients {
		client.Close()
		s.clients[i] = nil
	}
	s.clients = s.clients[:0]
	s.Writer.Close()
	s.conn.Close()	
}

func (s *Server) decodeBinaryMessage(data []byte) (id uint32, payload []byte) {
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
	if s == nil {
		return
	}
	pongWait := 60 * time.Second
	s.conn.SetPongHandler(func(string) error { s.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		msgType, data, err := s.conn.ReadMessage()
		if err != nil {
			break
		}

		if msgType == websocket.BinaryMessage {
			clientId, payload := s.decodeBinaryMessage(data)
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
		} else if msgType == websocket.TextMessage { 
			// Handle disconnects
			var status Status
			if err := json.Unmarshal(data, &status); err != nil {
				// Ignore
				continue
			}
			client := s.RemoveClientById(status.Id)
			client.Close()
		} else if msgType == websocket.CloseMessage {
			break
		} else if msgType == websocket.PingMessage {
			// Respond back with a pong
			var msg Message
			msg.Type = websocket.PongMessage
			s.Writer.Write(&msg)
		} else {
			// Ignore for now
		}
	}
	s.Remover <- s
}

// Repeat all messages sent by the client
func (s *Server) Repeat() {
	if s == nil {
		return
	}
	writeWait := 10 * time.Second

	for {
		msg := s.Writer.Read()
		if msg == nil {
			break
		}
		
		if msg.Type == websocket.TextMessage {
			// Handle disconnects
			var status Status
			if err := json.Unmarshal(msg.Data, &status); err != nil {
				// Ignore
				continue
			}
			client := s.RemoveClientById(status.Id)
			client.Close()
		} else {
			s.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := s.conn.WriteMessage(msg.Type, msg.Data); err != nil {
				break
			}
		}
	}
	s.Remover <- s
}

func (s *Server) ping() bool {
	if s == nil {
		return false
	}

	if s.Writer.Closed() {
		return false
	}

	// Ping the server and then ping all the clients
	var ping Message

	ping.Type = websocket.PingMessage

	s.Writer.Write(&ping)

	clients := []*Client{}

	cl := s.clientLock
	cl.Lock()
	if len(s.clients) > 0 {
		clients := make([]*Client, len(s.clients))
		copy(clients, s.clients)
	}
	cl.Unlock()

	for _, client := range clients {
		client.Writer.Write(&ping)
	}

	return true
}

func (s *Server) Ping() {
	pingPeriod := 15 * time.Second
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
	}()

	for {
		select {
		case <-ticker.C:
			if s.ping() == false {
				break
			}
		}
	}
}

func (s *Server) CompatibleWith(c *Client) bool {
	if s == nil {
		return false
	}

	if s.Filter != c.Filter {
		return false
	}

	cc := uint32(len(s.clients))
	if s.Limit < cc  {
		return false
	} else if s.Limit == cc {
		for _, client := range s.clients {
			if client == nil {
				return true
			}
		}
		return false
	}

	return true
}


func (s *Server) AddClient(c *Client) uint32 {
	if s == nil {
		return math.MaxUint32
	}

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
		id = uint32(len(s.clients))
		s.clients = append(s.clients, c)
	} else {
		s.clients[id] = c
	}
	cl.Unlock()
	return id
}

func (s *Server) ConnectClient(c *Client) {
	if s == nil {
		return
	}

	c.ServerWriter = s.Writer
	go c.Listen()
}

func (s *Server) RemoveClient(c *Client) {
	if c == nil {
		return
	}

	if s == nil {
		return
	}

	cl := s.clientLock
	cl.Lock()
	for idx, client := range s.clients {
		if client == c {
			s.clients[idx] = nil
			break
		}
	}
	cl.Unlock()
}

func (s *Server) RemoveClientById(id uint32) (client *Client) {
	if s == nil {
		return
	}

	cl := s.clientLock
	cl.Lock()
	if id < uint32(len(s.clients)) {
		client = s.clients[id]
		s.clients[id] = nil
	}
	cl.Unlock()
	return
}

