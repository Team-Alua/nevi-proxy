package clients

import (
	"encoding/binary"
	"encoding/json"
	"math"
	"time"

    "github.com/gorilla/websocket"

	"github.com/Team-Alua/nevi-proxy/isync"
)

// TODO: Maybe protect id from modification
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

	var msg Message
	msg.Type = websocket.TextMessage
	// Data to signify client disconnected
	var data Status
	data.Type = "CLIENT_DISCONNECT"
	data.Id = c.Id.Get()
	bData, _ := json.Marshal(data)
	msg.Data = bData
	c.ServerWriter.Get().Write(&msg)
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
			var msg Message
			msg.Type = msgType
			msg.Data = make([]byte, 4)
			// Add Client Id
			binary.BigEndian.PutUint32(msg.Data, c.Id.Get())
			msg.Data = append(msg.Data, data...)
			// Write to server
			c.ServerWriter.Get().Write(&msg)
		} else if msgType == websocket.CloseMessage {
			break
		} else if msgType == websocket.PingMessage {
			// Respond back with a pong
			var msg Message
			msg.Type = websocket.PongMessage
			c.Writer.Get().Write(&msg)
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

