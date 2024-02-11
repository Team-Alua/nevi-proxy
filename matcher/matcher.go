package matcher

import (
	"github.com/Team-Alua/nevi-proxy/clients"
	"github.com/Team-Alua/nevi-proxy/isync"
)


type Matcher struct {
	Servers *isync.List[*clients.Server]
	Clients *isync.List[*clients.Client]
}


func New() *Matcher {
	servers := isync.NewList[*clients.Server]()
	clients := isync.NewList[*clients.Client]()
	return &Matcher{Servers: servers, Clients: clients}
}

func (m *Matcher) matchClient(c *clients.Client) *clients.Server {
	if c == nil {
		return nil
	}

	for _, s := range m.Servers.Clone() {
		if s.CompatibleWith(c) {
			return s
		}
	}

	return nil
}

func (m *Matcher) MatchClients() {

	clients := make([]*clients.Client, 0)
	for _, c := range m.Clients.Clone() {
		if c == nil {
			continue
		}

		// Client disconnected before it was assigned
		// a server
		if c.Connected.Get() == false {
			continue
		}

		ms := m.matchClient(c)
		if ms == nil {
			clients = append(clients, c)
		} else {
			id := ms.Clients.Add(c)
			ms.ConnectClient(c, uint32(id))
		}
	}

	m.Clients.Set(clients)
}

