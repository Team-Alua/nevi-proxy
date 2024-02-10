package main

import (
    "github.com/gorilla/websocket"
)


type Matcher struct {
	servers []*Server
	clients []*Client
}


func NewMatcher() *Matcher {
	return &Matcher{servers: make([]*Server,0), clients:make([]*Client,0)}
}

func (m *Matcher) AddClient(c *Client) {
	m.clients = append(m.clients, c)
}

func (m *Matcher) AddServer(s *Server) {
	m.servers = append(m.servers, s)	
}

func (m *Matcher) RemoveServer(s *Server) {
	idx := -1
	for i, server := range m.servers {
		if server == s {
			idx = i
			break
		}
	}

	if idx == -1 {
		return
	}
	
	ret := make([]*Server, len(m.servers) - 1)

	for i, server := range m.servers {
		if i < idx {
			ret[i] = server
		} else if i > idx {
			ret[i - 1] = server
		}
	}

	m.servers = ret
}

func (m *Matcher) MatchClient(c *Client) *Server {
	if c == nil {
		return nil
	}

	for _, s := range m.servers {
		if s.CompatibleWith(c) {
			return s
		}
	}

	return nil
}

func (m *Matcher) MatchClients() {

	clients := make([]*Client, 0)
	for _, c := range m.clients {
		ms := m.MatchClient(c)
		if ms == nil {
			var msg Message
			msg.Type = websocket.TextMessage
			msg.Data = []byte("Cound not find a server")
			c.Writer.Write(&msg)
			clients = append(clients, c)
		} else {
			var msg Message
			msg.Type = websocket.TextMessage
			msg.Data = []byte("Found a server")
			c.Writer.Write(&msg)
			id := ms.AddClient(c)
			c.SetId(id)
			ms.ConnectClient(c)
		}
	}

	m.clients = clients
}

