package main

import (
    "github.com/gorilla/websocket"
)

type Client struct {
	Conn *websocket.Conn
	Filter string
}

