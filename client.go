package main

import (
    "github.com/gorilla/websocket"
)

type Client struct {
	Conn *websocket.Conn
	Id uint32
	Writer chan *Message
	ServerWriter chan *Message
	Filter string
}

// Listen for client messages
func (c *Client) Listen() {
	for {
		msgType, data, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}

		if msgType == websocket.BinaryMessage {
			var msg Message
			msg.Type = msgType
			msg.Data = data
			// Write to target client
			c.ServerWriter <- &msg
		} else {
			// Ignore for now
		}
	}
}

// Repeat all messages sent by the server
func (c *Client) Repeat() {
	for {
		msg, ok := <- c.Writer
		if !ok {
			break
		}
		c.Conn.WriteMessage(msg.Type, msg.Data)
	}
}

