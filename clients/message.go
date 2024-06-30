package clients

import (
    "github.com/gorilla/websocket"
    "encoding/json"
)

type Message struct {
    Type int
    Data []byte
    Error error
}

func NewTextMessage(data []byte) *Message {
    var msg Message
    msg.Type = websocket.TextMessage
    msg.Data = data
    return &msg
}

func NewBinaryMessage(data []byte) *Message {
    var msg Message
    msg.Type = websocket.BinaryMessage
    msg.Data = data
    return &msg
}

func NewPongMessage() *Message {
    var msg Message
    msg.Type = websocket.PongMessage
    return &msg
}

func NewPingMessage() *Message {
    var msg Message
    msg.Type = websocket.PingMessage
    return &msg
}

func NewJsonMessage(data interface{}) *Message {
    bData, err := json.Marshal(data)
    if err != nil {
        return nil
    }
    return NewTextMessage(bData)
}

