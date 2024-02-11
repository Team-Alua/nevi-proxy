package clients

import (
	"encoding/binary"
	"encoding/json"
	"math"
	"time"

    "github.com/gorilla/websocket"

	"github.com/Team-Alua/nevi-proxy/isync"
)

type Server struct {
	Filter string
	Limit uint32
	Remover chan *Server
	Writer *isync.SetGetter[*isync.ReadWriter[*Message]]

	conn *websocket.Conn
	Clients *isync.List[*Client]
}

func NewServer(conn *websocket.Conn, filter string, limit uint32) *Server {
	var s Server
	s.Writer = isync.NewSetGetter[*isync.ReadWriter[*Message]]()
	s.Clients = isync.NewList[*Client]()
	s.Filter = filter
	s.Limit = limit
	s.Writer.Set(isync.NewReadWriter[*Message]())
	s.conn = conn
	return &s
}

func (s *Server) Close() {
	if s == nil {
		return
	}
	clients := s.Clients.Clone()
	for _, client := range clients {
		client.Close()
	}
	s.Clients.Clear()
	s.Writer.Get().Close()
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
			client := s.Clients.Get(uint(clientId))
			if client != nil {
				msg := NewBinaryMessage(payload)
				client.Writer.Get().Write(msg)
			}
		} else if msgType == websocket.TextMessage { 
			// Handle disconnects
			var status Status
			if err := json.Unmarshal(data, &status); err != nil {
				// Ignore
				continue
			}
			client := s.Clients.RemoveByIndex(uint(status.Id))
			client.Close()
			status.Type = "SUCCESS"
			status.Id = 0
			s.Writer.Get().Write(NewJsonMessage(status))
		} else if msgType == websocket.CloseMessage {
			break
		} else if msgType == websocket.PingMessage {
			// Respond back with a pong
			msg := NewPongMessage()
			s.Writer.Get().Write(msg)
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
		msg := s.Writer.Get().Read()
		if msg == nil {
			break
		}

		s.conn.SetWriteDeadline(time.Now().Add(writeWait))
		if err := s.conn.WriteMessage(msg.Type, msg.Data); err != nil {
			break
		}
	}
	s.Remover <- s
}

func (s *Server) ping() bool {
	if s == nil {
		return false
	}

	writer := s.Writer.Get()

	if writer.Closed() {
		return false
	}

	// Ping the server and then ping all the clients
	ping := NewPingMessage()

	writer.Write(ping)

	clients := s.Clients.Clone()
	for idx, client := range clients {
		if !client.Connected.Get() {
			// Write client disconnected
			var status Status
			status.Type = "CLIENT_DISCONNECTED"
			status.Id = uint32(idx)
			writer.Write(NewJsonMessage(status))
			continue
		}
		client.Writer.Get().Write(ping)
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
	clients := s.Clients.Clone()	
	cc := uint32(len(clients))
	if s.Limit < cc  {
		return false
	} else if s.Limit == cc {
		for _, client := range clients {
			if client == nil {
				return true
			}
		}
		return false
	}

	return true
}

func (s *Server) ConnectClient(c *Client, id uint32) {
	if s == nil {
		return
	}
	c.Id.Set(id)
	c.ServerWriter.Set(s.Writer.Get())
	var status Status
	status.Type = "CLIENT_CONNECTED"
	status.Id = id
	s.Writer.Get().Write(NewJsonMessage(status))

	status.Type = "CONNECTED"
	status.Id = id
	c.Writer.Get().Write(NewJsonMessage(status))
}

