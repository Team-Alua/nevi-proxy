package clients

import (
	"encoding/binary"
	"encoding/json"
	"math"
	"time"

    "github.com/gorilla/websocket"

	"github.com/Team-Alua/nevi-proxy/isync"
)

type ServerRequest struct {
	Filter string `json:"filter"`
	Limit uint32 `json:"limit,omitempty"`
	Caps []string `json:"caps,omitempty"`
}

type Server struct {
	Filter string
	Limit uint32
	Rematch chan *Server
	Remover chan *Server

	caps []string
	clients *isync.Map[uint64, *Client]
	conn *websocket.Conn
	writer *isync.SetGetter[*isync.ReadWriter[*Message]]
}

func NewServer(conn *websocket.Conn, req *ServerRequest) *Server {
	var s Server
	s.Filter = req.Filter
	s.Limit = req.Limit

	s.caps = req.Caps
	s.clients = isync.NewMap[uint64, *Client]()
	s.conn = conn
	s.writer = isync.NewSetGetter[*isync.ReadWriter[*Message]]()
	s.writer.Set(isync.NewReadWriter[*Message]())

	return &s
}

func (s *Server) IsConnected() bool {
	if s == nil {
		return false
	}
	return !s.writer.Get().Closed()
}

func (s *Server) GetWriter() *isync.ReadWriter[*Message] {
	if s == nil {
		return nil
	}

	return s.writer.Get()
}


func (s *Server) Close() {
	if s == nil {
		return
	}

	clients := s.clients.Clone()
	for _, client := range clients {
		client.Close()
	}
	s.clients.Clear()
	s.writer.Get().Close()
	s.conn.Close()	
}

func (s *Server) decodeBinaryMessage(data []byte) (id uint64, payload []byte) {
	if len(data) < 8 {
		id = math.MaxUint64
		payload = data
		return
	}

	id = binary.BigEndian.Uint64(data[0:8])
	payload = data[8:]
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
		// Wait forever
		s.conn.SetReadDeadline(time.Time{})
		msgType, data, err := s.conn.ReadMessage()
		if err != nil {
			break
		}

		if msgType == websocket.BinaryMessage {
			clientId, payload := s.decodeBinaryMessage(data)
			client := s.clients.Get(clientId)
			if client != nil {
				msg := NewBinaryMessage(payload)
				client.GetWriter().Write(msg)
			}
		} else if msgType == websocket.TextMessage { 
			// Handle disconnects
			var status Status
			if err := json.Unmarshal(data, &status); err != nil {
				// Ignore
				continue
			}
			// Have it disconnect and then
			// wait for the next round of pings to 
			// accept new clients
			client := s.clients.Get(status.Id)
			client.Close()
			s.Rematch <- s
			status.Type = "SUCCESS"
			status.Id = 0
			s.writer.Get().Write(NewJsonMessage(status))
		} else if msgType == websocket.CloseMessage {
			break
		} else if msgType == websocket.PingMessage {
			// Respond back with a pong
			msg := NewPongMessage()
			s.writer.Get().Write(msg)
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
		msg := s.writer.Get().Read()
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

func (s *Server) checkOnClients() {
	writer := s.writer.Get()

	clients := s.clients.Clone()
	for idx, client := range clients {
		if !client.IsConnected() {
			// Remove the client from our array
			s.clients.Delete(idx)
			// Write client disconnected
			var status Status
			status.Type = "CLIENT_DISCONNECTED"
			status.Id = idx
			writer.Write(NewJsonMessage(status))
		}
	}
}

func (s *Server) Ping() {
	if s == nil {
		return
	}

	clientCheckPeriod := 500 * time.Millisecond
	t2 := time.NewTicker(clientCheckPeriod)
	defer func() {
		t2.Stop()
	}()
	

	for s.IsConnected() {
		<-t2.C
		s.checkOnClients()
	}
}


func (s *Server) IsAvailable() bool {
	return uint32(s.clients.Count()) < s.Limit 
}

func (s *Server) hasCap(capability string) bool {
	for _, c := range s.caps {
		if c == capability {
			return true
		}	
	}
	return false
}


func (s *Server) CountMatchingCaps(caps []string) (count int) {
	for _, c := range caps {
		if s.hasCap(c) {
			count += 1
		}
	}
	return
}

func (s *Server) CompatibleWith(c *Client) bool {
	if s == nil {
		return false
	}

	if !s.IsAvailable() {
		return false
	}

	if s.Filter != c.Filter {
		return false
	}

	rc := c.GetRequiredCaps()
	if s.CountMatchingCaps(rc) < len(rc) {
		return false
	}

	return true
}

func (s *Server) ConnectClient(c *Client) {
	if s == nil {
		return
	}
	cId := c.Id.Get()
	c.Server.Set(s)
	s.clients.Set(cId, c)

	var status Status
	status.Type = "CLIENT_CONNECTED"
	status.Id = cId
	s.writer.Get().Write(NewJsonMessage(status))

	status.Type = "CONNECTED"
	status.Id = cId
	c.GetWriter().Write(NewJsonMessage(status))
}

