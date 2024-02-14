package matcher

import (
	"github.com/Team-Alua/nevi-proxy/clients"
	"github.com/Team-Alua/nevi-proxy/isync"
)


type Matcher struct {
	Servers *isync.List[*clients.Server]
	clients *isync.List[*clients.Client]
}


func New() *Matcher {
	servers := isync.NewList[*clients.Server]()
	clients := isync.NewList[*clients.Client]()
	return &Matcher{Servers: servers, clients: clients}
}


func (m *Matcher) pair(s *clients.Server, c *clients.Client) bool {
	pairable := s.CompatibleWith(c)
	if pairable {
		s.ConnectClient(c)
	}
	return pairable
}

func (m *Matcher) MatchClient(c *clients.Client) {
	if c == nil {
		return
	}

	if !c.Connected.Get() {
		return
	}

	for _, s := range m.Servers.Clone() {
		if m.pair(s, c) {
			return
		}
	}

	m.clients.Add(c)
}

func (m *Matcher) MatchServer(s *clients.Server) {
	if s == nil {
		return
	}

	for _, c := range m.clients.Clone() {
		// Iterate through clients
		// until no more can be paired
		if !s.IsAvailable() {
			break
		}

		if !c.Connected.Get() || m.pair(s, c) {
			m.clients.Remove(c)
		}
	}
}

