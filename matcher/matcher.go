package matcher

import (
	"github.com/Team-Alua/nevi-proxy/clients"
)


type Matcher struct {
	servers []*clients.Server
	clients []*clients.Client
}


func New() *Matcher {
	return &Matcher{servers: make([]*clients.Server,0), clients:make([]*clients.Client,0)}
}

func (m *Matcher) AddClient(c *clients.Client) {
	m.clients = append(m.clients, c)
}

func (m *Matcher) AddServer(s *clients.Server) {
	m.servers = append(m.servers, s)	
}

func (m *Matcher) RemoveServer(s *clients.Server) {
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
	
	ret := make([]*clients.Server, len(m.servers) - 1)

	for i, server := range m.servers {
		if i < idx {
			ret[i] = server
		} else if i > idx {
			ret[i - 1] = server
		}
	}

	m.servers = ret
}

func (m *Matcher) matchClient(c *clients.Client) *clients.Server {
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

	clients := make([]*clients.Client, 0)
	for _, c := range m.clients {
		ms := m.matchClient(c)
		if ms == nil {
			clients = append(clients, c)
		} else {
			id := ms.AddClient(c)
			c.Id.Set(id)
			ms.ConnectClient(c)
		}
	}

	m.clients = clients
}

