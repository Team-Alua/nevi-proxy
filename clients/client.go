package clients

import (
	"encoding/binary"
	"encoding/json"
	"math"
	"time"

    "github.com/gorilla/websocket"

	"github.com/Team-Alua/nevi-proxy/isync"
)

type Client struct {
	Filter string
	Id *isync.SetGetter[uint32]
	ServerWriter *isync.SetGetter[*isync.ReadWriter[*Message]]
	Writer *isync.SetGetter[*isync.ReadWriter[*Message]]

	conn *websocket.Conn
}

func NewClient(conn *websocket.Conn, filter string) *Client {
	var c Client
	c.Id.Set(math.MaxUint32)
	c.Writer.Set(isync.NewReadWriter[*Message]())
	c.Filter = filter
	c.conn = conn
	return &c
}

func (c *Client) Close() {
	if c == nil {
		return
	}

	if c.Id.Get() < math.MaxUint32 {
		c.Id.Set(math.MaxUint32)
		c.ServerWriter.Set(nil)
		c.Writer.Get().Close()
		c.conn.Close()	
	}
}

func (c *Client) NotifyDisconnect() {
	if c == nil {
		return
	}

	// Data to signify client disconnected
	var data Status
	data.Type = "CLIENT_DISCONNECT"
	data.Id = c.Id.Get()
	bData, _ := json.Marshal(data)
	msg := NewTextMessage(bData)
	c.ServerWriter.Get().Write(msg)
}


// Listen for client messages
func (c *Client) Listen() {
	if c == nil {
		return
	}

	pongWait := 60 * time.Second
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		msgType, data, err := c.conn.ReadMessage()
		if err != nil {
			break
		}

		if msgType == websocket.BinaryMessage {	
			bData := data
			data := make([]byte, 4)
			// Add Client Id
			binary.BigEndian.PutUint32(data, c.Id.Get())
			data = append(data, bData...)
			msg := NewBinaryMessage(data)
			c.ServerWriter.Get().Write(msg)
		} else if msgType == websocket.CloseMessage {
			break
		} else if msgType == websocket.PingMessage {
			// Respond back with a pong
			msg := NewPongMessage()
			c.Writer.Get().Write(msg)
		} else {
			// Ignore for now
		}
	}
	c.NotifyDisconnect()
}

// Repeat all messages sent by the server
func (c *Client) Repeat() {
	if c == nil {
		return
	}

	writeWait := 10 * time.Second

	for {
		msg := c.Writer.Get().Read()
		if msg == nil {
			break
		}
		c.conn.SetWriteDeadline(time.Now().Add(writeWait))
		if err := c.conn.WriteMessage(msg.Type, msg.Data); err != nil {
			panic(err)
			break
		}
	}
	c.NotifyDisconnect()
}

