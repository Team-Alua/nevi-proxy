package main

import (
    "github.com/gorilla/websocket"
)

type Server struct {
	Conn *websocket.Conn
	Filter string
	Limit uint64
	Clients []*Client
}


func (s *Server) CompatibleWith(c *Client) bool {
	if s == nil {
		return false
	}

	if s.Filter != c.Filter {
		return false
	}

	if s.Limit <= uint64(len(s.Clients)) {
		return false
	}

	return true
}

