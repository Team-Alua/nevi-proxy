package main

import (
    "github.com/gorilla/websocket"
)



type Client struct {
	Filter string
	ServerWriter *SyncReadWriter
	Writer *SyncReadWriter

	conn *websocket.Conn
}

func NewClient(conn *websocket.Conn, filter string) *Client {
	var c Client
	c.Writer = NewSyncReadWriter()
	c.Filter = filter
	c.conn = conn
	return &c
}

// Listen for client messages
func (c *Client) Listen() {
	for {
		msgType, data, err := c.conn.ReadMessage()
		if err != nil {
			break
		}

		if msgType == websocket.BinaryMessage {
			var msg Message
			msg.Type = msgType
			msg.Data = data
			// Write to server
			c.ServerWriter.Write(&msg)
		} else {
			// Ignore for now
		}
	}
}

// Repeat all messages sent by the server
func (c *Client) Repeat() {
	for {
		msg := c.Writer.Read()
		if msg == nil {
			break
		}
		c.conn.WriteMessage(msg.Type, msg.Data)
	}
}

