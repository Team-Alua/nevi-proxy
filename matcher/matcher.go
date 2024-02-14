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

func (m *Matcher) MatchClient(c *clients.Client) {
	if c == nil {
		return
	}

	if !c.Connected.Get() {
		return
	}

	for _, s := range m.Servers.Clone() {
		if s.CompatibleWith(c) {
			s.ConnectClient(c)
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
		if s.CompatibleWith(c) {
			s.ConnectClient(c)
			m.clients.Remove(c)
		}
	}
}

