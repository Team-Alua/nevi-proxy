package clients

import (
	"encoding/binary"
	"math"
	"time"

    "github.com/gorilla/websocket"

	"github.com/Team-Alua/nevi-proxy/isync"
)
type ClientCapabilityRequest struct {
	Required []string `json:"required,omitempty"`
	Optional []string `json:"optional,omitempty"`
}

type ClientRequest struct {
	Filter string `json:"filter"`
	Caps ClientCapabilityRequest `json:"caps"`
}

type Client struct {
	Filter string
	Id *isync.SetGetter[uint64]
	Server *isync.SetGetter[*Server]

	caps ClientCapabilityRequest
	conn *websocket.Conn
	connected *isync.SetGetter[bool]
	writer *isync.SetGetter[*isync.ReadWriter[*Message]]
}

func NewClient(conn *websocket.Conn, req *ClientRequest) *Client {
	var c Client

	c.Filter = req.Filter
	c.Id = isync.NewSetGetter[uint64]()
	c.Id.Set(math.MaxUint64)
	c.Server = isync.NewSetGetter[*Server]()

	c.caps = req.Caps
	c.conn = conn
	c.connected = isync.NewSetGetter[bool]()
	c.connected.Set(true)
	c.writer = isync.NewSetGetter[*isync.ReadWriter[*Message]]()
	c.writer.Set(isync.NewReadWriter[*Message]())

	return &c
}


func (c *Client) IsConnected() bool {
	if c == nil {
		return false
	}

	return c.connected.Get()
}

func (c *Client) GetWriter() *isync.ReadWriter[*Message] {
	if c == nil {
		return nil
	}
	return c.writer.Get()
}

func (c *Client) Close() {
	if c == nil {
		return
	}

	if c.connected.Exchange(false) {
		c.Id.Set(math.MaxUint32)
		c.Server.Set(nil)
		c.writer.Get().Close()
		c.conn.Close()	
	}
}

func (c *Client) GetRequiredCaps() []string {
	return c.caps.Required
}

func (c *Client) GetOptionalCaps() []string {
	return c.caps.Optional
}


// Listen for client messages
func (c *Client) Listen() {
	if c == nil {
		return
	}

	pongWait := 60 * time.Second
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		// Reset to wait forever
		c.conn.SetReadDeadline(time.Time{})
		msgType, data, err := c.conn.ReadMessage()
		if err != nil {
			break
		}

		if msgType == websocket.BinaryMessage {	
			bData := data
			data := make([]byte, 8)
			// Add Client Id
			binary.BigEndian.PutUint64(data, c.Id.Get())
			data = append(data, bData...)
			msg := NewBinaryMessage(data)
			s := c.Server.Get()
			s.GetWriter().Write(msg)
		} else if msgType == websocket.CloseMessage {
			break
		} else if msgType == websocket.PingMessage {
			// Respond back with a pong
			msg := NewPongMessage()
			c.writer.Get().Write(msg)
		} else {
			// Ignore for now
		}
	}
	c.Close()
}

// Repeat all messages sent by the server
func (c *Client) Repeat() {
	if c == nil {
		return
	}

	writeWait := 10 * time.Second

	for {
		msg := c.writer.Get().Read()
		if msg == nil {
			break
		}
		c.conn.SetWriteDeadline(time.Now().Add(writeWait))
		if err := c.conn.WriteMessage(msg.Type, msg.Data); err != nil {
			break
		}
	}
	c.Close()
}

