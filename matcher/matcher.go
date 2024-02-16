package matcher

import (
	"github.com/Team-Alua/nevi-proxy/clients"
	"github.com/Team-Alua/nevi-proxy/isync"
	"github.com/mroth/weightedrand/v2"
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

	if !c.IsConnected() {
		return
	}

	servers := m.Servers.Clone()


	// Check for all servers the client
	// can be serviced by
	cs := make([]*clients.Server, 0)
	for _, s := range servers {
		if s.CompatibleWith(c) {
			cs = append(cs, s)
		}
	}

	if len(cs) == 0 {
		m.clients.Add(c)
		return
	} 

	var server *clients.Server
	
	if len(cs) == 1 {
		server = cs[0]
	} else {
		choices := make([]weightedrand.Choice[*clients.Server, int], 0)

		for _, s := range cs {
			w := s.CountMatchingCaps(c.GetOptionalCaps()) + 1
			c := weightedrand.NewChoice(s, w)
			choices = append(choices, c)
		}

		chooser, _ := weightedrand.NewChooser(choices...)

		server = chooser.Pick()
	}

	server.ConnectClient(c)
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

		if !c.IsConnected() || m.pair(s, c) {
			m.clients.Remove(c)
		}
	}
}

