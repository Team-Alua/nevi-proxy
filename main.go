package main

import (

	"net/http"
	"encoding/json"

	"github.com/gorilla/websocket"

	"github.com/Team-Alua/nevi-proxy/matcher"
	"github.com/Team-Alua/nevi-proxy/clients"
)
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var clientChan chan *clients.Client
var serverChan chan *clients.Server

func clientTracker() {
	match := matcher.New()
	remover := make(chan *clients.Server)
	rematch := make(chan *clients.Server)
	var clientId uint32

	for {
		select {
		case s := <-rematch:
			// Prevent clients from staying in the 
			// queue forever if no more clients/servers
			// connect
			match.MatchServer(s)
		case c := <-clientChan:
			c.Id.Set(clientId)
			clientId += 1
			// Only repeat so we can send notifications
			// Create a dummy read writer
			c.ServerWriter.Set(nil)
			go c.Listen()
			go c.Repeat()
			match.MatchClient(c)
		case s := <-serverChan:
			s.Remover = remover
			s.Rematch = rematch
			// Setup everything so everyone can start talking to each other
			go s.Listen()
			go s.Repeat()
			go s.Ping()
			match.Servers.Add(s)
			match.MatchServer(s)
		case s := <-remover:
			match.Servers.Remove(s)
			s.Close()
		}
	}
}

type SortRequest struct {
	As string `json:"as"`
	Filter string `json:"filter"`
	Limit uint32 `json:"limit,omitempty"`
}

func NeviProxy(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
	
	if err != nil {
		panic(err)
	}

	_, msg, err := conn.ReadMessage()
	if err != nil {
		return
	}

	var sr SortRequest
	if err := json.Unmarshal(msg, &sr); err != nil {
		conn.Close()
		return
	}

	// TODO: Allow clients and servers
	// to specify features constraints
	// TODO: Limit the amount of clients a server can handle
	// TODO: Automatically disconnect if max amount of servers + clients is met
	if sr.As == "server" && sr.Limit > 0 {
		serverChan <- clients.NewServer(conn, sr.Filter, sr.Limit)
	} else if sr.As == "client" {
		clientChan <- clients.NewClient(conn, sr.Filter)
	} else {
		conn.Close()
	}
}


func main() {
	serverChan = make(chan *clients.Server)
	clientChan = make(chan *clients.Client)
	go clientTracker()
	http.HandleFunc("/nevi-proxy", NeviProxy)
	http.ListenAndServe(":8080", nil)
	return
}

