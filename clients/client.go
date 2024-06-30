package clients

import (
	"encoding/binary"
	"time"

    "github.com/gorilla/websocket"

	"github.com/Team-Alua/nevi-proxy/isync"
)

type Client struct {
	Tag uint64
	Id uint64
    Mailer chan<-[]byte
    conn *websocket.Conn
	connected *isync.SetGetter[bool]
	writer *isync.SetGetter[*isync.ReadWriter[*Message]]
}

func NewClient(conn *websocket.Conn) *Client {
	var c Client

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
		c.writer.Get().Close()
		c.conn.Close()	
	}
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
			binary.BigEndian.PutUint64(data, c.Id)
			data = append(data, bData...)
            c.Mailer <- data
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

