package main

import (
	"encoding/binary"
	"encoding/json"
	"math"
	"sync"
	"time"

    "github.com/gorilla/websocket"
)

// TODO: Maybe protect id from modification
type Client struct {
	Filter string
	ServerWriter *SyncReadWriter
	Writer *SyncReadWriter

	conn *websocket.Conn
	id uint32
	idLock sync.RWMutex
}

func NewClient(conn *websocket.Conn, filter string) *Client {
	var c Client
	c.SetId(math.MaxUint32)
	c.Writer = NewSyncReadWriter()
	c.Filter = filter
	c.conn = conn
	return &c
}

func (c *Client) Close() {
	if c == nil {
		return
	}

	if c.GetId() < math.MaxUint32 {
		c.SetId(math.MaxUint32)
		c.ServerWriter = nil
		c.Writer.Close()
		c.conn.Close()	
	}
}

func (c *Client) SetId(id uint32) {
	if c == nil {
		return
	}

	c.idLock.Lock()
	c.id = id
	c.idLock.Unlock()
}

func (c *Client) GetId() uint32 {
	if c == nil {
		return math.MaxUint32
	}

	c.idLock.RLock()
	defer c.idLock.RUnlock()
	return c.id
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
	data.Id = c.GetId()
	bData, _ := json.Marshal(data)
	msg.Data = bData
	c.ServerWriter.Write(&msg)
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
			binary.BigEndian.PutUint32(msg.Data, c.GetId())
			msg.Data = append(msg.Data, data...)
			// Write to server
			c.ServerWriter.Write(&msg)
		} else if msgType == websocket.CloseMessage {
			break
		} else if msgType == websocket.PingMessage {
			// Respond back with a pong
			var msg Message
			msg.Type = websocket.PongMessage
			c.Writer.Write(&msg)
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
		msg := c.Writer.Read()
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

