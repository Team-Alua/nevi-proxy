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
	Rematch chan *Server
	Remover chan *Server
	Writer *isync.SetGetter[*isync.ReadWriter[*Message]]

	conn *websocket.Conn
	clients *isync.Map[uint32, *Client]
}

func NewServer(conn *websocket.Conn, filter string, limit uint32) *Server {
	var s Server
	s.Writer = isync.NewSetGetter[*isync.ReadWriter[*Message]]()
	s.Filter = filter
	s.Limit = limit
	s.Writer.Set(isync.NewReadWriter[*Message]())
	s.conn = conn
	s.clients = isync.NewMap[uint32, *Client]()
	return &s
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
			client := s.clients.Get(clientId)
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
			// Have it disconnect and then
			// wait for the next round of pings to 
			// accept new clients
			client := s.clients.Get(status.Id)
			client.Close()
			s.Rematch <- s
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

	clients := s.clients.Clone()
	for _, client := range clients {
		if client.Connected.Get() {
			client.Writer.Get().Write(ping)
		}
	}

	return true
}

func (s *Server) checkOnClients() {
	writer := s.Writer.Get()

	if writer.Closed() {
		return
	}

	clients := s.clients.Clone()
	for idx, client := range clients {
		if !client.Connected.Get() {
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
	pingPeriod := 15 * time.Second
	t1 := time.NewTicker(pingPeriod)

	clientCheckPeriod := 500 * time.Millisecond
	t2 := time.NewTicker(clientCheckPeriod)
	defer func() {
		t1.Stop()
		t2.Stop()
	}()
	

	for {
		select {
		case <-t1.C:
			if s.ping() == false {
				break
			}
		case <-t2.C:
			s.checkOnClients()
		}
	}
}


func (s *Server) IsAvailable() bool {
	return uint32(s.clients.Count()) < s.Limit 
}

func (s *Server) CompatibleWith(c *Client) bool {
	if s == nil {
		return false
	}

	if s.Filter != c.Filter {
		return false
	}

	if !s.IsAvailable() {
		return false
	}

	return true
}

func (s *Server) ConnectClient(c *Client) {
	if s == nil {
		return
	}
	cId := c.Id.Get()
	c.ServerWriter.Set(s.Writer.Get())
	s.clients.Set(cId, c)

	var status Status
	status.Type = "CLIENT_CONNECTED"
	status.Id = cId
	s.Writer.Get().Write(NewJsonMessage(status))

	status.Type = "CONNECTED"
	status.Id = cId
	c.Writer.Get().Write(NewJsonMessage(status))
}

